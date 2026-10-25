# Change 022 实现记录

创建日期：2026-10-24
最终收尾：2026-10-25
状态：业务实现、普通自动验证、正式 bindings、独立构建与统一归档完成；Windows GUI 与真实模型 NEEDS_USER；race BLOCKED；未提交、未推送、未发布、未关闭。

## 事前治理

00/01/02 在业务源码修改前完成；随后按下列阶段执行 TDD。未执行 Git 写操作。

## Task 2：统一答案汇总解析器（2026-10-24）

### TDD RED

1. 先仅新增 `pkg/answersummary/parser_test.go` 与 `pkg/answersummary/numbering_test.go`。
2. 测试覆盖：固定末尾三列表格、单题/多题、八种题型、多空答案、单/多选大小写与常见分隔符规范化、Markdown 转义竖线、`待核对` 全轮禁用复制、重复题号、缺列/额外列/错误表头、尾随内容、正文伪汇总、空表、50 项上限、Label 80 字符上限、Value 2,000 字符上限、32 KiB 总量上限、CopyText 全 `ready`、legacy Label 保守提取、最近 64 个去重 Label 与 4 KiB 编号提示上限。
3. 执行 `GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary -count=1`，退出码 1；编译失败，首个错误为 `undefined: FormatNumberingContext`，随后为计划接口和类型尚不存在，符合 RED 预期。

### 最小实现

仅新增以下三个实现文件：

- `pkg/answersummary/types.go`：按批准计划定义八种稳定题型、`ready` / `review` 状态、`Item`、`Result`、`NumberingContext` 及四个公开函数所需类型。
- `pkg/answersummary/parser.go`：只解析最后一个、位于 fenced code 之外的精确 `## 最终答案汇总`，要求标题后立即出现固定表头、固定分隔行和三列表格，且表格必须结束整个回答；统一规范化选择题并验证重复、长度、数量和总量边界；Markdown `\|` 解码为值内竖线；任一 `review` 保留结构化项但不生成 CopyText。
- `pkg/answersummary/numbering.go`：编号提示只序列化安全 Label 与可选建议，不接收或派生答案；从最近项向前去重，最多 64 个 Label，并按 UTF-8 字节裁剪至 4 KiB；legacy 路径只从精确汇总段的表格首列或保守冒号格式提取 Label，不生成答案或 CopyText。

实现期间一次组合 heredoc 命令在 shell 解析阶段失败，检查确认三个实现文件均未生成；随后仅在同一白名单内使用文件写入工具创建目标文件，不存在部分文件或范围外改动。

### TDD GREEN

执行同一命令 `GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary -count=1`，退出码 0，结果：`ok Q-Solver/pkg/answersummary 0.358s`。

### 本阶段范围与状态

- Task 2 已完成；Task 3～Task 10 未实施。
- 本阶段未修改其他业务文件、配置、依赖、用户数据或既有 Change 016～021 改动。
- 未运行真实模型，未执行 commit、push、Tag 或 Release。

## 与计划的偏差（Task 2）

无接口或文件范围偏差。实现命令曾发生一次未落盘的 shell heredoc 解析失败，不影响最终文件或测试结论。

### 2026-10-24 Task 2 复核补强

复核发现初版仅在字段完全等于“待核对”时标记 `review`，可能把“待核对：截图不清”等说明性值误当作可复制答案。先新增包含匹配用例并得到预期 RED，随后统一改为 Label 或 Value 包含“待核对”即标记 `review`，且跳过选择题规范化；整轮 `CopyText` 保持为空。

## Task 3：收紧 Prompt 与当前 user multipart 编号摘要（2026-10-24）

### TDD RED

