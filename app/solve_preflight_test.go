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
	"sync/atomic"
	"testing"
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
	if !a.solveInternal(context.Background(), 1, []string{"data:image/png;base64,aW1hZ2U="}, 0, 0) {
		t.Fatal("keyless solve did not complete")
	}
	if calls.Load() != 1 {
		t.Fatalf("requests=%d", calls.Load())
	}
}
