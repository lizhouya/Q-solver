# Change 027 当前状态

日期：2026-10-25
状态：事前记录（已按实际源码校正）

> 本文档初稿曾基于会话摘要推测 `app/questionbank.go` 中的 `buildSystemPrompt`、
> `SealAndCreateNewSegment` 等符号，并断言“新分段不清空历史”。读取真实源码后确认
> 该断言**错误**，初稿已作废。以下内容全部基于本次实际读取的文件与行号。

## 1. 已核实的调用链

```
前端触发解题
→ app/solve.go: triggerSend
→ app/solve.go: solveInternal
→ app/solve.go: a.solver.Solve(ctx, req, callbacks)
→ pkg/solution/solution.go: Solver.Solve
→ pkg/solution/solution.go: buildSystemPrompt
→ pkg/llm/{openai,anthropic,openai_responses}.go: GenerateContentStream
```

关键文件与职责：

| 文件 | 职责 |
|---|---|
| `app/solve.go` | 截图缓冲、任务启动、epoch 校验、解题后上下文提交/回滚 |
| `app/questionbank_next_exam.go` | `StartNextExam`：封存分段 + 清空上下文 |
| `app/questionbank_continuation.go` | 继续历史分段：重建并替换模型上下文 |
| `pkg/solution/solution.go` | 消息组装、system prompt 构建、chatHistory 维护 |
| `pkg/domain/prompts.go` | `<SystemPolicy>` 与场景 PersonaCard 文本 |
| `pkg/llm/openai.go` | OpenAI Chat Completions 请求构造（含图片） |
| `pkg/llm/types.go` | `Message` / `ContentPart` 定义与 data URL 解析 |

## 2. 已核实的多图构造（正确）

`solution.go:132-139`：

```go
userParts := make([]llm.ContentPart, 0, partCapacity)
if numberingContext != "" {
    userParts = append(userParts, llm.TextPart(numberingContext))
}
for _, screenshot := range req.Screenshots {
    userParts = append(userParts, llm.ImagePart(screenshot))
}
currentUserMsg := llm.NewMultiPartMessage(llm.RoleUser, userParts)
```

- 本轮所有截图逐张加入**同一个** user 消息的 `Parts`。
- 无按模型名/品牌切换图片格式的分支。

`openai.go:66-90` 将图片序列化为 `ImageContentPart`，URL 为原始 `data:...;base64,` 串；
`openai.go:73-76` 使用 `ParseBase64DataURLStrict` 并校验 MIME 属于
`image/png|jpeg|gif|webp`（`types.go:161-168`）。

结论：**Q-Solver 的多图编码与 OpenAI Chat Completions 线格式正确**，
不存在“第二张图丢失”或“图片未编码”的代码路径。

## 3. 已核实的上下文机制

### 3.1 KeepContext 默认关闭

`pkg/config/config.go:23`、`config.go:55`：

```go
KeepContext bool `json:"keepContext,omitempty"`
...
KeepContext: false,
```

默认 `false`。多轮上下文是用户显式开启的功能。

### 3.2 开启时历史会累积图片（真实机制）

`solution.go:216-223`：

```go
if req.Config.KeepContext && strings.TrimSpace(response.Content) != "" {
    s.mu.Lock()
    if s.generation == solveGeneration {
        s.chatHistory = append(s.chatHistory, cloneMessage(currentUserMsg), llm.NewAssistantMessage(response.Content))
        s.trimChatHistoryLocked()
    }
    s.mu.Unlock()
}
```

`currentUserMsg` 含全部 base64 图片 part，被**整体**追加进 `chatHistory`。

`solution.go:160-163` 每轮重新发送：

```go
messagesToSend = append(messagesToSend, llm.NewSystemMessage(systemPrompt))
messagesToSend = append(messagesToSend, history...)
messagesToSend = append(messagesToSend, cloneMessage(currentUserMsg))
```

`openai.go:48-56` 对含 `Parts` 的历史消息同样调用 `toOpenAIParts`，
因此**历史中的旧截图会被逐张重新发送**。

上限：`solution.go:18` `MaxConversationRounds = 10`；`app/solve.go:13` `MaxScreenshots = 3`。
即最坏情况单次请求约 10 轮 × 3 张旧图 + 本轮图。

**该机制仅在 `KeepContext=true` 时生效。** 用户是否开启尚未确认，
因此**不能**断言这是本次现象的直接原因。

### 3.3 分段切换与 epoch 已经清空历史（原初稿断言错误）

`questionbank_next_exam.go:74-78`：

```go
a.solver.ClearHistory()
a.conversationRefs = nil
a.conversationEpoch++
clearedScreenshots := len(a.screenshotBuffer)
a.screenshotBuffer = nil
```

`app/solve.go:198-201`：

```go
currentEpoch := conversationEpoch == a.conversationEpoch
if !currentEpoch {
    a.solver.ClearHistory()
}
```

