package app

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/task"
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
