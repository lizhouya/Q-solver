# Change 038 解题重试、中断与快速模式 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立后端权威的单槽解题尝试生命周期，使失败/中断轮次可复用原图原位重试，支持真实取消当前请求，并增加不修改持久场景的快速解题快捷键。

**Architecture:** `app` 持有一个不落盘的 `solveAttemptTracker`，普通、快速和重试共用同一启动管线；题库通过原子 `RestartRecord` 重用同一 Record。取消由 context 贯穿 Provider，并以独立 `solution-cancelled` 终态路由到 Desktop/Mobile；前端只提交轮次身份，不回传图片或配置。

**Tech Stack:** Go 1.24.1+ module（当前环境 Go 1.27.1）、Wails v2.11.0、Vue 3、Pinia、Vite 4.5.14、Node.js 22+；不新增依赖。

**Spec:** `_dev/changes/038-solve-retry-cancel-fast-mode/design.md`

## Global Constraints

- 遵守 `AGENTS.md > CLAUDE.md > Change 038`；仅在 `dev/qsolver-custom` 工作，禁止修改 `main`。
- 每个任务先写失败测试并运行确认失败原因正确，再写最小实现；不得先写实现后补测试。
- 不新增依赖，不升级 Go、Wails、Vue、Pinia、Vite、Node、LLM SDK 或 runtime；bindings 和候选固定 Wails v2.11.0。
- 不读取、打印、复制或修改正式用户配置；Key、Token、Header、URL、截图内容和题库路径实值不得进入日志、文档或提交信息。
- Retry API 不接受图片、Config、Prompt、文件路径或凭据；Mobile 不新增控制端点。
- 不自动重试；每次重试由用户显式触发并可能产生新的模型调用费用。
- `build/bin/**` 保持 ignored，不使用 `git add -f`；`build/bin/releases/**` 没有 Release 授权，不得改变。
- 当前未授权 commit、push、稳定版晋升、Tag、Release 或真实外部 API；所有任务停在未暂存检查点。
- 每次 Git 检查前执行 `git status`、`git branch`、`git remote -v`、`git diff`；提交边界只记录，不执行 `git add` 或 `git commit`。
- Windows race 若继续在启动阶段返回 `0xc0000139`，记录为 `BLOCKED`，不能用普通测试结果替代。

---

## 文件责任图

| 文件 | 责任 |
|---|---|
| 新 `app/solve_attempt.go` | SolveMode、attempt 状态、深拷贝、目标绑定、终态和 stale 校验 |
| 新 `app/solve_attempt_test.go` | 单槽、并发、双击、失效和敏感字段边界 |
| `app/solve.go` | 普通/快速/重试统一启动、取消入口、in-flight 屏障和截图边界失效 |
| `app/app.go`、`questionbank_next_exam.go`、`questionbank_continuation.go`、`questionbank.go` | 配置/题库会话代际变化时失效旧 retry snapshot |
| `app/app.go` | tracker 所有权和生命周期装配 |
| `app/questionbank_solve_capture.go` | 首次 Begin 与 Retry Restart 的 capture 选择 |
| `pkg/questionbank/store.go` | terminal Record 原子重启与同 Record 完成 |
| `pkg/solution/solution.go` | `solution-cancelled` 终态，不提交取消轮 KeepContext |
| `pkg/config/config.go` / `manager.go` | 新 action 默认值与旧配置冲突安全迁移 |
| `pkg/shortcut/delegate.go` / `service.go` | `send_fast`、`cancel_solve` FIFO 分发 |
| `frontend/src/stores/solution.js` | typed retry/cancel/fast 状态、同轮重置、按钮幂等 |
| 新 `frontend/solve_retry_cancel_fast_regression_test.mjs` | Desktop 状态机与静态 UI 回归 |
| 新 `frontend/src/components/SolveControls.vue` | 运行中取消和失败轮重试按钮 |
| `frontend/src/components/SolveView.vue` / `ErrorView.vue` | 控制组件接入与 cancelled 文案 |
| `app/mobile.go`、`pkg/mobile/**` | mode/cancelled Snapshot 与安全快捷键提示 |
| `frontend/wailsjs/go/app/**`、`models.ts` | Wails v2.11.0 生成 bindings |
| `README.md` / `README_EN.md` | 用户行为、费用和限制说明 |

---

### Task 1：建立单槽 SolveAttempt 领域模型

**Files:**
- Create: `app/solve_attempt.go`
- Create: `app/solve_attempt_test.go`
- Modify: `app/app.go:22-62`

**Interfaces:**
- Produces: `SolveMode`、`SolveControlResult`、`solveAttemptTracker`。
- Produces exact methods: `Start`、`BindTarget`、`MarkCancelling`、`Finish`、`RetrySnapshot`、`Invalidate`。
- Consumed by: Tasks 3、4、5。

- [ ] **Step 1：写单槽、深拷贝、状态与 stale 红灯测试**

在 `app/solve_attempt_test.go` 写入：

```go
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

func TestSolveAttemptTrackerRejectsWrongTargetEpochAndTask(t *testing.T) {
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
    if tracker.MarkCancelling(999) {
        t.Fatal("old task must not mutate current attempt")
    }
}

func TestSolveAttemptTrackerInvalidationAndReturnedClone(t *testing.T) {
    tracker := newSolveAttemptTracker()
    tracker.Start(31, []string{"image"}, SolveModeNormal, "gen-solver", false, 1)
    tracker.BindTarget(31, desktopSolveTarget{epoch: 1, sessionID: "live", recordID: "task-31"})
    tracker.Finish(31, attemptFailed)
    snapshot, _ := tracker.RetrySnapshot("live", "task-31", 1)
    snapshot.screenshots[0] = "changed"
    again, _ := tracker.RetrySnapshot("live", "task-31", 1)
    if again.screenshots[0] != "image" {
        t.Fatal("returned retry snapshot aliases tracker memory")
    }
    tracker.Invalidate()
    if _, code := tracker.RetrySnapshot("live", "task-31", 1); code != solveNoRetryAvailable {
        t.Fatalf("invalidated retry code=%q", code)
    }
}
```

- [ ] **Step 2：运行红灯并确认缺失符号是唯一失败原因**

Run:

```bash
go test ./app -run '^TestSolveAttemptTracker' -count=1 -v
```

Expected: FAIL，明确缺少 `newSolveAttemptTracker`、`SolveModeNormal` 等；不得因现有测试失败或环境配置失败而继续。

- [ ] **Step 3：实现最小领域模型**

`app/solve_attempt.go` 使用以下精确外部形状：

```go
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
    solveRetryStale           = "solve_retry_stale"
    solveTaskRunning          = "solve_task_running"
    solveTaskDraining         = "solve_task_draining"
    solveRetryPrepareFailed   = "solve_retry_prepare_failed"
    solveCancelRequested      = "solve_cancel_requested"
    solveNoRunningTask        = "solve_no_running_task"
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

func newSolveAttemptTracker() *solveAttemptTracker { return &solveAttemptTracker{} }
```

方法实现固定为：

```go
func cloneSolveAttempt(attempt *solveAttempt) solveAttempt {
    cloned := *attempt
    cloned.screenshots = append([]string(nil), attempt.screenshots...)
    return cloned
}

func (t *solveAttemptTracker) Start(taskID int64, screenshots []string, mode SolveMode, normalDomainID string, keepContext bool, conversationEpoch uint64) {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.current = &solveAttempt{
        taskID: taskID,
        screenshots: append([]string(nil), screenshots...),
        mode: mode,
        normalDomainID: normalDomainID,
        keepContext: keepContext,
        conversationEpoch: conversationEpoch,
        state: attemptRunning,
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
```

所有输入/输出截图切片都深拷贝；启动管线在调用 `Start` 前保证图片数量为 1～3。`Finish(completed)` 清空，只有 failed/cancelled 保留；taskID 不匹配返回 false；Retry 只接受精确 target 和 epoch。

在 `App` 新增：

```go
solveAttempts *solveAttemptTracker
```

在 `NewApp()` 初始化 `solveAttempts: newSolveAttemptTracker()`。测试构造的 App 若 tracker 为 nil，后续 helper 必须惰性初始化，不能 panic。

- [ ] **Step 4：运行聚焦测试和 App 全包**

```bash
go test ./app -run '^TestSolveAttemptTracker' -count=1 -v
go test ./app -count=1
```

Expected: PASS。

- [ ] **Step 5：检查边界并记录建议提交归属**

```bash
git status
git branch
git remote -v
git diff -- app/app.go app/solve_attempt.go app/solve_attempt_test.go
git diff --check -- app/app.go app/solve_attempt.go app/solve_attempt_test.go
git diff --cached
```

