# Change 038 解题重试、中断与快速模式设计

## 1. 状态与目标

本设计于 2026-11-10 获用户批准，书面规格于 2026-11-11 获用户批准。详细 TDD 实施计划已完成并等待批准；业务源码尚未修改。

目标是用一个统一、后端权威的解题尝试生命周期实现：

1. 失败或中断后复用原始截图手动重试。
2. 真实取消当前 Provider 请求，而不是只停止前端等待。
3. 普通发送与快速发送共享主链路，快速发送只对单次请求使用 `gen-speed`。

## 2. 术语

- **普通发送（normal）**：使用当前持久化 `DomainId`。
- **快速发送（fast）**：单次请求覆盖 `DomainId=gen-speed`，不保存配置。
- **中断（cancel）**：取消 context 并等待 goroutine 收尾；不能从 token 断点继续。
- **重试（retry）**：复用同一尝试的图片、轮次身份和模式，重新发起完整请求。
- **活动任务**：当前仍可接受取消的任务。
- **in-flight 任务**：已启动但尚未执行 `CompleteTask` 的任务，包括正在取消的任务。

## 3. 核心状态机

```text
idle
  └─ start(normal|fast) → running
running
  ├─ cancel request → cancelling
  ├─ provider failure → failed(retry available)
  └─ success → completed(retry cleared)
cancelling
  └─ provider exits → cancelled(retry available)
failed|cancelled
  ├─ retry → running(same display target / same question-bank record)
  └─ new screenshot/send/session generation → invalidated
```

任何时刻最多存在一个 active attempt 和一个与其同源的 retry snapshot。`completed`、新原始发送、开始下一套、题库会话切换或 conversation epoch 变化都会让旧重试失效。

## 4. 后端领域模型

建议在 `app` 新增非持久化类型：

```go
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

type solveAttempt struct {
    target            desktopSolveTarget
    screenshots       []string
    mode              SolveMode
    normalDomainID    string
    keepContext       bool
    conversationEpoch uint64
    state             solveAttemptState
    taskID            int64
}
```

约束：

- `screenshots` 深拷贝，数量 1～3；不记录路径。
- 不复制 API Key、Headers、Token、Connection Profile 或完整 Config。
- `normalDomainID` 只用于普通重试；fast 始终解析为 `gen-speed`。
- attempt 受专用 mutex 保护；不得持锁调用 Provider、题库 Store、Wails Emit 或磁盘 I/O。
- 锁顺序固定：`conversationMu` → attempt mutex → TaskCoordinator；反向获取禁止。

## 5. 统一启动管线

现有 `triggerSend(bool)` 重构为内部启动方法，接收：

```go
type solveStartRequest struct {
    mode        SolveMode
    retryTarget *desktopSolveTarget
    screenshots []string
}
```

公开入口：

- `TriggerSend()`：normal。
- `TriggerQuickSend()`：fast。
- `ConfirmSolveWithoutResume()`：继续此前待确认的 mode，不能固定回 normal。
- `RetrySolve(sessionID, recordID)`：从后端 retry snapshot 构造请求。

普通/快速原始发送沿用“无待发送截图时自动截图”行为。Retry 入口绝不调用截图服务，也不使用当前 `screenshotBuffer`。

所有启动入口先检查 `HasInFlightTask()`，而不是只检查 `HasRunningTask()`；旧取消任务尚未退出时返回固定 `solve_task_draining`，不启动重叠 Provider。

## 6. 请求级快速模式

启动时读取当前 Config 副本：

- normal：保留当前 `DomainId`。
- fast：设置副本 `DomainId = "gen-speed"`。

禁止调用 `ConfigManager.Save`、`UpdateFromJSON`、subscriber 或修改活动 Connection Profile。连接、模型、Endpoint 和凭据继续使用重试当下的活动 Config，使用户修复连接设置后可以复用原图重试。

attempt 保存首次实际使用的 mode 和 normalDomainID。重试时：

- fast 继续使用 `gen-speed`。
- normal 使用 attempt 的 normalDomainID。
- 当前连接和模型重新读取。

快速模式只保证使用简短场景并减少输出要求，不承诺服务端或网络时延上限。

## 7. 重试授权与 API

新增结构化结果：

```go
type SolveControlResult struct {
    Success bool   `json:"success"`
    Code    string `json:"code,omitempty"`
    Message string `json:"message,omitempty"`
}
```

固定安全错误码：

- `solve_no_retry_available`
- `solve_retry_target_mismatch`
- `solve_task_running`
- `solve_task_draining`
- `solve_retry_stale`
- `solve_retry_prepare_failed`
- `solve_cancel_requested`
- `solve_no_running_task`

