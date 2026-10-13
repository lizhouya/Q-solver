# 回滚预案 — 06 · 007-clickthrough-toggle-feedback

> **状态**：未执行回滚  
> **日期**：2026-10-13

## 回滚范围

本变更无配置、数据库、依赖、协议或持久化迁移。回滚只涉及 Go 源码、测试和 Change 007 治理记录。

## 业务代码回滚步骤

1. 恢复 `StateManager.ToggleClickThrough() bool` 原签名和失败时返回当前状态的旧实现。
2. 恢复 `App.ToggleClickThrough()` 无返回值签名。
3. 恢复 `ServiceDelegate.ToggleClickThrough()` 无返回值签名。
4. 恢复 Shortcut `clickthrough` 分支为单次直接调用，不发 Toast。
5. 恢复 `app/phase4_test.go` fake delegate 原签名。
6. 移除 `pkg/state/manager_test.go` 和 `pkg/shortcut/service_test.go` 中 Change 007 专用测试。

## 治理回滚步骤

1. 从 `_dev/changes/INDEX.md` 移除 007 登记。
2. 删除 `_dev/changes/007-clickthrough-toggle-feedback/`。

## 回滚后验证

- `go test ./...`
- `go build ./...`
- `npm --prefix frontend run build`
- `git diff --check`
- 确认无 `go.mod`、`go.sum`、`frontend/wailsjs/runtime/*` 污染

## 数据影响

无。ClickThrough 状态仍为进程内窗口状态，回滚不需要数据恢复。

## 注意

不得为了回滚 Change 007 而回滚 Change 004～006 已提交内容，也不得顺手修改 `toggle_window`、GhostMode、WindowVisibility 或既有 Windows hook 的 vet 警告。
