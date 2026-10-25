package app

import (
	"fmt"
	"strings"

	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/questionbank"
	"Q-Solver/pkg/solution"
)

const reconstructedConversationWarning = "按题库记录重建上下文，可能与当时实际上下文不同"

type QuestionBankContinuation struct {
	Epoch          uint64                    `json:"epoch"`
	ContextQuality string                    `json:"contextQuality"`
	ContextWarning string                    `json:"contextWarning,omitempty"`
	Detail         QuestionBankSessionDetail `json:"detail"`
}

func (a *App) ContinueQuestionBankSession(id string) (QuestionBankContinuation, error) {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()

	id = strings.TrimSpace(id)
	if id == "" {
		return QuestionBankContinuation{}, fmt.Errorf("分段 ID 无效")
	}
	if a.taskManager != nil && a.taskManager.HasInFlightTask() {
		return QuestionBankContinuation{}, fmt.Errorf("请等待解题任务完全结束后再切换分段")
	}
	if len(a.screenshotBuffer) > 0 {
		return QuestionBankContinuation{}, fmt.Errorf("请先发送或清除待发送截图，再切换分段")
	}
	if a.solver == nil {
		return QuestionBankContinuation{}, fmt.Errorf("解题服务尚未就绪")
	}
	if a.questionBank == nil {
		return QuestionBankContinuation{}, fmt.Errorf("题库服务不可用")
	}

	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closing {
		return QuestionBankContinuation{}, fmt.Errorf("应用正在关闭")
	}
	if b.store == nil {
		return QuestionBankContinuation{}, fmt.Errorf("请先在设置中选择题库目录")
	}
	if b.captures > 0 || b.organizing {
		return QuestionBankContinuation{}, fmt.Errorf("请等待当前题目、整理或导出任务完成")
	}
	allowed, err := recentQuestionBankContains(b.store, id)
	if err != nil {
		return QuestionBankContinuation{}, err
	}
	if !allowed {
		return QuestionBankContinuation{}, fmt.Errorf("只能继续最近 20 个分段；更早记录仍保留在题库文件夹中")
	}

	snapshot, err := b.store.ConversationSnapshotByID(id)
	if err != nil {
		return QuestionBankContinuation{}, err
	}
	if a.configManager != nil && !a.configManager.Get().KeepContext {
		snapshot = &questionbank.ConversationSnapshot{Version: 1, Quality: questionbank.ConversationQualityPersisted}
	}
	conversation, err := resolveSolverConversation(b.store, snapshot)
	if err != nil {
		return QuestionBankContinuation{}, err
	}

	oldConversation := a.solver.SnapshotConversation()
	if err = a.solver.ReplaceConversation(conversation); err != nil {
		return QuestionBankContinuation{}, fmt.Errorf("恢复模型上下文失败: %w", err)
	}
	activation, err := b.store.ActivateHistoricalSession(id)
	if err != nil {
		if rollbackErr := a.solver.ReplaceConversation(oldConversation); rollbackErr != nil {
			return QuestionBankContinuation{}, fmt.Errorf("切换题库分段失败: %v；恢复原模型上下文也失败: %w", err, rollbackErr)
		}
		return QuestionBankContinuation{}, err
	}

	a.conversationRefs = append([]questionbank.ConversationTurnRef(nil), snapshot.Turns...)
	a.conversationEpoch++
	result := QuestionBankContinuation{
		Epoch:          a.conversationEpoch,
		ContextQuality: snapshot.Quality,
		Detail:         buildQuestionBankSessionDetail(activation.Current),
	}
	if snapshot.Quality == questionbank.ConversationQualityReconstructed {
		result.ContextWarning = reconstructedConversationWarning
	}
	return result, nil
}

func resolveSolverConversation(store *questionbank.Store, snapshot *questionbank.ConversationSnapshot) (solution.ConversationState, error) {
	if store == nil || snapshot == nil {
		return solution.ConversationState{}, fmt.Errorf("题库对话快照不可用")
	}
	resolved, err := store.ResolveConversation(*snapshot)
	if err != nil {
		return solution.ConversationState{}, err
	}
	turns := make([]solution.ConversationTurn, 0, len(resolved))
	for _, turn := range resolved {
		parts := make([]llm.ContentPart, 0, len(turn.Images))
		for _, image := range turn.Images {
			parts = append(parts, llm.ImagePart(image))
		}
		turns = append(turns, solution.ConversationTurn{
			User:      llm.NewMultiPartMessage(llm.RoleUser, parts),
			Assistant: llm.NewAssistantMessage(turn.Answer),
		})
	}
	return solution.ConversationState{Turns: turns}, nil
}

func (a *App) restoreCurrentQuestionBankConversation() {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()
	if a.solver == nil || a.questionBank == nil {
		return
	}

	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	_ = a.restoreCurrentQuestionBankConversationLocked(b)
}

func (a *App) restoreCurrentQuestionBankConversationLocked(b *questionBankService) error {
	if a.solver == nil || b == nil || b.store == nil {
		return nil
	}
	current, _ := b.store.Snapshot()
	if current == nil || (current.State != questionbank.Active && current.State != questionbank.Paused) {
		return nil
	}
	if a.configManager != nil && !a.configManager.Get().KeepContext {
		a.solver.ClearHistory()
		a.conversationRefs = nil
		if err := b.store.ClearCurrentConversation(); err != nil {
			b.message = "模型上下文已清空，但题库快照未保存，请重试保存"
			return err
		}
		return nil
	}
	snapshot, err := b.store.ConversationSnapshotByID(current.ID)
	if err == nil {
		var state solution.ConversationState
		state, err = resolveSolverConversation(b.store, snapshot)
		if err == nil {
			err = a.solver.ReplaceConversation(state)
		}
	}
	if err != nil {
		a.conversationRefs = nil
		b.message = fmt.Sprintf("当前题库分段已打开，但模型上下文恢复失败：%v；请在题库中重新选择该分段", err)
		return err
	}
	a.conversationRefs = append([]questionbank.ConversationTurnRef(nil), snapshot.Turns...)
	if snapshot.Quality == questionbank.ConversationQualityReconstructed {
		b.message = reconstructedConversationWarning
	}
	return nil
}
