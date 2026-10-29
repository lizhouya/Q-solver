# Change 024 题库整理失败恢复实施计划

> **执行约束：** 按 TDD 分阶段实施；未经授权不提交、不推送、不创建 Tag、不发布。

**Goal：** 恢复本次 TCL 综合笔试的可核对整理稿，并修复“整理全部失败仍被判定为准备完成、随后导出零题整理稿”的流程缺陷。

**Architecture：** 保留现有“准备整理 → 人工核对 → 纯本地导出”两阶段架构，在 questionbank 层统一计算整理结果状态、持久化脱敏失败分类并提供只重试失败项；App 和前端依据明确状态展示可达操作；Store 对零题整理稿做最终保护。历史 TCL 通过独立 `_02` 离线恢复，不回写 Session 或覆盖旧导出。

**Tech Stack：** Go 1.25.13、Wails v2.11.0、Vue 3、Pinia 3、Vite 4.5.14。

**Spec：** 本计划获批后固化到 `_dev/changes/024-question-bank-organize-failure-recovery/02-plan.md`。

## Context

目标文件 `输出/求职笔试/题库/2026-10-25_095652_TCL_综合笔试/TCL_综合笔试.md` 只有 25 条原稿链接和“题目数量：0”。只读核查确认原 Session `a6ef78f5c602e1950b991e3d` 的 25 条完整答案、`answerItems` 和 25 张截图仍在，但 25 条 `Questions` 全为空，且所有失败均被压缩成相同的 `OrganizeError`。

可确认的应用层故障链是：`questionBankOrganizeState` 把失败项排除在 pending 之外，`OrganizeReady` 又只判断 `pending == 0`，所以 `pending=0, failed=25, questions=0` 被显示成“准备完成”；最终 API 只拦截 pending，`ExportSession` 仍会写出零题整理稿。现有数据没有保留请求错误、空响应或具体解析阶段，因此不能断言 provider、非流式接口、额度、网络或某一种 JSON 偏差是本次 25 条失败的确定根因。

本 Change 同时交付两项结果：

1. 为当前 TCL 创建不覆盖旧目录的 `_02` 恢复版，透明标注题号错配、重复、缺失和待核对答案。
2. 修复未来整理流程，使全部失败、部分失败、待处理和全部成功拥有不同状态与操作，并保存安全诊断信息。

## Global Constraints

- 先创建并完成 Change 024 的 `00-request.md`、`01-current-state.md`、`02-plan.md`，之后才修改业务源码。
- 固定分支 `dev/qsolver-custom`、Wails CLI v2.11.0、`GOTOOLCHAIN=go1.25.13`。
- 不新增依赖，不升级 Wails runtime、Go module 或 npm 依赖。
- 不调用真实模型、不上传截图、不消费真实 API 额度；自动测试只用 fake provider、合成图片和临时目录。
- 不读取、记录或输出 API Key、Token、Authorization header、完整 BaseURL、Prompt、截图 Base64、第三方原始错误体或完整无效模型响应。
- 不删除、覆盖或迁移 `_记录`、截图、旧导出、正式 `Q-Solver.exe` 或旧验证程序。
- 不把模型内容称为“官方答案”或“标准答案”，只使用“参考答案”“原模型建议”“待核对”。
- 不执行 `git reset --hard`、`git clean`、`git add .` 或 `git add -A`。
- 本计划不构成 commit、push、Tag 或 Release 授权。

---

## 目标状态与接口契约

### 整理汇总

新增 questionbank 层纯汇总函数，对 `Completed` Record 统计：

```go
type OrganizeSummary struct {
    Outcome             string
    EligibleRecords     int
    OrganizedRecords    int
    StructuredQuestions int
    PendingRecords      int
    FailedRecords       int
}
```

稳定状态：

- `pending`：仍有尚未尝试或取消后遗留的记录。
- `complete`：至少有一道结构化题，且无 pending/failed。
- `partial`：至少有一道结构化题，无 pending，但仍有 failed。
- `all_failed`：有可整理记录，零结构化题，无 pending，且存在 failed。
- `no_eligible`：没有可整理的 completed 记录，只能导出原稿。

