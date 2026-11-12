# 实现过程记录 03

## 当前状态

2026-11-11 已按用户批准的 `implementation-plan.md` 完成业务实现、bindings、README、普通自动门禁和保护候选验证。用户确认当前测试版本没有问题，功能提交 `5564a45` 与治理提交 `2502b31` 已创建并推送；随后按用户“本地版本更新”授权完成稳定版晋升。race 环境阻塞继续披露，未发布。

## 实现日志

| 日期 | 步骤 | 改动内容 | 主要范围 | 结果 |
|---|---|---|---|---|
| 2026-11-10 | 只读现状审计 | 审计发送、取消、题库 Record、KeepContext、快捷键、Desktop/Mobile 状态 | `01-current-state.md`、架构审计 | 完成 |
| 2026-11-10 | 设计批准 | 确认后端单槽 attempt、同轮重试、真实取消和请求级 `gen-speed` | `design.md`、`02-plan.md` | 用户批准 |
| 2026-11-11 | 书面规格与详细计划 | 锁定状态机、接口、安全边界和 11 项 TDD 任务 | `design.md`、`implementation-plan.md` | 用户批准实施 |
| 2026-11-11 | SolveAttempt | 新增单槽状态机、截图深拷贝、task/target/epoch 校验、终态与失效 | `app/solve_attempt.go`、测试、`app/app.go` | 聚焦及 App 全包通过 |
| 2026-11-11 | QuestionBank 原位重启 | terminal Record clone → 修改 → 保存 → 交换；复用 ID、CreatedAt、Images | `pkg/questionbank/store.go`、捕获层、测试 | 原子性和拒绝矩阵通过 |
| 2026-11-11 | 统一 normal/fast/retry 管线 | normal 保留首次场景；fast 请求副本覆盖 `gen-speed`；retry 使用当前连接但复用首次 mode/KeepContext | `app/solve.go`、相关测试 | 通过 |
| 2026-11-11 | 取消生命周期 | `CancelSolve`、`solution-cancelled`、context 取消、in-flight draining 屏障 | `pkg/solution`、`app`、题库 capture、测试 | 通过 |
| 2026-11-11 | 快捷键与迁移 | 新增 `send_fast` / `cancel_solve`，默认值、冲突安全迁移和 FIFO 分发 | `pkg/config`、`pkg/shortcut`、测试 | 通过 |
| 2026-11-11 | Desktop 状态与控制 | typed mode/canRetry、同轮 reset、部分正文保留、重试/中断按钮、安全 Toast 和动态提示 | Pinia、Vue 组件、API、回归测试 | 19 个前端回归通过 |
| 2026-11-11 | Mobile 只读状态 | `StatusCancelled`、`Payload.SolveMode`、部分正文、中断提示、回桌面重试提示和动态快捷键 | `app/mobile.go`、`pkg/mobile/**` | Mobile/App/DOM 回归通过 |
| 2026-11-11 | bindings 与文档 | 固定 Wails v2.11.0 生成三个 App 方法和 `SolveControlResult`；同步中英文 README | `frontend/wailsjs/go/**`、README | 依赖/runtime 哈希不变 |
| 2026-11-11 | 自动门禁 | 前端、Vite、Go 测试、vet、build、mod verify、diff check | 全仓相关范围 | 普通门禁 PASS；race BLOCKED |
| 2026-11-11 | Commit Boundary Audit 回修 | 发现 Mobile 页面内历史会把 cancelled 条目重新显示为 completed；补充 status/solveMode 历史语义和真实 SSE 回归 | `pkg/mobile/web/index.html`、`web_dom_test.mjs` | 红灯复现后修复，全量门禁重跑通过；旧候选归档 transient |
| 2026-11-11 | 保护候选 | 固定 Wails v2.11.0、保护参数、独立候选、隔离 Profile 基础 GUI | `build/bin/testing/current/` | 最终候选通过，等待用户晋升授权 |
| 2026-11-11 | 本地稳定版晋升 | 复核候选/旧稳定版/Release 哈希；旧版归档、同卷替换根 EXE、验证版归档并清空测试区 | `build/bin/Q-Solver.exe`、`archive/{superseded,validation}/change-038/` | 晋升完成；Release 未变 |

## 关键实现结果

### 后端权威重试

- `RetrySolve(sessionID, recordID)` 只接收稳定目标 ID，不接收 Base64、路径、Config、Prompt 或凭据。
- 失败/取消 attempt 在内存中保留最多三张截图；新截图、清空、正常/快速新请求、题库会话变化、关闭 KeepContext 等边界使旧快照失效。
- QuestionBank retry 原位重启同一 Record；持久化失败时内存和磁盘终态均不被改成 running。

