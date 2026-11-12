# 总结与注意事项 06

## 当前摘要

Change 038 已完成后端单槽 SolveAttempt、原图同轮重试、真实 context 中断、请求级快速模式、快捷键迁移、Desktop/Mobile 状态与提示、Wails bindings 和双语文档。普通自动门禁及保护候选验证通过；race 为 `BLOCKED` 并已披露。用户于 2026-11-11 确认当前测试版本没有问题并授权提交、推送，功能提交 `5564a45` 与治理提交 `2502b31` 已推送；随后明确要求更新本地版本，稳定版已晋升，未发布。

## 已实现原则

| 主题 | 实际结果 |
|---|---|
| 重试 | 后端权威、手动触发、单槽有界、只收 session/record 目标、原位复用同一轮 |
| 图片 | 后端内存深拷贝最多三张；前端不回传 Base64；应用重启后不保证可重试 |
| 题库 | terminal Record clone/save/swap 原位重启；ID、CreatedAt、Images 不变 |
| KeepContext | 失败/取消不提交；重试成功只提交完整成功轮 |
| 中断 | `CancelSolve` 取消 Provider context；不是暂停续传；旧任务退出前保持 draining |
| 快速模式 | 单次请求使用 `gen-speed`；不修改默认场景、活动 Profile 或 subscriber |
| Desktop | typed target、迟到隔离、原轮 retry、取消部分正文与控制按钮 |
| Mobile | `cancelled` / `solveMode` / 动态提示；只读，不新增局域网控制面 |
| 快捷键 | Windows/macOS 默认值；缺失时安全迁移；冲突、显式空值和自定义绑定不被覆盖 |
| 费用 | 不自动重试；README 明确每次重试可能产生新的模型费用 |

## 验证摘要

- PASS：相关包与 `go test ./... -count=1`。
- PASS：`go vet ./...`、`go build ./...`、`go mod verify`。
- PASS：19 个前端回归、Mobile DOM 回归、Vite production build。
- PASS：固定 Wails v2.11.0 bindings 和保护候选；依赖/runtime/index 保护哈希不变。
- PASS：隔离 Profile 启动、WebView2、Ghost Mode 切换、实际新快捷键提示、无任务中断 Toast、正常退出、无 Node/Vite 和最终无残留。
- PASS（用户确认）：当前测试版本产品验收无问题；逐项真机/Provider 日志未由用户提供。
- BLOCKED：Go race 在所有目标包启动阶段返回 Windows `0xc0000139`，已披露且未伪造为 PASS。

## 候选与稳定版

- 晋升前候选：`build/bin/testing/current/Q-Solver-candidate-change038.exe`
- 本地根稳定版：`build/bin/Q-Solver.exe`
- 根稳定版大小：42,550,784 字节
- 根稳定版 SHA-256：`8A868263F29E170F1204E8BABC8064055D5533B870ED1A023309C99F1758D220`
- 被替代稳定版：`build/bin/archive/superseded/change-038/Q-Solver-stable-before-change038.exe`，SHA-256 `8B19B242F7917D4BC0267E45BB6248077E2BA5BD2D3CE428AC9E1B836C8B4994`。
- 通过候选归档：`build/bin/archive/validation/change-038/Q-Solver-candidate-change038.exe`，哈希与根稳定版一致。
- `testing/current/` 已清空；三个 Release 资产与清单未修改。
- 首版候选 `ABB231…67AB9` 因 Mobile 页面内历史 cancelled 语义回修失效，已归档到 `archive/transient/` 并通过哈希清单验证。

## 后续维护约束

1. 新增超时、失败恢复或请求级模式必须复用 `solveAttemptTracker` 与统一启动管线，不能另建旁路。
2. Retry API 继续禁止接收图片、路径、Config、Prompt 或凭据。
3. `solution-cancelled` 继续与 `solution-error` 分离；取消正文不得成为 completed 或答案摘要。
4. Mobile 继续只读。任何远程控制需求都必须单独建 Change 并重新审计 Token、CSRF、局域网暴露与授权边界。
5. 新快捷键迁移必须尊重显式空绑定和既有 ComboID owner。
6. race 环境阻塞继续保留在验证记录中；后续修复环境后可补跑，但不回写本轮为历史 PASS。

## 提交边界

### 功能提交

- `5564a45 feat: add retry cancel and fast solve controls`
- 范围：本 Change 的 Go 业务代码及测试、Vue/Pinia/API 与回归、Mobile Web、三个生成 bindings、README/README_EN。
- 未包含 `build/bin/**`、`frontend/dist/**`、隔离 Profile、截图证据、正式配置或 secret。

### 治理提交

- message：`chore: add change 038 governance records`
- 范围：`_dev/audits/038-solve-retry-cancel-fast-mode.md`、`_dev/changes/038-solve-retry-cancel-fast-mode/**`、`_dev/changes/INDEX.md`。

用户已授权本次提交与推送到 `origin/dev/qsolver-custom`，双提交现已完成。当前没有独立功能分支可合并，未制造 merge commit；用户随后单独授权并完成本地稳定版晋升。Tag、Release 仍未授权且未执行。

## 参考资源

- `design.md`
- `implementation-plan.md`
- `01-current-state.md`
- `_dev/audits/038-solve-retry-cancel-fast-mode.md`
- Change 035、036、037 治理档案
