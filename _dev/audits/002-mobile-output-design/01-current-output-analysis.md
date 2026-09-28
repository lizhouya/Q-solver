# 01 — 当前输出链路分析（Current Output Analysis）

> 只读设计审计。本节追踪 **AI 结果产生 → 桌面展示** 的完整调用链，回答 6 个关键问题，作为 Mobile Output 的插入点基础。
> 确定性标注：`Confirmed`（源码直接见证）/ `High Confidence`（沿调用链推断）。

---

## 一、核心解题调用链（Confirmed）

```
用户触发
  ├─ 全局快捷键 / 按钮 → app.TriggerScreenshot()（截图入 buffer ≤3）
  └─ TriggerSend()（solve.go:77-125）
        ├─ 校验 APIKey / Model（无则 emit require-api-key / open-settings）
        ├─ buffer 空则自动补一张截图（solve.go:91-105，F-009/D-9）
        ├─ a.EmitEvent("start-solving") + ("user-message", screenshots[0])
        ├─ taskManager.StartTask("solve") → goroutine solveInternal(ctx, screenshots)
        │
        └─ solveInternal(135-153)
              └─ solution.Request{Config, Screenshots} → solver.Solve（pkg/solution/solution.go:45）

Solve 内部（solution.go:45-139）
  ├─ 0. APIKey 校验（空 → emit require-api-key）
  ├─ 1. 组装 systemPrompt（Common Sense System + ScenePrompt + CandidateProfile 简历）
  ├─ 2. userParts = llm.ImagePart(每张截图)   ← 图片以 base64 进入 LLM（纯视觉，无文本输入）
  ├─ 3. messagesToSend = [system, user]       ← F-001：从不读取 s.chatHistory
  ├─ 4. provider.GenerateContentStream(ctx, messages, onChunk)
  │      ├─ chunk.Thinking → emit "solution-stream-thinking"
  │      └─ chunk.Content  → emit "solution-stream-chunk"
  ├─ 5. 错误处理 → emit "solution-error" / context canceled
  └─ 6. 成功 → emit "solution" (response.Content)；s.chatHistory 清空
```

## 二、前端接收与展示（Confirmed）

```
App.vue onMounted → events.js onMany(initEvents)（App.vue:87-258）
  ├─ "start-solving"   → solution/general loading 状态
  ├─ "user-message"    → solution.setUserScreenshot(screenshot)
  ├─ "solution-stream-start" → handleStreamStart(keepContext)   （App.vue:195-198）
  ├─ "solution-stream-thinking" → handleThinkingChunk(token)      （App.vue:201）
  ├─ "solution-stream-chunk"   → handleStreamChunk(token)          （App.vue:200）
  ├─ "solution"         → handleSolution(data)                     （App.vue:171-175）
  └─ "solution-error"   → handleError(handleUserCancellation)
       ↓
solution.js 状态机：
  ├─ history []：历史列表（新解题 unshift 到头）
  ├─ currentRounds：当前问题（rounds[]）回合
  ├─ streamBuffer / streamingHtml：节流渲染中间态
  ├─ thinkingBuffer / isThinking：思考链展示
  └─ 最终 stop → history item.rounds[last].aiResponse 收尾
```

---

## 三、6 个问题的逐条回答

### Q1. AI 答案（"solution"）在哪里产生？
- **`pkg/solution/solution.go:92`** `s.llmProvider.GenerateContentStream()` 的流式回调产生 chunk；
- **组装成为最终答案**：`response.Content`（solution.go:121-134），由流式 chunk 在 `handleStreamChunk`（solution.js:151-176）拼接 `streamBuffer` 累积。
- **证据**：solution.go:78-103（组装请求）、solution.go:92-103（流式事件分发）。

### Q2. 答案保存在哪里？
- 后端**不持久化答案**（除日志 `logger.Printf`）。最终 `response.Content` 仅通过 `cb.EmitEvent("solution", ...)` 一次性发出（solution.go:134），无存储。
- 前端**内存态保存**：`solution.history[0].rounds[last].aiResponse`（solution.js:71-73 / 163 / 222-224）。无 localStorage / 无数据库。
- **结论**：答案生命周期 = 内存 + 当轮展示；不做持久化。

