package app

import (
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/questionbank"
	"Q-Solver/pkg/solution"
	"context"
	"fmt"
	"strings"
	"time"
)

const MaxScreenshots = 3

func (a *App) TriggerScreenshot() {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return
	}

	if cfg.Model == "" {
		a.EmitEvent("toast", "请先选择模型")
		a.EmitEvent("open-settings", "model")
		return
	}

	if a.taskManager.HasRunningTask() {
		logger.Println("忽略截图：当前有任务正在运行")
		a.EmitEvent("toast", "正在处理中，请稍候...")
		return
	}

	if len(a.screenshotBuffer) >= MaxScreenshots {
		a.EmitEvent("toast", "最多截图 3 张图片，请先发送或删除")
		return
	}

	previewResult, err := a.GetScreenshotPreview(
		cfg.CompressionQuality,
		cfg.Sharpening,
		cfg.Grayscale,
		cfg.NoCompression,
		cfg.ScreenshotMode,
	)
	if err != nil {
		logger.Printf("截图失败: %v\n", err)
		a.EmitEvent("toast", "截图失败: "+err.Error())
		return
	}

	a.screenshotBuffer = append(a.screenshotBuffer, previewResult.Base64)
	a.EmitEvent("screenshot-taken", previewResult.Base64, len(a.screenshotBuffer))
}

func (a *App) RemoveScreenshot(index int) {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	if index < 0 || index >= len(a.screenshotBuffer) {
		return
	}
	a.screenshotBuffer = append(a.screenshotBuffer[:index], a.screenshotBuffer[index+1:]...)
	a.EmitEvent("screenshot-removed", index, len(a.screenshotBuffer))
}

func (a *App) RemoveLastScreenshot() {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	if len(a.screenshotBuffer) == 0 {
		return
	}
	index := len(a.screenshotBuffer) - 1
	a.screenshotBuffer = a.screenshotBuffer[:index]
	a.EmitEvent("screenshot-removed", index, len(a.screenshotBuffer))
}

func (a *App) ClearScreenshots() {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	a.screenshotBuffer = nil
	a.EmitEvent("screenshots-cleared")
}

func needsResumeConfirmation(resumePath, resumeContent string) bool {
	return strings.TrimSpace(resumePath) != "" && strings.TrimSpace(resumeContent) == ""
}

func (a *App) TriggerSend() {
	a.triggerSend(false)
}

func (a *App) ConfirmSolveWithoutResume() {
	a.triggerSend(true)
}

func (a *App) triggerSend(skipResumeWarning bool) {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return
	}

	if cfg.Model == "" {
		a.EmitEvent("toast", "请先选择模型")
		a.EmitEvent("open-settings", "model")
		return
	}

	if !skipResumeWarning && needsResumeConfirmation(cfg.ResumePath, cfg.ResumeContent) {
		a.EmitEvent("resume-warning-required")
		return
	}

	if len(a.screenshotBuffer) == 0 {
		previewResult, err := a.GetScreenshotPreview(
			cfg.CompressionQuality,
			cfg.Sharpening,
			cfg.Grayscale,
			cfg.NoCompression,
			cfg.ScreenshotMode,
		)
		if err != nil {
			logger.Printf("截图失败: %v\n", err)
			a.EmitEvent("toast", "截图失败: "+err.Error())
			return
		}
		a.screenshotBuffer = append(a.screenshotBuffer, previewResult.Base64)
	}

	if a.taskManager.HasRunningTask() {
		logger.Println("忽略重复触发：当前有任务正在运行")
		a.EmitEvent("toast", "正在处理中，请稍候...")
		return
	}

	screenshots := make([]string, len(a.screenshotBuffer))
	copy(screenshots, a.screenshotBuffer)
	a.screenshotBuffer = nil

	displayEpoch := a.mobileDisplayEpoch()
	conversationEpoch := a.conversationEpoch
	ctx, taskID := a.taskManager.StartTask("solve")
	go func() {
		defer a.taskManager.CompleteTask(taskID)
		a.solveInternal(ctx, taskID, screenshots, displayEpoch, conversationEpoch)
	}()
}

func (a *App) TriggerSolve() {
	a.TriggerSend()
}

func (a *App) TriggerDeleteScreenshot() {
	a.RemoveLastScreenshot()
}

func (a *App) solveInternal(ctx context.Context, taskID int64, screenshots []string, displayEpoch, conversationEpoch uint64) bool {
	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-api-key")
		return false
	}

	var target desktopSolveTarget
	capture := a.beginQuestionSolveCapture(ctx, cfg, screenshots, func(event string, data ...interface{}) {
		a.routeConversationSolutionEvent(displayEpoch, target, event, data...)
	})
	target = desktopSolveTarget{epoch: conversationEpoch, sessionID: capture.sessionID, recordID: capture.recordID}
	if target.sessionID == "" {
		target.sessionID = "live"
	}
	if target.recordID == "" {
		target.recordID = fmt.Sprintf("task-%d", taskID)
	}
	runtimeEmit(a.ctx, "desktop-solve-event", DesktopSolveEvent{
		Epoch:     target.epoch,
		SessionID: target.sessionID,
		RecordID:  target.recordID,
		Event:     "request-start",
		Images:    append([]string(nil), screenshots...),
	})
	req := solution.Request{Config: cfg, Screenshots: screenshots}
	if capture.store != nil && capture.recordID != "" {
		req.NumberingContext = capture.store.CurrentNumberingContext(capture.recordID)
	}
	conversationBefore := a.solver.SnapshotConversation()
	ok := a.solver.Solve(ctx, req, solution.Callbacks{EmitEvent: capture.Emit})

	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	currentEpoch := conversationEpoch == a.conversationEpoch
	if !currentEpoch {
		a.solver.ClearHistory()
	}
	refs := append([]questionbank.ConversationTurnRef(nil), a.conversationRefs...)
	answer := capture.Answer()
	if ok && strings.TrimSpace(answer) != "" {
		if cfg.KeepContext && currentEpoch && capture.RequiresPersistence() {
			refs = append(refs, questionbank.ConversationTurnRef{SessionID: capture.sessionID, RecordID: capture.recordID})
			if len(refs) > questionbank.MaxConversationTurns {
				refs = refs[len(refs)-questionbank.MaxConversationTurns:]
			}
		} else {
			refs = []questionbank.ConversationTurnRef{}
		}
	}
	snapshot := &questionbank.ConversationSnapshot{
		Version:   1,
		Quality:   questionbank.ConversationQualityPersisted,
		Turns:     refs,
		UpdatedAt: time.Now(),
	}
	persisted := capture.Complete(ok, snapshot)
	if persisted {
		a.conversationRefs = refs
		return ok
	}
	if ok && strings.TrimSpace(answer) != "" && capture.RequiresPersistence() {
		if !currentEpoch {
			return false
		}
		if err := a.solver.ReplaceConversation(conversationBefore); err != nil {
			logger.Printf("题库保存失败后恢复模型上下文失败: %v", err)
			a.EmitEvent("toast", "题库与模型上下文恢复失败，请停止继续解题并保留现场", "error")
		}
		return false
	}
	return ok
}

func (a *App) CancelRunningTask() bool {
	return a.taskManager.CancelCurrentTask()
}

func (a *App) IsInterruptThinkingEnabled() bool {
	return a.configManager.Get().InterruptThinking
}
