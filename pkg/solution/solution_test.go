package solution

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
)

// =====================================================================
// fakeProvider：mock LLM Provider，记录每次 Solve 收到的消息列表，支持 success/error。
// =====================================================================

type fakeProvider struct {
	received [][]llm.Message // 每次 GenerateContentStream 收到的完整 messages
	reply    string          // 成功返回的 content
	err      error           // 非 nil 时模拟失败
}

func (f *fakeProvider) GenerateContentStream(ctx context.Context, messages []llm.Message, onChunk llm.StreamCallback) (llm.Message, error) {
	cp := make([]llm.Message, len(messages))
	copy(cp, messages)
	f.received = append(f.received, cp)
	if f.err != nil {
		return llm.Message{}, f.err
	}
	return llm.NewAssistantMessage(f.reply), nil
}

func (f *fakeProvider) GenerateContent(ctx context.Context, model string, messages []llm.Message) (llm.Message, error) {
	return llm.Message{}, nil
}
func (f *fakeProvider) GetModels(ctx context.Context) ([]string, error) { return nil, nil }
func (f *fakeProvider) TestChat(ctx context.Context) error              { return nil }

// =====================================================================
// helpers
// =====================================================================

// roles 返回消息的角色序列。
func rolesOf(ms []llm.Message) []llm.Role {
	out := make([]llm.Role, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Role)
	}
	return out
}

// firstImageB64 返回消息中第一张图片 base64，无则 ""。
func firstImageB64(m llm.Message) string {
	for _, p := range m.Parts {
		if p.Type == llm.ContentImage {
			return p.Base64
		}
	}
	return ""
}

func solveOnce(s *Solver, keep bool, shots []string) bool {
	return s.Solve(context.Background(), Request{
		Config:      config.Config{APIKey: "k", KeepContext: keep},
		Screenshots: shots,
	}, Callbacks{})
}

// =====================================================================
// T-KC-01 KeepContext=false：连续两次 Solve，第二次请求不含第一次问答。
// =====================================================================
func TestTKC01KeepFalseIndependent(t *testing.T) {
	fp := &fakeProvider{reply: "A1"}
	s := NewSolver(fp)

	solveOnce(s, false, []string{"b1"})
	solveOnce(s, false, []string{"b2"})

	if len(fp.received) != 2 {
		t.Fatalf("应收到 2 次请求，实际 %d", len(fp.received))
	}
	if got, want := rolesOf(fp.received[1]), []llm.Role{llm.RoleSystem, llm.RoleUser}; !reflect.DeepEqual(got, want) {
		t.Fatalf("第二次请求应为 [system,user]，实际 %v", got)
	}
	if len(s.chatHistory) != 0 {
		t.Fatalf("KeepContext=false 后 history 应为空，实际 %d 条", len(s.chatHistory))
	}
}

// =====================================================================
// T-KC-02 KeepContext=true：第二次请求携 history（Sys,U1,A1,U2）。
// =====================================================================
func TestTKC02KeepTrueCarriesHistory(t *testing.T) {
	fp := &fakeProvider{reply: "A1"}
	s := NewSolver(fp)

	solveOnce(s, true, []string{"b1"})
	solveOnce(s, true, []string{"b2"})

	got := rolesOf(fp.received[1])
	want := []llm.Role{llm.RoleSystem, llm.RoleUser, llm.RoleAssistant, llm.RoleUser}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("第二次请求应为 [system,user,assistant,user]，实际 %v", got)
	}
	// 历史 user 第一条截图应为 b1（不放当前 b2 提前进 history）
	if img := firstImageB64(fp.received[1][1]); img != "b1" {
		t.Fatalf("历史 user 截图应为 b1，实际 %q", img)
	}
}

// =====================================================================
// T-KC-03：成功两轮后 history 为 [Sys,U0,A0,U1,A1]。
// =====================================================================
func TestTKC03HistoryAfterTwoRounds(t *testing.T) {
	fp := &fakeProvider{reply: "A"}
	s := NewSolver(fp)

	solveOnce(s, true, []string{"b1"})
	solveOnce(s, true, []string{"b2"})

	got := rolesOf(s.chatHistory)
	want := []llm.Role{llm.RoleSystem, llm.RoleUser, llm.RoleAssistant, llm.RoleUser, llm.RoleAssistant}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("history 应为 [system,u1,a1,u2,a2]，实际 %v", got)
	}
}

