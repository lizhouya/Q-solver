# Change 022 验证记录

创建日期：2026-10-24
最终收尾：2026-10-25
状态：业务实现、普通自动验证、正式 bindings、独立构建与统一归档完成；Windows GUI 与真实模型 NEEDS_USER；race BLOCKED；未提交、未推送、未发布、未关闭。

## Task 2 验证（2026-10-24）

### RED

- 命令：`GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary -count=1`
- 阶段：仅存在 `parser_test.go`、`numbering_test.go`，实现文件尚未创建。
- 退出码：1。
- 结果：`FAIL Q-Solver/pkg/answersummary [build failed]`。
- 关键原始错误：`pkg\answersummary\numbering_test.go:10:12: undefined: FormatNumberingContext`；后续错误继续指出 `NumberingContext`、`ExtractLegacyLabels` 等计划接口尚不存在。
- 判定：预期 RED，失败由缺少待实现接口导致，不是环境或既有 Change 016～021 故障。

### GREEN

- 命令：`GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary -count=1`
- 环境：Windows 11，仓库根目录，固定 `GOTOOLCHAIN=go1.25.13`。
- 退出码：0。
- 原始结果：`ok  \tQ-Solver/pkg/answersummary\t0.358s`。
- 覆盖结果：固定末尾三列表格、八种题型、单/多选规范化、多空答案、转义竖线、`待核对`、重复/长度/数量/总量限制、严格 CopyText、legacy 仅 Label、编号上下文 64 Label / 4 KiB 边界均通过。

### 未执行与结论边界

- 本阶段仅完成计划 Task 2 的包级聚焦测试；Task 3～Task 10 的跨包测试、`go test ./...`、`go vet ./...`、`go build ./...`、前端测试/构建、race、Wails/GUI 与独立验证构建均未在本阶段执行。
- 未调用真实模型；真实模型固定表格遵循情况仍为后续 `NEEDS_USER` 人工验收。
- 结论：Task 2 聚焦测试 GREEN，但 Change 022 整体仍在实施中，不满足 Close 条件。

### Task 2 复核补强

- RED：`GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary -run TestParseReviewDisablesWholeRoundCopy -count=1`，新增的“待核对-1”和“待核对：截图不清”两例失败，证明原实现会错误生成 CopyText。
- GREEN：`GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary -count=1`，结果 `ok`。
- `git diff --check -- pkg/answersummary`：通过。

## Task 3 验证（2026-10-24）

### RED

- 命令：`GOTOOLCHAIN=go1.25.13 go test ./pkg/domain ./pkg/solution -count=1`
- 阶段：只新增/修改 Prompt 契约与 Solver multipart 测试，实现尚未修改。
- 退出码：1。
- `pkg/solution` 关键原始错误：`unknown field NumberingContext in struct literal of type Request`（3 处），证明请求契约尚未接入。
- `pkg/domain` 失败明确列出旧 Prompt 缺少“单题和多题”“回答的最后一节”“必须且只能出现一次”、固定三列表头与分隔行、八种题型、纯答案格式、显式题号优先、仅无题号参考编号摘要及表格后不得输出等约束。
- 判定：预期 RED，失败直接对应 Task 3 尚未实现的契约，不是环境故障，也未调用真实模型。

### GREEN：聚焦验证

1. 命令：`GOTOOLCHAIN=go1.25.13 go test ./pkg/domain ./pkg/solution -count=1`
   - 退出码：0。
   - 原始结果：`ok Q-Solver/pkg/domain 0.328s`；`ok Q-Solver/pkg/solution 0.646s`。
2. 补充 `KeepContext=true` 已有历史断言后的复核命令：`GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary ./pkg/domain ./pkg/solution -count=1`
   - 退出码：0。
   - 原始结果：`ok Q-Solver/pkg/answersummary 0.336s`；`ok Q-Solver/pkg/domain 0.303s`；`ok Q-Solver/pkg/solution 0.663s`。

覆盖结论：