### 中断和并发边界

- `CancelSolve` 在目标校验与 context 取消期间持有 `conversationMu`，防止误取消新任务。
- `TaskCoordinator.HasInFlightTask()` 在旧 goroutine 完全退出前保持 true；普通发送、快速发送和重试均返回 draining。
- `solution-cancelled` 与普通网络错误分离；已流出正文保留，但不形成 completed、答案摘要或 KeepContext 新轮。

### 快速模式与快捷键

- `SolveModeFast` 只修改请求 Config 副本的 `DomainId=gen-speed`，不保存默认场景、不切换 Connection Profile、不通知配置 subscriber。
- Windows 默认 `Ctrl+Shift+J` / `Ctrl+Shift+X`；macOS 默认 `⌘⇧J` / `⌘⇧X`。
- 缺失 action 仅在组合空闲时补默认；显式空绑定、自定义绑定和已占用组合均保留。

### Desktop 与 Mobile

- Desktop 错误/中断轮次原位重试；运行中显示“中断解题”，cancelling 显示“正在中断…”。
- ScreenshotDock、Welcome、TopBar、设置页和 Mobile 使用实际绑定显示名。
- Mobile 仅同步 mode、cancelled、部分 Answer 与安全诊断，不新增重试/中断端点、Record ID、ComboID、图片或凭据。

## 与计划的偏差

1. `pkg/task/coordinator.go` 无需修改：既有 `GetCurrentTaskID`、`CancelCurrentTask`、`HasInFlightTask` 和 `CompleteTask` 足以实现屏障。
2. 候选 GUI 的初始自动截图被 Ghost Mode 正常排除；通过隔离 Profile 按 `F9` 关闭防录屏后才取得可审计画面。这是既有产品行为，不是候选空白。
3. 隔离 Profile 未配置模型，`F8` 被安全拦截并打开设置，因此自动 GUI 未逐项覆盖 ScreenshotDock、真实失败重试、运行中取消或快速 Provider 请求；用户随后确认当前测试版本没有问题，结合自动测试完成产品验收。
4. WebView2 production 未接受临时调试端口；未修改 DevTools/production 设置绕过。

## 产物边界

- 晋升前候选：`build/bin/testing/current/Q-Solver-candidate-change038.exe`
- 晋升后根稳定版：`build/bin/Q-Solver.exe`
- 大小：42,550,784 字节
- SHA-256：`8A868263F29E170F1204E8BABC8064055D5533B870ED1A023309C99F1758D220`
- 被替代稳定版：`build/bin/archive/superseded/change-038/Q-Solver-stable-before-change038.exe`，SHA-256 `8B19B242F7917D4BC0267E45BB6248077E2BA5BD2D3CE428AC9E1B836C8B4994`。
- 验证归档：`build/bin/archive/validation/change-038/Q-Solver-candidate-change038.exe`，哈希与根稳定版一致。
- `go version -m`：候选链接 `github.com/wailsapp/wails/v2 v2.11.0`。
- `testing/current/` 已清空；三个 Release 资产及清单未改变。
- 首版候选 `ABB231…67AB9` 在 Commit Boundary Audit 发现 Mobile 历史 cancelled 语义缺口后失效，已按政策移入 `archive/transient/Q-Solver-candidate-change038-before-mobile-history-fix.exe` 并通过 transient 清单验证。

## 用户验收与提交

- 2026-11-11：用户说明当前测试版本验证“没有什么问题”，并明确授权对本次修改执行提交和推送。
- 功能提交：`5564a45 feat: add retry cancel and fast solve controls`。
- 治理提交：`2502b31 chore: add change 038 governance records`；两个提交已推送至 `origin/dev/qsolver-custom`。
- 当前分支即 `dev/qsolver-custom`，没有独立功能分支可合并；遵守项目规则，不合并或修改 `main`。
- 用户随后明确要求“本地版本更新”，已完成根稳定版晋升；Tag、Release 仍未获授权且未执行。

## 遗留与披露

- race：所有目标包启动阶段统一返回 Windows `0xc0000139`，状态 `BLOCKED`，已向用户披露。
- 根稳定版现为 Change 038，SHA-256 `8A868263F29E170F1204E8BABC8064055D5533B870ED1A023309C99F1758D220`；旧稳定版与验证候选均已归档。
- 本 Change 未调用真实收费 API，未写入正式用户配置。
