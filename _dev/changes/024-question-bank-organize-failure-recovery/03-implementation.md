# Change 024 实现记录

日期：2026-10-25
状态：业务实现、固定 Wails v2.11.0 bindings、独立验证构建与普通自动验证完成；GUI 与真实 provider 待人工验收；未提交、未推送、未发布。

## 1. 题库层

### 1.1 统一整理结果状态

新增 `pkg/questionbank/organize_failure.go`，由 `SummarizeOrganizeState` 对 completed Record 统一统计：

- `pending`：仍有未结构化、且没有失败标记的记录；
- `complete`：已有结构化题目，且无 pending/failed；
- `partial`：已有结构化题目，无 pending，但仍有 failed；
- `all_failed`：存在可整理记录，零结构化题，无 pending，且有 failed；
- `no_eligible`：没有 completed Record。

App 草稿、后台完成消息、最终导出校验和 Store 保护均使用同一汇总语义，不再用 `pending == 0` 推导“准备完成”。

### 1.2 脱敏失败分类

`Record` 新增可选 `OrganizeFailure`，不提升 Session version。持久化内容仅包含稳定错误码、阶段、是否可重试、是否调用 provider、尝试时间、模型名和响应字节数。分类覆盖：

`input_invalid`、`request_timeout`、`request_failed`、`empty_response`、`response_too_large`、`invalid_json`、`extra_content`、`source_mismatch`、`schema_invalid`、`legacy_unknown`。

底层 provider 错误、无效响应正文、API Key、Token、Authorization、完整 BaseURL 和绝对路径均不写入 Session 或用户消息。旧记录只有 `OrganizeError` 时仍作为可重试失败项处理。

### 1.3 严格解析与有限兼容

保留严格 JSON schema、未知字段拒绝、来源 ID、题数、必填字段和图片索引校验。仅新增以下确定性本地规范化：

- 去除首尾空白；
- 去除 UTF-8 BOM；
- 当整个响应恰好是单一代码围栏时，解包无语言标签或 `json` 标签的内容。

仍拒绝 prose + JSON、`<think>` + JSON、围栏后的补充说明、多个 JSON 对象、截断 JSON 和自动修补字段。响应仍限制为 512 KiB。

### 1.4 只重试失败记录

`Store.RetryFailedSession` 与首次整理复用同一私有循环，但只选择 completed、零 Questions 且带失败标记的 Record。成功项和纯 pending 项不处理；重试成功后同时清空 `OrganizeError` 和 `OrganizeFailure`。

### 1.5 Store 零题保护

`Store.Export` 在创建导出目录、临时目录和导出快照之前拒绝 `RawOnly=false` 且结构化题目数为零的请求。`RawOnly=true` 继续只导出 `原始记录.md` 与 `assets/`。

## 2. App 层

`QuestionBankExportDraft` 增加 outcome 与整理计数字段；`QuestionBankExportRequest` 增加 `AllowPartial`。最终导出现在执行以下校验：

- pending：拒绝最终导出；
- 零结构化题：拒绝整理稿；
- partial 且未显式 `AllowPartial=true`：拒绝；
- complete：正常本地导出；
- RawOnly：不受整理状态阻塞。

新增公开入口 `RetryFailedQuestionBankRecords(sessionID string)`，复用既有配置、异步 worker、取消与状态事件。它拒绝无失败项、organizing、closing 和缺少模型配置等前置状态，并只调用失败重试路径。

后台完成消息区分完整成功、部分成功、全部失败和取消，不再把全失败显示为“准备完成”。

## 3. 前端

- API service 新增 `retryFailedQuestionBankRecords`；
- Pinia draft 增加 outcome 与计数归一化，并保留旧 draft 的兼容推导；
- `partial` 普通提交被拒绝，只有“导出已有整理结果”发送 `allowPartial=true`；
- `all_failed` 不提供整理稿导出，只提供“只重试失败记录”和“改为仅原稿”；
- `no_eligible` 只能改为仅原稿；
- “改为仅原稿”只修改本次弹窗，不写长期偏好；
- 草稿刷新保留弹窗内的 `autoOrganize`、临时目录和人工编辑元数据；
- 最终确认仍是纯本地导出，不再次调用模型。

## 4. 测试调整

新增题库层解析、失败分类、状态汇总、失败重试、脱敏和零题导出保护测试。调整旧 Store/App 测试中把零题整理稿当作正常前置的断言：只验证原稿、assets 或生命周期的场景改用 `RawOnly=true`；真正验证整理稿的场景先创建合法 Question。

App 公开重试入口使用临时 `APPDATA` 和本地 `httptest` OpenAI-compatible 端点验证异步 worker。测试端点只返回合成 JSON，不读取用户真实配置、不连接真实 provider、不上传截图。

## 5. TCL 离线恢复

创建独立目录：

`E:\DATA\求职_AI\输出\求职笔试\题库\2026-10-25_095652_TCL_综合笔试_02`

交付内容：

- `TCL_综合笔试.md`：27 个推定题位，前半试题、后半参考答案与解析；
- `原始记录.md`：从旧目录逐字节复制；
- `assets/`：25 张截图逐文件复制。

恢复稿包含语言 9 题、数字 9 题、图形 9 题；18 道语言/数字题已文字化，6 道图形题标记“题图待补”，图形第 4、6、8 题标记“原始材料缺失”。数字第 5、7 题和图形第 9 题保留冲突并标记待核对，所有图形题均不表述为确定答案。

视觉复核修正了事前推测：提交 1、3、4 的截图与回答可以对应；实际明确错位为提交 5/6，提交 7 与提交 5 是重复截图。恢复稿按实际证据记录，旧原稿和 Session 未修改。

## 6. 实施中的问题与处理

- 长 Bash/Python heredoc 在 Windows 命令长度下被截断，解析阶段失败且未创建目标目录；后续改为短命令复制、文件工具写稿和独立哈希验证。
- Go 测试源码曾误嵌真实 BOM，编译报 `invalid BOM in the middle of the file`；改为 Go 转义 `\uFEFF` 后通过。
- Store 零题保护暴露 8 个固化旧缺陷语义的测试；按测试真实目标分别改为 RawOnly 或构造结构化题。
- PATH 中 Wails CLI 为 v2.16.0，未使用。首次精确运行 v2.11.0 和从缓存源码编译临时 CLI 被权限分类器拒绝；用户随后明确授权后，`go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0` 核验为 v2.11.0，并成功生成 bindings 与独立验证 EXE。Wails 生成的 `models.ts` 空行带有尾随制表符，定向清理后 `git diff --check` 通过，实际模型差异只保留新 DTO 字段。

## 7. 未执行的外部操作

- 未调用真实模型；
- 未读取或修改用户真实 API 配置、密钥或 Mobile Token；
- 未覆盖正式或历史验证程序；
- 未提交、未推送、未创建 Tag、未创建 Release。
