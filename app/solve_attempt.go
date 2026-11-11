package app

import "sync"

type SolveMode string

const (
	SolveModeNormal SolveMode = "normal"
	SolveModeFast   SolveMode = "fast"
)

type solveAttemptState string

const (
	attemptRunning    solveAttemptState = "running"
	attemptCancelling solveAttemptState = "cancelling"
	attemptFailed     solveAttemptState = "failed"
	attemptCancelled  solveAttemptState = "cancelled"
	attemptCompleted  solveAttemptState = "completed"
)

const (
	solveNoRetryAvailable    = "solve_no_retry_available"
	solveRetryTargetMismatch = "solve_retry_target_mismatch"
	solveRetryStale          = "solve_retry_stale"
	solveTaskRunning         = "solve_task_running"
	solveTaskDraining        = "solve_task_draining"
	solveRetryPrepareFailed  = "solve_retry_prepare_failed"
	solveCancelRequested     = "solve_cancel_requested"
	solveNoRunningTask       = "solve_no_running_task"
)

type SolveControlResult struct {
	Success bool   `json:"success"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func solveControlSuccess(code, message string) SolveControlResult {
	return SolveControlResult{Success: true, Code: code, Message: message}
}

func solveControlFailure(code, message string) SolveControlResult {
	return SolveControlResult{Success: false, Code: code, Message: message}
}

func safeSolveControlMessage(code string) string {
	switch code {
	case solveNoRetryAvailable:
		return "当前没有可重试的题目"
	case solveRetryTargetMismatch, solveRetryStale:
		return "该题目已过期，请重新截图"
	case solveTaskRunning:
		return "当前解题仍在进行"
	case solveTaskDraining:
		return "正在结束上一次解题，请稍候"
	case solveRetryPrepareFailed:
		return "重试准备失败，原记录未修改"
	case solveNoRunningTask:
		return "当前没有正在进行的解题"
	default:
		return "解题控制操作失败"
	}
}

type solveAttempt struct {
	taskID            int64
	target            desktopSolveTarget
	screenshots       []string
	mode              SolveMode
	normalDomainID    string
	keepContext       bool
	conversationEpoch uint64
	state             solveAttemptState
}

type solveAttemptTracker struct {
	mu      sync.Mutex
	current *solveAttempt
}

func newSolveAttemptTracker() *solveAttemptTracker {
	return &solveAttemptTracker{}
}

func cloneSolveAttempt(attempt *solveAttempt) solveAttempt {
	cloned := *attempt
	cloned.screenshots = append([]string(nil), attempt.screenshots...)
	return cloned
}

func (t *solveAttemptTracker) Start(taskID int64, screenshots []string, mode SolveMode, normalDomainID string, keepContext bool, conversationEpoch uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current = &solveAttempt{
		taskID:            taskID,
		screenshots:       append([]string(nil), screenshots...),
		mode:              mode,
		normalDomainID:    normalDomainID,
		keepContext:       keepContext,
		conversationEpoch: conversationEpoch,
		state:             attemptRunning,
	}
}

func (t *solveAttemptTracker) BindTarget(taskID int64, target desktopSolveTarget) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.current == nil || t.current.taskID != taskID || t.current.state != attemptRunning {
		return false
	}
	t.current.target = target
	return true
}

func (t *solveAttemptTracker) MarkCancelling(taskID int64) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.current == nil || t.current.taskID != taskID {
		return false
	}
	switch t.current.state {
	case attemptRunning:
		t.current.state = attemptCancelling
		return true
	case attemptCancelling:
		return true
	default:
		return false
	}
}

func (t *solveAttemptTracker) Finish(taskID int64, terminal solveAttemptState) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.current == nil || t.current.taskID != taskID {
		return false
	}
	switch terminal {
	case attemptCompleted:
		t.current = nil
	case attemptFailed, attemptCancelled:
		t.current.state = terminal
	default:
		return false
	}
	return true
}

func (t *solveAttemptTracker) RetrySnapshot(sessionID, recordID string, conversationEpoch uint64) (solveAttempt, string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.current == nil || (t.current.state != attemptFailed && t.current.state != attemptCancelled) {
		return solveAttempt{}, solveNoRetryAvailable
	}
	if t.current.target.sessionID != sessionID || t.current.target.recordID != recordID {
		return solveAttempt{}, solveRetryTargetMismatch
	}
	if t.current.conversationEpoch != conversationEpoch {
		return solveAttempt{}, solveRetryStale
	}
	return cloneSolveAttempt(t.current), ""
}

func (t *solveAttemptTracker) Invalidate() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current = nil
}
