package solution

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
)

func TestSnapshotAndReplaceConversationDeepCopy(t *testing.T) {
	s := NewSolver(&fakeProvider{reply: "unused"})
	input := ConversationState{Turns: []ConversationTurn{{
		User:      llm.NewMultiPartMessage(llm.RoleUser, []llm.ContentPart{llm.ImagePart("image-1")}),
		Assistant: llm.NewAssistantMessage("answer-1"),
	}}}

	if err := s.ReplaceConversation(input); err != nil {
		t.Fatalf("ReplaceConversation 失败: %v", err)
	}

	input.Turns[0].User.Parts[0].Base64 = "mutated-input"
	input.Turns[0].Assistant.Content = "mutated-input"
	first := s.SnapshotConversation()
	if got := first.Turns[0].User.Parts[0].Base64; got != "image-1" {
		t.Fatalf("ReplaceConversation 应深拷贝输入，实际图片为 %q", got)
	}
	if got := first.Turns[0].Assistant.Content; got != "answer-1" {
		t.Fatalf("ReplaceConversation 应深拷贝输入，实际回答为 %q", got)
	}

	first.Turns[0].User.Parts[0].Base64 = "mutated-snapshot"
	first.Turns[0].Assistant.Content = "mutated-snapshot"
	second := s.SnapshotConversation()
	if got := second.Turns[0].User.Parts[0].Base64; got != "image-1" {
		t.Fatalf("SnapshotConversation 应深拷贝输出，实际图片为 %q", got)
	}
	if got := second.Turns[0].Assistant.Content; got != "answer-1" {
		t.Fatalf("SnapshotConversation 应深拷贝输出，实际回答为 %q", got)
	}
}

func TestReplaceConversationValidationIsAtomic(t *testing.T) {
	s := NewSolver(&fakeProvider{reply: "unused"})
	original := ConversationState{Turns: []ConversationTurn{{
		User:      llm.NewUserMessage("question"),
		Assistant: llm.NewAssistantMessage("answer"),
	}}}
	if err := s.ReplaceConversation(original); err != nil {
		t.Fatalf("准备历史失败: %v", err)
	}

	tests := []struct {
		name  string
		state ConversationState
	}{
		{
			name: "拒绝 system",
			state: ConversationState{Turns: []ConversationTurn{{
				User: llm.NewSystemMessage("system"), Assistant: llm.NewAssistantMessage("answer"),
			}}},
		},
		{
			name: "拒绝角色错序",
			state: ConversationState{Turns: []ConversationTurn{{
				User: llm.NewAssistantMessage("wrong"), Assistant: llm.NewUserMessage("wrong"),
			}}},
		},
		{
			name: "拒绝空 assistant",
			state: ConversationState{Turns: []ConversationTurn{{
				User: llm.NewUserMessage("question"), Assistant: llm.NewAssistantMessage(""),
			}}},
		},
		{
			name:  "拒绝超过最大轮数",
			state: ConversationState{Turns: make([]ConversationTurn, MaxConversationRounds+1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := s.ReplaceConversation(tt.state); err == nil {
				t.Fatal("ReplaceConversation 应拒绝非法历史")
			}
			if got := s.SnapshotConversation(); !reflect.DeepEqual(got, original) {
				t.Fatalf("非法替换不应改变原历史，实际 %+v", got)
			}
		})
	}
}

func TestSolveRebuildsCurrentSystemAroundRestoredTurns(t *testing.T) {
	fp := &fakeProvider{reply: "new-answer"}
	s := NewSolver(fp)
	if err := s.ReplaceConversation(ConversationState{Turns: []ConversationTurn{{
		User:      llm.NewUserMessage("old-question"),
		Assistant: llm.NewAssistantMessage("old-answer"),
	}}}); err != nil {
		t.Fatalf("准备历史失败: %v", err)
	}

	ok := s.Solve(context.Background(), Request{
		Config: config.Config{
			APIKey:        "k",
			KeepContext:   true,
			ResumeContent: "current-system-marker",
		},
		Screenshots: []string{"current-image"},
	}, Callbacks{})
	if !ok {
		t.Fatal("Solve 失败")
	}

	got := fp.received[0]
	wantRoles := []llm.Role{llm.RoleSystem, llm.RoleUser, llm.RoleAssistant, llm.RoleUser}
	if roles := rolesOf(got); !reflect.DeepEqual(roles, wantRoles) {
		t.Fatalf("发送角色序列错误: got %v want %v", roles, wantRoles)
	}
	if got[0].Content == "" || !contains(got[0].Content, "current-system-marker") {
		t.Fatal("每次 Solve 应使用当前配置重建 system")
	}
	for _, message := range s.chatHistory {
		if message.Role == llm.RoleSystem {
			t.Fatal("内部历史不得保存 system")
		}
	}
}

func contains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}

type blockingProvider struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	reply   string
}

func (p *blockingProvider) GenerateContentStream(ctx context.Context, messages []llm.Message, onChunk llm.StreamCallback) (llm.Message, error) {
	p.once.Do(func() { close(p.started) })
	select {
	case <-p.release:
		return llm.NewAssistantMessage(p.reply), nil
	case <-ctx.Done():
		return llm.Message{}, ctx.Err()
	}
}

func (p *blockingProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) {
	return llm.Message{}, nil
}
func (p *blockingProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *blockingProvider) TestChat(context.Context) error              { return nil }

func TestInFlightSolveDoesNotOverwriteReplacedConversation(t *testing.T) {
	provider := &blockingProvider{
		started: make(chan struct{}),
		release: make(chan struct{}),
		reply:   "stale-answer",
	}
	s := NewSolver(provider)
	done := make(chan bool, 1)
	go func() {
		done <- solveOnce(s, true, []string{"stale-question"})
	}()
	<-provider.started

	replacement := ConversationState{Turns: []ConversationTurn{{
		User:      llm.NewUserMessage("restored-question"),
		Assistant: llm.NewAssistantMessage("restored-answer"),
	}}}
	if err := s.ReplaceConversation(replacement); err != nil {
		t.Fatalf("ReplaceConversation 失败: %v", err)
	}
	close(provider.release)
	if ok := <-done; !ok {
		t.Fatal("模型请求本身应成功")
	}
	if got := s.SnapshotConversation(); !reflect.DeepEqual(got, replacement) {
		t.Fatalf("过期 Solve 不得覆盖替换后的历史，实际 %+v", got)
	}
}

func TestKeepFalseInvalidatesInFlightKeepTrueCommit(t *testing.T) {
	provider := &blockingProvider{
		started: make(chan struct{}),
		release: make(chan struct{}),
		reply:   "stale-answer",
	}
	s := NewSolver(provider)
	done := make(chan bool, 1)
	go func() {
		done <- solveOnce(s, true, []string{"stale-question"})
	}()
	<-provider.started

	// 第二次请求不必真的进入 Provider；KeepContext=false 在调用 Provider 前就必须清空并推进代际。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.Solve(ctx, Request{Config: config.Config{APIKey: "k", KeepContext: false}}, Callbacks{})
	close(provider.release)
	<-done

	if got := s.SnapshotConversation(); len(got.Turns) != 0 {
		t.Fatalf("KeepContext=false 后过期请求不得回写，实际 %+v", got)
	}
}
