# Change 022 实施方案：答案复制与分段题号同步

日期：2026-10-24
事前状态（2026-10-24）：用户已批准，事前方案已固化；最新实施与验证状态见 03/04。

## 1. 方案概述

新增独立纯 Go 包 `pkg/answersummary` 作为答案汇总解析、规范化、复制文本和编号提示的唯一事实源。Prompt 要求同一次模型请求在完整回答末尾输出唯一固定三列表格；Solver 将当前题库 Session 的受限编号摘要作为 user multipart 的第一个 text part，图片紧随其后，不改变一次请求语义。App 在 capture 最终终态只解析一次，将完整 Answer、结构化 AnswerItems 与 ConversationSnapshot 在题库候选 Session 中原子保存，保存成功后再把同一结构化结果同步给 Desktop 与 Mobile。

Desktop 和 Mobile 继续只显示完整 Markdown 中的一份汇总表，新增“复制答案”但保留“复制全文”。复制答案只使用 Go 端生成的 `answerCopyText`，任何待核对或无效结构均不回退复制全文。编号参考只来自同一 Session 的已校验 AnswerItems 或旧汇总中保守提取的 Label，不依赖 Solver history 或 `conversationRefs`。同时修复 `KeepContext=false` 切换时 Solver history 与持久化会话引用的一致性，并补齐 Mobile 的答题端快捷键提示。

## 2. 数据与接口契约

### 2.1 `pkg/answersummary`

新增不依赖 App、Mobile 或 questionbank 的纯 Go 包：

```go
const (
    KindSingleChoice   = "single_choice"
    KindMultipleChoice = "multiple_choice"
    KindFill           = "fill"
    KindJudgment       = "judgment"
    KindShortAnswer    = "short_answer"
    KindCalculation    = "calculation"
    KindCode           = "code"
    KindOther          = "other"

    StatusReady  = "ready"
    StatusReview = "review"
)

type Item struct {
    Label  string `json:"label"`
    Kind   string `json:"kind"`
    Value  string `json:"value"`
    Status string `json:"status"`
}

type Result struct {
    Items    []Item `json:"items,omitempty"`
    CopyText string `json:"copyText,omitempty"`
}

type NumberingContext struct {
    PreviousLabels     []string `json:"previousLabels,omitempty"`
    SuggestedNextLabel string   `json:"suggestedNextLabel,omitempty"`
}

func Parse(markdown string) (Result, error)
func BuildCopyText(items []Item) (string, error)
func ExtractLegacyLabels(markdown string) []string
func FormatNumberingContext(ctx NumberingContext) string
```

约束：

- 每轮最多 50 项。
- Label 最多 80 个 Unicode 字符。
- 单项 Value 最多 2,000 字符。
- 全部结构化内容最多 32 KiB。
- 编号提示最多保留最近 64 个去重 Label，序列化后最多 4 KiB。
- `Parse` 只解析回答末尾最后一个精确标题及紧随其后的固定三列表格，不扫描正文选项字母。
- `BuildCopyText` 仅在所有 Item 均为 `ready` 时按原顺序以 `\n` 连接 Value；任一 `review` 或非法项返回错误和空字符串。
- `ExtractLegacyLabels` 只从明确“最终答案汇总”段保守提取受限 Label，不为旧自由 Markdown 生成 CopyText。

### 2.2 题库持久化

`pkg/questionbank.Record` 新增：

```go
AnswerItems []answersummary.Item `json:"answerItems,omitempty"`
```

保持 Session schema version 2；可选字段对旧 JSON 向后兼容，不改写旧记录。完整 `Answer` 永远保留，`AnswerItems` 不得替代或裁剪它。

Store 新增：

```go
func (s *Store) FinishWithConversationResult(
    recordID string,
    answerText string,
    items []answersummary.Item,
    status string,
    snapshot *ConversationSnapshot,
) error

func (s *Store) CurrentNumberingContext(excludeRecordID string) answersummary.NumberingContext

func (s *Store) ClearCurrentConversation() error
```

保留旧 `FinishWithConversation` 作为兼容包装器。

### 2.3 传输与界面 DTO

- `solution.Request` 新增 `NumberingContext answersummary.NumberingContext`。
- Desktop 终态事件、历史 DTO 和 Mobile `Payload` 增加同名 `answerItems` 与 `answerCopyText` 字段。
- `answerCopyText` 必须由 Go `BuildCopyText` 生成；Desktop/Mobile 不自行解析或规范化答案。
- Mobile Bridge 对旧的一参数 `Publish("solution", text)` 保持兼容，Snapshot 深拷贝切片。

## 3. 固定 Prompt 与 multipart 规则

固定汇总格式：