- Prompt 同时约束单题/多题唯一末尾固定三列表格、八种题型和纯答案格式。
- 显式截图完整题号/章节号优先；只有截图无题号时才参考编号摘要；不确定项标记“待核对”。
- 空编号上下文的当前 user 仍只有三张图片；非空时严格为 `TextPart(FormatNumberingContext)` 后跟原图 `ImagePart`，三图顺序保持不变。
- `KeepContext=false` 仍可携带业务编号摘要且不带历史；`KeepContext=true` 在已有历史时角色序列仍为 `[system,user,assistant,user]`，编号摘要只属于最后的当前 user。
- system 不包含格式化动态摘要；成功、Provider 失败和 context 取消场景每次 Solve 都只记录一次 Provider 调用，没有新增第二次请求或补偿重试。

### GREEN：全量 Go 回归与构建

- `GOTOOLCHAIN=go1.25.13 go test ./... -count=1`：退出码 0，全部 Go 包通过。
- `GOTOOLCHAIN=go1.25.13 go build ./...`：退出码 0，无输出。

### 未执行与结论边界

- 未调用真实模型；模型实际遵循固定表格的稳定性仍为 `NEEDS_USER`。
- Task 3 不涉及前端、Wails bindings 或 GUI，因此本阶段未运行前端构建、Wails 打包、Windows GUI/clipboard 人工验收或独立验证程序。
- Task 4～Task 10 尚未实施；Change 022 整体仍在实施中，不能 Close。
- 未执行 commit、push、Tag 或 Release。

### 差异检查

- tracked 实现文件执行 `git diff --check -- pkg/domain/prompts.go pkg/solution/solution.go pkg/solution/solution_test.go`，并对未跟踪的 `pkg/domain/prompts_test.go`、`03-implementation.md`、`04-validation.md` 使用等价 `git diff --no-index --check`：通过，无 trailing whitespace 错误。
- Git 输出既有工作树的 `LF will be replaced by CRLF` 提示；该提示不代表空白错误，未修改行尾配置或其他文件。

## Task 4 验证（2026-10-24）

### RED：计划 API 尚不存在

- 命令：`GOTOOLCHAIN=go1.25.13 go test ./pkg/questionbank -run 'Test(FinishWithConversationResult|CurrentNumberingContext)' -count=1`
- 阶段：只新增 Task 4 测试，实现尚未修改。
- 退出码：1。
- 关键原始错误：`s.FinishWithConversationResult undefined`、`record.AnswerItems undefined`、`s.CurrentNumberingContext undefined`。
- 判定：预期 RED，失败直接对应 Task 4 尚未实现的字段和方法，不是环境或既有 Change 016～021 故障。

### GREEN：聚焦验证

1. 初次最小实现后的同一聚焦命令退出码 0：`PASS`，`ok Q-Solver/pkg/questionbank 1.948s`。
2. 安全补强 RED：加入合法结构化 Label `答案：LABEL-SECRET` 后，同一测试报告实际 `PreviousLabels:[1 2 答案：LABEL-SECRET 第二章-3]`，证明编号上下文可能携带答案样文本。
3. 增加保守 Label 过滤后，同一聚焦命令退出码 0：`PASS`，`ok Q-Solver/pkg/questionbank 1.232s`。
4. 最终补充旧包装与失败后调用方修改隔离断言后执行：`GOTOOLCHAIN=go1.25.13 go test ./pkg/questionbank -run 'Test(FinishWithConversation|CurrentNumberingContext)' -count=1`，退出码 0，结果 `ok Q-Solver/pkg/questionbank 1.363s`。

覆盖结论：

- `Answer`、`AnswerItems`、`ConversationSnapshot` 在同一个 clone candidate 中经一次保存提交；save hook 执行时 current 仍是 running 且无半完成字段。
- 首次保存失败时 current 不半更新，完整 candidate 进入 `pendingCompletion`；调用方随后修改原 Item 切片或 Snapshot 不影响待重试数据；`Flush` 后内存和磁盘同时得到完整三字段结果。
- 旧 `FinishWithConversation` 仍可用并保持 `AnswerItems` 为空；Session schema version 仍为 2。
- 编号账本只读取 current Session 内、排除记录之前的 completed Records；`KeepContext=false` 不影响业务编号；有效结构化项优先，无效/缺失结构化项只走 `ExtractLegacyLabels`；不包含 Answer、Item Value、failed/后续记录或不安全 Label。
- 最近 64 个 Label 跨记录去重；纯 ASCII 十进制安全递增，章节号与 `uint64` 最大值不猜测。
- 新 Session 不继承旧分段编号；active Store reopen、ended/exported 分段经真实打开或 `ActivateHistoricalSession` 续接后编号仍可恢复。
- 旧 v1/v2 JSON 均可读取 legacy Label；查询前后 `session.json` 字节不变，证明只读查询不改写磁盘。

