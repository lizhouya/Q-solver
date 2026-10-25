package task

import (
	"context"
	"sync"
	"testing"
)

func TestHasInFlightTaskRemainsTrueAfterCancelUntilComplete(t *testing.T) {
	tc := NewTaskCoordinator()
	ctx, taskID := tc.StartTask("cancelled")

	if !tc.HasInFlightTask() {
		t.Fatal("started task should be in flight")
	}
	if !tc.CancelCurrentTask() {
		t.Fatal("CancelCurrentTask should cancel the current task")
	}
	if tc.HasRunningTask() {
		t.Fatal("cancelled current task should not be reported as running")
	}
	if !tc.HasInFlightTask() {
		t.Fatal("cancelled task should remain in flight until CompleteTask")
	}
	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			t.Fatalf("cancelled task context error = %v, want %v", ctx.Err(), context.Canceled)
		}
	default:
		t.Fatal("CancelCurrentTask should cancel the task context")
	}

	tc.CompleteTask(taskID)
	if tc.HasInFlightTask() {
		t.Fatal("completed task should no longer be in flight")
	}
}

func TestReplacedTaskRemainsInFlightUntilItCompletes(t *testing.T) {
	tc := NewTaskCoordinator()
	oldCtx, oldTaskID := tc.StartTask("old")
	_, newTaskID := tc.StartTask("new")

	select {
	case <-oldCtx.Done():
	case <-context.Background().Done():
		t.Fatal("unreachable")
	default:
		t.Fatal("starting a new task should cancel the previous task context")
	}
	if !tc.HasRunningTask() {
		t.Fatal("replacement task should be running")
	}
	if !tc.HasInFlightTask() {
		t.Fatal("started tasks should be in flight")
	}

	tc.CompleteTask(newTaskID)
	if tc.HasRunningTask() {
		t.Fatal("completed current task should not be reported as running")
	}
	if !tc.HasInFlightTask() {
		t.Fatal("replaced task should remain in flight after current task completes")
	}

	tc.CompleteTask(oldTaskID)
	if tc.HasInFlightTask() {
		t.Fatal("all completed tasks should be removed from in-flight tracking")
	}
}

func TestCompleteTaskIsIdempotentAndDoesNotRemoveAnotherTask(t *testing.T) {
	tc := NewTaskCoordinator()
	_, firstTaskID := tc.StartTask("first")

	tc.CompleteTask(firstTaskID)
	tc.CompleteTask(firstTaskID)
	if tc.HasInFlightTask() {
		t.Fatal("duplicate completion should leave the completed task removed")
	}

	_, secondTaskID := tc.StartTask("second")
	tc.CompleteTask(firstTaskID)
	if !tc.HasInFlightTask() {
		t.Fatal("duplicate completion of an old task must not remove another task")
	}
	if !tc.IsTaskRunning(secondTaskID) {
		t.Fatal("duplicate completion of an old task must not affect the current task")
	}

	tc.CompleteTask(secondTaskID)
	if tc.HasInFlightTask() {
		t.Fatal("second task should be removed after completion")
	}
}

func TestTaskCoordinatorConcurrentStartAndComplete(t *testing.T) {
	const taskCount = 128

	tc := NewTaskCoordinator()
	ids := make(chan int64, taskCount)
	var startWG sync.WaitGroup
	startWG.Add(taskCount)
	for i := 0; i < taskCount; i++ {
		go func() {
			defer startWG.Done()
			_, taskID := tc.StartTask("concurrent")
			ids <- taskID
		}()
	}
	startWG.Wait()
	close(ids)

	seen := make(map[int64]struct{}, taskCount)
	for taskID := range ids {
		if _, exists := seen[taskID]; exists {
			t.Fatalf("duplicate task ID %d", taskID)
		}
		seen[taskID] = struct{}{}
	}
	if len(seen) != taskCount {
		t.Fatalf("started %d unique tasks, want %d", len(seen), taskCount)
	}
	if !tc.HasInFlightTask() {
		t.Fatal("started tasks should be in flight before completion")
	}

	var completeWG sync.WaitGroup
	completeWG.Add(len(seen) * 2)
	for taskID := range seen {
		taskID := taskID
		for i := 0; i < 2; i++ {
			go func() {
				defer completeWG.Done()
				tc.CompleteTask(taskID)
				_ = tc.HasInFlightTask()
				_ = tc.HasRunningTask()
			}()
		}
	}
	completeWG.Wait()

	if tc.HasInFlightTask() {
		t.Fatal("all concurrently completed tasks should be removed")
	}
	if tc.HasRunningTask() {
		t.Fatal("current task should not be running after completion")
	}
}
