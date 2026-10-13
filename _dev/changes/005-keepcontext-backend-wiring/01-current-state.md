# 现状记录 — 01 · 005-keepcontext-backend-wiring

> **阶段**：设计期现状核实（只读源码，未改代码）

## 当前 Solve 调用链（KeepContext 相关）

```
app.TriggerSend()
  → taskManager.StartTask("solve")   // HasRunningTask 检查 + 新任务取消旧任务
    → go solveInternal(ctx, screenshots)
      → solution.Solver.Solve(ctx, req, cb)   (solution.go:45)
          ├─ 组装 systemPrompt
          │    = GetSystemBehaviorPrompt() + <ScenePrompt> + <CandidateProfile>
          ├─ currentUserMsg = llm.NewMultiPartMessage(RoleUser, [ImagePart(截图)...])
          ├─ messagesToSend = [system, currentUserMsg]        ★ 从不读 chatHistory
          ├─ provider.GenerateContentStream(messagesToSend, onChunk)
          │     └─ onChunk → cb.EmitEvent(solution-stream-thinking / chunk)
          ├─ 成功 → cb.EmitEvent("solution", response.Content)
          │     └─ s.chatHistory = []llm.Message{}             ★ 成功返还时强制清空
          └─ 失败 → cb.EmitEvent("solution-error", ...)（不清空）
```

## chatHistory 字段

`solution.go:27`：
```go
type Solver struct {
    llmProvider llm.Provider
    chatHistory []llm.Message
}
```

`NewSolver`（30-35）初始化 `chatHistory: make([]llm.Message, 0)`。

## 已存在但是「死代码」的能力

| 函数 | 位置 | 是否进入 Solve 主链 | 现状 |
|---|---|---|---|
| `chatHistory` 字段 | :27 | ❌ 从未被读 | Solve 直接 `messagesToSend=[system,currentUser]` |
| `ClearHistory()` | :41-43 | ⚠️ 被 app.onConfigChanged 在 `!KeepContext` 时调用 | **唯一实际生效的很清入口** |
| `ensureSystemPrompt(prompt)` | :141-159 | ❌ 从未被调用 | 维护 history[0] 为当前 system |
| `trimChatHistory()` | :162-181 | ❌ 从未被调用 | 按 MaxConversationRounds 裁剪 |
| `MaxConversationRounds` | :14 | ❌ 死常量 | 值 10 |

## 关键细节

### 1. KeepContext 在 app 层如何被消费（app/app.go:117-134）

```go
func (a *App) onConfigChanged(new, old) {
    if !newConfig.KeepContext && a.solver != nil {
        a.solver.ClearHistory()
    }
}
```

- **KeepContext=false 时**：每次 config 变更都会 `ClearHistory()` → chatHistory 空 → 每题独立。
- **KeepContext=true 时**：不 ClearHistory，理论上应保留历史——但 `Solve()` 读完历史且成功返还时 `s.chatHistory=[]` 强制清空，所以本轮仍无法携带。

### 2. `ClearHistory` 的两处触发
- 仅 `app.onConfigChanged` 在 `!KeepContext` 时调用（app.go:123)
- `Solve()` 内部 137 行也清空（这是当前 bug 根源之一）

### 3. 消息类型语义（pkg/llm/types.go）

用户消息是 **multipart**：
```go
// ContentPart{Type, Text/Base64}
// NewMultiPartMessage(RoleUser, parts) → Message{Role:user, Parts:[{Type:image, Base64:"data:image/...;base64,..."}]}
```

`Message` 支持：`Content`（纯文本）+ `Parts`（多模态）+ `Thinking`（仅 Assistant）。**把 current user message（含截图 base64）直接存入 chatHistory，会完整保留图片内容**（不丢）。

## 并发与锁（solution.go:25-28）

`Solver` 只有 `llmProvider` + `chatHistory` 两字段，**无 mutex、无 solve lock、无 service-level serialization**。

当前唯一串行化来自 **`task.Coordinator`**：`app.TriggerSend` → `StartTask` 先 `HasRunningTask()` 检查，且新任务会 cancel 旧任务 → goroutine 内单当前任务。所以**在产品运行路径上并发 Solve 会被 task manager 拦截**（同一时刻至多一个未经的 Solve goroutine）。

**风险点**：`Solve()` 本身不持锁；若未来某处绕过 task manager 直接调 `Solve()`，`chatHistory` 存在 data-race。但当前产品路径不会。需在设计中记录为「未来并发隐患」，不擅自引入大型并发重构。

## 5. trimChatHistory 实际语义（solution.go:162-181）**

```go
if len(s.chatHistory) <= 1 { return }            // 仅 system，无需裁剪
nonSystemMsgs := len(s.chatHistory) - 1          // 去掉 system
maxNonSystemMsgs := MaxConversationRounds * 2    // 每轮 = 1 user + 1 assistant
if nonSystemMsgs <= maxNonSystemMsgs { return }
startIndex := len(s.chatHistory) - maxNonSystemMsgs
newHistory = [chatHistory[0]] + chatHistory[startIndex:]   // 保留 system + 最近 N 轮
```

确认：**1 round = user + assistant = 2 条非 system 消息**；system 不计作普通 round，恒为 history[0]。

### 例：MaxConversationRounds=2，已有 System,U1,A1,U2,A2，U3 到来
- `nonSystemMsgs = 4 > maxNonSystemMsgs(=4)`? 等于 4，`<=` 不裁剪 → 保持全部。
- 发给模型的最终 history = `[System, U1, A1, U2, A2, U3]` → **保留最近 2 轮 + 当前 user**。
- （若已有 U1,A1,U2,A2,U3,A3 共 6 条非 system，`6 > 4` → 裁剪为 `[System, U2, A2, U3, A3]`，保留最近 2 轮。）

## 根因结论

`chatHistory` 的**存储、system 维护、裁剪全套能力已实现**，唯一缺的是：
1. `Solve()` 应读取 `chatHistory` 作为消息前缀（而不是只发 system+current）；
2. 成功后应把本轮 `(user, assistant)` 追加到 `chatHistory`（而 .137 行清空）；
3. 应在**发请求前**调 `ensureSystemPrompt` 保证 history[0] 为当前 system prompt，并在**追加后**调 `trimChatHistory()` 裁剪。

## 相关源码

- `pkg/solution/solution.go:25-181`（Solver / Solve / ClearHistory / ensureSystemPrompt / trimChatHistory）
- `pkg/llm/types.go:35-91`（Message / ContentPart / NewMultiPartMessage / ImagePart）
- `app/app.go:117-125`（onConfigChanged → ClearHistory 触发点）
- `app/solve.go:107-125`（TaskCoordinator 串行化）