1. 先只修改 `pkg/domain/prompts_test.go` 与 `pkg/solution/solution_test.go`，未改实现。
2. Prompt 契约测试新增：单题/多题回答最后一节唯一固定三列表格、八种固定题型、选择题及其他题型纯答案格式、显式截图题号优先、无题号才参考编号摘要、无法确定标记“待核对”、表格后不得继续输出。
3. Solver 测试新增：空编号上下文仍只有原图；非空上下文必须为当前 user multipart 的第一个 text part，随后是三张原图；`KeepContext=false/true`（含已有历史）均保持消息边界；动态摘要不进入 system；Provider 失败或 context 取消均只调用一次且不改变 multipart 图片顺序。
4. 执行 `GOTOOLCHAIN=go1.25.13 go test ./pkg/domain ./pkg/solution -count=1`，退出码 1。`pkg/solution` 因 `Request` 尚无 `NumberingContext` 字段而编译失败；`pkg/domain` 同时明确报告旧 Prompt 缺少固定表格、八种题型、纯答案格式、编号优先级等新增约束，符合预期 RED。

### 最小实现

- `pkg/domain/prompts.go`：把原先“多题可用列表或表格、单题不强制”的规则收紧为单题和多题均必须在回答最后输出且只输出一次固定 `题号 / 题型 / 可填写答案` 三列表格；限定八种题型与纯答案格式；明确截图显式完整题号/章节号优先，仅无题号时参考当前 user 中的编号摘要，无法确定时标记“待核对”。纯 OCR、纯翻译等非解题场景边界与变式题隔离规则保持不变。
- `pkg/solution/solution.go`：`Request` 增加 `answersummary.NumberingContext`。每次 Solve 只调用一次 `answersummary.FormatNumberingContext`；结果非空时先追加 `llm.TextPart`，再按原顺序追加所有 `llm.ImagePart`；结果为空时不增加 text part。system 仍仅由静态行为 Prompt、场景 Prompt 和既有配置内容构建，未注入动态编号摘要，也未增加任何模型请求或重试。

### TDD GREEN

- 首次 GREEN：`GOTOOLCHAIN=go1.25.13 go test ./pkg/domain ./pkg/solution -count=1`，退出码 0；结果为 `ok Q-Solver/pkg/domain 0.328s`、`ok Q-Solver/pkg/solution 0.646s`。
- 补充已有历史的 `KeepContext=true` 角色序列断言后，执行 `GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary ./pkg/domain ./pkg/solution -count=1`，退出码 0；三包均通过。
- 全量 `GOTOOLCHAIN=go1.25.13 go test ./... -count=1` 与 `GOTOOLCHAIN=go1.25.13 go build ./...` 均退出码 0。

### 本阶段范围与状态

- Task 3 已完成；Task 4～Task 10 未在本阶段实施。
- 只修改本 Task 批准的四个 Go 文件并追加本 Change 的 03/04 记录；未修改依赖、配置、用户数据或 Change 016～021 的既有内容。
- 全部测试使用 fake provider 和合成字符串；未调用真实模型，未上传截图，未执行 commit、push、Tag 或 Release。

## 与计划的偏差（Task 3）

无接口、请求次数或文件范围偏差。真实模型能否稳定遵循固定表格仍按计划保留为后续 `NEEDS_USER` 人工验收。

## Task 4：题库原子持久化与当前分段编号账本（2026-10-24）

### TDD RED

1. 先只在 `pkg/questionbank/continuation_test.go` 增加 Task 4 聚焦测试，覆盖三字段单候选保存、失败后的 `pendingCompletion` 与重试、当前分段编号聚合、新分段、`KeepContext=false`、Store reopen、ended/exported 分段真实续接、旧 v1/v2 JSON 以及只读查询不改写磁盘。
2. 执行 `GOTOOLCHAIN=go1.25.13 go test ./pkg/questionbank -run 'Test(FinishWithConversationResult|CurrentNumberingContext)' -count=1`，退出码 1。编译错误包含 `s.FinishWithConversationResult undefined`、`record.AnswerItems undefined`、`s.CurrentNumberingContext undefined`，符合计划 API 尚未实现的 RED 预期。

### 最小实现