### Q3. 答案如何传输到前端？
- **Wails runtime 事件总线**（`app/app.go:95-97` `runtime.EventsEmit(ctx, event, data...)`）→ WebView2 前端 `EventsOn`（services/events.js / `wailsjs/runtime/runtime`）。
- 事件传输路径：Go → Wails IPC channel → 前端监听。

### Q4. 是否有可用的事件机制？
- **是**：`app.EmitEvent(name, data...)` 是万能发射口；事件全集见 App.vue:87-258（start-solving / solution-stream-* / solution / solution-error / toast …）。
- 支持**多参数**（`EmitEvent(name, a, b)`），可携带 uid /截图 / 状态。

### Q5. 当前输出是否与 UI 强绑定？
- **强绑定但可分离**：
  - 后端只依赖 `cb.EmitEvent`（solution.Callbacks.EmitEvent，solution.go:16-18）——这是**抽象边界**，可替换为任意传输（WebSocket / HTTP / 文件）。
  - 前端 `solution.js` 把事件对应到 `history` / `streamingHtml` 等 Vue 响应式状态——UI 层是唯一消费方。
  - 所以：**后端 emit 是解耦点**；前端接收后「喂给 UI」是为唯一消费路径。

### Q6. 最小插入点在哪？
- **方案 a（后端旁路）**：新增一个 `mobileBridge`（可选模块），在 `solution.go` 事件分发旁路 — 把同一份 `cb.EmitEvent` 复制一份到 WebSocket/HTTP。**改动点**：只加一个适配器，`solution.Solve` 核心不改。
  - 优点：不动业务核心，插入点最小；零 UI 改动。
  - 缺点：若不封装，会让 `solution.go` 耦合两套输出逻辑。
- **方案 b（app 层复用）**：`app.EmitEvent` 是一层薄封装（app.go:95-97），理论上可在 `App` 结构上再挂一个「移动端转发器」，`EmitEvent` 内多一路发送。**改动点**：app.go`EmitEvent` + 初始化。
  - 优点：单点截获所有事件（含 toast / settings），天然覆盖全事件。
  - 缺点：事件语义是「桌面 UI 事件」，并非「移动端语义事件」，需过滤/重映射。
- **方案 c（前端侧二次推）**：前端监听已有事件后，再转发到移动端。**改动点**：前端 js + 新增网关。
  - 优点：不动 Go，纯前端。
  - 缺点：WebView 与 Mobile 网络共存复杂，依赖前端存活状态。

> **本审计不选型**：最小插入点会判断为「后端事件总线的旁路/镜像」（方案 a/b），第三种方案受前端二进制生命周期制约，标记 `Needs`。最终选型留待 `06-user-decisions.md` 及用户决策。

---

## 四、关键帧（供 03 数据结构设计的输入）

| 后端事件 | 语义 | 触发点 | 频率 |
|---|---|---|---|
| `start-solving` | 开始解题 | app/solve.go:117 | 1 次/题 |
| `user-message` | 用户提交的截图 | app/solve.go:118 | 1 次/题 |
| `solution-stream-start` | 流开始（清除缓存） | solution.go:88-90 | 1 次/题 |
| `solution-stream-thinking` | 思考链 chunk | solution.go:99 | 每 chunk |
| `solution-stream-chunk` | 内容 chunk | solution.go:101 | 每 chunk |
| `solution` | 最终答案（一次性） | solution.go:134 | 1 次/题 |
| `solution-error` | 错误 | solution.go:109/117/128 | 偶发 |
| `screenshot-taken` | 截图入 buffer | app/solve.go:52 | 每张 |

> 这些事件天然构成「等待 → 解题中 → 流式 → 最终」的状态流，可直接映射到 Mobile 状态同步（见 03）。

---

*（本节完成。所有证据来自当前 dev/qsolver-custom 分支源码；确定性均 Confirmed。）*