```markdown
## 最终答案汇总
| 题号 | 题型 | 可填写答案 |
|---|---|---|
| 12 | 单选 | B |
| 13 | 多选 | AC |
| 14 | 填空 | 第一空；第二空 |
```

- 单题和多题都必须输出，且为回答最后一节、只出现一次。
- 题型仅允许八种固定中文值，并映射为稳定 Go 枚举。
- 截图显式完整题号/章节号永远优先；仅在截图确实无题号时参考编号摘要。
- 编号摘要是当前 user multipart 的第一个 text part，图片紧随其后；无摘要时仍只有图片。
- 动态摘要不进入 system，不包含答案、旧完整回答、题干、Prompt、Config 或 secret。
- `KeepContext=false` 可携带当前 Session 的业务编号摘要，但不恢复聊天历史。
- 1～3 张图片始终只发起一次模型请求，顺序不变；解析失败也不发补偿请求。

## 4. 分步实施计划与文件白名单

### Task 1：事前治理档案

创建或修改：

- `_dev/changes/022-answer-copy-numbering-sync/00-request.md`～`06-notes.md`
- `_dev/audits/022-answer-copy-numbering-sync.md`
- `_dev/changes/INDEX.md`

完成标志：00/01/02 完整固化批准计划；03～06 与审计仅初始化为尚未实施；INDEX 分配 022。只有本任务完成后才允许修改业务源码。

### Task 2：以 TDD 建立统一答案汇总解析器

创建：

- `pkg/answersummary/types.go`
- `pkg/answersummary/parser.go`
- `pkg/answersummary/numbering.go`
- `pkg/answersummary/parser_test.go`
- `pkg/answersummary/numbering_test.go`

先写 RED 表驱动测试，覆盖单题、多题、所有题型、多空答案、选项大小写/分隔符规范化、Markdown 转义竖线、待核对、重复题号、缺列、额外尾随内容、超限及正文伪汇总。随后最小实现 Parse、BuildCopyText、FormatNumberingContext 和 ExtractLegacyLabels，运行：

```text
GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary -count=1
```

### Task 3：收紧 Prompt，并把编号摘要放入当前 multipart

修改：

- `pkg/domain/prompts.go`
- `pkg/domain/prompts_test.go`
- `pkg/solution/solution.go`
- `pkg/solution/solution_test.go`

先锁定 Prompt 契约和 multipart 顺序。验证 `KeepContext=false/true`、失败、取消和三张图片场景仍只有一次请求；图片顺序不变；system 不含动态摘要。真实模型遵循情况保留为 `NEEDS_USER`。

### Task 4：接入题库原子边界与编号账本

修改：

- `pkg/questionbank/types.go`
- `pkg/questionbank/continuation.go`
- `pkg/questionbank/store.go`
- 相关 `pkg/questionbank/*_test.go`

先用 RED 测试证明 Answer、AnswerItems、ConversationSnapshot 同一候选 Session 一次保存；保存失败进入既有 pending completion，内存 current 不出现半完成，Retry 保留全部字段。`CurrentNumberingContext` 只读取同一 current Session 中较早 completed Records，排除当前 running Record，优先用已校验 Items，旧记录仅保守提取 Label。仅在最近可明确解析的纯数字题号上生成建议值，复杂章节号不猜测。验证新 Session、`KeepContext=false`、Store reopen 及 ended/exported 真正续接。

### Task 5：capture 终态只解析一次并同步 Desktop/Mobile

修改：

- `app/solve.go`
- `app/questionbank_solve_capture.go`
- `app/mobile.go`
- `app/questionbank_history.go`
- 相关 `app/*_test.go`
- `pkg/mobile/state.go`
- `pkg/mobile/bridge.go`
- 相关 `pkg/mobile/*_test.go`

先建题库 Record，再读取排除当前 Record 的编号上下文。capture 收到最终 `solution` 后只调用一次 Parse；成功时完整 Answer 与 Items 原子保存，失败时保存完整 Answer 和空结构化结果。Store 成功后才发布终态。历史 DTO 从持久化 Items 重算 CopyText；旧记录仅在严格符合新契约时临时解析且不回写。保持 typed target、历史浏览绑定、Mobile 序号、清屏和旧 Publish 兼容。

### Task 6：修复 KeepContext 关闭时引用一致性

修改：

- `app/app.go`
- `app/solve.go`
- `pkg/questionbank/continuation.go`
- 相关 App/Store 测试

先重现 `true -> false -> true` 与 in-flight 场景。在 `conversationMu` 边界内同时清 Solver history 和 `conversationRefs`，推进 `conversationEpoch`；旧 epoch 完成不得追加旧引用。`ClearCurrentConversation` 对 active/paused current 先克隆、清 Conversation 并原子保存，成功后替换内存；无 current 或 ended/exported/discarded 幂等不改归档。持久化清理失败时仍清运行时模型历史并明确报告；后续 `KeepContext=false` Record 保存空快照。应用启动且配置为 false 时不恢复旧 turns。

