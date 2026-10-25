package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/questionbank"
	"Q-Solver/pkg/solution"
)

func completedRecord(t *testing.T, store *questionbank.Store, imageURL, answer string) string {
	t.Helper()
	id, err := store.Begin([]string{imageURL}, "old-model", "old-scene")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Finish(id, answer, questionbank.Completed); err != nil {
		t.Fatal(err)
	}
	return id
}

func continuationApp(t *testing.T) (*App, *questionbank.Store, string, string) {
	t.Helper()
	a, store, imageURL := archiveApp(t)
	a.solver = solution.NewSolver(nil)
	firstRecord := completedRecord(t, store, imageURL, "第一段答案")
	first, _ := store.Snapshot()
	if err := store.End(); err != nil {
		t.Fatal(err)
	}
	completedRecord(t, store, imageURL, "当前段答案")
	return a, store, first.ID, firstRecord
}

func TestContinueQuestionBankSessionSwitchesStoreSolverAndEpoch(t *testing.T) {
	a, store, targetID, targetRecordID := continuationApp(t)
	before, _ := store.Snapshot()

	result, err := a.ContinueQuestionBankSession(targetID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Epoch != 1 || result.Detail.ID != targetID || result.Detail.State != questionbank.Active {
		t.Fatalf("续接结果错误: %+v", result)
	}
	if result.ContextQuality != questionbank.ConversationQualityReconstructed || result.ContextWarning != reconstructedConversationWarning {
		t.Fatalf("旧记录重建提示缺失: %+v", result)
	}
	current, dirty := store.Snapshot()
	if dirty || current.ID != targetID || current.State != questionbank.Active {
		t.Fatalf("题库 current 未切换: %+v dirty=%v", current, dirty)
	}
	paused, err := store.SnapshotByID(before.ID)
	if err != nil || paused.State != questionbank.Paused {
		t.Fatalf("原活动段未暂停: %+v err=%v", paused, err)
	}
	conversation := a.solver.SnapshotConversation()
	if len(conversation.Turns) != 1 || conversation.Turns[0].Assistant.Content != "第一段答案" || len(conversation.Turns[0].User.Parts) != 1 {
		t.Fatalf("Solver 上下文未替换: %+v", conversation)
	}
	if len(a.conversationRefs) != 1 || a.conversationRefs[0].RecordID != targetRecordID {
		t.Fatalf("App turn refs 未同步: %+v", a.conversationRefs)
	}
}

func TestContinueQuestionBankSessionRejectsUnsafeStateWithoutChangingCurrent(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*App)
		want    string
	}{
		{name: "in-flight", prepare: func(a *App) { _, _ = a.taskManager.StartTask("solve") }, want: "完全结束"},
		{name: "pending screenshots", prepare: func(a *App) { a.screenshotBuffer = []string{"pending"} }, want: "待发送截图"},
		{name: "capture", prepare: func(a *App) { a.questionBank.captures = 1 }, want: "整理或导出"},
		{name: "organizing", prepare: func(a *App) { a.questionBank.organizing = true }, want: "整理或导出"},
		{name: "exporting", prepare: func(a *App) {
			a.questionBank.organizing = true
			a.questionBank.operation = "exporting"
		}, want: "整理或导出"},
		{name: "closing", prepare: func(a *App) { a.questionBank.closing = true }, want: "应用正在关闭"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, store, targetID, _ := continuationApp(t)
			before, _ := store.Snapshot()
			tt.prepare(a)
			_, err := a.ContinueQuestionBankSession(targetID)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("未按预期拒绝: %v", err)
			}
			after, _ := store.Snapshot()
			if after.ID != before.ID || a.conversationEpoch != 0 || len(a.solver.SnapshotConversation().Turns) != 0 {
				t.Fatalf("拒绝后状态发生变化: before=%s after=%s epoch=%d", before.ID, after.ID, a.conversationEpoch)
			}
		})
	}
}

func TestContinueQuestionBankSessionWaitsForCancelledGoroutineCompletion(t *testing.T) {
	a, _, targetID, _ := continuationApp(t)
	_, taskID := a.taskManager.StartTask("solve")
	if !a.taskManager.CancelCurrentTask() {
		t.Fatal("取消任务失败")
	}
	if _, err := a.ContinueQuestionBankSession(targetID); err == nil {
		t.Fatal("取消后 goroutine 未完成时允许了切换")
	}
	a.taskManager.CompleteTask(taskID)
	if _, err := a.ContinueQuestionBankSession(targetID); err != nil {
		t.Fatalf("goroutine 完成后仍拒绝切换: %v", err)
	}
}

func reopenQuestionBankStore(t *testing.T, root, sessionID string) *questionbank.Store {
	t.Helper()
	reopened, err := questionbank.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = reopened.Open(sessionID); err != nil {
		t.Fatal(err)
	}
	return reopened
}

