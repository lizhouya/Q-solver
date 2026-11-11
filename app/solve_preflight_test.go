package app

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/solution"
	"Q-Solver/pkg/task"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestTriggerSendWaitsForResumeConfirmationBeforeConsumingScreenshots(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	manager := config.NewConfigManager()
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Patch(func(cfg *config.Config) {
		cfg.APIKey = "test-key"
		cfg.Model = "test-model"
		cfg.ResumePath = "resume.pdf"
		cfg.ResumeContent = ""
	}); err != nil {
		t.Fatal(err)
	}

	coordinator := task.NewTaskCoordinator()
	a := &App{
		configManager:    manager,
		taskManager:      coordinator,
		screenshotBuffer: []string{"pending-image"},
	}

	a.TriggerSend()

	if coordinator.HasInFlightTask() {
		t.Fatal("简历确认前不应启动解题任务")
	}
	if len(a.screenshotBuffer) != 1 || a.screenshotBuffer[0] != "pending-image" {
		t.Fatal("简历确认前不应消费待发送截图")
	}
}

type resumeModeBlockingProvider struct {
	started chan struct{}
	release chan struct{}
}

func (p *resumeModeBlockingProvider) GenerateContentStream(ctx context.Context, _ []llm.Message, _ llm.StreamCallback) (llm.Message, error) {
	close(p.started)
	select {
	case <-p.release:
		return llm.NewAssistantMessage("answer"), nil
	case <-ctx.Done():
		return llm.Message{}, ctx.Err()
	}
}
func (p *resumeModeBlockingProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) {
	return llm.Message{}, nil
}
func (p *resumeModeBlockingProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *resumeModeBlockingProvider) TestChat(context.Context) error              { return nil }

func TestQuickSendResumeConfirmationPreservesFastMode(t *testing.T) {
	manager := configuredSolveManager(t)
	if err := manager.Patch(func(cfg *config.Config) {
		cfg.ResumePath = "resume.pdf"
		cfg.ResumeContent = ""
	}); err != nil {
		t.Fatal(err)
	}
	provider := &resumeModeBlockingProvider{started: make(chan struct{}), release: make(chan struct{})}
	coordinator := task.NewTaskCoordinator()
	a := &App{
		configManager:    manager,
		taskManager:      coordinator,
		solver:           solution.NewSolver(provider),
		solveAttempts:    newSolveAttemptTracker(),
		screenshotBuffer: []string{"pending-image"},
	}

	a.TriggerQuickSend()
	if a.pendingSolveMode != SolveModeFast || coordinator.HasInFlightTask() {
		t.Fatalf("fast mode was not retained before confirmation: mode=%q", a.pendingSolveMode)
	}
	a.ConfirmSolveWithoutResume()
	select {
	case <-provider.started:
	case <-time.After(2 * time.Second):
		t.Fatal("confirmed fast solve did not start")
	}
	a.solveAttempts.mu.Lock()
	attempt := cloneSolveAttempt(a.solveAttempts.current)
	a.solveAttempts.mu.Unlock()
	if attempt.mode != SolveModeFast || attempt.normalDomainID == "gen-speed" {
		t.Fatalf("confirmed mode=%q normalDomainID=%q", attempt.mode, attempt.normalDomainID)
	}
	close(provider.release)
	deadline := time.Now().Add(2 * time.Second)
	for coordinator.HasInFlightTask() && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	if coordinator.HasInFlightTask() {
		t.Fatal("confirmed solve did not finish")
	}
}

func TestNeedsResumeConfirmation(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		want    bool
	}{
		{name: "absent resume", want: false},
		{name: "blank resume path", path: "  ", want: false},
		{name: "unparsed resume", path: "resume.pdf", want: true},
		{name: "blank parsed content", path: "resume.pdf", content: "  ", want: true},
		{name: "parsed resume", path: "resume.pdf", content: "parsed markdown", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := needsResumeConfirmation(test.path, test.content); got != test.want {
				t.Fatalf("needsResumeConfirmation() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestKeylessSolveInternalReachesLocalTemplateServer(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "" {
			t.Error("keyless request sent default authorization")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if messages, ok := body["messages"].([]any); !ok || len(messages) < 2 {
			t.Error("missing system and screenshot messages")
		}
		_, _ = w.Write([]byte(`{"answer":"local answer"}`))
	}))
	defer server.Close()
	manager := configuredSolveManager(t)
	if err := manager.Patch(func(c *config.Config) {
		c.APIKey = ""
		c.APIProtocol = "custom_json"
		c.BaseURL = server.URL + "/generate"
		c.RequestTemplate = `{"messages":"{{openai_messages}}"}`
		c.ResponseJSONPointer = "/answer"
	}); err != nil {
		t.Fatal(err)
	}
	provider, err := llm.NewProvider(manager.Get())
	if err != nil {
		t.Fatal(err)
	}
	a := &App{configManager: manager, solver: solution.NewSolver(provider)}
	cfg := manager.Get()
	run := solveRunRequest{screenshots: []string{"data:image/png;base64,aW1hZ2U="}, mode: SolveModeNormal, normalDomainID: cfg.DomainId, keepContext: cfg.KeepContext}
	if !a.solveInternal(context.Background(), 1, run, 0, 0) {
		t.Fatal("keyless solve did not complete")
	}
	if calls.Load() != 1 {
		t.Fatalf("requests=%d", calls.Load())
	}
}