### Task 7：Desktop 增加可靠“复制答案”

创建或修改：

- 新建 `frontend/src/utils/clipboard.js`
- `frontend/src/components/ActionBar.vue`
- `frontend/src/components/SolveView.vue`
- `frontend/src/stores/solution.js`
- `frontend/history_round_selection_regression_test.mjs`
- 新建或扩展 clipboard/answer summary 前端回归测试

实时轮、归档轮和逐轮历史都携带 Items/CopyText；reset 清理这些字段。SolveView 只在 completed 时呈现 ActionBar。ActionBar 保留“复制全文”，新增“复制答案”；含 review 时禁用并提示。统一 `copyText()` 依次尝试 Wails `ClipboardSetText()` 并检查布尔结果、`navigator.clipboard.writeText()`、隐藏 textarea + `document.execCommand('copy')`，仅确认成功才提示成功。复制答案只使用后端 CopyText，空值不回退完整 Markdown。

### Task 8：Mobile 补齐答题端快捷键与答案复制

修改：

- `app/mobile.go`
- `pkg/mobile/web/index.html`
- `pkg/mobile/web_dom_test.mjs`
- Mobile hints/Bridge/Server 相关 Go 测试

快捷键提示增加 `clear_mobile_display`（清空同步端）和 `start_next_exam`（开始下一套），继续只传 `{label,keyName}`，空值跳过。标题改为“答题端快捷键”，说明按键在 Desktop 使用。答案卡仅在 completed Snapshot/历史项有结构化结果时显示“复制答案”，只使用 payload CopyText，Clipboard API 失败后用 textarea fallback。历史选择始终绑定当前项；重连 Snapshot 与 SSE 一致。保持开始下一套不清当前 Mobile 答案，清屏屏障不变。

### Task 9：Bindings、文档与回归边界

修改：

- `frontend/wailsjs/go/models.ts`（由固定 CLI 生成）
- 若生成器产生必要差异：`frontend/wailsjs/go/app/App.js`、`App.d.ts`
- `README.md`
- `README_EN.md`

使用 Wails v2.11.0 全量再生成 bindings；本 Change 不新增公开 App 命令，只接受 DTO/模型字段必要差异。审计 Change 016～021 API 保留，`go.mod`、`go.sum`、Wails runtime 和用户快捷键配置不变。文档说明固定汇总、两种复制、待核对禁用、题号优先级、`KeepContext=false` 编号摘要与“答题端快捷键”。

### Task 10：自动验证、独立构建与治理收尾

依次运行并记录真实结果：

```text
GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary -count=1
GOTOOLCHAIN=go1.25.13 go test ./pkg/domain ./pkg/solution ./pkg/questionbank ./pkg/mobile ./app -count=1
node frontend/history_round_selection_regression_test.mjs
node frontend/questionbank_history_regression_test.mjs
node frontend/mobile_store_regression_test.mjs
node frontend/settings_regression_test.mjs
node pkg/mobile/web_dom_test.mjs
GOTOOLCHAIN=go1.25.13 go test ./... -count=1
GOTOOLCHAIN=go1.25.13 go vet ./...
GOTOOLCHAIN=go1.25.13 go build ./...
npm --prefix frontend run build
GOTOOLCHAIN=go1.25.13 go test -race ./pkg/task ./pkg/solution ./pkg/questionbank ./pkg/mobile ./app -count=1
git diff --check
```

全程只使用合成图片、假 provider 和本地 fixtures。构建前记录并保护 bindings、`go.mod`、`go.sum`、正式程序和旧验证程序哈希。独立构建 `Q-Solver-change022-answer-copy-numbering-validation.exe`，不得覆盖正式程序；权限允许时复制到统一归档并以 LF-only 更新 `SHA256SUMS` 后全量校验。更新 03～06、审计和 INDEX；GUI、真实模型、race 或归档未完成项必须如实记为 `NEEDS_USER` / `BLOCKED`。

## 5. 自动测试矩阵

- 解析：八类题型、单/多题、多空、规范化、转义、伪汇总、结构错误、重复、待核对、长度与总量限制。
- 编号：去重与截断、纯数字建议、复杂章节号不猜、新 Session 清空、旧记录 Label 保守提取。
- Solver：text/image part 顺序、0/1～3 图、KeepContext 开关、失败/取消、请求次数。
- Store：原子保存、pending completion/Retry、旧 JSON、reopen、续接、排除 running Record、只读不改写。
- App：终态解析一次、保存后发布、typed target、不完整终态不复制、历史轮不串写。
- KeepContext：`true -> false -> true`、in-flight 旧 epoch、启动恢复、持久化清理失败。
- Desktop：实时/归档/历史 CopyText、review 禁用、clipboard 三层 fallback、成功/失败提示、选择目标稳定。
- Mobile：Snapshot/SSE/历史绑定、深拷贝、旧 Publish、hints 白名单与改绑、空绑定、序号与清屏屏障。
- 兼容：Change 016～021 累积 API、bindings、依赖与正式程序哈希。

