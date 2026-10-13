# 实现计划 — 02 · 005-keepcontext-backend-wiring

> **阶段**：设计期（最小修改方案，未实现）

## 设计目标

把既有 `chatHistory` 能力正确接线到 `Solve()`，让 KeepContext 真正生效。**默认保留实际发送给模型的 user message（含截图 base64）作为历史**，不丢图片、不做压缩。

## 修改范围（优先限制）

- 业务：仅 `pkg/solution/solution.go`
- 测试：`pkg/solution/*_test.go`（新增）

**不改**：frontend / pkg/mobile / app / config schema / LLM provider / pkg/window / tray / shortcut。

## 核心改动（`Solve()` 内请求构造 + 收尾）

### 请求消息构造顺序（§7 需求已确认）

```
KeepContext OFF（或 history 空）:
  ensureSystemPrompt(systemPrompt)          // 建立/对齐 history[0]
  messagesToSend = [history[0], currentUserMsg]

KeepContext ON（history 有历史）:
  ensureSystemPrompt(systemPrompt)          // 把旧 system 替换为当前 prompt，不重复累积
  trimChatHistory()                          // 进入前裁剪到 MaxConversationRounds
  messagesToSend = [history...] + [currentUserMsg]
  // 满足：system 唯一、当前 user 只出现一次、无 partial chunk 重复
```

- 从 `messagesToSend[:len(messagesToSend)-1]` 即 history 前缀（含对齐后 system + 历史 user/assistant）。
- `currentUserMsg` 只追加一次，追加在历史之后。

### 成功后才写历史（§5 需求）

```
成功后（response.Content 非空）:
  append(chatHistory, currentUserMsg)        // user
  append(chatHistory, llm.NewAssistantMessage(response.Content))  // assistant
  trimChatHistory()                          // 控制轮数
```

- **不在 solution-stream-start / thinking / chunk 阶段写历史**。
- assistant message 只保存完整 final content，不含 thinking（可按 llm.NewAssistantMessage 用 Content，不填 Thinking，避免体积膨胀）。
- 失败（err / 空 response / context canceled）→ **不写任何 user/assistant**（D-005-005）。

### 移除强制清空

- 删除 `Solve()` 成功路径的 `s.chatHistory = []llm.Message{}`（:137）。
- `ClearHistory()` 保留为**唯一显式清空入口**（由 app.onConfigChanged 在 !KeepContext 时调用）。

## Phase 划分

- **Phase 1 — 请求 history wiring**：`ensureSystemPrompt` + 读取 history 作为 prefils + 追加 current user。
- **Phase 2 — 成功后 history append**：追加 (user, assistant) + `trimChatHistory`；移除 :137 强制清空。
- **Phase 3 — trim / ClearHistory regression tests**：T-KC-01~10。
- **Phase 4 — full regression validation**（见 04-validation）。

## 并发（§5 结论）

当前产品路径由 task.Coordinator 串行化（单当前 Solve）。`Solver` 无锁。**本 Change 不引入锁**；在 01 记录「未来并发隐患」，不作为本 Change 范围。如需最小正确性保证（如测试并发触发），实现阶段再评估用 `sync.Mutex` 包住 history 读写——写入 Plan 风险项。

## 不做

- ❌ 图片摘要 / OCR 替换 / token 计算 / 图片去重（这些属未来优化 Non-goal）
- ❌ 新 conversation abstraction（LD-5-1）
- ❌ history 持久化

## 否决/回退

见 `06-rollback.md`。任何前端、mobile、pkg/config 或 schema 变更即属越界，立即停止。