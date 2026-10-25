package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Q-Solver/pkg/answersummary"
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/questionbank"
	"Q-Solver/pkg/solution"
)

type persistenceFailureProvider struct {
	store *questionbank.Store
	err   error
}

func (p *persistenceFailureProvider) GenerateContentStream(_ context.Context, _ []llm.Message, onChunk llm.StreamCallback) (llm.Message, error) {
	if onChunk != nil {
		onChunk(llm.StreamChunk{Type: llm.ChunkContent, Content: "answer"})
	}
	session, _ := p.store.Snapshot()
	manifest := filepath.Join(p.store.Root(), "_记录", session.ID, "session.json")
	backup := manifest + ".bak"
	if p.err = os.Rename(manifest, backup); p.err == nil {
		p.err = os.Mkdir(manifest, 0700)
	}
	return llm.NewAssistantMessage("answer"), nil
}

func (p *persistenceFailureProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) {
	return llm.Message{}, nil
}
func (p *persistenceFailureProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *persistenceFailureProvider) TestChat(context.Context) error              { return nil }

type numberingCaptureProvider struct {
	store    *questionbank.Store
	calls    int
	messages []llm.Message
	answer   string
	t        *testing.T
}

func (p *numberingCaptureProvider) GenerateContentStream(_ context.Context, messages []llm.Message, _ llm.StreamCallback) (llm.Message, error) {
	p.calls++
	p.messages = messages
	if p.store != nil {
		session, _ := p.store.Snapshot()
		if len(session.Records) != 2 || session.Records[1].Status != questionbank.Running {
			p.t.Fatalf("模型调用前必须已创建当前 running Record: %+v", session.Records)
		}
	}
	return llm.NewAssistantMessage(p.answer), nil
}

func (p *numberingCaptureProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) {
	return llm.Message{}, nil
}
func (p *numberingCaptureProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *numberingCaptureProvider) TestChat(context.Context) error              { return nil }

func configuredSolveManager(t *testing.T) *config.ConfigManager {
	t.Helper()
	t.Setenv("APPDATA", t.TempDir())
	manager := config.NewConfigManager()
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Patch(func(cfg *config.Config) {
		cfg.APIKey = "test-key"
		cfg.Model = "test-model"
		cfg.DomainId = "general-assistant"
		cfg.KeepContext = false
		cfg.OutputMode = "desktop"
	}); err != nil {
		t.Fatal(err)
	}
	return manager
}

func TestSolveCreatesRecordBeforeReadingNumberingContext(t *testing.T) {
	a, store, image := archiveApp(t)
	previousID, err := store.BeginWithContext([]string{image}, "test-model", "general-assistant", false)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.FinishWithConversationResult(previousID, "previous", []answersummary.Item{{
		Label: "7", Kind: answersummary.KindSingleChoice, Value: "A", Status: answersummary.StatusReady,
	}}, questionbank.Completed, &questionbank.ConversationSnapshot{
		Version: 1, Quality: questionbank.ConversationQualityPersisted,
	}); err != nil {
		t.Fatal(err)
	}

	a.configManager = configuredSolveManager(t)
	answer := "解答\n\n## 最终答案汇总\n| 题号 | 题型 | 可填写答案 |\n|---|---|---|\n| 8 | 单选 | B |"
	provider := &numberingCaptureProvider{store: store, answer: answer, t: t}
	a.solver = solution.NewSolver(provider)

	if ok := a.solveInternal(context.Background(), 2, []string{image}, 0, 0); !ok {
		t.Fatal("有效回答应解题成功")
	}
	if provider.calls != 1 {
		t.Fatalf("不得新增模型调用: %d", provider.calls)
	}
	currentUser := provider.messages[len(provider.messages)-1]
	if len(currentUser.Parts) != 2 || currentUser.Parts[0].Type != llm.ContentText ||
		!strings.Contains(currentUser.Parts[0].Text, "7") || !strings.Contains(currentUser.Parts[0].Text, "8") ||
		currentUser.Parts[1].Type != llm.ContentImage || currentUser.Parts[1].Base64 != image {
		t.Fatalf("编号摘要未在当前 Record 创建后注入 user multipart: %+v", currentUser.Parts)
	}
	session, dirty := store.Snapshot()
	if dirty || len(session.Records) != 2 || len(session.Records[1].AnswerItems) != 1 || session.Records[1].AnswerItems[0].Value != "B" {
		t.Fatalf("结构化终态未原子保存: session=%+v dirty=%v", session, dirty)
	}
}

func TestSolveWithoutQuestionBankUsesEmptyNumberingContext(t *testing.T) {
	manager := configuredSolveManager(t)
	answer := "解答\n\n## 最终答案汇总\n| 题号 | 题型 | 可填写答案 |\n|---|---|---|\n| 1 | 单选 | C |"
	provider := &numberingCaptureProvider{answer: answer, t: t}
	a := &App{configManager: manager, solver: solution.NewSolver(provider)}

	if ok := a.solveInternal(context.Background(), 3, []string{"synthetic-image"}, 0, 0); !ok {
		t.Fatal("无题库时仍应解题成功")
	}
	currentUser := provider.messages[len(provider.messages)-1]
	if len(currentUser.Parts) != 1 || currentUser.Parts[0].Type != llm.ContentImage {
		t.Fatalf("无 Store/capture 时编号上下文必须为空: %+v", currentUser.Parts)
	}
}