## 6. Windows GUI 与真实模型人工验收

1. Mobile 卡片显示桌面实际绑定，包含“清空同步端”“开始下一套”，改绑后更新，空绑定不显示，标题为“答题端快捷键”。
2. 单题也输出固定汇总；各题型只复制可填写值，多题按显示顺序逐行复制。
3. “复制全文”仍复制完整 Markdown；“复制答案”只复制结构化值。待核对或契约无效时不得复制全文冒充答案。
4. 同一 Session 连续截图：显式题号保持原样；无题号时安全续号；章节或编号规则切换时不强接旧数字。
5. `KeepContext=false`、重启和真正续接后，编号参考仍来自当前 Session；旧完整 Answer 不进入 system。
6. Desktop/Mobile 浏览历史时复制所选项，后台新题完成不改变选择。
7. 三张图只发一次请求；完整回答、截图和 AnswerItems 同步落盘；解析或落盘失败不产生半完成。
8. 开始下一套后新 Session 编号重新开始，旧题库、截图、导出和 Mobile 当前显示不变。

## 7. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 模型未遵循表格契约 | 结构化复制不可用 | 完整 Answer 保留；严格解析失败即禁用复制，不二次调用模型 |
| 从正文误抓答案 | 用户复制错误值 | 只解析末尾最后一个精确标题与紧随固定三列表格 |
| 半完成持久化 | Answer、Items、快照不一致 | 候选 Session 原子保存；失败走 pending completion，成功后才发布终态 |
| 编号强行续接 | 题号错误 | 显式截图优先；只对纯数字生成建议；复杂规则待核对 |
| `KeepContext` 旧引用复活 | 旧对话进入新快照 | 同锁清 history/refs、推进 epoch、拒绝旧 epoch 追加 |
| 历史浏览串写 | 复制了非当前轮答案 | 保持 typed target，每轮独立携带 Items/CopyText |
| clipboard 假成功 | 用户误以为已复制 | 三层实现逐一检查结果，仅确认成功才提示 |
| Mobile 序号被破坏 | SSE/清屏回归 | 保持 `seq`、`clearSeq`、`controlSeq` 语义并加回归测试 |
| bindings 累积噪音 | 误删既有接口或混入依赖 | 固定 CLI 全量生成，前后哈希与 API 审计，不升级依赖 |
| race/GUI 环境受限 | 验收不完整 | 如实标记 `BLOCKED` / `NEEDS_USER`，不关闭 Change |

## 8. 回滚预案

- 定向撤销 `pkg/answersummary`、Prompt/multipart 编号摘要、Record 可选字段、capture 结构化终态、KeepContext 清理修复、Desktop/Mobile 复制入口及 hints 白名单差异。
- 保留原 `Record.Answer`、Session/Record、截图和旧导出；`answerItems` 为可选字段，旧代码可忽略，无需迁移或删除数据。
- 仅删除 Change 022 新验证程序；若已归档，仅移除对应 SHA-256 条目并重新校验其余程序。
- 保留 Change 022 治理档案并记录回滚原因；不得回退 Change 016～021 或修改用户私人配置。
- 不使用 `git reset --hard`、`git clean`、`git add .`、`git add -A` 或批量覆盖式恢复。

## 9. 实施与 Git 边界

- 本方案批准不等于 commit、push、Tag 或 Release 授权。
- 实施必须按 TDD 分阶段进行，先 RED 再 GREEN，并同步更新 03/04。
- 不得越出本方案文件白名单；新增必要文件或范围变化须先更新治理记录并取得确认。
- 不读取或输出真实 API 配置、Mobile Token 或私人配置。
- 事前建档阶段不运行测试、构建、Wails 生成、真实模型或 Git 写操作。
## 实施细化（接口语义不变）

- Task 6 的启动、Retry 与真正续接恢复逻辑在 `app/questionbank_continuation.go`；将该文件纳入定向修改白名单，只增加 KeepContext=false 守卫，不重构其他续接行为。
- ClearCurrentConversation 必须保存明确的空快照而不是 nil，防止触发旧数据重建；失败时保留空快照候选供 Retry，原记录仍保留。
- 验证证据包括 `pre-build-protected.json`、`tmp/bindings-before/` 和本 Change 专用本地日志；均不含用户配置或密钥。