Expected: 暂存区为空；只出现本任务文件。建议归入未来功能提交，未经授权不执行暂存或 commit。

---

### Task 2：题库 terminal Record 原子原位重启

**Files:**
- Modify: `pkg/questionbank/store.go:337-404`
- Modify: `pkg/questionbank/store_test.go`

**Interfaces:**
- Produces: `func (s *Store) RestartRecord(recordID, model, scene string, keepContext bool) error`。
- Consumed by: Task 4 的 retry capture。

- [ ] **Step 1：写状态矩阵、同 Record 与原子失败红灯测试**

在 `pkg/questionbank/store_test.go` 增加：

```go
func TestRestartRecordReusesImagesAndClearsTerminalOutput(t *testing.T) {
    store := bank(t)
    image := shot(t, false)
    recordID, err := store.BeginWithContext([]string{image}, "old-model", "gen-solver", true)
    if err != nil { t.Fatal(err) }
    if err = store.Append(recordID, "partial"); err != nil { t.Fatal(err) }
    if err = store.Finish(recordID, "", Failed); err != nil { t.Fatal(err) }
    before, _ := store.Snapshot()
    images := append([]string(nil), before.Records[0].Images...)

    if err = store.RestartRecord(recordID, "new-model", "gen-speed", false); err != nil {
        t.Fatal(err)
    }
    after, dirty := store.Snapshot()
    record := after.Records[0]
    if dirty || len(after.Records) != 1 || record.ID != recordID || record.Status != Running {
        t.Fatalf("restart changed record identity: %+v dirty=%v", after.Records, dirty)
    }
    if !reflect.DeepEqual(record.Images, images) || record.Answer != "" || len(record.AnswerItems) != 0 {
        t.Fatalf("restart did not preserve images and clear output: %+v", record)
    }
    if record.Model != "new-model" || record.Scene != "gen-speed" || record.KeepContext == nil || *record.KeepContext {
        t.Fatalf("restart semantics not updated: %+v", record)
    }
}

func TestRestartRecordRejectsRunningCompletedAndMissing(t *testing.T) {
    store := bank(t)
    image := shot(t, false)
    runningID, _ := store.Begin([]string{image}, "model", "scene")
    if err := store.RestartRecord(runningID, "model", "scene", false); err == nil {
        t.Fatal("running record restart should fail")
    }
    _ = store.Finish(runningID, "answer", Completed)
    if err := store.RestartRecord(runningID, "model", "scene", false); err == nil {
        t.Fatal("completed record restart should fail")
    }
    if err := store.RestartRecord("0123456789abcdef01234567", "model", "scene", false); err == nil {
        t.Fatal("missing record restart should fail")
    }
}

func TestRestartRecordSaveFailureLeavesMemoryAndDiskTerminal(t *testing.T) {
    store := bank(t)
    image := shot(t, false)
    recordID, _ := store.Begin([]string{image}, "model", "scene")
    _ = store.Finish(recordID, "", Cancelled)
    before, _ := store.Snapshot()
    store.saveSessionHook = func(*Session, string) error { return errors.New("synthetic restart failure") }
    if err := store.RestartRecord(recordID, "new", "gen-speed", false); err == nil {
        t.Fatal("restart should surface persistence failure")
    }
    after, _ := store.Snapshot()
    if !reflect.DeepEqual(before, after) {
        t.Fatalf("failed restart changed memory: before=%+v after=%+v", before, after)
    }
}
```

测试文件新增 `reflect` import；目录、图片和可写分段只复用该文件已有 `bank(t)` 与 `shot(t, false)`，不创建第二套 fixture。

- [ ] **Step 2：运行红灯**

```bash
go test ./pkg/questionbank -run '^TestRestartRecord' -count=1 -v
```

Expected: FAIL，仅因为 `RestartRecord` 不存在。

- [ ] **Step 3：实现 clone→修改→保存→交换事务**

实现核心顺序：

```go
func (s *Store) RestartRecord(recordID, model, scene string, keepContext bool) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    if err := s.ensureWriteAllowedLocked(); err != nil { return err }
    if s.current == nil || (s.current.State != Active && s.current.State != Paused) {
        return fmt.Errorf("当前分段不可重试")
    }
    working := cloneSession(s.current)
    record := findRecord(working, recordID)
    if record == nil { return fmt.Errorf("未找到可重试题目记录") }
    if record.Status != Failed && record.Status != Cancelled && record.Status != Interrupted {
        return fmt.Errorf("当前题目状态不可重试")
    }
    if len(record.Images) < 1 || len(record.Images) > 3 {
        return fmt.Errorf("可重试题目图片数量无效")
    }
    record.Status = Running
    record.Model = model
    record.Scene = scene
    keep := keepContext
    record.KeepContext = &keep
    record.Answer = ""
    record.AnswerItems = nil
    record.Questions = nil
    record.OrganizeError = ""
    record.OrganizeFailure = nil
    record.MetadataSuggestion = nil
    base := s.currentBase
    if base == "" { base = s.recordsRoot }
    if err := s.saveSessionLocked(working, base, s.pending); err != nil { return err }
    s.current = working
    s.currentBase = base
    s.dirty = false
    s.lastSave = time.Now()
    return nil
}
```

`keep` 为逃逸到堆的局部 bool，Record clone 和后续 JSON 保存均持有独立值；不新增通用 pointer helper。

- [ ] **Step 4：运行聚焦与 QuestionBank 全包**

```bash
go test ./pkg/questionbank -run '^TestRestartRecord' -count=1 -v
go test ./pkg/questionbank -count=1
```

Expected: PASS。

- [ ] **Step 5：边界检查**

```bash
git diff --check -- pkg/questionbank
git diff --cached
```

Expected: 不改变 export schema、图片文件或依赖；暂存区为空。建议归入未来功能提交。

---

### Task 3：统一 normal/fast 启动管线

**Files:**
- Modify: `app/solve.go:93-172`
- Modify: `app/solve_preflight_test.go`
- Modify: `app/solve_persistence_test.go`
- Modify: `pkg/solution/solution.go:24-28,115-253`
- Modify: `pkg/solution/solution_test.go`

**Interfaces:**
- Consumes: `SolveMode` 和 tracker Task 1。
- Produces: `TriggerQuickSend()`、`startSolve(solveStartRequest) SolveControlResult`、`startSolveLocked(solveStartRequest) SolveControlResult`、内部 `solveRunRequest`。
- Preserves: `TriggerSend()`、`TriggerSolve()`、`ConfirmSolveWithoutResume()`。

- [ ] **Step 1：写 fast 单次覆盖、normal 保持、Config 不变红灯测试**

在 `app/solve_persistence_test.go` 增加捕获 system prompt 的 Provider：

```go
type modeCaptureProvider struct {
    calls    int
    messages [][]llm.Message
}
func (p *modeCaptureProvider) GenerateContentStream(_ context.Context, messages []llm.Message, _ llm.StreamCallback) (llm.Message, error) {
    p.calls++
    p.messages = append(p.messages, messages)
    return llm.NewAssistantMessage("A"), nil
}
func (p *modeCaptureProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) { return llm.Message{}, nil }
func (p *modeCaptureProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *modeCaptureProvider) TestChat(context.Context) error { return nil }

func TestSolveModeFastUsesGenSpeedWithoutChangingConfig(t *testing.T) {
    manager := configuredSolveManager(t)
    if err := manager.Patch(func(c *config.Config) { c.DomainId = "gen-step" }); err != nil { t.Fatal(err) }
    provider := &modeCaptureProvider{}
    app := &App{configManager: manager, solver: solution.NewSolver(provider), taskManager: task.NewTaskCoordinator(), solveAttempts: newSolveAttemptTracker()}
    before := manager.Get()
    req := solveRunRequest{screenshots: []string{"image"}, mode: SolveModeFast, normalDomainID: before.DomainId, keepContext: before.KeepContext}
    if !app.solveInternal(context.Background(), 1, req, 0, 0) { t.Fatal("fast solve failed") }
    system := provider.messages[0][0].Content
    if !strings.Contains(system, "Speed Solver") || strings.Contains(system, "Step-by-Step Tutor") {
        t.Fatalf("fast prompt=%q", system)
    }
    after := manager.Get()
    if !reflect.DeepEqual(before, after) { t.Fatal("fast solve mutated persisted config") }
}

func TestSolveModeNormalUsesSelectedDomain(t *testing.T) {
    manager := configuredSolveManager(t)
    _ = manager.Patch(func(c *config.Config) { c.DomainId = "gen-step" })
    provider := &modeCaptureProvider{}
    app := &App{configManager: manager, solver: solution.NewSolver(provider), solveAttempts: newSolveAttemptTracker()}
    req := solveRunRequest{screenshots: []string{"image"}, mode: SolveModeNormal, normalDomainID: "gen-step"}
    if !app.solveInternal(context.Background(), 2, req, 0, 0) { t.Fatal("normal solve failed") }
    if !strings.Contains(provider.messages[0][0].Content, "Step-by-Step Tutor") { t.Fatal("selected domain missing") }
}
```