`RetrySolve(sessionID, recordID)` 只接受当前 retry snapshot 的目标。它不接受图片、Config、Prompt、Key 或任意文件路径。错误文案不包含截图、URL、Record 路径或底层响应正文。

重试按钮只代表手动发起新的模型请求，可能产生费用；不自动循环，不在网络层静默重发。

## 8. 题库 Record 原位重启

新增 Store 原子方法，概念签名：

```go
func (s *Store) RestartRecord(recordID, model, scene string, keepContext bool) error
```

规则：

1. 只操作当前 active/paused 可写分段中的 Record。
2. 只允许 `failed`、`cancelled`、`interrupted`。
3. Record 必须有 1～3 个既有图片引用。
4. 候选副本中将 Status 设为 running，清空 Answer、AnswerItems、Questions、OrganizeError、OrganizeFailure 和 MetadataSuggestion。
5. 更新 Model、Scene、KeepContext；保留 ID、CreatedAt 和 Images。
6. 先完整落盘候选 Session，成功后再交换内存状态；失败保持旧 Record 不变。
7. 原位 restart 不新增 Record，不改变已有图片文件。

App retry capture 使用同一 `recordID`。无题库模式使用 attempt 原 target，确保 Desktop 同一轮原位刷新。

## 9. KeepContext 与题号上下文

- `Solver.Solve` 失败/取消时不提交 user/assistant 对，现有约束保留。
- Retry 仅在最终成功后提交一次。
- attempt 的 conversation epoch 必须等于当前 epoch；否则返回 stale。
- 新的正常请求、题库续接、开始下一套或清历史使旧 retry snapshot 失效。
- 同一 Record retry 时 `CurrentNumberingContext(recordID)` 继续排除当前 Record，不重复占号。
- KeepContext 开关按首次 attempt 语义保留；连接和模型使用重试时当前配置。

## 10. 中断语义

新增 App 控制入口 `CancelSolve() SolveControlResult`，保留 `CancelRunningTask()` 作为旧 binding 兼容包装，直到前端和生成 bindings 完成迁移。

流程：

1. attempt 从 running 原子切换到 cancelling。
2. 调用 `TaskCoordinator.CancelCurrentTask()`。
3. UI 立即显示“正在中断…”，按钮禁用。
4. Provider 因 context cancellation 退出。
5. Solver 发出 `solution-cancelled`，不再把用户取消包装成普通 `solution-error`。
6. 题库 capture 结束为 `cancelled`；部分 Answer 不作为完整答案或 KeepContext。
7. goroutine `CompleteTask` 后 attempt 进入 cancelled，可手动重试。

重复取消幂等；没有任务时返回 `solve_no_running_task`。中断不强制杀 goroutine，不允许旧任务完全退出前启动新请求。

## 11. 事件契约

`solutionOutputEvents` 新增 `solution-cancelled`。`DesktopSolveEvent` 增加非敏感字段：

```go
Mode     string `json:"mode,omitempty"`
CanRetry bool   `json:"canRetry,omitempty"`
```

Desktop typed 事件：

- `request-start`：携带 mode 和原始图片，retry 时目标与失败轮相同。
- `solution-stream-start/thinking/chunk`：行为不变。
- `solution-error`：失败终态，按后端快照设置 CanRetry。
- `solution-cancelled`：中断终态，CanRetry=true。
- `solution`：成功终态，CanRetry=false 并清除 retry snapshot。

前端继续用 epoch/sessionId/recordId 拒绝迟到事件。retry 的 `request-start` 命中既有轮次时必须重置 thinking、正文、答案摘要、错误和 status，而不是追加新轮。

## 12. Desktop UX

### 运行中

当前活动轮在 waiting、thinking、streaming 状态显示：

- normal：“AI 正在回复…”
- fast：“快速解题中…”
- 按钮：“中断解题”
- 点击后：“正在中断…”且禁用

### 失败/中断

行内错误卡片和旧全局 ErrorView 都支持：

- 主按钮“重试本题”。
- 诊断 retryable=true 时使用强调样式。
- retryable=false 时使用次级样式并提示先检查配置；仍允许修复设置后手动重试。
- cancelled 显示“已中断，以上内容可能不完整”，不是“请求出错”。
- 重试中按钮禁用，防止双击。
- 后端拒绝 stale/target mismatch 时保留原卡片并显示安全 Toast。

前端不保存图片重试缓存，不向 Retry API 传 Base64。

## 13. 快捷键

新增 action：

| Action | Windows 默认 | macOS 默认 | 行为 |
|---|---|---|---|
| `send_fast` | `Ctrl+Shift+J` | `⌘⇧J` | 快速发送/自动截图 |
| `cancel_solve` | `Ctrl+Shift+X` | `⌘⇧X` | 中断当前解题 |

迁移规则：