- `pkg/questionbank/types.go`：为 `Record` 增加 `AnswerItems []answersummary.Item \`json:"answerItems,omitempty"\``；`currentSessionVersion` 保持 2，旧 v1/v2 JSON 缺少该字段时按零值读取。
- `pkg/questionbank/store.go`：新增 `cloneAnswerItems`，隔离调用方切片；沿用既有 `pendingCompletion`、`Flush` 和写阻断机制，不改变既有 schema 或恢复语义。
- `pkg/questionbank/continuation.go`：
  - 保留旧 `FinishWithConversation`，作为向 `FinishWithConversationResult` 传 `nil items` 的兼容包装。
  - 新方法只在 clone candidate 上同时写入 `Answer`、`AnswerItems`、`Status` 和 `ConversationSnapshot`，并调用一次 `saveSessionLocked`；成功后整体替换 current，失败时把完整 candidate 放入 `pendingCompletion`，current 继续保持 running，`Flush` 重试后再整体提交。
  - 新增只读 `CurrentNumberingContext(excludeRecordID)`：仅遍历 current Session 中排除记录之前的 completed Records；已通过 `answersummary.BuildCopyText` 校验的结构化项优先，否则只调用 `answersummary.ExtractLegacyLabels`；从末尾去重并保留最近 64 个 Label；仅对 ASCII 纯十进制且未发生 `uint64` 溢出的最近 Label 给出 `n+1`。
  - 编号账本不读取答案 Value；对结构化 Label 额外过滤 `待核对`、包含“答案”或包含 `|` 的不安全值，避免把答案样文本带入编号提示。
- `pkg/questionbank/continuation_test.go`：新增原子保存/失败重试、调用方切片与快照隔离、旧包装不写 `AnswerItems`、结构化优先与 legacy 回退、completed/记录顺序边界、最近 64 个跨记录去重、复杂/溢出编号不猜、新 Session、`KeepContext=false`、active reopen、ended/exported 续接、旧 v1/v2 JSON 与只读不改写磁盘测试。

### 安全补强 RED/GREEN

初次最小实现后聚焦测试已 GREEN。随后加入“合法结构化 Item 的 Label 为 `答案：LABEL-SECRET`”用例，测试按预期 RED：实际编号上下文包含该 Label，而期望仅保留安全题号。最小修复只在结构化 Label 收集处增加上述不安全 Label 过滤；复跑同一聚焦测试恢复 GREEN。该补强不把 Answer 或 Item Value 传入编号上下文，也不扩大 legacy 提取范围。

### 本阶段范围与状态

- Task 4 已完成；未开始 Task 5 或其他层接线。
- 本阶段仅修改 `pkg/questionbank/types.go`、`pkg/questionbank/store.go`、`pkg/questionbank/continuation.go`、`pkg/questionbank/continuation_test.go`，并追加本 Change 的 03/04 记录；未覆盖 Change 016～021 已有叠加改动。
- 未修改配置、依赖或用户数据，未运行真实模型，未生成 Wails bindings，未执行 commit、push、Tag 或 Release。

## 与计划的偏差（Task 4）

无接口、schema version、持久化边界或文件范围偏差。除批准计划外，仅增加了保守的结构化 Label 安全过滤，以及对调用方在保存失败后修改原 `items` / `snapshot` 不会污染待重试 candidate 的测试。

## Task 5：结构化终态与历史 DTO

上次代理受 429 中断，仅留下部分测试（含一个未使用 import）；本轮在主会话接续，不将代理部分结果当作完成。新增 capture/历史/Mobile 测试先因 DTO 字段缺失 RED，然后完成以下接线并通过 App、Mobile 测试：

- 创建 Record 后取得编号摘要，再调用一次 Solver；无题库时保持空摘要。
- capture 在 solution 事件解析一次，延迟到 Complete 成功持久化后发布完整 Answer + Result；无题库分支也只发布一次。解析失败保留全文、复制文本为空。
- DesktopSolveEvent、历史详情与 Mobile Payload 携带 answerItems/answerCopyText；CopyText 在 Go 重新校验生成，不信任外传 CopyText。
- Mobile Snapshot/GetState/每个订阅者各自复制 Items；错误、开始、新流和 legacy 单参数终态清除旧结构化值。
- 历史旧格式只有严格新表格可临时恢复复制，不写回旧文件。