- [ ] **Step 2：红灯确认**

```bash
go test ./app -run '^TestSolveMode' -count=1 -v
```

Expected: FAIL，缺少 `solveRunRequest` 或新签名；不是现有 Provider/题库失败。

- [ ] **Step 3：实现统一请求结构和入口**

在 `app/solve.go` 定义：

```go
type solveStartRequest struct {
    mode        SolveMode
    retryTarget *desktopSolveTarget
    screenshots []string
}

type solveRunRequest struct {
    screenshots    []string
    mode           SolveMode
    normalDomainID string
    keepContext    bool
    retryTarget    *desktopSolveTarget
}
```

入口严格为：

```go
func (a *App) TriggerSend()      { a.triggerSend(SolveModeNormal, false) }
func (a *App) TriggerQuickSend() { a.triggerSend(SolveModeFast, false) }
func (a *App) ConfirmSolveWithoutResume() { a.confirmPendingSolveWithoutResume() }
```

简历内容缺失确认流程必须记住待确认 mode，不能把 fast 降级为 normal。在 App 增加非敏感 `pendingSolveMode`，受 `conversationMu` 保护；取消弹窗时前端不启动请求。

统一启动函数签名固定为：

```go
func (a *App) startSolve(request solveStartRequest) SolveControlResult {
    a.conversationMu.Lock()
    defer a.conversationMu.Unlock()
    return a.startSolveLocked(request)
}

func (a *App) startSolveLocked(request solveStartRequest) SolveControlResult
```

`startSolveLocked` 只允许在持有 `conversationMu` 时调用，并按以下顺序执行：

1. 验证 BaseURL/Model。
2. 处理简历内容缺失确认。
3. 检查 `HasInFlightTask()`；running 返回 `solve_task_running`，已取消但未 Complete 返回 `solve_task_draining`。
4. 原始发送无图时截图；retry 绝不进入截图分支。
5. 原始发送深拷贝并消费 screenshotBuffer；retry 再次调用 `RetrySnapshot` 校验 target/epoch，并仅使用快照中的 screenshots、mode、normalDomainID、keepContext。
6. 原始发送从当前 Config 固定 normalDomainID/KeepContext；retry 保留首次 attempt 的 normalDomainID/KeepContext，但连接、模型、Endpoint 和凭据仍读取重试当下的 Config。
7. `StartTask` 后调用 tracker.Start；retry 的新 attempt 使用前一步固定的首次语义。
8. target 建立后调用 tracker.BindTarget；retry 沿用原 target。
9. goroutine 完成 `solveInternal` 后先调用 `CompleteTask(taskID)`，再按 `ctx.Err()!=nil`、solve 结果依次标记 cancelled、failed 或 completed；旧 taskID 不能清理新 attempt。

`routeConversationSolutionEvent` 增加显式 `mode SolveMode` 参数。它给 Desktop envelope 填 `Mode`；路由 `solution-stream-start` 到 Mobile 时，把 `string(mode)` 作为 start data，其他事件保持原 data，避免把 mode 冒充答案文本。

`solveInternal` 内读取当前 Config，再按 `solveRunRequest` 覆盖：

```go
if req.mode == SolveModeFast {
    cfg.DomainId = "gen-speed"
} else {
    cfg.DomainId = req.normalDomainID
}
cfg.KeepContext = req.keepContext
```

不得调用 ConfigManager 写方法。

- [ ] **Step 4：更新现有调用并运行聚焦测试**

把现有直接 `solveInternal(ctx, id, screenshots, ...)` 测试改为显式 `solveRunRequest`，不改变原断言。

```bash
go test ./app -run '^(TestSolveMode|TestTriggerSendWaitsForResumeConfirmation|TestKeylessSolveInternal|TestSolveCreatesRecord)' -count=1 -v
go test ./pkg/solution -count=1
```

Expected: PASS；Keyless 与 numbering 行为不变。

- [ ] **Step 5：检查范围**

```bash
git diff --check -- app/solve.go app/solve_preflight_test.go app/solve_persistence_test.go pkg/solution
git diff --cached
```

Expected: Config/Connection Profile 无差异，暂存区为空。建议归入未来功能提交。

---

### Task 4：接入 RetrySolve 和题库同 Record capture

**Files:**
- Modify: `app/solve.go`
- Modify: `app/questionbank_solve_capture.go`
- Modify: `app/app.go`
- Modify: `app/questionbank_next_exam.go`
- Modify: `app/questionbank_continuation.go`
- Modify: `app/questionbank.go`
- Create: `app/solve_retry_test.go`
- Modify: `app/solve_persistence_test.go`
- Modify: `app/questionbank_next_exam_test.go`
- Modify: `app/questionbank_continuation_test.go`
- Modify: `app/questionbank_delete_test.go`
- Modify: `app/error_diagnostics_test.go`
- Modify: `app/mobile.go:12-23,190-231`

**Interfaces:**
- Consumes: tracker、`Store.RestartRecord`、`solveRunRequest`。
- Produces: `RetrySolve(sessionID, recordID string) SolveControlResult`。
- Produces Desktop fields: `Mode string`、`CanRetry bool`。

- [ ] **Step 1：写无截图调用、同 Record、target mismatch 和双击红灯测试**

`app/solve_retry_test.go` 先定义完全本地的 Provider 和等待 helper：

```go
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
            if p.calls == 2 { p.secondImage = part.Base64 }
        }
    }
    resultErr := p.results[p.calls-1]
    if resultErr != nil { return llm.Message{}, resultErr }
    if onChunk != nil { onChunk(llm.StreamChunk{Type: llm.ChunkContent, Content: "answer"}) }
    return llm.NewAssistantMessage("answer"), nil
}
func (p *scriptedSolveProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) { return llm.Message{}, nil }
func (p *scriptedSolveProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *scriptedSolveProvider) TestChat(context.Context) error { return nil }

func waitForNoInFlight(t *testing.T, coordinator *task.TaskCoordinator) {
    t.Helper()
    deadline := time.Now().Add(2 * time.Second)
    for coordinator.HasInFlightTask() && time.Now().Before(deadline) { runtime.Gosched() }
    if coordinator.HasInFlightTask() { t.Fatal("solve task did not finish") }
}

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
func (p *blockingSolveProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) { return llm.Message{}, nil }
func (p *blockingSolveProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *blockingSolveProvider) TestChat(context.Context) error { return nil }

func retryableAppWithoutQuestionBank(t *testing.T) (*App, *blockingSolveProvider) {
    t.Helper()
    manager := configuredSolveManager(t)
    provider := newBlockingSolveProvider()
    app := &App{
        configManager: manager,
        solver: solution.NewSolver(provider),
        taskManager: task.NewTaskCoordinator(),
        solveAttempts: newSolveAttemptTracker(),
    }
    app.solveAttempts.Start(1, []string{"image"}, SolveModeNormal, "gen-solver", false, 0)
    app.solveAttempts.BindTarget(1, desktopSolveTarget{sessionID: "live", recordID: "task-1"})
    app.solveAttempts.Finish(1, attemptFailed)
    return app, provider
}
```

测试文件显式 import `runtime`、`sync`、`time`；所有等待都有 2 秒上限。随后写入：

```go
func TestRetrySolveReusesBackendImagesAndSameQuestionBankRecord(t *testing.T) {
    app, store, image := archiveApp(t)
    app.configManager = configuredSolveManager(t)
    provider := &scriptedSolveProvider{results: []error{errors.New("synthetic network failure"), nil}}
    app.solver = solution.NewSolver(provider)
    app.taskManager = task.NewTaskCoordinator()
    app.solveAttempts = newSolveAttemptTracker()

    app.startSolve(solveStartRequest{mode: SolveModeNormal, screenshots: []string{image}})
    waitForNoInFlight(t, app.taskManager)
    failed, _ := store.Snapshot()
    if len(failed.Records) != 1 || failed.Records[0].Status != questionbank.Failed { t.Fatalf("failed=%+v", failed) }

    result := app.RetrySolve(failed.ID, failed.Records[0].ID)
    if !result.Success { t.Fatalf("retry result=%+v", result) }
    waitForNoInFlight(t, app.taskManager)
    completed, _ := store.Snapshot()
    if len(completed.Records) != 1 || completed.Records[0].ID != failed.Records[0].ID || completed.Records[0].Status != questionbank.Completed {
        t.Fatalf("retry duplicated record: %+v", completed.Records)
    }
    if provider.imageInputs != 2 || provider.secondImage != image { t.Fatal("retry did not reuse backend image") }
}

func TestRetrySolveRejectsWrongTargetAndConcurrentDoubleClick(t *testing.T) {
    app, provider := retryableAppWithoutQuestionBank(t)
    defer close(provider.release)
    if result := app.RetrySolve("wrong", "target"); result.Code != solveRetryTargetMismatch { t.Fatalf("result=%+v", result) }
    first := app.RetrySolve("live", "task-1")
    second := app.RetrySolve("live", "task-1")
    if !first.Success || second.Code != "solve_task_running" { t.Fatalf("first=%+v second=%+v", first, second) }
}

func TestRetrySolveStaleAfterConversationEpochChange(t *testing.T) {
    app, provider := retryableAppWithoutQuestionBank(t)
    defer close(provider.release)
    app.conversationEpoch++
    result := app.RetrySolve("live", "task-1")
    if result.Code != solveRetryStale { t.Fatalf("result=%+v", result) }
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
```

