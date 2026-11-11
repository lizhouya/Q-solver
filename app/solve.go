package app

import (
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/questionbank"
	"Q-Solver/pkg/solution"
	"context"
	"fmt"
	"strings"
	"time"
)

const MaxScreenshots = 3

type solveStartRequest struct {
	mode        SolveMode
	retryTarget *desktopSolveTarget
	screenshots []string
}

type solveRunRequest struct {
	screenshots    []string
	mode           SolveMode
	normalDomainID string
	keepContext    bool
	retryTarget    *desktopSolveTarget
	capture        *questionSolveCapture
	target         desktopSolveTarget
}

func (a *App) TriggerScreenshot() {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	cfg := a.configManager.Get()

	if strings.TrimSpace(cfg.BaseURL) == "" {
		a.EmitEvent("toast", "请先配置模型接口地址")
		a.EmitEvent("open-settings", "api")
		return
	}

	if cfg.Model == "" {
		a.EmitEvent("toast", "请先选择模型")
		a.EmitEvent("open-settings", "api")
		return
	}

	if a.taskManager != nil && a.taskManager.HasInFlightTask() {
		logger.Println("忽略截图：当前有任务尚未完全结束")
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

	a.appendScreenshotLocked(previewResult.Base64)
	a.syncMobileScreenshotState(len(a.screenshotBuffer))
	a.EmitEvent("screenshot-taken", previewResult.Base64, len(a.screenshotBuffer))
}

func (a *App) RemoveScreenshot(index int) {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	if index < 0 || index >= len(a.screenshotBuffer) {
		return
	}
	a.screenshotBuffer = append(a.screenshotBuffer[:index], a.screenshotBuffer[index+1:]...)
	a.syncMobileScreenshotState(len(a.screenshotBuffer))
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
	a.syncMobileScreenshotState(len(a.screenshotBuffer))
	a.EmitEvent("screenshot-removed", index, len(a.screenshotBuffer))
}

func (a *App) ClearScreenshots() {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	a.screenshotBuffer = nil
	if a.taskManager == nil || !a.taskManager.HasInFlightTask() {
		a.invalidateSolveRetryLocked()
	}
	a.syncMobileScreenshotState(0)
	a.EmitEvent("screenshots-cleared")
}

func (a *App) invalidateSolveRetryLocked() {
	if a.solveAttempts != nil {
		a.solveAttempts.Invalidate()
	}
}

func (a *App) appendScreenshotLocked(image string) {
	a.invalidateSolveRetryLocked()
	a.screenshotBuffer = append(a.screenshotBuffer, image)
}

func needsResumeConfirmation(resumePath, resumeContent string) bool {
	return strings.TrimSpace(resumePath) != "" && strings.TrimSpace(resumeContent) == ""
}

func (a *App) TriggerSend() {
	a.triggerSend(SolveModeNormal, false)
}

func (a *App) TriggerQuickSend() {
	a.triggerSend(SolveModeFast, false)
}

func (a *App) ConfirmSolveWithoutResume() {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	mode := a.pendingSolveMode
	if mode == "" {
		mode = SolveModeNormal
	}
	a.pendingSolveMode = ""
	a.startSolveLockedWithResume(solveStartRequest{mode: mode}, true)
}

func (a *App) triggerSend(mode SolveMode, skipResumeWarning bool) {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	a.startSolveLockedWithResume(solveStartRequest{mode: mode}, skipResumeWarning)
}

func (a *App) startSolve(request solveStartRequest) SolveControlResult {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	return a.startSolveLocked(request)
}

func (a *App) startSolveLocked(request solveStartRequest) SolveControlResult {
	return a.startSolveLockedWithResume(request, false)
}

func (a *App) RetrySolve(sessionID, recordID string) SolveControlResult {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	if a.taskManager == nil {
		return solveControlFailure(solveRetryPrepareFailed, safeSolveControlMessage(solveRetryPrepareFailed))
	}
	if a.taskManager.HasRunningTask() {
		return solveControlFailure(solveTaskRunning, safeSolveControlMessage(solveTaskRunning))
	}
	if a.taskManager.HasInFlightTask() {
		return solveControlFailure(solveTaskDraining, safeSolveControlMessage(solveTaskDraining))
	}
	if a.solveAttempts == nil {
		return solveControlFailure(solveNoRetryAvailable, safeSolveControlMessage(solveNoRetryAvailable))
	}
	snapshot, code := a.solveAttempts.RetrySnapshot(sessionID, recordID, a.conversationEpoch)
	if code != "" {
		return solveControlFailure(code, safeSolveControlMessage(code))
	}
	target := snapshot.target
	return a.startSolveLockedWithResume(solveStartRequest{
		mode:        snapshot.mode,
		retryTarget: &target,
		screenshots: append([]string(nil), snapshot.screenshots...),
	}, true)
}

func (a *App) startSolveLockedWithResume(request solveStartRequest, skipResumeWarning bool) SolveControlResult {
	if a.configManager == nil {
		return solveControlFailure(solveRetryPrepareFailed, "解题配置不可用")
	}
	cfg := a.configManager.Get()

	if strings.TrimSpace(cfg.BaseURL) == "" {
		a.EmitEvent("toast", "请先配置模型接口地址")
		a.EmitEvent("open-settings", "api")
		return solveControlFailure(solveRetryPrepareFailed, "请先配置模型接口地址")
	}
	if cfg.Model == "" {
		a.EmitEvent("toast", "请先选择模型")
		a.EmitEvent("open-settings", "api")
		return solveControlFailure(solveRetryPrepareFailed, "请先选择模型")
	}
	if !skipResumeWarning && needsResumeConfirmation(cfg.ResumePath, cfg.ResumeContent) {
		a.pendingSolveMode = normalizedSolveMode(request.mode)
		a.EmitEvent("resume-warning-required")
		return solveControlFailure(solveRetryPrepareFailed, "请确认是否忽略未解析的简历")
	}
	if a.taskManager == nil {
		return solveControlFailure(solveRetryPrepareFailed, "解题服务尚未就绪")
	}
	if a.taskManager.HasRunningTask() {
		logger.Println("忽略重复触发：当前有任务正在运行")
		a.EmitEvent("toast", safeSolveControlMessage(solveTaskRunning))
		return solveControlFailure(solveTaskRunning, safeSolveControlMessage(solveTaskRunning))
	}
	if a.taskManager.HasInFlightTask() {
		logger.Println("忽略重复触发：上一个任务尚未完全结束")
		a.EmitEvent("toast", safeSolveControlMessage(solveTaskDraining))
		return solveControlFailure(solveTaskDraining, safeSolveControlMessage(solveTaskDraining))
	}

	mode := normalizedSolveMode(request.mode)
	screenshots := append([]string(nil), request.screenshots...)
	normalDomainID := cfg.DomainId
	keepContext := cfg.KeepContext
	if request.retryTarget != nil {
		if a.solveAttempts == nil {
			return solveControlFailure(solveNoRetryAvailable, safeSolveControlMessage(solveNoRetryAvailable))
		}
		snapshot, code := a.solveAttempts.RetrySnapshot(request.retryTarget.sessionID, request.retryTarget.recordID, a.conversationEpoch)
		if code != "" {
			return solveControlFailure(code, safeSolveControlMessage(code))
		}
		mode = snapshot.mode
		screenshots = append([]string(nil), snapshot.screenshots...)
		normalDomainID = snapshot.normalDomainID
		keepContext = snapshot.keepContext
	} else if len(screenshots) == 0 {
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
				return solveControlFailure(solveRetryPrepareFailed, "截图失败")
			}
			a.appendScreenshotLocked(previewResult.Base64)
			a.syncMobileScreenshotState(len(a.screenshotBuffer))
		}
		screenshots = append([]string(nil), a.screenshotBuffer...)
		a.screenshotBuffer = nil
		a.syncMobileScreenshotState(0)
	}
	if len(screenshots) < 1 || len(screenshots) > MaxScreenshots {
		return solveControlFailure(solveRetryPrepareFailed, "每次解题需要 1～3 张截图")
	}

	displayEpoch := a.mobileDisplayEpoch()
	conversationEpoch := a.conversationEpoch
	runRequest := solveRunRequest{
		screenshots:    screenshots,
		mode:           mode,
		normalDomainID: normalDomainID,
		keepContext:    keepContext,
		retryTarget:    request.retryTarget,
	}
	solveCfg := cfg
	if mode == SolveModeFast {
		solveCfg.DomainId = "gen-speed"
	} else {
		solveCfg.DomainId = normalDomainID
	}
	solveCfg.KeepContext = keepContext
	if _, err := llm.NewProvider(solveCfg); err != nil {
		a.EmitEvent("toast", "模型连接配置无效，请检查设置", "error")
		return solveControlFailure(solveRetryPrepareFailed, safeSolveControlMessage(solveRetryPrepareFailed))
	}

	ctx, taskID := a.taskManager.StartTask("solve")
	var target desktopSolveTarget
	capture, err := a.beginQuestionSolveCapture(ctx, solveCfg, screenshots, request.retryTarget, func(event string, data ...interface{}) {
		a.routeSolveAttemptEvent(taskID, displayEpoch, target, mode, event, data...)
	})
	if err != nil {
		a.taskManager.CompleteTask(taskID)
		return solveControlFailure(solveRetryPrepareFailed, safeSolveControlMessage(solveRetryPrepareFailed))
	}
	if request.retryTarget != nil {
		target = *request.retryTarget
	} else {
		target = desktopSolveTarget{epoch: conversationEpoch, sessionID: capture.sessionID, recordID: capture.recordID}
		if target.sessionID == "" {
			target.sessionID = "live"
		}
		if target.recordID == "" {
			target.recordID = fmt.Sprintf("task-%d", taskID)
		}
	}
	if a.solveAttempts == nil {
		a.solveAttempts = newSolveAttemptTracker()
	}
	a.solveAttempts.Start(taskID, screenshots, mode, normalDomainID, keepContext, conversationEpoch)
	a.solveAttempts.BindTarget(taskID, target)
	a.pendingSolveMode = ""
	runRequest.capture = capture
	runRequest.target = target
	a.emitSolveRequestStart(target, mode, screenshots)

	go func() {
		ok := a.solveInternal(ctx, taskID, runRequest, displayEpoch, conversationEpoch)
		a.taskManager.CompleteTask(taskID)
		terminal := attemptFailed
		if ok {
			terminal = attemptCompleted
		} else if ctx.Err() != nil {
			terminal = attemptCancelled
		}
		a.solveAttempts.Finish(taskID, terminal)
	}()
	return solveControlSuccess("", "")
}

