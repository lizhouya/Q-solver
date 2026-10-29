# Change 027 请求记录

日期：2026-10-25
状态：事前记录

## 1. 用户原始反馈

用户在 Change 026 验证程序中分别测试不同模型后反馈：

> “分别测试不同的模型，APMIX.AI-openai/claude-sonnet-4-6-free 识别出两道完全不匹配的题目，实际为5道题目，deep seek 识别正确，gpt模型没有识别出来，应该是没有额度了，先解决修复Claude的模型问题”

观察事实：

- 实际截图含 5 道题，Claude 返回 2 道**完全不相关**的题（LLaMA 相关）。
- DeepSeek 在同一 Q-Solver/CCR 流程中识别正确。
- GPT 返回鉴权类错误，用户推测为额度问题（未证实）。
- 用户明确要求：“先解决修复Claude的模型问题”。

“完全不相关”而非“格式错误”是关键特征：它更像模型**未看到图片**后基于文本
提示自行生成，而非图片被拒绝。但此为推断，非结论。

## 2. 已核实的代码事实

详见 `01-current-state.md`。摘要：

1. **多图构造正确**。`solution.go:132-139` 将本轮全部截图加入同一 user 消息；
   `openai.go:66-90` 以 base64 data URL 序列化并校验 MIME。不存在丢图路径。
2. **分段切换已清空历史**。`questionbank_next_exam.go:74-78` 调用
   `ClearHistory()` 并递增 `conversationEpoch`；`solve.go:198-201` 按 epoch 拦截迟到提交。
   不存在“新分段携带旧历史”缺陷。
3. **KeepContext 默认关闭**（`config.go:55`）。
4. **KeepContext 开启时历史会累积图片**（`solution.go:216-223`），
   每轮全量重发（`solution.go:160-163` + `openai.go:48-56`）。
   上限 `MaxConversationRounds=10`、`MaxScreenshots=3`。
   **仅在用户显式开启该功能时触发。**
5. **prompt 已禁止编造**（`prompts.go:156`、`:170`），
   但**无**“无法看到图片时显式报错”的指令。

## 3. 问题分层

### 3.1 已排除：Q-Solver 丢图或编码错误

代码层证据充分（§2.1），无此缺陷。

### 3.2 已排除：跨分段历史污染

代码层证据充分（§2.2），无此缺陷。原初稿此断言错误，已作废。

### 3.3 待验证：KeepContext 历史图片累积

机制真实存在（§2.4），但是否触发取决于用户设置，**当前未知**。

若触发，后果可能包括：

- 请求携带大量旧截图，模型被旧题干扰。
- 图片总数超过 20 时，Claude 侧启用更严格的单图尺寸限制，
  超限图片被**拒绝**而非缩放（Anthropic Vision 文档）。
- 旧截图与当前截图混在一起，模型可能对旧题作答或混淆题号。

这与“返回不相关题目”的现象一致，但**尚不能确认**。

### 3.4 待验证：上游未收到图片

若 §3.3 未触发，则图片可能在 CCR 或上游被丢弃/忽略。
Q-Solver 侧不可见，需用户核对 CCR 配置。

### 3.5 可改进：缺少视觉失败显式指令

即使上游丢图，当前 prompt 也未要求模型在“看不到图片”时明确报错。
增加该指令可让失败可见，而非静默生成无关内容。

## 4. 拟议范围

### 4.1 拟修复

1. **多图/历史请求规模保护**（若 §3.3 确认）：
   在保留 KeepContext 语义的前提下，避免把旧截图无限制地重发给上游。
   具体方案待 Task 3 的 RED 测试确定，不在事前预设实现。
2. **视觉失败显式指令**：在 `<SystemPolicy>` 增加一条规则，
   要求模型在无法读取任何图片时明确说明，而不是凭文本提示生成题目。

### 4.2 不修复（超出 Q-Solver 范围）

- CCR alias 与上游路由配置。
- APMIX 套餐与视觉透传能力。
- 用户 CCR 侧的缓存或 fallback 行为。

### 4.3 明确不做

- 不根据模型名推断视觉能力。
- 不修改 CCR 入站协议或新增 adapter。
- 不破坏“继续历史分段”功能。
- 不为“返回答案是否匹配”实现启发式后处理。

## 5. 验收标准

### 5.1 自动验证

- loopback 多图测试：N 张合成截图 → payload 恰好 N 个 image part。
- 历史规模测试：KeepContext 开启、多轮解题 → 断言请求中图片数量符合**新确定的**
  上限策略，且当前轮截图始终完整存在。
- 分段隔离回归：`StartNextExam` 后首轮请求不含上一分段内容。
- prompt 测试：`<SystemPolicy>` 含视觉失败指令。
- 全量 `go test ./...`、前端回归、Vite build、`go vet`、`go mod verify` 通过。
- race：若仍 `0xc0000139`，记 `BLOCKED`。

### 5.2 人工验收（NEEDS_USER）

1. Direct 三协议多图解题正常。
2. CCR + DeepSeek 多图解题正常。
3. KeepContext 开启下连续多轮解题，答案不串题。
4. `StartNextExam` 后新分段答案不含旧分段内容。
5. 上游丢图时模型明确报错而非编造。

### 5.3 用户核对清单（CCR 侧）

- `APMIX.AI-openai/claude-sonnet-4-6-free` 的实际上游模型。
- 该路由是否透传 OpenAI-compatible 图片输入。
- 是否存在缓存或 fallback 到 text-only 模型。
- 官方 `claude-sonnet-4-6` 是否需要额外订阅才能使用视觉。

## 6. 安全与授权边界

- 不调用真实 CCR、真实模型或真实 API；不消费额度。
- 不读取真实配置、密钥、Token。
- 自动测试只用 loopback `httptest`、fake provider、合成图片。
- 不删除、覆盖或迁移题库 `_记录`、截图、旧导出。
- 不覆盖 Change 026 验证程序或正式 `Q-Solver.exe`。
- 不清理 Change 024/025/026 未提交工作树。
- 未授权 commit、push、Tag、Release 或发布。
- 未确认根因前不预写 PASS。