在下列既有成功路径测试中，边界调用前插入 seed，既有成功断言后插入失效断言：

| 测试 | 边界调用前 | 成功断言后 |
|---|---|---|
| `TestStartNextExamSealsThenClearsRuntimeState` | `seedRetryableAttempt(a, a.conversationEpoch)` | `assertRetryInvalidated(t, a, 7)` |
| `TestContinueQuestionBankSessionSwitchesStoreSolverAndEpoch` | `seedRetryableAttempt(a, a.conversationEpoch)` | `assertRetryInvalidated(t, a, 0)` |
| `TestDiscardQuestionBankSegmentPhysicallyDeletesThenClearsContext` | `seedRetryableAttempt(a, a.conversationEpoch)` | `assertRetryInvalidated(t, a, 7)` |
| `TestKeepContextOffClearsRefsAndStartupHistory` | `seedRetryableAttempt(a, a.conversationEpoch)` | `assertRetryInvalidated(t, a, 7)` |

在 `TestStartNextExamPersistenceFailureHasZeroRuntimeSideEffects` 和 `TestContinueQuestionBankSessionRollsBackSolverWhenStoreRejects` 的失败调用前同样 seed，失败断言后增加：

```go
if _, code := a.solveAttempts.RetrySnapshot("old-session", "old-record", a.conversationEpoch); code != "" {
    t.Fatalf("rejected boundary invalidated retry snapshot: %q", code)
}
```

这锁定“仅在边界操作成功后失效”。

测试 helper 必须使用 fake Provider；不得启动 Wails 或真实网络。

- [ ] **Step 2：运行红灯**

```bash
go test ./app -run '^TestRetrySolve' -count=1 -v
```

Expected: FAIL，缺少 RetrySolve/startSolve/retry capture；题库 Store Task 2 已为绿。

- [ ] **Step 3：实现结构化 Retry API 与 capture 分支**

`RetrySolve` 顺序：

```go
func (a *App) RetrySolve(sessionID, recordID string) SolveControlResult {
    a.conversationMu.Lock()
    defer a.conversationMu.Unlock()
    if a.taskManager.HasRunningTask() { return solveControlFailure(solveTaskRunning, safeSolveControlMessage(solveTaskRunning)) }
    if a.taskManager.HasInFlightTask() { return solveControlFailure(solveTaskDraining, safeSolveControlMessage(solveTaskDraining)) }
    if a.solveAttempts == nil { return solveControlFailure(solveNoRetryAvailable, safeSolveControlMessage(solveNoRetryAvailable)) }
    snapshot, code := a.solveAttempts.RetrySnapshot(sessionID, recordID, a.conversationEpoch)
    if code != "" { return solveControlFailure(code, safeSolveControlMessage(code)) }
    return a.startSolveLocked(solveStartRequest{
        mode: snapshot.mode,
        retryTarget: &snapshot.target,
        screenshots: append([]string(nil), snapshot.screenshots...),
    })
}
```

`startSolveLocked` 的 retry 分支必须在创建题库 capture 或启动 task 前再次按 `retryTarget + conversationEpoch` 获取快照；这是锁内的二次授权校验，也是 `normalDomainID`、`keepContext` 的唯一 retry 来源。不得用重试时当前 `DomainId/KeepContext` 覆盖首次 attempt 语义；若二次校验失败，直接返回安全错误码，不修改 Record、不调用 Provider。

在 `app/solve.go` 增加仅供持有 `conversationMu` 的调用点使用的 helper：

```go
func (a *App) invalidateSolveRetryLocked() {
    if a.solveAttempts != nil {
        a.solveAttempts.Invalidate()
    }
}

func (a *App) appendScreenshotLocked(image string) {
    a.invalidateSolveRetryLocked()
    a.screenshotBuffer = append(a.screenshotBuffer, image)
}
```

`TriggerScreenshot` 和原始发送的自动截图成功后通过 `appendScreenshotLocked` 接受新图；`ClearScreenshots` 无论缓冲是否为空都调用 `invalidateSolveRetryLocked`。原始 normal/fast 启动由后续 `tracker.Start` 覆盖旧槽。`onConfigChanged` 关闭 KeepContext、`StartNextExam`、`ContinueQuestionBankSession`、`DiscardQuestionBankSegment` 只在各自持久化/切换成功、即将推进 `conversationEpoch` 时调用失效 helper；操作拒绝或落盘回滚失败时不得失效。

`beginQuestionSolveCapture` 拆为首次 Begin 和 retry target 分支。retry 且有 Store/recordID 时先调用 `RestartRecord`；失败返回固定 `solve_retry_prepare_failed`，不得调用 Provider。无 Store 时沿用 target.recordID，不创建新显示轮。

`request-start` 在 retry 时使用同一个 target；`DesktopSolveEvent` 填入 Mode。`solveInternal` 的 callback 在转发 `solution-error` / `solution-cancelled` 前，先对匹配 taskID 调用 `Finish(attemptFailed)` / `Finish(attemptCancelled)`，再用 `RetrySnapshot(target.sessionID, target.recordID, target.epoch)` 是否成功计算 `CanRetry`。`newDesktopSolveEvent(target, mode, canRetry, event, data...)` 写入 envelope，并对 `solution-error` 和 `solution-cancelled` 都只附加 `llm.CleanDiagnostic` 后的 ErrorDetail；普通流事件和成功事件固定 `CanRetry=false`。即使终态事件已使快照可见，`RetrySolve` 仍先由 `HasInFlightTask()` 返回 draining，直到旧 goroutine Complete。

- [ ] **Step 4：运行聚焦、App 和 QuestionBank 测试**

```bash
go test ./app -run '^(TestRetrySolve|TestDesktopSolveEvent|TestScreenshotBoundaries|TestStartNextExam|TestContinueQuestionBankSession|TestDiscardQuestionBankSegment|TestKeepContextOff|TestSolveCreatesRecord|TestSolveRollsBackConversation)' -count=1 -v
go test ./app ./pkg/questionbank -count=1
```

Expected: PASS；每次 retry Provider 调用数增加 1，但 Record 数不增加。

- [ ] **Step 5：安全与边界检查**

搜索公开 Retry 方法参数不得出现 `image`、`base64`、`config.Config`、`APIKey`：

```bash
git diff --check -- app pkg/questionbank
git diff --cached
```

Expected: 暂存区为空；建议归入未来功能提交。

---

### Task 5：取消生命周期和 solution-cancelled 终态

**Files:**
- Modify: `pkg/solution/solution.go:183-202`
- Modify: `pkg/solution/error_diagnostics_test.go`
- Modify: `app/solve.go`
- Create: `app/solve_cancel_test.go`
- Modify: `app/questionbank_solve_capture.go`

**Interfaces:**
- Consumes existing: `TaskCoordinator.GetCurrentTaskID() int64`、`CancelCurrentTask() bool`、`HasInFlightTask() bool`。
- Produces: `CancelSolve() SolveControlResult`。
- Preserves: `CancelRunningTask() bool` 兼容包装。
- Produces event: `solution-cancelled`。

- [ ] **Step 1：写 Solver 独立取消事件红灯**

更新 `pkg/solution/error_diagnostics_test.go` 的取消分支：

```go
if cancelled {
    if events[len(events)-1].name != "solution-cancelled" {
        t.Fatalf("cancel event=%q", events[len(events)-1].name)
    }
    detail := events[len(events)-1].args[1].(llm.ErrorDiagnostic)
    if detail.Code != llm.ErrorCancelled { t.Fatalf("detail=%+v", detail) }
}
```

并增加 KeepContext 断言：取消后 `SnapshotConversation().Turns` 长度不变。