func TestRestoreCurrentQuestionBankConversationUsesPersistedSnapshot(t *testing.T) {
	_, store, image := archiveApp(t)
	recordID, err := store.BeginWithContext([]string{image}, "old-model", "old-scene", true)
	if err != nil {
		t.Fatal(err)
	}
	current, _ := store.Snapshot()
	snapshot := &questionbank.ConversationSnapshot{
		Version: 1,
		Quality: questionbank.ConversationQualityPersisted,
		Turns:   []questionbank.ConversationTurnRef{{SessionID: current.ID, RecordID: recordID}},
	}
	if err = store.FinishWithConversation(recordID, "persisted answer", questionbank.Completed, snapshot); err != nil {
		t.Fatal(err)
	}

	reopened := reopenQuestionBankStore(t, store.Root(), current.ID)
	a := &App{
		questionBank: &questionBankService{store: reopened},
		solver:       solution.NewSolver(&fakeQuestionBankProvider{}),
	}
	a.restoreCurrentQuestionBankConversation()

	got := a.solver.SnapshotConversation()
	if len(got.Turns) != 1 || got.Turns[0].Assistant.Content != "persisted answer" {
		t.Fatalf("启动时没有恢复持久化模型上下文: %+v", got)
	}
	if len(a.conversationRefs) != 1 || a.conversationRefs[0].RecordID != recordID {
		t.Fatalf("启动时没有同步引用账本: %+v", a.conversationRefs)
	}
	if a.questionBank.message != "" {
		t.Fatalf("精确持久化快照不应显示 legacy 警告: %q", a.questionBank.message)
	}
}

func TestRestoreCurrentQuestionBankConversationMarksLegacyRebuild(t *testing.T) {
	_, store, image := archiveApp(t)
	recordID, err := store.Begin([]string{image}, "old-model", "old-scene")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Finish(recordID, "legacy answer", questionbank.Completed); err != nil {
		t.Fatal(err)
	}
	current, _ := store.Snapshot()

	reopened := reopenQuestionBankStore(t, store.Root(), current.ID)
	a := &App{
		questionBank: &questionBankService{store: reopened},
		solver:       solution.NewSolver(&fakeQuestionBankProvider{}),
	}
	a.restoreCurrentQuestionBankConversation()

	got := a.solver.SnapshotConversation()
	if len(got.Turns) != 1 || got.Turns[0].Assistant.Content != "legacy answer" {
		t.Fatalf("启动时没有按旧记录重建模型上下文: %+v", got)
	}
	if a.questionBank.message != reconstructedConversationWarning {
		t.Fatalf("旧记录重建警告错误: %q", a.questionBank.message)
	}
}

func TestRestoreCurrentQuestionBankConversationFailsClosed(t *testing.T) {
	_, store, image := archiveApp(t)
	recordID, err := store.BeginWithContext([]string{image}, "old-model", "old-scene", true)
	if err != nil {
		t.Fatal(err)
	}
	current, _ := store.Snapshot()
	snapshot := &questionbank.ConversationSnapshot{
		Version: 1,
		Quality: questionbank.ConversationQualityPersisted,
		Turns:   []questionbank.ConversationTurnRef{{SessionID: current.ID, RecordID: recordID}},
	}
	if err = store.FinishWithConversation(recordID, "persisted answer", questionbank.Completed, snapshot); err != nil {
		t.Fatal(err)
	}
	reopened := reopenQuestionBankStore(t, store.Root(), current.ID)
	persisted, err := reopened.SnapshotByID(current.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(store.Root(), "_记录", current.ID, persisted.Records[0].Images[0])); err != nil {
		t.Fatal(err)
	}

	a := &App{
		questionBank:     &questionBankService{store: reopened},
		solver:           solution.NewSolver(&fakeQuestionBankProvider{}),
		conversationRefs: []questionbank.ConversationTurnRef{{SessionID: "stale", RecordID: "stale"}},
	}
	a.restoreCurrentQuestionBankConversation()

	if got := a.solver.SnapshotConversation(); len(got.Turns) != 0 {
		t.Fatalf("损坏历史不应被部分恢复: %+v", got)
	}
	if len(a.conversationRefs) != 0 {
		t.Fatalf("恢复失败后不应保留旧引用: %+v", a.conversationRefs)
	}
	if !strings.Contains(a.questionBank.message, "模型上下文恢复失败") {
		t.Fatalf("恢复失败缺少可操作提示: %q", a.questionBank.message)
	}
}

