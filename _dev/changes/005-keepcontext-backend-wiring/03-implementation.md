# 实现记录 — 03 · 005-keepcontext-backend-wiring

> **状态**：Implementation 已完成（2026-10-13）

## Phase 1 — 请求 history wiring ✅

**状态**：Done

- `KeepContext=true`：
  - 调用 `ensureSystemPrompt(systemPrompt)`，保证历史首条为当前有效 System Prompt。
  - 调用 `trimChatHistory()`，复用既有轮次上限。
  - 复制 `chatHistory` 到独立 `messagesToSend`，再追加一次 `currentUserMsg`。
  - 当前 user 在请求成功前不会提前进入 `chatHistory`。
- `KeepContext=false`：
  - 不读取旧 history，直接构造 `[current system, current user]`。
  - 在 `Solve()` 内清空可能残留的 history。
  - 成功后不保存本轮上下文。

满足：system 唯一、当前 user 仅出现一次、streaming chunk/thinking 不进入 history。

## Phase 2 — 成功后 history append ✅

**状态**：Done

- 仅在 Provider 成功返回非空最终响应后追加：
  - `currentUserMsg`
  - `llm.NewAssistantMessage(response.Content)`
- 追加后调用 `trimChatHistory()`。
- 移除原成功路径无条件清空 history 的行为。
- Provider error、context canceled、空响应路径均在历史写入前返回，不保存失败轮次。
- assistant history 仅保存最终 `Content`，不保存 streaming chunk 或 `Thinking`。

## Phase 3 — trim / ClearHistory regression tests ✅

**状态**：Done

新增 `pkg/solution/solution_test.go`，覆盖 T-KC-01～T-KC-10：

- OFF 连续请求独立，且 history 为空。
- ON 第二轮携带 `[system, U1, A1, U2]`。
- 两轮成功后 history 保存完整两轮。
- 超过 `MaxConversationRounds` 后精确保留最近 N 轮并丢弃最旧轮。
- `ClearHistory()` 后下一次请求不携带旧轮次。
- Provider error 不写入失败轮。
- System Prompt 从 A 更新为 B 时，Provider 只收到唯一的当前 B。
- true → false 与 false → true 切换语义。
- multipart user message 的图片 `Parts` 和 base64 原值保留。

测试使用同包 fake Provider 捕获实际发送消息，不为测试新增生产 getter 或接口。

## Phase 4 — full regression validation ✅

**状态**：Done

- [x] `go test ./pkg/solution/... -count=1`
- [x] `go test ./pkg/llm/... -count=1`
- [x] `go test ./app/... -count=1`
- [x] `go test ./pkg/... -count=1`
- [x] `go build ./...`
- [x] `go vet ./pkg/solution/...`
- [x] `npm --prefix frontend run build`
- [x] `git diff --check`
- [x] 最终边界仅包含 `pkg/solution/*` 与本 Change 归档目录

## 并发处理

未引入 mutex 或新的 conversation abstraction。当前仍依赖既有 `task.Coordinator` 对主产品路径的串行化；`Solver` 被外部并发直接调用的潜在风险继续作为已知后续事项，不在本 Change 扩展处理。

## 变更日志

| 日期 | Phase | 内容 |
|---|---|---|
| 2026-10-13 | 1 | 接通 ON/OFF 请求构造与当前 System Prompt 对齐 |
| 2026-10-13 | 2 | 成功后追加完整 user/assistant round，并复用既有裁剪 |
| 2026-10-13 | 3 | 新增 T-KC-01～10 回归测试并强化最近轮次及 Prompt 替换断言 |
| 2026-10-13 | 4 | 完成 Go、Frontend、vet、diff 与边界验证 |
