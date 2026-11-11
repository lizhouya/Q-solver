package app

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/questionbank"
	"Q-Solver/pkg/solution"
	"Q-Solver/pkg/task"
)

type scriptedSolveProvider struct {
	mu          sync.Mutex
	results     []error
	calls       int
	imageInputs int
	secondImage string
}

func (p *scriptedSolveProvider) GenerateContentStream(_ context.Context, messages []llm.Message, onChunk llm.StreamCallback) (llm.Message, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	current := messages[len(messages)-1]
	for _, part := range current.Parts {
		if part.Type == llm.ContentImage {
			p.imageInputs++
			if p.calls == 2 {
				p.secondImage = part.Base64
			}
		}
	}
	resultErr := p.results[p.calls-1]
	if resultErr != nil {
		return llm.Message{}, resultErr
	}
	if onChunk != nil {
		onChunk(llm.StreamChunk{Type: llm.ChunkContent, Content: "answer"})
	}
	return llm.NewAssistantMessage("answer"), nil
}

func (p *scriptedSolveProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) {
	return llm.Message{}, nil
}
func (p *scriptedSolveProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *scriptedSolveProvider) TestChat(context.Context) error              { return nil }

type blockingSolveProvider struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingSolveProvider() *blockingSolveProvider {
	return &blockingSolveProvider{started: make(chan struct{}), release: make(chan struct{})}
}

func (p *blockingSolveProvider) GenerateContentStream(ctx context.Context, _ []llm.Message, _ llm.StreamCallback) (llm.Message, error) {
	p.once.Do(func() { close(p.started) })
	select {
	case <-p.release:
		return llm.NewAssistantMessage("answer"), nil
	case <-ctx.Done():
		return llm.Message{}, ctx.Err()
	}
}
func (p *blockingSolveProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) {
	return llm.Message{}, nil
}
func (p *blockingSolveProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *blockingSolveProvider) TestChat(context.Context) error              { return nil }

func waitForNoInFlight(t *testing.T, coordinator *task.TaskCoordinator) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for coordinator.HasInFlightTask() && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	if coordinator.HasInFlightTask() {
		t.Fatal("solve task did not finish")
	}
}

func retryableAppWithoutQuestionBank(t *testing.T) (*App, *blockingSolveProvider) {
	t.Helper()
	manager := configuredSolveManager(t)
	provider := newBlockingSolveProvider()
	app := &App{
		configManager: manager,
		solver:        solution.NewSolver(provider),
		taskManager:   task.NewTaskCoordinator(),
		solveAttempts: newSolveAttemptTracker(),
	}
	app.solveAttempts.Start(1, []string{"image"}, SolveModeNormal, "gen-solver", false, 0)
	app.solveAttempts.BindTarget(1, desktopSolveTarget{sessionID: "live", recordID: "task-1"})
	app.solveAttempts.Finish(1, attemptFailed)
	return app, provider
}

func TestRetrySolveReusesBackendImagesAndSameQuestionBankRecord(t *testing.T) {
	app, store, image := archiveApp(t)
	app.configManager = configuredSolveManager(t)
	provider := &scriptedSolveProvider{results: []error{errors.New("synthetic network failure"), nil}}
	app.solver = solution.NewSolver(provider)
	app.taskManager = task.NewTaskCoordinator()
	app.solveAttempts = newSolveAttemptTracker()

	start := app.startSolve(solveStartRequest{mode: SolveModeNormal, screenshots: []string{image}})
	if !start.Success {
		t.Fatalf("start result=%+v", start)
	}
	waitForNoInFlight(t, app.taskManager)
	failed, _ := store.Snapshot()
	if len(failed.Records) != 1 || failed.Records[0].Status != questionbank.Failed {
		t.Fatalf("failed=%+v", failed)
	}

	result := app.RetrySolve(failed.ID, failed.Records[0].ID)
	if !result.Success {
		t.Fatalf("retry result=%+v", result)
	}
	waitForNoInFlight(t, app.taskManager)
	completed, _ := store.Snapshot()
	if len(completed.Records) != 1 || completed.Records[0].ID != failed.Records[0].ID || completed.Records[0].Status != questionbank.Completed {
		t.Fatalf("retry duplicated record: %+v", completed.Records)
	}
	if provider.imageInputs != 2 || provider.secondImage != image {
		t.Fatal("retry did not reuse backend image")
	}
}

func TestRetrySolveRejectsWrongTargetAndConcurrentDoubleClick(t *testing.T) {
	app, provider := retryableAppWithoutQuestionBank(t)
	defer close(provider.release)
	if result := app.RetrySolve("wrong", "target"); result.Code != solveRetryTargetMismatch {
		t.Fatalf("result=%+v", result)
	}
	first := app.RetrySolve("live", "task-1")
	if !first.Success {
		t.Fatalf("first=%+v", first)
	}
	<-provider.started
	second := app.RetrySolve("live", "task-1")
	if second.Code != solveTaskRunning {
		t.Fatalf("second=%+v", second)
	}
}

func TestRetrySolveStaleAfterConversationEpochChange(t *testing.T) {
	app, provider := retryableAppWithoutQuestionBank(t)
	defer close(provider.release)
	app.conversationEpoch++
	result := app.RetrySolve("live", "task-1")
	if result.Code != solveRetryStale {
		t.Fatalf("result=%+v", result)
	}
}

func TestDesktopSolveEventIncludesModeAndRetryAuthority(t *testing.T) {
	target := desktopSolveTarget{epoch: 5, sessionID: "session", recordID: "record"}
	event := newDesktopSolveEvent(target, SolveModeFast, true, "solution-error", "network")
	if event.Mode != "fast" || !event.CanRetry || event.Event != "solution-error" {
		t.Fatalf("event=%+v", event)
	}
	success := newDesktopSolveEvent(target, SolveModeNormal, false, "solution", "answer")
	if success.Mode != "normal" || success.CanRetry {
		t.Fatalf("success event=%+v", success)
	}
}

func seedRetryableAttempt(app *App, epoch uint64) {
	app.solveAttempts = newSolveAttemptTracker()
	app.solveAttempts.Start(101, []string{"old-image"}, SolveModeNormal, "gen-solver", false, epoch)
	app.solveAttempts.BindTarget(101, desktopSolveTarget{epoch: epoch, sessionID: "old-session", recordID: "old-record"})
	app.solveAttempts.Finish(101, attemptFailed)
}

func assertRetryInvalidated(t *testing.T, app *App, epoch uint64) {
	t.Helper()
	if _, code := app.solveAttempts.RetrySnapshot("old-session", "old-record", epoch); code != solveNoRetryAvailable {
		t.Fatalf("old retry snapshot survived boundary: %q", code)
	}
}

func TestScreenshotBoundariesInvalidateRetrySnapshot(t *testing.T) {
	app := &App{solveAttempts: newSolveAttemptTracker(), screenshotBuffer: []string{"pending"}}
	seedRetryableAttempt(app, 4)
	app.conversationMu.Lock()
	app.appendScreenshotLocked("new-image")
	app.conversationMu.Unlock()
	assertRetryInvalidated(t, app, 4)

	seedRetryableAttempt(app, 4)
	app.ClearScreenshots()
	assertRetryInvalidated(t, app, 4)
}