- [ ] **Step 2：写 App cancelling/in-flight/幂等红灯**

`app/solve_cancel_test.go`：

```go
func TestCancelSolveWaitsForInFlightCompletionAndKeepsRetry(t *testing.T) {
    manager := configuredSolveManager(t)
    provider := newBlockingSolveProvider()
    app := &App{
        configManager: manager,
        solver: solution.NewSolver(provider),
        taskManager: task.NewTaskCoordinator(),
        solveAttempts: newSolveAttemptTracker(),
    }
    result := app.startSolve(solveStartRequest{mode: SolveModeNormal, screenshots: []string{"image"}})
    if !result.Success { t.Fatalf("start=%+v", result) }
    <-provider.started
    first := app.CancelSolve()
    second := app.CancelSolve()
    if !first.Success || first.Code != solveCancelRequested { t.Fatalf("first=%+v", first) }
    if second.Code != solveCancelRequested { t.Fatalf("idempotent second=%+v", second) }
    if !app.taskManager.HasInFlightTask() { t.Fatal("cancel must not erase in-flight task") }
    if result := app.RetrySolve("live", "task-1"); result.Code != solveTaskDraining {
        t.Fatalf("retry before unwind=%+v", result)
    }
    waitForNoInFlight(t, app.taskManager)
    if _, code := app.solveAttempts.RetrySnapshot("live", "task-1", app.conversationEpoch); code != "" {
        t.Fatalf("cancelled attempt not retryable: %q", code)
    }
}
```

- [ ] **Step 3：运行红灯**

```bash
go test ./pkg/solution -run 'Cancel|Cancelled' -count=1 -v
go test ./app -run '^TestCancelSolve' -count=1 -v
```

Expected: Solver 仍发 `solution-error`，App 缺 `CancelSolve`。

- [ ] **Step 4：实现取消终态**

Solver：

```go
if errors.Is(ctx.Err(), context.Canceled) {
    detail := llm.DiagnoseError(llm.NewProviderError(llm.ErrorCancelled, llm.StageStreamResponse, "", nil))
    if cb.EmitEvent != nil { cb.EmitEvent("solution-cancelled", detail.Message, detail) }
    return false
}
```

Capture 将 `solution-cancelled` 映射为 `questionbank.Cancelled`，终止后仍 passthrough；普通 `solution-error` 映射 Failed。

App `CancelSolve`：

```go
func (a *App) CancelSolve() SolveControlResult {
    a.conversationMu.Lock()
    defer a.conversationMu.Unlock()

    taskID := a.taskManager.GetCurrentTaskID()
    if taskID == 0 || a.solveAttempts == nil || !a.solveAttempts.MarkCancelling(taskID) {
        return solveControlFailure(solveNoRunningTask, safeSolveControlMessage(solveNoRunningTask))
    }
    if a.taskManager.CancelCurrentTask() {
        return solveControlSuccess(solveCancelRequested, "正在中断解题")
    }
    if a.taskManager.HasInFlightTask() {
        return solveControlSuccess(solveCancelRequested, "正在中断解题")
    }
    return solveControlFailure(solveNoRunningTask, safeSolveControlMessage(solveNoRunningTask))
}
```

- `MarkCancelling` 对同 taskID 的 cancelling 状态幂等，第二次调用不会再次取消 context。
- goroutine 收尾按 Task 3 的固定顺序先 `CompleteTask(taskID)`，再把匹配 attempt 标为 cancelled；在此之前 retry/new send 均返回 draining。
- `CancelRunningTask()` 只调用 `CancelSolve()` 并返回 `Success`，供旧 binding 使用。

- [ ] **Step 5：运行聚焦与全包**

```bash
go test ./pkg/solution -count=1
go test ./pkg/task -count=1
go test ./app -run 'Cancel|RetrySolve' -count=1 -v
go test ./app -count=1
```

Expected: PASS；取消后无 completed Record、无 KeepContext 新轮。

- [ ] **Step 6：边界检查**

```bash
git diff --check -- pkg/task pkg/solution app
git diff --cached
```

Expected: 暂存区为空。建议归入未来功能提交。

---

### Task 6：新增快速/中断快捷键与冲突安全迁移

**Files:**
- Modify: `pkg/config/config.go:111-203`
- Modify: `pkg/config/config_test.go`
- Modify: `pkg/config/manager.go:55-92`
- Modify: `pkg/config/manager_test.go`
- Modify: `pkg/shortcut/delegate.go`
- Modify: `pkg/shortcut/service.go:79-140`
- Modify: `pkg/shortcut/service_test.go`

**Interfaces:**
- Adds actions: `send_fast`、`cancel_solve`。
- Delegate adds: `TriggerQuickSend()`、`CancelSolve() SolveControlResult` 不能直接 import app；因此接口方法应为 `TriggerQuickSend()` 和 `CancelRunningTask() bool`，结构化结果留在 Wails App API。

- [ ] **Step 1：写默认值和唯一性红灯**

在 `pkg/config/config_test.go`：

```go
func TestDefaultFastAndCancelSolveShortcuts(t *testing.T) {
    windows := getDefaultShortcutsForOS("windows")
    if got := windows["send_fast"]; got.ComboID != "74+160+162" || got.KeyName != "Ctrl+Shift+J" { t.Fatalf("send_fast=%+v", got) }
    if got := windows["cancel_solve"]; got.ComboID != "88+160+162" || got.KeyName != "Ctrl+Shift+X" { t.Fatalf("cancel_solve=%+v", got) }
    mac := getDefaultShortcutsForOS("darwin")
    if got := mac["send_fast"]; got.ComboID != "Cmd+Shift+J" || got.KeyName != "⌘⇧J" { t.Fatalf("mac send_fast=%+v", got) }
    if got := mac["cancel_solve"]; got.ComboID != "Cmd+Shift+X" || got.KeyName != "⌘⇧X" { t.Fatalf("mac cancel_solve=%+v", got) }
}
```

扩展唯一性测试，确保两个 ComboID 只有对应 owner。

- [ ] **Step 2：写旧配置迁移红灯**

在 `manager_test.go` 复用现有 `isolatedManager` / `writeShortcutConfig`：

```go
func TestLoadMigratesSolveControlShortcutsSafely(t *testing.T) {
    if runtime.GOOS != "windows" { t.Skipf("Windows-only migration test on %s", runtime.GOOS) }
    defaults := map[string]shortcut.KeyBinding{
        "send_fast":    {ComboID: "74+160+162", KeyName: "Ctrl+Shift+J"},
        "cancel_solve": {ComboID: "88+160+162", KeyName: "Ctrl+Shift+X"},
    }
    for action, binding := range defaults {
        t.Run(action+" free default", func(t *testing.T) {
            cm := isolatedManager(t)
            writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{"screenshot": {ComboID: "119", KeyName: "F8"}})
            if err := cm.Load(); err != nil { t.Fatal(err) }
            if got := cm.Get().Shortcuts[action]; got != binding { t.Fatalf("want %+v got %+v", binding, got) }
        })
        t.Run(action+" occupied default", func(t *testing.T) {
            cm := isolatedManager(t)
            occupied := shortcut.KeyBinding{ComboID: binding.ComboID, KeyName: "Legacy Binding"}
            writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{"legacy_action": occupied})
            if err := cm.Load(); err != nil { t.Fatal(err) }
            got := cm.Get().Shortcuts
            if got["legacy_action"] != occupied || got[action] != (shortcut.KeyBinding{}) {
                t.Fatalf("migration stole occupied combo: %+v", got)
            }
        })
        for _, explicit := range []shortcut.KeyBinding{{}, {ComboID: "77+160+162", KeyName: "Ctrl+Shift+M"}} {
            cm := isolatedManager(t)
            writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{action: explicit})
            if err := cm.Load(); err != nil { t.Fatal(err) }
            if got := cm.Get().Shortcuts[action]; got != explicit { t.Fatalf("explicit want %+v got %+v", explicit, got) }
        }
    }
}
```

该测试同时覆盖 action 缺失补默认、默认组合被占用时新增 action 为空、显式自定义和显式空值原样保留；只写测试目录中的加密 fixture。

- [ ] **Step 3：写 Service 分发红灯**

扩展 `mockDelegate`：

```go
quickSendCalls int
cancelCalls int
func (m *mockDelegate) TriggerQuickSend() { m.quickSendCalls++ }
func (m *mockDelegate) CancelRunningTask() bool { m.cancelCalls++; return true }
```

测试：

```go
func TestHandleTriggerFastSendAndCancel(t *testing.T) {
    d := &mockDelegate{}
    s := newTestService(d)
    s.handleTrigger("send_fast")
    s.handleTrigger("cancel_solve")
    if d.quickSendCalls != 1 || d.cancelCalls != 1 { t.Fatalf("delegate=%+v", d) }
}
```