func TestSolveRollsBackConversationWhenQuestionBankCompletionCannotPersist(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	a, store, image := archiveApp(t)
	manager := config.NewConfigManager()
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Patch(func(cfg *config.Config) {
		cfg.APIKey = "test-key"
		cfg.Model = "test-model"
		cfg.DomainId = "general-assistant"
		cfg.KeepContext = true
		cfg.OutputMode = "desktop"
	}); err != nil {
		t.Fatal(err)
	}
	a.configManager = manager
	provider := &persistenceFailureProvider{store: store}
	a.solver = solution.NewSolver(provider)

	if ok := a.solveInternal(context.Background(), 1, []string{image}, 0, 0); ok {
		t.Fatal("题库完成记录保存失败时不应报告整轮成功")
	}
	if provider.err != nil {
		t.Fatalf("注入题库持久化失败点失败: %v", provider.err)
	}
	if got := a.solver.SnapshotConversation(); len(got.Turns) != 0 {
		t.Fatalf("题库持久化失败后 Solver 保留了未归档轮次: %+v", got)
	}
	if len(a.conversationRefs) != 0 {
		t.Fatalf("题库持久化失败后 App 引用账本被推进: %+v", a.conversationRefs)
	}
	session, dirty := store.Snapshot()
	if !dirty {
		t.Fatal("题库持久化失败后应保持未保存阻断状态")
	}
	if session.Conversation != nil {
		t.Fatalf("失败保存不应把新对话快照伪装成已提交: %+v", session.Conversation)
	}
	if len(session.Records) != 1 || session.Records[0].Status != questionbank.Running {
		t.Fatalf("失败保存后应保留原内存记录供故障处理: %+v", session.Records)
	}
	if !store.HasPendingCompletion() {
		t.Fatal("失败保存后未保留可重试完成态")
	}

	manifest := filepath.Join(store.Root(), "_记录", session.ID, "session.json")
	if err := os.Remove(manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(manifest+".bak", manifest); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveQuestionBankSession(); err != nil {
		t.Fatalf("题库重试保存失败: %v", err)
	}

	recovered, dirty := store.Snapshot()
	if dirty || store.HasPendingCompletion() || recovered.Records[0].Status != questionbank.Completed || recovered.Conversation == nil {
		t.Fatalf("重试后题库完成态未恢复: session=%+v dirty=%v", recovered, dirty)
	}
	conversation := a.solver.SnapshotConversation()
	if len(conversation.Turns) != 1 || conversation.Turns[0].Assistant.Content != "answer" {
		t.Fatalf("重试后 Solver 上下文未同步恢复: %+v", conversation)
	}
	if len(a.conversationRefs) != 1 || a.conversationRefs[0].RecordID != recovered.Records[0].ID {
		t.Fatalf("重试后 App 引用账本未同步恢复: %+v", a.conversationRefs)
	}
}

func TestKeepContextOffClearsRefsAndStartupHistory(t *testing.T) {
	a, store, _, _ := nextExamApp(t)
	a.configManager = configuredSolveManager(t)
	a.llmService = llm.NewService(a.configManager.Get(), a.configManager)
	a.onConfigChanged(config.Config{KeepContext: false}, config.Config{KeepContext: true})
	if len(a.conversationRefs) != 0 || len(a.solver.SnapshotConversation().Turns) != 0 {
		t.Fatal("OFF did not clear refs and solver")
	}
	a.onConfigChanged(config.Config{KeepContext: true}, config.Config{KeepContext: false})
	a.restoreCurrentQuestionBankConversation()
	if len(a.conversationRefs) != 0 || len(a.solver.SnapshotConversation().Turns) != 0 {
		t.Fatal("history resurrected on restore")
	}
	current, _ := store.Snapshot()
	snapshot, err := store.ConversationSnapshotByID(current.ID)
	if err != nil || len(snapshot.Turns) != 0 {
		t.Fatalf("persisted refs not cleared: %+v %v", snapshot, err)
	}
}

type contextSwitchProvider struct {
	app   *App
	calls int
}

func (p *contextSwitchProvider) GenerateContentStream(_ context.Context, _ []llm.Message, _ llm.StreamCallback) (llm.Message, error) {
	p.calls++
	p.app.onConfigChanged(config.Config{KeepContext: false}, config.Config{KeepContext: true})
	p.app.onConfigChanged(config.Config{KeepContext: true}, config.Config{KeepContext: false})
	return llm.NewAssistantMessage("completed while context was switched"), nil
}
func (p *contextSwitchProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) {
	return llm.Message{}, nil
}
func (p *contextSwitchProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *contextSwitchProvider) TestChat(context.Context) error              { return nil }
func TestKeepContextSwitchDuringSolveDoesNotResurrectRefs(t *testing.T) {
	a, store, image, _ := nextExamApp(t)
	a.configManager = configuredSolveManager(t)
	if err := a.configManager.Patch(func(c *config.Config) { c.KeepContext = true }); err != nil {
		t.Fatal(err)
	}
	a.llmService = llm.NewService(a.configManager.Get(), a.configManager)
	provider := &contextSwitchProvider{app: a}
	a.solver.SetProvider(provider)
	epoch := a.conversationEpoch
	if !a.solveInternal(context.Background(), 99, []string{image}, 0, epoch) {
		t.Fatal("record should still complete")
	}
	if provider.calls != 1 || len(a.conversationRefs) != 0 || len(a.solver.SnapshotConversation().Turns) != 0 {
		t.Fatal("in-flight history resurrected")
	}
	current, _ := store.Snapshot()
	if current.Conversation == nil || len(current.Conversation.Turns) != 0 {
		t.Fatal("old refs persisted")
	}
}