### GREEN：最终包级、全量回归、静态检查与构建

- `GOTOOLCHAIN=go1.25.13 go test ./pkg/answersummary ./pkg/questionbank -count=1`：退出码 0；`ok Q-Solver/pkg/answersummary 0.345s`，`ok Q-Solver/pkg/questionbank 4.600s`。
- `GOTOOLCHAIN=go1.25.13 go test ./... -count=1`：退出码 0；全部 Go 包通过，其中 `Q-Solver/app 6.754s`、`Q-Solver/pkg/questionbank 5.754s`。
- `GOTOOLCHAIN=go1.25.13 go vet ./pkg/answersummary ./pkg/questionbank`：退出码 0，无输出。
- `GOTOOLCHAIN=go1.25.13 go build ./...`：退出码 0，无输出。

### 未执行与结论边界

- Task 4 不涉及前端、Wails bindings 或 GUI，未运行前端构建、bindings 生成、Wails 打包或 Windows GUI 人工验收。
- 未调用真实模型；未修改配置、依赖或用户数据；未执行 commit、push、Tag 或 Release。
- Task 5～Task 10 尚未在本阶段实施；Change 022 整体仍在实施中，不能 Close。

### 差异与格式检查

- `git diff --check -- pkg/questionbank/types.go pkg/questionbank/store.go`：退出码 0。
- 对未跟踪的 `pkg/questionbank/continuation.go`、`pkg/questionbank/continuation_test.go`、`03-implementation.md`、`04-validation.md` 分别执行等价 `git diff --no-index --check /dev/null <file>`：均无空白错误。
- `gofmt -d pkg/questionbank/types.go pkg/questionbank/store.go pkg/questionbank/continuation.go pkg/questionbank/continuation_test.go`：无差异。
- Git 仅输出既有 `LF will be replaced by CRLF` 工作树提示；未修改行尾配置。
- 最终 `git status --short --branch` 与事前叠加工作树文件集合一致，仅本 Task 白名单文件内容及 Change 022 的 03/04 记录发生本轮修改；未产生构建产物或范围外新文件。

## Task 5/6 实际验证

- 初始中断状态：`go test ./app ./pkg/mobile -count=1` 因 app/questionbank_test.go 未使用 import 编译失败，不能视为功能 RED。
- Task 5 补充测试 RED：Payload/历史 DTO 的 AnswerItems、AnswerCopyText 字段未定义。
- Task 5 GREEN：`GOTOOLCHAIN=go1.25.13 go test ./app ./pkg/mobile -count=1` 通过。
- Task 6 RED：Store ClearCurrentConversation 未定义；App 独立测试明确报 `OFF did not clear refs and solver`（修正测试夹具 nil llmService 后确认）。
- Task 6 GREEN：`GOTOOLCHAIN=go1.25.13 go test ./app ./pkg/questionbank -count=1` 通过；加入 in-flight、保存失败 Retry 用例后复跑通过。
- 本阶段没有真实模型请求、用户配置读取或写入、GUI 操作或 Git 写操作。

## Task 7/8 验证

- RED：clipboard helper 文件不存在；历史轮 answerCopyText 为 undefined；新 Mobile 提示为空；页面缺少“答题端快捷键”。
- GREEN：`node frontend/clipboard_regression_test.mjs`、`node frontend/history_round_selection_regression_test.mjs`、`node pkg/mobile/web_dom_test.mjs` 通过。
- 全部 `frontend/*regression_test.mjs` 七份脚本通过。
- `GOTOOLCHAIN=go1.25.13 go test ./app ./pkg/mobile -count=1` 的首次集成有 DOM 夹具失败；修正测试隔离后 `go test ./pkg/mobile -count=1` 通过，App 当次已通过。
- `npm --prefix frontend run build` 通过；仍有既有大于 500 kB 的非阻断 bundle 警告。