// =====================================================================
// T-KC-04：超过 MaxConversationRounds 后，仅保留最近轮次。
// =====================================================================
func TestTKC04TrimKeepsRecentRounds(t *testing.T) {
	fp := &fakeProvider{reply: "A"}
	s := NewSolver(fp)
	totalRounds := MaxConversationRounds + 2

	for i := 0; i < totalRounds; i++ {
		if !solveOnce(s, true, []string{fmt.Sprintf("b%d", i)}) {
			t.Fatalf("第 %d 轮失败", i)
		}
	}

	wantLen := 1 + MaxConversationRounds*2
	if len(s.chatHistory) != wantLen {
		t.Fatalf("history 应为 system + 最近 %d 轮，共 %d 条；实际 %d", MaxConversationRounds, wantLen, len(s.chatHistory))
	}
	if s.chatHistory[0].Role != llm.RoleSystem {
		t.Fatalf("history[0] 应为 system，实际 %v", s.chatHistory[0].Role)
	}

	firstRetainedRound := totalRounds - MaxConversationRounds
	for i := 0; i < MaxConversationRounds; i++ {
		userIndex := 1 + i*2
		assistantIndex := userIndex + 1
		if s.chatHistory[userIndex].Role != llm.RoleUser || s.chatHistory[assistantIndex].Role != llm.RoleAssistant {
			t.Fatalf("保留轮次 %d 应为 [user,assistant]，实际 %v", i, rolesOf(s.chatHistory[userIndex:assistantIndex+1]))
		}
		wantImage := fmt.Sprintf("b%d", firstRetainedRound+i)
		if got := firstImageB64(s.chatHistory[userIndex]); got != wantImage {
			t.Fatalf("保留轮次 %d 的截图应为 %q，实际 %q", i, wantImage, got)
		}
	}
}

// trimChatHistory 直接单测：system 恒为位置 0，短 history 不裁剪。
func TestTKC04TrimDirect(t *testing.T) {
	s := &Solver{}
	s.chatHistory = []llm.Message{
		llm.NewSystemMessage("sys"),
		llm.NewUserMessage("u1"), llm.NewAssistantMessage("a1"),
	}
	s.trimChatHistory()
	if got := rolesOf(s.chatHistory); !reflect.DeepEqual(got, []llm.Role{llm.RoleSystem, llm.RoleUser, llm.RoleAssistant}) {
		t.Fatalf("无裁剪时保持 [sys,u1,a1]，实际 %v", got)
	}
}

// =====================================================================
// T-KC-05：ClearHistory 后旧上下文消失。
// =====================================================================
func TestTKC05ClearHistory(t *testing.T) {
	fp := &fakeProvider{reply: "A1"}
	s := NewSolver(fp)

	solveOnce(s, true, []string{"b1"})
	if len(s.chatHistory) == 0 {
		t.Fatal("第一轮后 history 不应为空")
	}
	s.ClearHistory()
	if len(s.chatHistory) != 0 {
		t.Fatalf("ClearHistory 后应为空，实际 %d 条", len(s.chatHistory))
	}

	solveOnce(s, true, []string{"b2"})
	if got := rolesOf(fp.received[1]); !reflect.DeepEqual(got, []llm.Role{llm.RoleSystem, llm.RoleUser}) {
		t.Fatalf("ClearHistory 后第二次请求应为 [system,user]，实际 %v", got)
	}
}

// =====================================================================
// T-KC-06：Provider error，失败轮不写 history。
// =====================================================================
func TestTKC06ErrorDoesNotWriteHistory(t *testing.T) {
	fp := &fakeProvider{reply: "A1"}
	s := NewSolver(fp)

	solveOnce(s, true, []string{"b1"}) // 成功 1 轮 -> [sys,u1,a1]

	fp.err = errTestInjected
	if solveOnce(s, true, []string{"b2"}) {
		t.Fatal("失败轮 Solve 应返回 false")
	}
	fp.err = nil

	if got := rolesOf(s.chatHistory); !reflect.DeepEqual(got, []llm.Role{llm.RoleSystem, llm.RoleUser, llm.RoleAssistant}) {
		t.Fatalf("失败后 history 应保持 [sys,u1,a1]，实际 %v", got)
	}
}

func TestTKC06EmptyResponseDoesNotWriteHistory(t *testing.T) {
	fp := &fakeProvider{reply: "A1"}
	s := NewSolver(fp)

	solveOnce(s, true, []string{"b1"})
	fp.reply = ""
	if solveOnce(s, true, []string{"b2"}) {
		t.Fatal("空响应轮 Solve 应返回 false")
	}

	if got := rolesOf(s.chatHistory); !reflect.DeepEqual(got, []llm.Role{llm.RoleSystem, llm.RoleUser, llm.RoleAssistant}) {
		t.Fatalf("空响应后 history 应保持 [sys,u1,a1]，实际 %v", got)
	}
}