## Task 6：关闭上下文的一致性

先以 Store 缺失接口及 App 旧 refs 未清理得到 RED，再实现并通过测试。

- true→false 同锁清 Solver、refs，并推进 conversationEpoch；旧 epoch 请求仍保存完整结果，但不重新追加历史引用，不回滚已清理上下文。
- Store ClearCurrentConversation 写入明确的空 ConversationSnapshot，不能设为 nil（nil 会触发旧记录重建）。保存失败保留空快照 candidate 以便 Retry，不丢弃原记录。
- 启动/Retry 在 KeepContext=false 时不恢复历史。
- 实际恢复实现位于 app/questionbank_continuation.go，本阶段定向增加 false 配置守卫；这是批准计划 Task 6 的启动/恢复要求所需文件，未增加公开 API 或无关重构。
- 覆盖 true→false→true、在途请求、Store 清理失败后 Retry、reopen 不复活等测试。

## Task 2/4 复核补充

编号提示过滤所有包含“待核对”的 Label（包括“待核对-1”）；对应测试先 RED 后 GREEN，避免不确定题号进入下轮编号参考。

## Task 7/8：Desktop 与 Mobile 复制、实际快捷键提示

- Desktop round 与归档 round 保存结构化结果，按所选 Record 复制；legacy 全文复制保持可用。ActionBar 区分复制全文/复制答案，后者无安全 CopyText 时禁用，不回退全文。
- 新建 clipboard helper：Wails 布尔结果、浏览器 Clipboard API、textarea execCommand 三层；检查失败，恢复焦点，清理临时节点。异步反馈用版本标记绑定当前内容，切换轮次不误报已复制。
- Mobile 答案卡新增复制按钮，使用 Go 提供的 CopyText；历史条目带结构化结果，浏览旧项时后台新帧不切走选择。清屏、新流、错误清理旧复制目标；失败有明确提示。
- Mobile 提示加入 clear_mobile_display/start_next_exam 实际 KeyName，空绑定隐藏；标题明确为“答题端快捷键”，提示按键在答题电脑使用。
- 初次 Go→SSE→DOM 集成测试暴露测试脚本重建 DOM 后继续使用旧 context 的夹具污染；把新复制用例放到可选真实 SSE 夹具之前，明确隔离实例后，DOM 与 Mobile Go 测试通过。

## 实施后自审补强

- 已取消请求在 provider 仍返回成功文本时，旧 Solver 会发布 solution；新增专项测试 RED 后，在 provider 返回后检查 ctx.Err，取消/超时不再产生可复制终态。
- 编号参考限制为字母、数字和有限编号标点，拒绝引号、尖括号、冒号等指令/标记结构；仍仅作为 user 数据，不提升为 system。对应注入样式 Label 测试先 RED 后 GREEN。
- 正式 bindings 完整再生成；App.js/App.d.ts 与事前副本一致，models.ts 只新增答案项类型与历史 DTO 字段，并移除生成器纯空白 Tab 行。

## Task 9/10：交付收尾

- 更新 README 与 README_EN 的复制/编号使用说明；公开 API 不变，模型类型新增答案项与历史复制文本。
- 普通 Go test/vet/build、全部前端回归、Mobile DOM、Vite 和固定 Wails v2.11.0 独立构建通过；工具链 go1.25.13。
- 最终程序已复制到统一验证目录，12 条 SHA256SUMS 全部通过；正式 EXE、旧程序、Go 依赖和 runtime 共 28 项保护哈希保持不变。
- race 仍为 Windows 启动错误 0xc0000139；GUI/真实模型未经用户验收，Change 不关闭。