- [ ] **Step 4：运行红灯**

```bash
go test ./pkg/config -run 'FastAndCancel|Shortcut' -count=1 -v
go test ./pkg/shortcut -run 'FastSendAndCancel' -count=1 -v
```

Expected: 新 action 缺失，delegate 未实现。

- [ ] **Step 5：实现默认值、通用迁移 helper 和分发**

避免复制两个长迁移函数，新增包内 helper：

```go
func migrateDefaultShortcut(cfg *Config, action string, binding shortcut.KeyBinding, explicitlyConfigured bool) {
    if cfg.Shortcuts == nil { cfg.Shortcuts = make(map[string]shortcut.KeyBinding) }
    if explicitlyConfigured { return }
    for other, existing := range cfg.Shortcuts {
        if other != action && existing.ComboID != "" && existing.ComboID == binding.ComboID {
            cfg.Shortcuts[action] = shortcut.KeyBinding{}
            return
        }
    }
    cfg.Shortcuts[action] = binding
}
```

Load persisted metadata必须分别检测 `send_fast` / `cancel_solve` 是否存在。macOS 默认同样进入默认 Map；Windows VK ComboID 使用升序 `74+160+162` / `88+160+162`。

Service switch：

```go
case "send_fast":
    s.delegate.TriggerQuickSend()
case "cancel_solve":
    if !s.delegate.CancelRunningTask() {
        s.delegate.EmitEvent("toast", map[string]string{"text": "当前没有正在进行的解题", "type": "info"})
    }
```

不得在快捷键 worker 等待任务退出。

- [ ] **Step 6：运行聚焦与 Config/Shortcut 全包**

```bash
go test ./pkg/config ./pkg/shortcut -count=1
```

Expected: PASS。

- [ ] **Step 7：边界检查**

```bash
git diff --check -- pkg/config pkg/shortcut
git diff --cached
```

Expected: 不出现依赖变化；暂存区为空。建议归入未来功能提交。

---

### Task 7：Pinia typed retry/cancel/fast 状态机与 API

**Files:**
- Modify: `frontend/src/services/api.js`
- Modify: `frontend/src/stores/solution.js`
- Modify: `frontend/src/App.vue`
- Create: `frontend/solve_retry_cancel_fast_regression_test.mjs`
- Modify: `frontend/llm_error_diagnostics_regression_test.mjs`
- Modify: `frontend/questionbank_continuation_regression_test.mjs`

**Interfaces:**
- API: `retrySolve(sessionId, recordId)`、`cancelSolve()`、`triggerQuickSend()`。
- Store: `retryRound(round)`、`cancelCurrentSolve()`、`solveControlBusy`、`currentSolveMode`。
- Handles event: `solution-cancelled`。

- [ ] **Step 1：写同轮 retry reset 与 cancelled 红灯**

新回归文件沿用现有 `fs`/`vm`/Pinia 加载方式，并定义：

```js
function deferred() {
  let resolve
  let reject
  const promise = new Promise((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

function makeSolutionStore(overrides = {}) {
  setActivePinia(createPinia())
  const api = {
    getQuestionBankRecordImage: async () => '',
    retrySolve: async () => ({ success: false, code: 'solve_no_retry_available' }),
    cancelSolve: async () => ({ success: false, code: 'solve_no_running_task' }),
    ...overrides,
  }
  const context = vm.createContext({
    defineStore, ref, reactive, computed, api,
    document: { getElementById: () => null },
    renderMarkdownWithLatex: text => text || '',
    requestAnimationFrame: fn => { fn(); return 1 },
    cancelAnimationFrame() {}, setTimeout, clearTimeout, console,
  })
  vm.runInContext(solutionSource + '\nglobalThis.useSolutionStore = useSolutionStore', context)
  return { store: context.useSolutionStore(), api }
}
```

随后验证同轮状态：

```js
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'request-start', mode: 'fast', images: ['image'] })
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'solution-stream-chunk', text: 'partial' })
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'solution-cancelled', text: '模型请求已取消', canRetry: true })
assert.equal(store.currentRounds[0].status, 'cancelled')
assert.equal(store.currentRounds[0].aiResponse, 'partial')
assert.equal(store.currentRounds[0].canRetry, true)
assert.equal(store.currentSolveMode, 'fast')

const roundsBefore = store.history[0].rounds.length
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'request-start', mode: 'fast', images: ['image'] })
assert.equal(store.history[0].rounds.length, roundsBefore)
assert.equal(store.currentRounds[0].aiResponse, '')
assert.equal(store.currentRounds[0].error, null)
assert.equal(store.currentRounds[0].status, 'running')
```

- [ ] **Step 2：写 API 防双击和安全 target 红灯**

在同一 VM harness 中注入只记录安全参数的 API：

```js
const retryGate = deferred()
const calls = []
const store = makeSolutionStore({
  retrySolve: async (sessionId, recordId) => {
    calls.push(['retry', sessionId, recordId])
    return retryGate.promise
  },
  cancelSolve: async () => {
    calls.push(['cancel'])
    return { success: true, code: 'solve_cancel_requested' }
  },
}).store
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'request-start', images: ['image'] })
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'solution-error', text: 'network', canRetry: true })
const round = store.currentRounds[0]
const firstRetry = store.retryRound(round)
const secondRetry = await store.retryRound(round)
assert.equal(secondRetry, false)
assert.deepEqual(calls, [['retry', 'session-a', 'record-a']])
assert.equal(JSON.stringify(calls).includes('image'), false)
retryGate.resolve({ success: true })
assert.equal(await firstRetry, true)

assert.equal(await store.cancelCurrentSolve(), true)
assert.equal(await store.cancelCurrentSolve(), false)
assert.equal(calls.filter(call => call[0] === 'cancel').length, 1)
```

测试 helper `makeSolutionStore(overrides)` 必须将 overrides 合并进 mock api；`deferred()` 沿用项目其他前端回归的实现。连续调用只允许一次 bridge 调用，参数只有 sessionId/recordId，不含 image/config。

- [ ] **Step 3：运行红灯**

```bash
node frontend/solve_retry_cancel_fast_regression_test.mjs
```

Expected: FAIL，缺 `solution-cancelled`、`canRetry`、action/API。

- [ ] **Step 4：实现 store 与 API**

`api.js`：

```js
retrySolve: (sessionId, recordId) => callApp('RetrySolve', sessionId, recordId),
cancelSolve: () => callApp('CancelSolve'),
triggerQuickSend: () => callApp('TriggerQuickSend'),
```

Store 规则：

- `beginDesktopRequest` 命中现有 record 时清空 thinking/aiResponse/answerItems/answerCopyText/error，保留图片目标并设置 running。
- `solution-error` 从 event.canRetry 设置 round.canRetry。
- `solution-cancelled` 设置 cancelled，保留已有 aiResponse，写入非 error 的 interruption 元数据，清 stream target。
- `retryRound` 只从 round 所属 typed target 取 sessionId/recordId；legacy round 返回 false。
- `solveControlBusy` 防双击；`cancelCurrentSolve` 在结构化成功后立即把当前 typed round 设为 `cancelling`，因此第二次调用在事件返回前也直接 false；API 返回失败码时恢复原状态并由 App/UI 显示固定安全 Toast。
- stale typed 事件继续由三元目标拒绝。

`App.vue` typed switch 增加 solution-cancelled，状态文字“已中断”；删除旧事件路径把取消复位成“正在思考”的 `handleUserCancellation` 逻辑，避免双状态机。

- [ ] **Step 5：运行前端聚焦回归**

```bash
node frontend/solve_retry_cancel_fast_regression_test.mjs
node frontend/llm_error_diagnostics_regression_test.mjs
node frontend/questionbank_continuation_regression_test.mjs
```

Expected: PASS。

- [ ] **Step 6：边界检查**

```bash
git diff --check -- frontend/src frontend/solve_retry_cancel_fast_regression_test.mjs frontend/llm_error_diagnostics_regression_test.mjs frontend/questionbank_continuation_regression_test.mjs
git diff --cached
```

Expected: Store 不包含 Base64 retry payload 或 localStorage；暂存区为空。建议归入未来功能提交。

---

### Task 8：Desktop 控制组件与全显示端快捷键提示

**Files:**
- Create: `frontend/src/components/SolveControls.vue`
- Modify: `frontend/src/components/SolveView.vue`
- Modify: `frontend/src/components/ErrorView.vue`
- Modify: `frontend/src/components/SettingsModal.vue`
- Modify: `frontend/src/components/WelcomeView.vue`
- Modify: `frontend/src/components/TopBar.vue`
- Modify: `frontend/src/components/ScreenshotDock.vue`
- Modify: `frontend/src/stores/settings.js`
- Modify: `frontend/shortcut_hint_sync_regression_test.mjs`
- Modify: `frontend/settings_regression_test.mjs`
- Modify: `frontend/screenshot_dock_regression_test.mjs`