`QuestionBankExportDraft` 保留现有字段以兼容前端，并新增 `organizeOutcome`、`eligibleOrganize`、`organizedRecords`、`structuredQuestions`。`OrganizeReady` 改为 `PendingRecords == 0 && StructuredQuestions > 0`，因此只有 `complete` 和 `partial` 为 true。

### 安全失败记录

保留现有 `Record.OrganizeError` 作为安全中文文案；新增可选结构，不提升 Session version、不批量迁移旧文件：

```go
type OrganizeFailure struct {
    Code           string    `json:"code"`
    Stage          string    `json:"stage"`
    Retryable      bool      `json:"retryable"`
    ProviderCalled bool      `json:"providerCalled"`
    AttemptedAt    time.Time `json:"attemptedAt"`
    Model          string    `json:"model,omitempty"`
    ResponseBytes  int       `json:"responseBytes,omitempty"`
}
```

固定脱敏分类：`input_invalid`、`request_timeout`、`request_failed`、`empty_response`、`response_too_large`、`invalid_json`、`extra_content`、`source_mismatch`、`schema_invalid`、`legacy_unknown`。旧记录只有 `OrganizeError` 时视为 `legacy_unknown`，仍可重试。成功重试同时清除 `OrganizeError` 和 `OrganizeFailure`；用户取消不制造虚假失败。存储失败只作为当前操作安全错误上报，因为存储本身失败时不能声称已持久化分类。

### 解析兼容边界

继续保留 `DisallowUnknownFields`、来源 ID、题数、字段长度、图片索引和尾随内容校验。只允许同一响应的确定性本地规范化：去除首尾空白/UTF-8 BOM，以及解包“整个响应恰好是一个可选 `json` 标签的代码围栏”的情形；不从 prose、`<think>`、多个 JSON 或截断文本中猜取对象，不修补来源 ID 或图片索引，也不增加第二次模型调用。

### 重试与导出

- 保留 `OrganizeSession` 作为处理所有未结构化 completed 记录的兼容入口。
- 新增 `RetryFailedSession`，只选择 `Questions` 为空且已有失败标记的 completed 记录；成功项和纯 pending 项都不处理。
- 新增 Wails API `RetryFailedQuestionBankRecords(sessionID string)`，复用现有配置、异步 worker、取消和更新事件。
- `QuestionBankExportRequest` 新增 `AllowPartial bool`。部分导出必须由界面明确标记；普通完成态不需要该标记。
- App 层拒绝 `AutoOrganize=true && StructuredQuestions==0`；部分失败且 `AllowPartial=false` 也拒绝。
- Store 层在创建临时导出目录前再次拒绝 `!RawOnly && StructuredQuestions==0`，防止旧入口或未来调用者生成空整理稿。
- `RawOnly=true` 始终可用，只生成 `原始记录.md` 和 `assets/`。
- 最终确认仍只做本地导出，不再次调用模型。

---

### Task 1：建立 Change 024 事前治理档案

**Files：**
- Create: `_dev/changes/024-question-bank-organize-failure-recovery/00-request.md`～`06-notes.md`
- Create: `_dev/audits/024-question-bank-organize-failure-recovery.md`
- Modify: `_dev/changes/INDEX.md`

- [ ] **Step 1：再次只读核对分支、工作树、编号和原始 TCL 哈希。** 如工作树出现并非本任务产生的修改，停止对冲突文件操作并如实记录，不清理用户改动。
- [ ] **Step 2：创建并完整填写 `00-request.md`。** 固定目标、验收、非目标、数据保护、Git 授权边界和 `_02` 恢复语义。
- [ ] **Step 3：填写 `01-current-state.md`。** 记录已确认的状态误判链、严格 JSON 契约、25 条数据现状以及“provider 具体根因不可恢复”的证据边界。
- [ ] **Step 4：填写 `02-plan.md`。** 固化本计划的状态机、失败结构、重试选择器、导出保护、文件白名单、测试矩阵和回滚。
- [ ] **Step 5：初始化 `03`～`06` 和审计为未实施，登记 INDEX 为“进行中”。** 不预写 PASS；完成前三份文档后才进入业务代码。