func TestContinueQuestionBankSessionRejectsDirtyStore(t *testing.T) {
	a, store, image := archiveApp(t)
	a.solver = solution.NewSolver(nil)
	targetRecord := completedRecord(t, store, image, "目标答案")
	target, _ := store.Snapshot()
	if err := store.End(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Begin([]string{image}, "model", "scene"); err != nil {
		t.Fatal(err)
	}
	current, _ := store.Snapshot()
	if err := store.Append(current.Records[0].ID, "尚未节流落盘"); err != nil {
		t.Fatal(err)
	}
	if _, dirty := store.Snapshot(); !dirty {
		t.Fatal("测试前置未制造 dirty Store")
	}

	_, err := a.ContinueQuestionBankSession(target.ID)
	if err == nil || !strings.Contains(err.Error(), "未落盘") {
		t.Fatalf("dirty Store 未拒绝切换: %v", err)
	}
	after, _ := store.Snapshot()
	if after.ID != current.ID || a.conversationEpoch != 0 || len(a.conversationRefs) != 0 || len(a.solver.SnapshotConversation().Turns) != 0 {
		t.Fatalf("dirty 拒绝后状态变化: current=%+v refs=%+v epoch=%d", after, a.conversationRefs, a.conversationEpoch)
	}
	_ = targetRecord
}

func TestContinueQuestionBankSessionRejectsTargetRunningRecord(t *testing.T) {
	a, store, image := archiveApp(t)
	a.solver = solution.NewSolver(nil)
	completedRecord(t, store, image, "目标答案")
	target, _ := store.Snapshot()
	if err := store.End(); err != nil {
		t.Fatal(err)
	}
	target.Records[0].Status = questionbank.Running
	data, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(store.Root(), "_记录", target.ID, "session.json")
	if err = os.WriteFile(manifest, data, 0600); err != nil {
		t.Fatal(err)
	}
	completedRecord(t, store, image, "当前答案")
	before, _ := store.Snapshot()

	_, err = a.ContinueQuestionBankSession(target.ID)
	if err == nil || !strings.Contains(err.Error(), "未完成记录") {
		t.Fatalf("running 目标未拒绝切换: %v", err)
	}
	after, _ := store.Snapshot()
	if after.ID != before.ID || a.conversationEpoch != 0 || len(a.solver.SnapshotConversation().Turns) != 0 {
		t.Fatalf("running 目标拒绝后状态变化: before=%s after=%s", before.ID, after.ID)
	}
}

func TestContinueQuestionBankSessionRejectsOutsideRecentTwenty(t *testing.T) {
	a, store, image := archiveApp(t)
	a.solver = solution.NewSolver(nil)
	var oldest string
	for i := 0; i < recentQuestionBankSessionLimit+1; i++ {
		completedRecord(t, store, image, "答案")
		current, _ := store.Snapshot()
		if i == 0 {
			oldest = current.ID
		}
		if err := store.End(); err != nil {
			t.Fatal(err)
		}
	}

	_, err := a.ContinueQuestionBankSession(oldest)
	if err == nil || !strings.Contains(err.Error(), "最近 20") {
		t.Fatalf("第 21 个分段未拒绝续接: %v", err)
	}
	if a.conversationEpoch != 0 || len(a.solver.SnapshotConversation().Turns) != 0 {
		t.Fatalf("非最近分段拒绝后模型状态变化: epoch=%d", a.conversationEpoch)
	}
}

func TestContinueQuestionBankSessionRollsBackSolverWhenStoreRejects(t *testing.T) {
	a, store, image := archiveApp(t)
	targetRecord := completedRecord(t, store, image, "只读答案")
	target, _ := store.Snapshot()
	if err := store.DiscardCurrent(); err != nil {
		t.Fatal(err)
	}
	currentRecord := completedRecord(t, store, image, "当前答案")
	current, _ := store.Snapshot()

	a.solver = solution.NewSolver(nil)
	oldConversation := solution.ConversationState{Turns: []solution.ConversationTurn{{
		User:      llm.NewMultiPartMessage(llm.RoleUser, []llm.ContentPart{llm.ImagePart(image)}),
		Assistant: llm.NewAssistantMessage("原模型上下文"),
	}}}
	if err := a.solver.ReplaceConversation(oldConversation); err != nil {
		t.Fatal(err)
	}
	a.conversationRefs = []questionbank.ConversationTurnRef{{SessionID: current.ID, RecordID: currentRecord}}
	a.conversationEpoch = 9

	_, err := a.ContinueQuestionBankSession(target.ID)
	if err == nil || !strings.Contains(err.Error(), "永久只读") {
		t.Fatalf("discarded Store 拒绝未上抛: %v", err)
	}
	after, _ := store.Snapshot()
	conversation := a.solver.SnapshotConversation()
	if after.ID != current.ID || len(conversation.Turns) != 1 || conversation.Turns[0].Assistant.Content != "原模型上下文" || a.conversationEpoch != 9 {
		t.Fatalf("Store 拒绝后跨组件状态未回滚: current=%+v conversation=%+v epoch=%d", after, conversation, a.conversationEpoch)
	}
	if len(a.conversationRefs) != 1 || a.conversationRefs[0].RecordID != currentRecord {
		t.Fatalf("Store 拒绝后引用账本变化: %+v", a.conversationRefs)
	}
	_ = targetRecord
}