## 自审补强验证

- `TestCancelledRequestCannotPublishSuccessfulProviderReply`：先 FAIL（cancelled reply published as completed），增加 provider 返回后的 ctx.Err 检查后 `go test ./pkg/solution ./app -count=1` 通过。
- `TestNumberingContextRejectsMarkupInLabels`：先 FAIL（markup entered numbering context），字符集校验后 `go test ./pkg/answersummary ./pkg/questionbank ./pkg/solution -count=1` 通过。

## 最终验证汇总（2026-10-25）

证据：`tmp/final-validation.log`、`tmp/wails-build.log`、`tmp/race.log`；仅包含合成测试与构建输出。

| 项目 | 结果 |
|---|---|
| `GOTOOLCHAIN=go1.25.13 go test ./... -count=1` | PASS |
| `GOTOOLCHAIN=go1.25.13 go vet ./...` | PASS |
| `GOTOOLCHAIN=go1.25.13 go build ./...` | PASS |
| 全部七份 `frontend/*regression_test.mjs` | PASS |
| `node pkg/mobile/web_dom_test.mjs`（含真实 SSE 集成） | PASS |
| `npm --prefix frontend run build` | PASS；既有 >500 kB bundle warning 保留 |
| Wails v2.11.0 首次生成 bindings 与生产构建 | PASS，18.224s |
| Wails 最终 `-skipbindings` 生产重建 | PASS，10.595s |
| 统一归档 12 条 SHA256SUMS | 全部 OK |
| 构建前 28 项旧产物/依赖/runtime 哈希 | 全部不变 |

### 独立验证程序

- 构建输出：`build/bin/Q-Solver-change022-answer-copy-numbering-validation.exe`
- 统一归档：`build/bin/archive/validation/Q-Solver-change022-answer-copy-numbering-validation.exe`
- 大小：23,440,896 字节。
- SHA-256：`5cde3df3a8bdd160f7740ffeebc6b68c7dcb3f87bbfc5ff7226ce6bf2950e751`。
- 正式 `build/bin/Q-Solver.exe` 保持原哈希 `5dcabb17933bfa9ab4dc15f289b4b8e50af99879c316b58831b09bbb4e39aa11`，未覆盖。

### Bindings 与依赖

本次生成前备份了三份 bindings。App.js 和 App.d.ts 逐字节不变；models.ts 新增 answersummary.Item、历史 DTO 的 answerItems/answerCopyText 及转换代码，清除生成器空白行 Tab。go.mod、go.sum、runtime 哈希不变；frontend/package.json/package-lock.json 无本次差异。

### Race

`GOTOOLCHAIN=go1.25.13 go test -race ./pkg/task ./pkg/solution ./pkg/questionbank ./pkg/mobile ./app -count=1`：退出码 1，五包均在程序启动阶段返回 `exit status 0xc0000139`。状态 **BLOCKED**，没有得到 race 安全性结论。

### 人工验收

Windows GUI 与真实模型 **NEEDS_USER**，没有使用用户 API 或真实截图进行验证：

1. 启动本次独立验证程序，同步页面重新加载后检查“答题端快捷键”，修改/清空绑定后应同步。
2. 单题、多题分别检查末尾“题号 / 题型 / 可填写答案”汇总；点击复制答案，粘贴后只含答案值，多题一行一题；复制全文仍保留解析。
3. 待核对/格式不合规时按钮禁用，失败不提示已复制。
4. 同一分段连续截图检查题号；显式新章节编号优先；关闭 KeepContext、重启与真正续接后再检查编号参考。
5. Desktop/Mobile 浏览旧轮期间新题完成，旧轮复制目标保持不变；点击最新历史条目恢复跟随。
6. 开始下一套后桌面清理、新分段按当前截图编号；题库原始数据、旧导出与 Mobile 当前显示保持不变。

未提交、未推送、未发布，不能以本次构建通过替代人工验收或关闭 Change。

### 最终文档与差异检查

`git diff --check` 最终退出码 0；保留 Git 行尾转换提示，但无空白错误。Change 022 七份生命周期文档与审计齐全，状态未提前关闭。