**Interfaces:**
- Presentational emits: `retry`、`cancel`。
- Store computed: `fastSendShortcut`、`cancelSolveShortcut`。

- [ ] **Step 1：写 UI 静态和 SSR 红灯约束**

扩展回归，要求：

```js
assert.match(solveView, /<SolveControls/)
assert.match(solveView, /@retry="solution\.retryRound\(round\)"/)
assert.match(solveView, /@cancel="solution\.cancelCurrentSolve"/)
assert.match(controls, /重试本题/)
assert.match(controls, /中断解题/)
assert.match(controls, /正在中断/)
assert.match(settings, /send_fast/)
assert.match(settings, /cancel_solve/)
assert.match(dock, /fastSendShortcut/)
assert.doesNotMatch(controls, /暂停后继续|恢复生成/)
```

SSR 渲染 `SolveControls` 三种状态：running normal、running fast、cancelled retryable；断言按钮文字和 disabled。

- [ ] **Step 2：运行红灯**

```bash
node frontend/solve_retry_cancel_fast_regression_test.mjs
node frontend/shortcut_hint_sync_regression_test.mjs
node frontend/settings_regression_test.mjs
node frontend/screenshot_dock_regression_test.mjs
```

Expected: 缺组件和 action。

- [ ] **Step 3：实现 SolveControls**

组件 props：

```js
const props = defineProps({
  running: Boolean,
  cancelling: Boolean,
  cancelled: Boolean,
  canRetry: Boolean,
  retryable: { type: Boolean, default: null },
  mode: { type: String, default: 'normal' },
  busy: Boolean,
})
const emit = defineEmits(['retry', 'cancel'])
```

规则：running 显示 cancel；cancelling 显示禁用“正在中断…”；failed/cancelled 且 canRetry 显示 retry；`retryable=false` 使用次级样式和“请先检查配置”的 title。按钮必须为 `type="button"`，带 `aria-busy`。

- [ ] **Step 4：接入 SolveView/ErrorView 和快捷键提示**

- typed round 行内 error/cancelled 下挂 SolveControls。
- loading/thinking/streaming 下只对当前 stream target显示 cancel。
- 旧 ErrorView 增加 `retry` emit，并由 `canRetry` 控制按钮显示；删除“按截图快捷键重试”的错误文案。
- Settings 新增 action 两行。
- Welcome/TopBar 显示快速键；TopBar 空间不足时 title 保留完整说明。
- ScreenshotDock 在有截图时显示普通发送和快速发送；三张上限提示不变。

Settings Store：

```js
const fastSendShortcut = computed(() => shortcuts.send_fast?.keyName || 'Ctrl+Shift+J')
const cancelSolveShortcut = computed(() => shortcuts.cancel_solve?.keyName || 'Ctrl+Shift+X')
```

macOS 使用配置返回值；fallback 根据 isMacOS 选择。

- [ ] **Step 5：运行聚焦和全部现存前端回归**

```bash
node frontend/solve_retry_cancel_fast_regression_test.mjs
node frontend/shortcut_hint_sync_regression_test.mjs
node frontend/settings_regression_test.mjs
node frontend/screenshot_dock_regression_test.mjs
for test_file in frontend/*_test.mjs; do node "$test_file" || exit 1; done
```

Expected: 全部 PASS。

- [ ] **Step 6：边界检查**

```bash
git diff --check -- frontend
git diff --cached
```

Expected: 无浏览器 confirm、无本地持久化 retry 数据；暂存区为空。建议归入未来功能提交。

---

### Task 9：Mobile cancelled/fast 状态和安全提示

**Files:**
- Modify: `app/mobile.go:40-90,190-231`
- Modify: `app/mobile_test.go`
- Modify: `pkg/mobile/state.go:8-57`
- Modify: `pkg/mobile/bridge.go:143-212`
- Modify: `pkg/mobile/bridge_server_test.go`
- Modify: `pkg/mobile/web/index.html`
- Modify: `pkg/mobile/web_dom_test.mjs`

**Interfaces:**
- Adds `StatusCancelled = "cancelled"`。
- Adds `Payload.SolveMode string`。
- Adds solution output event `solution-cancelled`。
- Mobile remains read-only。

- [ ] **Step 1：写 Bridge cancelled/fast 红灯**

`bridge_server_test.go`：

```go
func TestBridgeFastModeAndCancelledPreservePartialAnswer(t *testing.T) {
    b := newTestBridge(t)
    b.Publish("solution-stream-start", "fast")
    b.Publish("solution-stream-chunk", "B")
    detail := llm.DiagnoseError(context.Canceled)
    b.Publish("solution-cancelled", "模型请求已取消", detail)
    state := b.GetState()
    if state.Status != StatusCancelled || state.Payload.SolveMode != "fast" || state.Payload.Answer != "B" {
        t.Fatalf("state=%+v", state)
    }
    if len(state.Payload.AnswerItems) != 0 || state.Payload.AnswerCopyText != "" {
        t.Fatal("cancelled snapshot must not expose completed answer summary")
    }
}
```

扩展 `app/mobile_test.go` 的 hints 测试：

```go
func TestMobileShortcutHintsIncludeFastAndCancelWithoutComboIDs(t *testing.T) {
    hints := mobileShortcutHints(map[string]shortcut.KeyBinding{
        "send_fast":    {ComboID: "74+160+162", KeyName: "Ctrl+Shift+J"},
        "cancel_solve": {ComboID: "88+160+162", KeyName: "Ctrl+Shift+X"},
    })
    encoded, err := json.Marshal(hints)
    if err != nil { t.Fatal(err) }
    text := string(encoded)
    if !strings.Contains(text, "快速解题") || !strings.Contains(text, "中断解题") ||
        !strings.Contains(text, "Ctrl+Shift+J") || !strings.Contains(text, "Ctrl+Shift+X") {
        t.Fatalf("hints=%s", text)
    }
    if strings.Contains(text, "74+160+162") || strings.Contains(text, "88+160+162") || strings.Contains(strings.ToLower(text), "combo") {
        t.Fatalf("mobile hints leaked combo ids: %s", text)
    }
}
```

- [ ] **Step 2：写 Mobile DOM 红灯**

在 `web_dom_test.mjs`：

```js
diagnosticSource.emit('message', snapshot('cancel-session', 1, 10, 'partial', 0, '', {
  status: 'cancelled', payload: { answer: 'partial', solveMode: 'fast' }
}))
assert.match(elements.get('status-badge').textContent, /已中断/)
assert.match(elements.get('answer-text').innerHTML, /partial/)
assert.equal(elements.get('error-card').style.display, 'none')
assert.match(elements.get('waiting-text').textContent + elements.get('answer-card').textContent, /内容可能不完整|已中断/)
```

错误状态提示必须包含“在桌面端重试”，但不能生成按钮或 fetch 命令端点。

- [ ] **Step 3：运行红灯**

```bash
go test ./pkg/mobile ./app -run 'Mobile|BridgeFastModeAndCancelled|ShortcutHint' -count=1 -v
node pkg/mobile/web_dom_test.mjs
```

Expected: 缺 StatusCancelled / solveMode / hint。

- [ ] **Step 4：实现 Snapshot、Bridge、Router 和 Web UI**

- `solutionOutputEvents` 增加 solution-cancelled。
- mobile hint 定义增加 send_fast/cancel_solve。
- App 的 `routeConversationSolutionEvent(expectedDisplay, target, mode, event, data...)` 只在路由 `solution-stream-start` 到 Bridge 时传 `string(mode)`；Desktop envelope 始终由独立 `Mode` 字段携带模式。
- `routeSolutionEvent`、`routeTaskSolutionEvent`、`routeConversationSolutionEvent` 的 mobile-only hidden 终态分支都纳入 `solution-cancelled`，以 `success=false` 收敛桌面 loading。
- Bridge start 从 `data[0]` 白名单解析 normal/fast，非法值归 normal；无参数的旧调用保持 normal 兼容。
- cancelled 保留 partial Answer，清 AnswerItems/AnswerCopyText，使用清洗诊断。
- Web cancelled 不调用 `showError`；显示部分答案和中断提示。
- error 继续诊断，并新增纯文字“可在桌面端重试本题”。
- 不新增 POST、WebSocket、control action 或 URL 参数。

- [ ] **Step 5：运行 Mobile 聚焦与回归**

```bash
go test ./pkg/mobile ./app -count=1
node pkg/mobile/web_dom_test.mjs
```