var errTestInjected = &testErrInjected{}

type testErrInjected struct{}

func (e *testErrInjected) Error() string { return "mock injected failure" }

// =====================================================================
// T-KC-07：System Prompt 从 A 更新为 B，Provider 仅收到唯一的当前 system。
// =====================================================================
func TestTKC07EnsurePromptReplacesSystem(t *testing.T) {
	fp := &fakeProvider{reply: "A1"}
	s := NewSolver(fp)

	request := func(resumeContent, image string) bool {
		return s.Solve(context.Background(), Request{
			Config: config.Config{
				APIKey:        "k",
				KeepContext:   true,
				ResumeContent: resumeContent,
			},
			Screenshots: []string{image},
		}, Callbacks{})
	}

	if !request("prompt-marker-A", "b1") {
		t.Fatal("使用 System A 的首轮 Solve 失败")
	}
	if !request("prompt-marker-B", "b2") {
		t.Fatal("使用 System B 的第二轮 Solve 失败")
	}

	if len(fp.received) != 2 {
		t.Fatalf("应收到 2 次请求，实际 %d", len(fp.received))
	}
	second := fp.received[1]
	if got, want := rolesOf(second), []llm.Role{llm.RoleSystem, llm.RoleUser, llm.RoleAssistant, llm.RoleUser}; !reflect.DeepEqual(got, want) {
		t.Fatalf("第二轮请求应为 [system,u1,a1,u2]，实际 %v", got)
	}

	systemCount := 0
	for _, message := range second {
		if message.Role != llm.RoleSystem {
			continue
		}
		systemCount++
		if !strings.Contains(message.Content, "prompt-marker-B") {
			t.Fatalf("Provider 应收到当前 System B，实际未包含 B marker")
		}
		if strings.Contains(message.Content, "prompt-marker-A") {
			t.Fatal("Provider 不应收到旧 System A")
		}
	}
	if systemCount != 1 {
		t.Fatalf("第二轮请求应仅含 1 条 system，实际 %d", systemCount)
	}
}

// =====================================================================
// T-KC-08：true→false，旧 history 被清除。
// =====================================================================
func TestTKC08ToggleTrueToFalse(t *testing.T) {
	fp := &fakeProvider{reply: "A1"}
	s := NewSolver(fp)

	solveOnce(s, true, []string{"b1"})
	if len(s.chatHistory) == 0 {
		t.Fatal("切换到 false 前应存在历史")
	}
	solveOnce(s, false, []string{"b2"})

	if got := rolesOf(fp.received[1]); !reflect.DeepEqual(got, []llm.Role{llm.RoleSystem, llm.RoleUser}) {
		t.Fatalf("false 请求应为 [system,user]，实际 %v", got)
	}
	if len(s.chatHistory) != 0 {
		t.Fatalf("false 后 history 应清空，实际 %d 条", len(s.chatHistory))
	}
}

// =====================================================================
// T-KC-09：false→true，不恢复已清除轮次。
// =====================================================================
func TestTKC09ToggleFalseToTrue(t *testing.T) {
	fp := &fakeProvider{reply: "A2"}
	s := NewSolver(fp)

	solveOnce(s, false, []string{"b1"}) // false：无历史

	solveOnce(s, true, []string{"b2"}) // true：从当前轮重建

	if got := rolesOf(fp.received[1]); !reflect.DeepEqual(got, []llm.Role{llm.RoleSystem, llm.RoleUser}) {
		t.Fatalf("false→true 第二轮应视为全新 [system,user]，实际 %v", got)
	}
}

// =====================================================================
// T-KC-10：multimodal user message，历史保留 Parts/截图。
// =====================================================================
func TestTKC10MultimodalPreserved(t *testing.T) {
	fp := &fakeProvider{reply: "A1"}
	s := NewSolver(fp)

	b64 := "data:image/png;base64,BBB=="
	if !solveOnce(s, true, []string{b64}) {
		t.Fatal("多模态首轮失败")
	}

	u := s.chatHistory[1]
	if u.Role != llm.RoleUser {
		t.Fatalf("history[1] 应为 user，实际 %v", u.Role)
	}
	if len(u.Parts) != 1 || u.Parts[0].Type != llm.ContentImage {
		t.Fatalf("history user 应保留 multipart image，实际 %+v", u.Parts)
	}
	if got := u.Parts[0].Base64; got != b64 {
		t.Fatalf("history user 截图应保持原值，实际 %q", got)
	}
}
