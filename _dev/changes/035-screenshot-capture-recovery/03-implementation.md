# 实现过程记录 03

## 实现日志

| 时间 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-11-03 | 事前归档 | 完成需求、现状和实施方案 | `00-request.md`、`01-current-state.md`、`02-plan.md` | 用户批准短设计后才进入业务实现 |
| 2026-11-03 | TDD 红灯 | 新增调度器和 Dock 回归测试并确认旧实现失败 | `dispatcher_test.go`、`screenshot_dock_regression_test.mjs` | Go 缺少调度器类型；Dock 缺少响应式快捷键和继续截图提示 |
| 2026-11-03 | 串行调度 | 新增单 worker FIFO，将 Manager 触发改为快速入队 | `dispatcher.go`、`service.go` | 解除 Hook 回调对截图/压缩的同步等待，同时保持动作顺序 |
| 2026-11-03 | Dock 恢复 | 增加继续截图/已满提示并使用 `storeToRefs` | `ScreenshotDock.vue` | 显示真实截图、删除、发送快捷键；保留 `1/3～3/3` 计数 |
| 2026-11-03 | 自动验证 | 完成聚焦、全量、前端、Vite 和 Wails 候选构建 | 代码、测试与构建产物 | 必要自动验证通过；race 为既有环境 BLOCKED |
| 2026-11-03 | GUI 验收 | 用户确认 Change 035 候选测试通过 | GUI 候选 | 快速三图、提示和既有流程按本轮验收口径通过 |
| 2026-11-03 | 稳定版晋升 | 归档旧稳定版和候选，原子替换根稳定版 | `build/bin/**` | SHA-256、目录不变量和 Release 保护均通过 |
| 2026-11-03 | 晋升后验证 | 根稳定版运行 8 秒、WebView2、无 Node、正常退出 | `build/bin/Q-Solver.exe` | 退出码 0，无强制终止和残留进程 |
| 2026-11-03 | 边界审计 | 核对文件白名单、保护哈希和双提交规划 | 审计与 Change 文档 | 用户已授权 commit 与 push，仅目标 origin/dev/qsolver-custom |
| 2026-11-03 | 功能提交 | 提交功能代码与对应测试 | Commit `e23f86a` | `fix: restore reliable multi-screenshot shortcuts` |
| 2026-11-03 | 治理提交 | 本文件、审计、INDEX 与其余 Change 档案随治理提交落地 | 治理文件 | `chore: add change 035 governance records`；不要求文档记录自身提交哈希 |

## 与计划的偏差

- 生产实现未修改 `app/solve.go` 或 `pkg/screen`，与计划一致。
- 首次独立启动因用户当时正在运行根稳定版而暂缓；用户完成候选 GUI 验收并退出后，晋升后的根稳定版已补做独立启动与正常退出验证并通过。
- race 检查补充执行，但在测试程序启动阶段返回既有环境错误 `0xc0000139`。该补充项保持 BLOCKED，不冒充 PASS；项目要求的普通测试、静态检查和构建均已通过。

## 关键决策纪要

1. 调度位置选在 `shortcut.Service`，跨平台 Manager 均复用，Windows Hook 立即受益。
2. 使用单 worker FIFO，不为每个动作创建无序 goroutine。
3. handler 在队列锁外执行，截图耗时不会阻塞后续动作入队。
4. 停止时拒绝新动作并丢弃尚未执行队列；正在执行的动作完成后 worker 退出。
5. 保持显式发送，第 3 张截图不会自动解题。
6. 不扩大锁重构；现有 `conversationMu` 继续保证截图缓冲顺序。

## 遗留未完成项

- race 环境 `0xc0000139` 未在本 Change 解决，作为已知环境限制保留。
- Tag、Release 未授权且未执行。