### Task 2：以 TDD 建立失败分类、有限响应规范化和统一汇总

**Files：**
- Modify: `pkg/questionbank/types.go`
- Modify: `pkg/questionbank/organize.go`
- Create: `pkg/questionbank/organize_failure.go`
- Create: `pkg/questionbank/organize_failure_test.go`
- Modify: `pkg/questionbank/history_test.go`

- [ ] **Step 1：写失败测试。** 覆盖空响应、超长响应、非 JSON、额外正文、来源错配、题数/字段/图片索引错误、请求错误、超时、用户取消和旧 `OrganizeError`。
- [ ] **Step 2：运行聚焦测试确认 RED。** 预期失败点为尚无 typed failure、统一汇总和代码围栏兼容。
- [ ] **Step 3：实现固定分类和安全文案映射。** 不把底层 `err.Error()`、响应体或路径写入 Session；记录 provider 是否调用、模型、响应字节数和时间。
- [ ] **Step 4：实现严格优先的解析。** 先按原契约解析；仅对完整单一 JSON 围栏/BOM 做一次本地规范化后复用同一校验，不增加 provider 调用。
- [ ] **Step 5：实现 `SummarizeOrganizeState`。** 让 App、导出保护和测试使用同一事实源，避免前后端重复推导。
- [ ] **Step 6：运行聚焦测试转 GREEN。** 额外断言 fake 错误中的伪密钥、URL、绝对路径和响应体不出现在 Session、状态消息或测试快照中。

### Task 3：以 TDD 增加“只重试失败项”

**Files：**
- Modify: `pkg/questionbank/organize.go`
- Modify: `pkg/questionbank/store_test.go`
- Modify: `pkg/questionbank/history_test.go`

- [ ] **Step 1：建立混合 Session RED 测试。** 同时包含成功项、失败项和纯 pending 项。
- [ ] **Step 2：实现共享私有循环和两个选择器。** `OrganizeSession` 保持处理全部未结构化记录；`RetryFailedSession` 只处理失败项。
- [ ] **Step 3：验证调用与数据不变式。** 失败重试只调用一次 fake provider；成功 Questions 不变；pending 不被顺带处理；Answer、AnswerItems、Images、Record ID 不变；重试成功后清错，再次重试零调用。
- [ ] **Step 4：覆盖部分成功、再次失败、取消和保存失败。** 已完成结果保留，取消后不新增调用或虚假失败，保存失败不宣称成功。

### Task 4：以 TDD 阻止零题整理稿并保护部分导出

**Files：**
- Modify: `pkg/questionbank/export.go`
- Modify: `pkg/questionbank/segment_test.go`
- Modify: `app/questionbank.go`
- Modify: `app/questionbank_test.go`

- [ ] **Step 1：写 Store RED 测试。** 全失败且 `RawOnly=false` 必须在建目录前失败，不追加 Exports；同一 Session 显式 `RawOnly=true` 成功；部分成功保留“尚未结构化的提交”。
- [ ] **Step 2：实现 Store 最终保护。** 复用统一汇总统计，不再让 `renderExport` 的零题占位成为可正常交付的整理稿。
- [ ] **Step 3：写 App 状态表测试。** 覆盖 `pending/complete/partial/all_failed/no_eligible` 以及字段计数。
- [ ] **Step 4：更新最终导出校验。** pending 拒绝；零题拒绝；partial 必须 `AllowPartial=true`；完整成功正常导出；所有最终确认均零 provider 调用。
- [ ] **Step 5：修正后台完成文案。** 区分整理成功、部分失败、全部失败、取消、RawOnly 和本地导出，不再把实际启用过整理的失败场景描述成“未启用模型整理”。
- [ ] **Step 6：更新原有固化缺陷的测试。** 将 `TestQuestionBankOrganizedExportRequiresPreparationButAllowsRecordedFailure` 改为“全失败禁止整理稿、显式 RawOnly 仍成功”，保留最终本地导出不调用模型的断言。

