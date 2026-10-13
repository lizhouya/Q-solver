# 回滚方案 — 06 · 005-keepcontext-backend-wiring

> **状态**：Implementation 完成后的可执行回滚方案（2026-10-13）

## 回退范围

本 Change 的业务与测试回退仅涉及：

- `pkg/solution/solution.go`
- `pkg/solution/solution_test.go`（本 Change 新增）

归档文档位于 `_dev/changes/005-keepcontext-backend-wiring/`，是否随业务回退一并移除由后续回退请求明确决定；默认保留作为审计记录。

## 文件级回退步骤

1. 恢复 `pkg/solution/solution.go` 到 Change 005 前版本。
2. 删除本 Change 新增的 `pkg/solution/solution_test.go`。
3. 重新执行：
   - `go test ./pkg/solution/... -count=1`
   - `go test ./pkg/... -count=1`
   - `go test ./app/... -count=1`
   - `go build ./...`
   - `git diff --check`
4. 用 `git status --short` 确认未波及范围外文件。

> 回滚时不得使用会覆盖其他未提交工作的目录级或破坏性命令；应只处理上列两个明确文件。

## 回滚后的行为

- `Solve()` 恢复为每次只发送当前 System Prompt 与当前 user message。
- 成功路径恢复为不保留可复用问答历史。
- `ClearHistory()`、`ensureSystemPrompt()`、`trimChatHistory()` 的既有定义保持不变，但不再由本 Change 接入主请求链。

## 不涉及

- `frontend/`
- `app/`
- `pkg/mobile/`
- `pkg/config/`
- `pkg/window/`
- `pkg/tray/`
- `pkg/shortcut/`
- LLM Provider 实现
- `go.mod` / `go.sum`
- `frontend/wailsjs/`

## 影响面评估

回滚会重新关闭 KeepContext 的后端实际功能，但不会改变配置 schema、前端开关、Mobile 协议、窗口/托盘/快捷键或 Provider 行为。由于历史仅驻留内存，不存在磁盘数据迁移或清理步骤。