结论：`StartNextExam` **已经**清空模型历史并递增 epoch；跨轮迟到的提交也会被 epoch 拦截。
不存在“新分段携带旧分段历史”的缺陷。

### 3.4 继续历史分段是有意的功能

`questionbank_continuation.go:62-77` 会按题库快照重建并 `ReplaceConversation`，
并在 `KeepContext=false` 时降级为空快照（`:66-68`）。
这是用户在 Change 020 明确批准的功能，**不得**为修本次问题而破坏。

## 4. 已核实的 prompt 现状

真实 system prompt 由 `solution.go:227-250` 组装：

- `domain.GetSystemBehaviorPrompt()`（`prompts.go:145-192`）
- 可选 `<ScenePrompt>`（`prompts.go:197+` 的 PersonaCard）
- 可选 `<CandidateProfile>`（简历内容）

其中**已经存在**防编造约束：

- `prompts.go:156`：`如果信息不足，明确指出缺失条件，不要编造内容。`
- `prompts.go:170`：`只识别并汇总当前用户消息中截图所含的全部待解原题；历史对话只可作为当前题目的上下文，不得重新作答或汇总历史题目。`
- `prompts.go:171`：多图同题合并、重叠只作答一次。
- `prompts.go:182`：`固定表格必须结束整个回答`。

**不存在**的约束：

- 没有“若无法看到图片，请明确报错”的显式视觉失败指令。
- 没有“先确认能否看到图片”的前置检查。

结论：prompt 已禁止编造，但**未针对“图片未到达”这一具体失败模式**给出显式指令。

## 5. 已核实的图片传输边界

- 请求体格式正确（见 §2）。
- 图片以 base64 data URL 内联，不依赖外链可达性。
- 未发现 Q-Solver 侧丢弃、截断或改写图片的代码。
- CCR 模式固定 `OpenAIAdapter`（`factory.go:17-26`），不按模型名换协议。

因此：**若上游未收到图片，原因在 Q-Solver 之外**（CCR 路由、上游 provider 的
OpenAI-compatible 视觉透传、或该 alias 的实际指向）。

## 6. 尚不能确认的事项

| 事项 | 状态 | 说明 |
|---|---|---|
| 用户是否开启 KeepContext | 未知 | 决定 §3.2 机制是否触发 |
| 用户当时的分段轮次 | 未知 | 决定历史图片累积量 |
| `claude-sonnet-4-6-free` 实际上游 | 未知 | CCR 侧配置，Q-Solver 不可见 |
| 该 route 是否透传视觉 | 未知 | APMIX 文档未声明 |
| 是否为 CCR 缓存或 fallback | 未知 | 需用户核对 CCR 配置 |

**不得**把以上任一项写成已确认根因。

## 6.1 后续实测补充（2026-10-25）

Task 2 的 loopback 测量（`pkg/solution/conversation_images_test.go`）
把 §3.2 从「机制推断」升级为「已确认事实」，并修正了上限的算法：

| 场景 | 请求图片总数 |
|---|---|
| KeepContext=false，单轮 2 图 | 2 |
| KeepContext=true，第 1 / 2 / 3 轮 | 2 / 4 / 6 |
| KeepContext=true，第 13 轮 | **22** |

关键修正：`trimChatHistoryLocked`（`solution.go:264-278`）在**发送前**把
history 裁到 `MaxConversationRounds*2` 条消息，但当前轮在
`solution.go:163` 之后才追加，因此请求实际携带 **11 轮**图片而非 10 轮；
`MaxScreenshots=3` 时理论上限为 `10*3+3 = 33` 张，
**超过** Anthropic 文档所述 20 张阈值。

该上限已在 Change 027 的 Task 3 收敛为 10 轮（见 `03-implementation.md`）。

仍需强调：同一请求会同时发往 Claude 与 DeepSeek，而仅 Claude 异常，
因此 §3.4 的「上游未收到图片」与 §6 表中的 CCR 侧因子**依然待验证**，
本 Change 不宣称已定位唯一根因。

## 7. 公开资料（外部能力边界）

- APMIX 官方模型 ID 为 `claude-sonnet-4-6` / `anthropic/claude-sonnet-4-6`，
  需 Starter/Pro/Max 套餐，无免费层；模型页与文档**未声明**视觉输入支持。
- 用户截图中的 `APMIX.AI-openai/claude-sonnet-4-6-free` 不符合官方 ID 形态，
  很可能是 CCR 自定义聚合路由名。
- Anthropic 官方 Vision 文档：Claude 原生支持多图（200k 上下文模型上限 100 张，
  其他 600 张），支持 base64/URL/Files API；单图最大 10MB；
  超过 20 张图时启用更严格的单图尺寸限制，超限图片会被**拒绝**而非缩放。

最后一条与 §3.2 相关：若 KeepContext 累积导致请求图片数超过 20，
上游可能直接拒绝或降级处理，表现为答案与题目不符。
这仍是**待验证假设**，需先确认用户设置。