### Task 5：接入 App/Wails 的失败重试 API

**Files：**
- Modify: `app/questionbank.go`
- Modify: `app/questionbank_test.go`
- Modify: `frontend/src/services/api.js`

- [ ] **Step 1：为 App 前置校验写 RED 测试。** 无失败项、busy/closing、无题库目录、无模型配置时拒绝且不改变 Session。
- [ ] **Step 2：实现 `RetryFailedQuestionBankRecords`。** 复用现有 target/config/worker/cancel/event 机制，不重新封存或创建导出目录。
- [ ] **Step 3：验证安全消息。** 完全恢复、仍部分失败、仍全部失败和取消均只显示白名单文案，不拼接第三方原始错误。
- [ ] **Step 4：接入前端 API 封装。** 保持现有 `PrepareQuestionBankSegment` 作为首次/继续准备入口，新方法只用于显式失败重试。

### Task 6：修复导出弹窗状态与可达操作

**Files：**
- Modify: `frontend/src/stores/questionbank.js`
- Modify: `frontend/src/components/QuestionBankExportModal.vue`
- Modify: `frontend/questionbank_history_regression_test.mjs`

- [ ] **Step 1：扩展 draft 归一化和 fake API fixture。** 加入 outcome/计数/AllowPartial/retry API。
- [ ] **Step 2：写前端 RED 回归。** 全失败不得显示“准备完成”；部分失败显示成功/失败/题数；显式重试只调用一次；RawOnly 不写长期偏好；正常两阶段流程不回归。
- [ ] **Step 3：实现全失败 UX。** 显示“全部整理失败”，提供“只重试失败记录”和“改为仅原稿”；不提供整理稿导出动作。
- [ ] **Step 4：实现部分失败 UX。** 提供“只重试失败记录”“导出已有整理结果”“改为仅原稿”；部分导出请求携带 `allowPartial=true`。
- [ ] **Step 5：实现 pending/complete/no_eligible UX。** pending 继续准备，complete 正常确认，no_eligible 只允许原稿；后台工作期间保持防重复和关闭语义。
- [ ] **Step 6：修复草稿刷新保持。** `reloadExportDraft()` 保留当前弹窗的 `autoOrganize` 选择、目录和人工编辑字段；“改为仅原稿”只影响当前请求，不调用偏好保存 API。
- [ ] **Step 7：运行 Node 回归转 GREEN。** 模板继续不用 `v-html`，人工命名和建议刷新行为保持。

### Task 7：生成 TCL `_02` 离线恢复稿

**Output：**
- Create: `输出/求职笔试/题库/2026-10-25_095652_TCL_综合笔试_02/TCL_综合笔试.md`
- Copy byte-for-byte: `原始记录.md`、`assets/`

- [ ] **Step 1：执行 fail-closed 预检。** `_02` 必须不存在；记录原目录、25 张 assets 和 Session 文件的 SHA-256、大小与数量，确认仍是 25 Records、25 answers、25 answerItems、25 screenshots、0 Questions。
- [ ] **Step 2：建立题号映射表。** 以截图可见的 `0/9`～`8/9` 映射为每组第 1～9 题；截图题干优先于原模型裸编号，提交顺序仅作辅助。
- [ ] **Step 3：处理已知错配。** 提交 6/7 的重复截图只恢复一个题位但保留两份原资料；提交 1/3/4 的截图与答案循环错配按截图内容重排；不改原始记录或 answerItems。
- [ ] **Step 4：创建目标并逐字节复制原稿和 assets。** 复制后逐文件哈希比对，不更新 Session 的 Exports 或 Questions。
- [ ] **Step 5：编写无图恢复稿。** 使用“语言推理-01～09、数字推理-01～09、图形推理-01～09”；前半 `## 试题`，后半 `## 参考答案与解析`，共 27 个推定题位和对应答案位。
- [ ] **Step 6：按证据等级恢复。** 18 道语言/数字题完整文字化；6 道有截图但依赖图形的题写“题图待补”并链接原始提交；图形第 4、6、8 题写“原始材料缺失”。
- [ ] **Step 7：标注答案风险。** 语言第 4 题待核对；数字第 5、7 题同时保留原模型结论和复核建议但不写成确定答案；所有图形题待核对；图形第 9 题明确记录截图疑似选择 D 与原模型 B 冲突。
- [ ] **Step 8：验证恢复稿。** 不嵌图片、不出现“官方/标准答案”断言；每题来源可追溯；旧目录与 Session 哈希完全不变。最终只称为“离线恢复稿/待人工核对”，不称为重新模型整理成功。

