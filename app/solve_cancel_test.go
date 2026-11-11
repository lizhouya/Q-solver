package app

import (
	"testing"

	"Q-Solver/pkg/solution"
	"Q-Solver/pkg/task"
)

func TestCancelSolveWaitsForInFlightCompletionAndKeepsRetry(t *testing.T) {
	manager := configuredSolveManager(t)
	provider := newBlockingSolveProvider()
	app := &App{
		configManager: manager,
		solver:        solution.NewSolver(provider),
		taskManager:   task.NewTaskCoordinator(),
		solveAttempts: newSolveAttemptTracker(),
	}
	result := app.startSolve(solveStartRequest{mode: SolveModeNormal, screenshots: []string{"image"}})
	if !result.Success {
		t.Fatalf("start=%+v", result)
	}
	<-provider.started

	first := app.CancelSolve()
	second := app.CancelSolve()
	if !first.Success || first.Code != solveCancelRequested {
		t.Fatalf("first=%+v", first)
	}
	if !second.Success || second.Code != solveCancelRequested {
		t.Fatalf("idempotent second=%+v", second)
	}
	if !app.taskManager.HasInFlightTask() {
		t.Fatal("cancel must not erase in-flight task")
	}
	if result := app.RetrySolve("live", "task-1"); result.Code != solveTaskDraining {
		t.Fatalf("retry before unwind=%+v", result)
	}
	waitForNoInFlight(t, app.taskManager)
	if _, code := app.solveAttempts.RetrySnapshot("live", "task-1", app.conversationEpoch); code != "" {
		t.Fatalf("cancelled attempt not retryable: %q", code)
	}
}

func TestCancelSolveWithoutRunningTaskReturnsSafeCode(t *testing.T) {
	app := &App{taskManager: task.NewTaskCoordinator(), solveAttempts: newSolveAttemptTracker()}
	result := app.CancelSolve()
	if result.Success || result.Code != solveNoRunningTask || result.Message != safeSolveControlMessage(solveNoRunningTask) {
		t.Fatalf("result=%+v", result)
	}
}