func (a *App) routeSolveAttemptEvent(taskID int64, displayEpoch uint64, target desktopSolveTarget, mode SolveMode, event string, data ...interface{}) {
	if a.solveAttempts != nil {
		switch event {
		case "solution-error":
			a.solveAttempts.Finish(taskID, attemptFailed)
		case "solution-cancelled":
			a.solveAttempts.Finish(taskID, attemptCancelled)
		}
	}
	canRetry := false
	if (event == "solution-error" || event == "solution-cancelled") && a.solveAttempts != nil {
		_, code := a.solveAttempts.RetrySnapshot(target.sessionID, target.recordID, target.epoch)
		canRetry = code == ""
	}
	a.routeConversationSolutionEvent(displayEpoch, target, mode, canRetry, event, data...)
}

func (a *App) emitSolveRequestStart(target desktopSolveTarget, mode SolveMode, screenshots []string) {
	envelope := newDesktopSolveEvent(target, mode, false, "request-start")
	envelope.Images = append([]string(nil), screenshots...)
	runtimeEmit(a.ctx, "desktop-solve-event", envelope)
}

func normalizedSolveMode(mode SolveMode) SolveMode {
	if mode == SolveModeFast {
		return SolveModeFast
	}
	return SolveModeNormal
}

func (a *App) TriggerSolve() {
	a.TriggerSend()
}