### Task 8：Bindings、文档和全量验证

**Files：**
- Regenerate as needed: `frontend/wailsjs/go/app/App.js`、`App.d.ts`、`frontend/wailsjs/go/models.ts`
- Modify: `README.md`、`README_EN.md`
- Update: Change 024 的 `03`～`06`、审计、INDEX

- [ ] **Step 1：更新用户文档。** 说明全部失败/部分失败、显式失败重试、RawOnly 和“最终确认不再次调用模型”；不声称已解决某个未经证实的 provider 根因。
- [ ] **Step 2：使用固定 Wails v2.11.0 生成必要 bindings。** 审计生成差异，确保既有 App API 没有丢失，`go.mod/go.sum/package*.json` 无依赖变化。
- [ ] **Step 3：运行聚焦验证。**

```text
GOTOOLCHAIN=go1.25.13 go test ./pkg/questionbank ./app -count=1
node frontend/questionbank_history_regression_test.mjs
```

- [ ] **Step 4：运行全量验证。**

```text
GOTOOLCHAIN=go1.25.13 go test ./... -count=1
GOTOOLCHAIN=go1.25.13 go vet ./...
GOTOOLCHAIN=go1.25.13 go build ./...
npm --prefix frontend run build
git diff --check
```

- [ ] **Step 5：执行 Wails 集成构建。** 使用独立 Change 024 验证程序名，不覆盖正式或旧验证程序；构建前后核对受保护文件哈希。
- [ ] **Step 6：按环境条件运行 race。** 若继续出现既有 Windows 启动问题，记录实际输出并标为 `BLOCKED`，不得写成 PASS。
- [ ] **Step 7：完成数据保护验证。** 比对 TCL 原目录与 Session 的预检/后检哈希；确认 `_02` 是唯一新增题库目录。
- [ ] **Step 8：回写治理记录。** 如实记录 RED/GREEN、测试、构建、未执行项、`BLOCKED` 和 `NEEDS_USER`；未经另行授权保持未提交状态。

## 人工验收矩阵

1. **全部失败：** 弹窗显示全部失败；不能导出整理稿；可只重试失败项或切为仅原稿。
2. **部分失败：** 明确显示成功/失败/题目数；重试只触及失败项；明确选择后可导出已有结果。
3. **全部成功：** 保持准备后人工核对、最终纯本地导出。
4. **取消后继续：** 已完成结果保留，pending 可继续，取消不新增失败记录。
5. **本次 RawOnly：** 只影响当前弹窗，不修改长期 `autoOrganize` 偏好。
6. **历史重导出：** 新目录使用 `_02/_03`，旧输出不变。
7. **API 配置调整后重试：** 可从正常 UI 只重试失败记录；成功项不重复消费额度。
8. **TCL 恢复稿：** 27 个推定题位、18 道文字化、6 道题图待补、3 道材料缺失，全部风险标记清楚。
9. **真实 provider：** 只在用户自行验收时执行，自动验证不调用；实际失败分类应能区分 input/request/response 阶段。

## 回滚

- 代码回滚仅定向撤销 Change 024 白名单中的源码、测试、前端、文档和生成 bindings；新增可选 JSON 字段无需迁移，旧代码可忽略。
- 不删除或清空任何重试后成功保存的 Questions、原 Answer、AnswerItems、截图、Session 或旧导出。
- TCL 原目录和 `_记录` 从未修改；默认保留 `_02` 作为待复核副本。若将来需要删除 `_02`，必须先确认其中没有后续人工内容并取得单独授权。
- 保留 Change 024 治理档案并记录回滚原因；不回退 Change 016～023。
