package app

import "testing"

func TestSolveAttemptTrackerRetainsOnlyFailedOrCancelledAttempt(t *testing.T) {
	tracker := newSolveAttemptTracker()
	images := []string{"image-one", "image-two"}
	tracker.Start(11, images, SolveModeNormal, "gen-solver", true, 7)
	images[0] = "mutated"
	target := desktopSolveTarget{epoch: 7, sessionID: "session-a", recordID: "record-a"}
	if !tracker.BindTarget(11, target) {
		t.Fatal("current task should accept target")
	}
	if !tracker.Finish(11, attemptFailed) {
		t.Fatal("failed current task should become retryable")
	}
	snapshot, code := tracker.RetrySnapshot("session-a", "record-a", 7)
	if code != "" || snapshot.screenshots[0] != "image-one" || !snapshot.keepContext {
		t.Fatalf("retry snapshot = %+v, code=%q", snapshot, code)
	}

	tracker.Start(12, []string{"new"}, SolveModeFast, "gen-solver", false, 7)
	tracker.BindTarget(12, desktopSolveTarget{epoch: 7, sessionID: "session-a", recordID: "record-b"})
	tracker.Finish(12, attemptCompleted)
	if _, code = tracker.RetrySnapshot("session-a", "record-b", 7); code != solveNoRetryAvailable {
		t.Fatalf("completed attempt must not remain retryable: %q", code)
	}
}

func TestSolveAttemptTrackerRejectsWrongTargetEpochAndOldTask(t *testing.T) {
	tracker := newSolveAttemptTracker()
	tracker.Start(21, []string{"image"}, SolveModeFast, "gen-solver", false, 9)
	target := desktopSolveTarget{epoch: 9, sessionID: "session-a", recordID: "record-a"}
	tracker.BindTarget(21, target)
	tracker.Finish(21, attemptCancelled)

	if _, code := tracker.RetrySnapshot("session-x", "record-a", 9); code != solveRetryTargetMismatch {
		t.Fatalf("wrong session code=%q", code)
	}
	if _, code := tracker.RetrySnapshot("session-a", "record-a", 10); code != solveRetryStale {
		t.Fatalf("wrong epoch code=%q", code)
	}

	tracker.Start(22, []string{"new-image"}, SolveModeNormal, "gen-step", true, 9)
	if tracker.MarkCancelling(21) {
		t.Fatal("old task must not cancel current attempt")
	}
	if tracker.Finish(21, attemptFailed) {
		t.Fatal("old task must not finish current attempt")
	}
	if tracker.current == nil || tracker.current.taskID != 22 || tracker.current.state != attemptRunning {
		t.Fatalf("old task mutated current attempt: %+v", tracker.current)
	}
}

func TestSolveAttemptTrackerCancellationIsIdempotent(t *testing.T) {
	tracker := newSolveAttemptTracker()
	tracker.Start(31, []string{"image"}, SolveModeNormal, "gen-solver", false, 1)
	if !tracker.MarkCancelling(31) || !tracker.MarkCancelling(31) {
		t.Fatal("cancelling the current task should be idempotent")
	}
	if !tracker.Finish(31, attemptCancelled) {
		t.Fatal("cancelled task should become retryable")
	}
}

func TestSolveAttemptTrackerInvalidationAndReturnedClone(t *testing.T) {
	tracker := newSolveAttemptTracker()
	tracker.Start(41, []string{"image"}, SolveModeNormal, "gen-solver", false, 1)
	tracker.BindTarget(41, desktopSolveTarget{epoch: 1, sessionID: "live", recordID: "task-41"})
	tracker.Finish(41, attemptFailed)
	snapshot, _ := tracker.RetrySnapshot("live", "task-41", 1)
	snapshot.screenshots[0] = "changed"
	again, _ := tracker.RetrySnapshot("live", "task-41", 1)
	if again.screenshots[0] != "image" {
		t.Fatal("returned retry snapshot aliases tracker memory")
	}
	tracker.Invalidate()
	if _, code := tracker.RetrySnapshot("live", "task-41", 1); code != solveNoRetryAvailable {
		t.Fatalf("invalidated retry code=%q", code)
	}
}
