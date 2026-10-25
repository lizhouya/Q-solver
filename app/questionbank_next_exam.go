package app

import "fmt"

type StartNextExamResult struct {
	SessionID              string `json:"sessionId"`
	PreviousState          string `json:"previousState"`
	SessionState           string `json:"sessionState"`
	Sealed                 bool   `json:"sealed"`
	ConversationEpoch      uint64 `json:"conversationEpoch"`
	ClearedScreenshotCount int    `json:"clearedScreenshotCount"`
}

// StartNextExam closes the current exam boundary and clears desktop-only
// conversation state. The next question creates its own segment lazily.
func (a *App) StartNextExam(confirmed bool) (StartNextExamResult, error) {
	if !confirmed {
		return StartNextExamResult{}, fmt.Errorf("请确认开始下一套笔试")
	}

	a.conversationMu.Lock()
	result, err := a.startNextExamLocked()
	a.conversationMu.Unlock()
	if err != nil {
		return StartNextExamResult{}, err
	}

	// Events must run without conversationMu or questionBank.mu held. In
	// particular notifyQuestionBank takes questionBank.mu itself.
	a.EmitEvent("screenshots-cleared")
	a.notifyQuestionBank()
	return result, nil
}

// startNextExamLocked requires conversationMu. It follows the lock order
// conversationMu -> TaskCoordinator -> questionBank.mu -> Store -> Solver.
func (a *App) startNextExamLocked() (StartNextExamResult, error) {
	if a.taskManager == nil {
		return StartNextExamResult{}, fmt.Errorf("解题服务尚未就绪")
	}
	if a.taskManager.HasInFlightTask() {
		return StartNextExamResult{}, fmt.Errorf("请等待解题任务完全结束后再开始下一套")
	}
	if a.solver == nil {
		return StartNextExamResult{}, fmt.Errorf("解题服务尚未就绪")
	}
	if a.questionBank == nil {
		return StartNextExamResult{}, fmt.Errorf("题库服务不可用")
	}

	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closing {
		return StartNextExamResult{}, fmt.Errorf("应用正在关闭")
	}
	if b.store == nil {
		return StartNextExamResult{}, fmt.Errorf("请先在设置中选择题库目录")
	}
	if b.captures > 0 {
		return StartNextExamResult{}, fmt.Errorf("请等待当前题目处理完成后再开始下一套")
	}
	if b.organizing || b.operation == "organizing" || b.operation == "exporting" {
		return StartNextExamResult{}, fmt.Errorf("请等待整理或导出任务完成后再开始下一套")
	}

	sealed, err := b.store.SealCurrentForNext()
	if err != nil {
		return StartNextExamResult{}, err
	}

	// Everything below is infallible in-memory cleanup and only runs after the
	// Store commit (or an idempotently safe ended/exported result) succeeded.
	a.solver.ClearHistory()
	a.conversationRefs = nil
	a.conversationEpoch++
	clearedScreenshots := len(a.screenshotBuffer)
	a.screenshotBuffer = nil
	b.message = "已开始下一套笔试；下一题将自动创建新分段"

	return StartNextExamResult{
		SessionID:              sealed.SessionID,
		PreviousState:          sealed.PreviousState,
		SessionState:           sealed.SessionState,
		Sealed:                 sealed.Sealed,
		ConversationEpoch:      a.conversationEpoch,
		ClearedScreenshotCount: clearedScreenshots,
	}, nil
}