func (a *App) TriggerDeleteScreenshot() {
	a.RemoveLastScreenshot()
}

func (a *App) solveInternal(ctx context.Context, taskID int64, run solveRunRequest, displayEpoch, conversationEpoch uint64) bool {
	cfg := a.configManager.Get()
	if run.mode == SolveModeFast {
		cfg.DomainId = "gen-speed"
	} else {
		cfg.DomainId = run.normalDomainID
	}
	cfg.KeepContext = run.keepContext

	if strings.TrimSpace(cfg.BaseURL) == "" {
		a.EmitEvent("toast", "请先配置模型接口地址")
		a.EmitEvent("open-settings", "api")
		return false
	}

	if _, err := llm.NewProvider(cfg); err != nil {
		a.EmitEvent("toast", "模型连接配置无效，请检查设置", "error")
		return false
	}

	capture := run.capture
	target := run.target
	if capture == nil {
		var err error
		capture, err = a.beginQuestionSolveCapture(ctx, cfg, run.screenshots, run.retryTarget, func(event string, data ...interface{}) {
			a.routeSolveAttemptEvent(taskID, displayEpoch, target, run.mode, event, data...)
		})
		if err != nil {
			return false
		}
		if run.retryTarget != nil {
			target = *run.retryTarget
		} else {
			target = desktopSolveTarget{epoch: conversationEpoch, sessionID: capture.sessionID, recordID: capture.recordID}
			if target.sessionID == "" {
				target.sessionID = "live"
			}
			if target.recordID == "" {
				target.recordID = fmt.Sprintf("task-%d", taskID)
			}
		}
		if a.solveAttempts != nil {
			a.solveAttempts.BindTarget(taskID, target)
		}
		a.emitSolveRequestStart(target, run.mode, run.screenshots)
	}
	req := solution.Request{Config: cfg, Screenshots: run.screenshots}
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

func (a *App) CancelSolve() SolveControlResult {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()

	if a.taskManager == nil {
		return solveControlFailure(solveNoRunningTask, safeSolveControlMessage(solveNoRunningTask))
	}
	taskID := a.taskManager.GetCurrentTaskID()
	if taskID == 0 || a.solveAttempts == nil || !a.solveAttempts.MarkCancelling(taskID) {
		return solveControlFailure(solveNoRunningTask, safeSolveControlMessage(solveNoRunningTask))
	}
	if a.taskManager.CancelCurrentTask() {
		return solveControlSuccess(solveCancelRequested, "正在中断解题")
	}
	if a.taskManager.HasInFlightTask() {
		return solveControlSuccess(solveCancelRequested, "正在中断解题")
	}
	return solveControlFailure(solveNoRunningTask, safeSolveControlMessage(solveNoRunningTask))
}

func (a *App) CancelRunningTask() bool {
	return a.CancelSolve().Success
}

func (a *App) IsInterruptThinkingEnabled() bool {
	return a.configManager.Get().InterruptThinking
}