- 对旧配置分别检测 action 是否显式存在。
- 未显式配置且默认 ComboID 未被占用时补默认。
- 默认被其他 action 占用时写入空 binding，不夺取组合键。
- 用户显式空 binding 保持为空。
- 录制冲突继续由 Shortcut Service 统一拒绝。
- action 仍进入 Change 035 单 worker FIFO；Hook 回调不得执行截图、Provider 或取消等待。

设置页、Welcome、TopBar、ScreenshotDock 和 Mobile Hints 使用实际 KeyName。ScreenshotDock 未满三张时保留截图/删除/普通发送，并新增快速发送；取消提示只在任务运行状态显示或进入完整快捷键列表。

## 14. Mobile

Mobile 保持只读，不新增 HTTP 命令、表单、重试 token 或 Record ID。

`Snapshot.Status` 增加 `cancelled`；Payload 可增加 `SolveMode`（`normal|fast`），不包含场景 Prompt。Bridge：

- start 时记录 mode。
- cancelled 时保留已接收的 Answer，清空答案摘要，状态设为 cancelled。
- error 继续携带清洗后的诊断。
- done 行为不变。

Web UI：

- fast 显示“快速处理中”。
- cancelled 显示“已中断，内容可能不完整”。
- error 显示“可在桌面端重试本题”。
- Hints 增加快速解题和中断解题实际按键。

ClearDisplay、Reset、clearSeq、seq、截图数量和 SSE 重连语义保持不变。

## 15. 并发与失效规则

- App 启动任何新请求前检查 `HasInFlightTask()`。
- retry/cancel 使用版本或 attempt identity 防止双击。
- 不在锁内调用 Wails、Provider、Store 或 Bridge。
- 旧 taskID 的 Complete 不能清理新 attempt。
- 迟到的旧 chunk/error/solution 必须同时通过 task/target/epoch 校验才可写 UI、Mobile 或题库。
- ClearScreenshots 只清待发送截图；若当前失败 retry snapshot 存在，应显式失效，避免“清除”后仍可重试旧图。
- 新截图、普通/快速原始发送、开始下一套、题库续接和 conversation clear 都使旧 retry snapshot 失效。

## 16. 安全与隐私

- retry snapshot 仅内存、单槽、有界，最多三张图；进程退出即消失。
- 不在日志、Toast、DTO 或治理记录输出图片、Key、Header、URL、题库路径或上游响应正文。
- Retry API 不接收图片或配置，防止前端构造任意请求。
- Mobile 不传图片、目标 ID、ComboID 或控制 token。
- 自动测试只使用 fake Provider / `httptest`。
- 每次手动重试可能产生新模型费用，UI 明示；禁止自动重试。

## 17. 兼容性

- 新快捷键 Map key 对旧版本可忽略。
- 题库不新增必须字段；Restart 复用现有 Record schema。
- `solution-cancelled` 是新增事件；旧前端可忽略，当前前端与 Mobile 同步升级。
- 保留旧 `CancelRunningTask` binding 兼容；新 UI 使用结构化 `CancelSolve`。
- Wails runtime 固定 v2.11.0，不升级依赖。

## 18. 测试矩阵

### Go

- attempt：single-slot、clone、target、stale、double-click、success invalidation。
- task/solution：cancel before request、during thinking、during content、late success、KeepContext。
- questionbank：restart 状态矩阵、原子失败、图片引用、同 Record、numbering。
- shortcut/config：默认值、显式空值、冲突、迁移、FIFO。
- mobile：cancelled、mode、重放、Reset/ClearDisplay、诊断清洗。
- app：normal/fast/retry/cancel 端到端 fake Provider。

### Frontend

- typed retry 同轮重置，不新增 round。
- retry/cancel 按钮状态、双击、Toast、安全诊断。
- partial answer 在 cancelled 下保留。
- normal/fast 状态和实际快捷键提示。
- Mobile DOM 的 cancelled/error/fast 及 SSE 去重。
- 现有所有回归继续通过。

### 人工

- 题库开/关，KeepContext 开/关，desktop/mobile/dual。
- 1～3 张截图，网络断开/恢复，Provider 503，流中途断开。
- 普通/快速快捷键，自定义绑定，冲突，鼠标穿透和窗口隐藏。
- 中断后立即重试、修复 Connection Profile 后重试、开始下一套后旧按钮失效。

## 19. 关闭门禁

- 必要自动测试、Vite、Go build/vet/mod verify、Wails v2.11.0 保护构建和候选启动完成。
- 必要 GUI 人工验收 PASS；FAIL/BLOCKED/NEEDS_USER 不冒充通过。
- race 若因环境 `0xc0000139` 继续 BLOCKED，单独披露，不替代普通测试。
- 完成 Commit Boundary Audit；commit、push、稳定版晋升、Tag 和 Release 分别等待明确授权。
