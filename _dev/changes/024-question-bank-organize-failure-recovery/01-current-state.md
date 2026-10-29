# Change 024 现状快照：题库整理失败恢复

日期：2026-10-25

## 仓库与数据基线

- 分支：`dev/qsolver-custom`
- HEAD：`43022e1`，与 `origin/dev/qsolver-custom` 一致。
- 事前工作区：clean。
- 下一可用编号：024。
- 原导出：`输出/求职笔试/题库/2026-10-25_095652_TCL_综合笔试`。
- Session：`a6ef78f5c602e1950b991e3d`。
- 现状：25 Records、25 answers、25 answerItems、25 screenshots、25 organizeError、0 Questions。
- 原 `原始记录.md` SHA-256：`b938d2e732f1a948effb608d4d3453667fa720b3496b0da80050536992779fe8`。
- 原 Session SHA-256：`f93e663bf9c45c3f8de156ac84a18dd25239f632c3c62534c6a010369cba3c83`。

## 已确认的故障链

1. `pkg/questionbank/organize.go` 对请求或解析失败统一写入 `OrganizeError`，具体阶段丢失。
2. `app/questionbank.go` 的状态计算把有 `OrganizeError` 的无 Questions 记录计入 failed，而不是 pending。
3. `QuestionBankExportDraft.OrganizeReady` 仅为 `pending == 0`。
4. 本次得到 `pending=0, failed=25, questions=0, organizeReady=true`。
5. 前端因此显示“准备完成”，最终提交直接进入纯本地导出。
6. `pkg/questionbank/export.go` 对零题仍写出“暂无结构化整理结果”的整理稿。

## 不能确认的事项

现有数据没有保存 provider 原始错误、HTTP 状态、空响应标记、解析阶段或无效响应。无法区分请求失败、超时、空 Choices/Content、非流式兼容、代码围栏、额外字段、来源错配或 schema 校验失败。旧成功 Session 曾使用同名 `deepseek-flash` 完成整理，因此不能断言该模型必然不支持整理。

## 可复用现有能力

- `OrganizeSession` 已逐 Record 保存，并跳过已有 Questions 的成功项。
- `PrepareQuestionBankSegment` 和 `ExportQuestionBankSegment` 已形成两阶段流程。
- `ExportSession` 已使用临时目录 + rename，并通过 `_02/_03` 避免覆盖。
- `RawOnly` 已支持只导出原稿和截图。
- 导出稿已按“试题 → 参考答案与解析 → 尚未结构化提交”组织。

## 当前 UI 缺口

- 全部失败仍显示“准备完成”。
- 没有可达的“只重试失败记录”动作。
- `reloadExportDraft()` 不保留本次 `autoOrganize` 选择。
- 最终状态文案可能把实际整理失败描述成“未启用模型整理”。

## TCL 离线核查结论

原卷高概率为语言、数字、图形三组各 9 题，共 27 个推定题位。18 道语言/数字题可文字化；6 道图形题有截图但不能无损转为纯文字；图形第 4、6、8 题材料缺失。提交 6/7 重复；提交 1/3/4 存在截图与答案错配；截图 `0/9`～`8/9` 应映射为第 1～9 题。数字第 5、7 题和全部图形题需要明确待核对。