Expected: PASS；Reset/ClearDisplay、截图 Dock、SSE 重连测试仍通过。

- [ ] **Step 6：安全扫描与边界检查**

确认 Mobile Snapshot/HTML 不出现 `recordId`、`retryToken`、`comboID`、图片字段或调用 RetrySolve 的代码。

```bash
git diff --check -- app/mobile.go pkg/mobile
git diff --cached
```

Expected: 暂存区为空。建议归入未来功能提交。

---

### Task 10：固定 Wails v2.11.0 bindings、双语说明与自动门禁

**Files:**
- Modify generated: `frontend/wailsjs/go/app/App.js`
- Modify generated: `frontend/wailsjs/go/app/App.d.ts`
- Modify generated: `frontend/wailsjs/go/models.ts`
- Modify: `README.md`
- Modify: `README_EN.md`
- Modify: Change 038 `03-implementation.md`、`04-validation.md`

**Interfaces:**
- Binding methods: `TriggerQuickSend()`、`RetrySolve(string,string)`、`CancelSolve()`。
- Generated DTO: `SolveControlResult`。

- [ ] **Step 1：完成生成前只读保护清单**

记录普通文件 SHA-256，不读取用户配置：

```bash
git status
git branch
git remote -v
git diff
git diff --cached
git diff -- go.mod go.sum frontend/wailsjs frontend/dist
```

在仓库外临时目录记录 `go.mod`、`go.sum`、`frontend/wailsjs/**`、`frontend/dist/**` 和 `.git/index` 哈希；目录本身不得传给 `sha256sum`。

- [ ] **Step 2：固定 v2.11.0 生成 bindings**

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 generate module -nocolour
```

Expected:

- App.js/App.d.ts 增加三个方法。
- models.ts 增加 SolveControlResult。
- `go.mod`、`go.sum`、runtime 和 Git index 内容不变。
- 若只有纯空白 Tab 噪音，定向清除空白，不手工改生成类型语义。

- [ ] **Step 3：更新 README 中英文**

说明必须包含：

- 错误卡可复用原图手动重试；每次重试可能产生新费用。
- 中断不是暂停续传；部分内容可能不完整。
- 普通/快速默认快捷键及可重绑定。
- 快速模式使用 `gen-speed` 但不修改默认场景，也不保证网络/服务端固定时延。
- Mobile 只显示状态和提示，不能远程重试/取消。

不得记录真实 URL、Key 或测试配置。

- [ ] **Step 4：运行全部自动门禁**

```bash
go test ./pkg/task ./pkg/solution ./pkg/questionbank ./pkg/shortcut ./pkg/mobile ./app -count=1
go test ./... -count=1
go vet ./...
go build ./...
go mod verify
for test_file in frontend/*_test.mjs; do node "$test_file" || exit 1; done
npm --prefix frontend run build
git diff --check
```

Expected: 全部普通门禁 PASS；Vite 仅允许既有大 chunk warning。

- [ ] **Step 5：执行相关 race 并如实记录**

```bash
go test -race ./pkg/task ./pkg/solution ./pkg/questionbank ./pkg/shortcut ./pkg/mobile ./app -count=1
```

Expected: 若环境可运行则 PASS；若启动阶段 `0xc0000139`，逐包记录 BLOCKED，不禁用 race、不写 PASS。

- [ ] **Step 6：复核生成保护边界**

比较生成前后保护清单。Expected：依赖/runtime 无语义变化，frontend/dist 仅为 ignored 构建产物，Git 暂存区为空。

- [ ] **Step 7：记录建议提交边界**

更新 03/04 记录真实命令、退出结果和警告。建议功能提交包含业务代码、测试、bindings 和 README；治理提交包含 `_dev/audits/038*`、Change 038、INDEX。未经授权不暂存、不提交。

---

### Task 11：保护候选、GUI 验收与 Commit Boundary Audit

**Files:**
- Modify: `_dev/changes/038-solve-retry-cancel-fast-mode/03-implementation.md`
- Modify: `_dev/changes/038-solve-retry-cancel-fast-mode/04-validation.md`
- Modify: `_dev/changes/038-solve-retry-cancel-fast-mode/05-rollback.md`
- Modify: `_dev/changes/038-solve-retry-cancel-fast-mode/06-notes.md`
- Modify: `_dev/audits/038-solve-retry-cancel-fast-mode.md`
- Modify: `_dev/changes/INDEX.md`
- Generate ignored: `build/bin/testing/current/Q-Solver-candidate-change038.exe`

**Interfaces:**
- No new product interface。
- Produces candidate evidence and future dual-commit whitelist。

- [ ] **Step 1：冻结构建产物与 Git 保护基线**

按 `_dev/build-artifact-policy.md`：

- 盘点 `build/bin/**` 路径、大小、SHA-256。
- 冻结 `releases/**`、稳定版、go.mod/go.sum、bindings/runtime、dist、Git index 和暂存区。
- 确认根目录唯一 EXE，复制为 testing/current 稳定前副本。
- `testing/current/` 同时只允许一个 Change 038 候选。

- [ ] **Step 2：固定 Wails v2.11.0 构建唯一候选**

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build \
  -s \
  -o Q-Solver-candidate-change038.exe \
  -skipbindings \
  -skipembedcreate \
  -nosyncgomod \
  -m \
  -nocolour \
  -v 1 \
  -ldflags "-s -w" \
  -tags prod
```

将候选放入 testing/current，生成并验证 SHA256SUMS。`go version -m` 必须证明链接 Wails v2.11.0。

- [ ] **Step 3：独立启动烟测**

使用空隔离 APPDATA/LOCALAPPDATA 启动候选，验证：Started=true、WebView2 子进程存在、无 Node/Vite 子进程、正常窗口关闭、ExitCode=0、无残留候选进程。不得读取正式用户配置。

- [ ] **Step 4：固定隔离 Profile GUI 验收**

在同一隔离 Profile 完成：

1. 1～3 张截图普通发送。
2. 快速快捷键使用 gen-speed，设置页默认场景不变。
3. waiting/thinking/content 三阶段中断；状态为已中断，部分内容提示不完整。
4. 网络失败后点击重试，不调用截图、不新增题库 Record。
5. 重试双击被阻止；新截图/开始下一套后旧重试失效。
6. KeepContext on/off 各一轮，不重复历史。
7. 自定义快速/中断快捷键保存并重启恢复；冲突安全拒绝。
8. desktop/mobile/dual 状态、提示、SSE 重连、Reset/ClearDisplay。
9. 三图 Dock、删除补拍、发送和 Change 035/036 行为不回归。

不使用真实收费 Provider时，可用本地隔离 fake/httptest 验证网络失败；任何真实调用需另行授权。

- [ ] **Step 5：根据结果处理候选**

- 必要项 PASS：保留 current，等待用户明确晋升授权。
- FAIL：根稳定版不变，候选进入 failed 或保留 current 供修复。
- BLOCKED/NEEDS_USER：不得 Close 或晋升。
- 不修改 releases，不使用 `git add -f`。

- [ ] **Step 6：Commit Boundary Audit**

审计：

- 功能白名单与治理白名单分离。
- 无其他 Change、用户配置、日志、二进制、dist、依赖噪音或 secret。
- 所有测试证据真实，race/真实 API/GUI 状态单列。
- 列出两个完整建议 message，但不执行：

```text
feat: add retry cancel and fast solve controls

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

```text
chore: add change 038 governance records

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

- [ ] **Step 7：最终只读 Git 检查并停止**

```bash
git status
git branch
git remote -v
git diff
git diff --cached
git diff --check
```

Expected: 仅 Change 038 预期未提交修改，暂存区为空。报告候选路径、大小、SHA-256、验证结果和遗留项；等待用户分别授权 commit、push 和稳定版晋升。

---

## 计划自审清单

- [x] Spec §3～§19 每项均映射到 Task 1～11。
- [x] 所有新增类型、方法、事件、action 和 DTO 名称跨任务一致。
- [x] 没有要求前端或 Mobile 传图片、Config、Record 路径或凭据。
- [x] 没有自动重试、断点续传、Mobile 控制端点或第二套 Profile。
- [x] 每个实现任务都有红灯命令、最小实现、聚焦绿灯和边界检查。
- [x] 所有 Git 步骤只记录建议边界，未授权时不暂存、不提交、不推送、不晋升。

规格映射：§3～4 → Task 1；§5～6 → Task 3～4；§7 → Task 1、4、7；§8 → Task 2、4；§9～10 → Task 3～5；§11～12 → Task 4、5、7、8；§13 → Task 6、8；§14 → Task 9；§15～17 → Task 1、3～6、9、10；§18～19 → Task 10～11。
