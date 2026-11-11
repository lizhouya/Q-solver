# 现状快照 — 01

## 1. 涉及文件清单

| 文件 | 当前职责 | 关键事实 |
|---|---|---|
| `app/solve.go` | 截图缓冲、发送、任务启动、Solver/题库收尾 | 发送时复制并立即清空截图缓冲；已有 `CancelRunningTask`，没有 Retry API |
| `app/app.go` | App 依赖和运行态所有权 | 保存 `TaskCoordinator`、Solver、题库、截图缓冲和会话代际；没有解题尝试状态 |
| `pkg/task/coordinator.go` | 当前任务 context、取消和 in-flight 跟踪 | 取消后 `HasRunningTask=false`，但 `HasInFlightTask=true` 直到 goroutine Complete |
| `pkg/solution/solution.go` | 构建 Prompt、流式调用和 KeepContext 提交 | Provider 使用传入 context；失败/取消不提交 user/assistant 历史 |
| `app/questionbank_solve_capture.go` | 解题事件与题库 Record 生命周期桥接 | 请求前 Begin；错误标为 failed/cancelled；终态 Record 不能原位重新运行 |
| `pkg/questionbank/store.go` | Record 图片与状态落盘 | Begin 保存 1～3 张图片；Finish 只接受 running Record |
| `pkg/domain/prompts.go` | 内置场景 | 已有 `gen-speed`“秒杀技巧”，无需新增第二套持久场景 |
| `pkg/shortcut/service.go` | 快捷键 FIFO action 调度 | send 触发后端发送；无 send_fast / cancel_solve |
| `pkg/config/config.go` | 默认快捷键和兼容迁移 | Windows 已有 F8、Ctrl+J 等映射；新 action 必须冲突安全迁移 |
| `frontend/src/stores/solution.js` | 带轮次流状态、错误与历史 | typed 事件按 epoch/session/record 隔离；错误无可重试状态 |
| `frontend/src/components/SolveView.vue` | 当前轮、行内错误、loading 与答案 | 行内错误没有按钮；运行中没有中断入口 |
| `frontend/src/components/ErrorView.vue` | 旧全局错误视图 | 提示“按截图快捷键重试”，实际会重新截图 |
| `app/mobile.go`、`pkg/mobile/**` | Desktop/Mobile 路由和 SSE Snapshot | cancellation 目前作为 solution-error；Mobile 只读且不传截图 |

## 2. 关键代码现状

1. `triggerSend` 在启动 goroutine 前把 `screenshotBuffer` 置空，失败后 App 无权威图片重试对象。
2. `TaskCoordinator.CancelCurrentTask` 已取消 context；现有 OpenAI、Responses、Anthropic 和 Custom JSON Provider 均把 context 传入 SDK/HTTP。
3. `Solver.Solve` 只在完整成功且答案非空时提交 KeepContext；失败、取消和空响应不会追加历史。
4. 题库捕获会在请求前创建 Record 并保存图片；错误或取消后 Record 变为终态，现有 Store 没有 restart 操作。
5. Desktop typed event 已包含 `epoch/sessionId/recordId`，可以作为重试目标身份，但必须由后端再次校验。
6. `gen-speed` 已要求尽量只输出最终答案和必要的一句理由；全局答案汇总协议仍适用。
7. API 已暴露 `CancelRunningTask`，但前端没有入口；旧事件路径对取消仍存在“正在思考”式复位逻辑。

## 3. 相关配置

- 主配置继续加密保存；本 Change 不读取或记录真实配置值。
- 当前快捷键 Map 可向后兼容未知/新增 action；配置加载已有按 action 检测显式配置和冲突的迁移模式。
- `InterruptThinking` 字段存在但没有前端产品链路，本 Change 不复用该字段，避免把一次性控制误建模为持久偏好。

## 4. 依赖与构建现状

- Go module 指令 1.24.1；当前开发环境曾验证为 Go 1.27.1。
- Wails module/runtime 固定 v2.11.0。
- Vue 3 + Pinia + Vite；不新增 npm 或 Go 依赖。
- Change 037 收尾后工作区 clean，HEAD 与 `origin/dev/qsolver-custom` 一致。

## 5. 运行行为现状

- 最多三张截图；普通发送默认 `Ctrl+J`，截图默认 F8。
- 快捷键业务动作经单 worker FIFO，避免 Windows Hook 被截图/压缩阻塞。
- 发送消费截图后 Desktop/Mobile 待发送数量归零。
- Mobile 仅接收答案、简短分析、错误诊断、安全快捷键显示名和截图数量，不接收图片或控制凭据。
- 网络失败卡片只显示诊断；用户必须重新截图或另行触发发送，无法保证复用原图。

## 6. 已知现状问题

- Windows `go test -race` 在进程启动阶段可能返回 `0xc0000139`，须单列为环境 BLOCKED，不能冒充通过。
- 快速场景只能减少 Prompt 要求和输出长度，无法保证上游排队与网络时延。
- Wails 构建可能产生 bindings/runtime/依赖噪音，必须固定 v2.11.0 并执行保护哈希。
- 真实 provider、收费调用和正式用户配置不属于自动验证范围。

关联审计：`_dev/audits/038-solve-retry-cancel-fast-mode.md`。
