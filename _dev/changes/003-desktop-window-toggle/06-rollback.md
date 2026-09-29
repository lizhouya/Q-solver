# 回滚预案 — 06 · 003-desktop-window-toggle

> 设计阶段预填；回滚方案面向「若实现引入回归」时如何安全复原。

## 1. 回滚触发条件

- 实现后出现下列任一情况即停线回滚：
  - 手动快捷键导致窗口无法恢复（用户被锁在隐藏态）
  - 移动 SSE 中断（说明误伤了 `pkg/mobile`）
  - 与原 GhostMode 隐身冲突（F9 失效或被清除）
  - 系统托盘无法创建、或托盘交互导致崩溃

## 2. 回滚策略

| 阶段 | 回滚动作 |
|---|---|
| 代码未提交（实现阶段） | `git checkout -- <本次改动文件>` 或从提交前状态重置 |
| 已提交（未 push） | `git revert -m 1 <commit>` 或 `git reset --hard <上一点的提交>` |
| 已发布构建 | 用上一个稳定二进制覆盖；必要时卸载托盘（`Shell_NotifyIcon(NIM_DELETE)`） |

## 3. 精确回滚清单（本次改动文件）

| 文件 | 回滚动作 |
|---|---|
| `pkg/state/*`（WindowVisibilityManager、Visible 字段） | 删除新增代码文件/字段 |
| `pkg/platform/platform_windows.go`（ShowWindow / SysTrayIcon） | 恢复原版或移除新增函数 |
| `app/*`（HideWindow/ShowWindow/ToggleWindowVisible/WindowVisible 绑定） | 移除绑定方法 |
| `pkg/shortcut/*`（ActionToggleWindow） | 移除 action 及注册表项 |
| `frontend/settings`（隐藏键录制） | 移除 UI 改动 |
| 前端生成物 `wailsjs/go/*` | 重新 `wails generate module` |

## 4. 验收/测试命令（回滚后需重跑防回归）

```bash
go build ./...
go vet ./...
go test ./app/... ./pkg/...
cd frontend && npm run build
```

## 5. 记录

| 日期 | 动作 | 结果 |
|---|---|---|
| — | 未发生回滚 | — |