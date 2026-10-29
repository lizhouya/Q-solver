# 变更请求 00

## 状态

已批准 / 待实施。

## 背景

当前题库整理稿使用 Markdown：先集中列出全部试题，再在文件后半部分集中列出全部参考答案与解析。题目与答案距离较远，复习时需要反复跳转。

## 用户目标

- 继续保留 Markdown 格式。
- `原始记录.md` 继续完整保存截图、原答案及失败/取消记录。
- 整理稿改为固定逐题解析：每道题的参考答案、知识点、解析要点、来源和核对说明紧跟在该题下面。
- 优化层级和分隔，使长题干、代码、公式与多段解析保持可读。

## 已批准设计

- 不增加布局选择器，也不同时生成多套整理稿。
- 保持现有两个文件契约：`原始记录.md` 与 `<题库名称>.md`。
- 整理稿不再生成末尾独立的“参考答案与解析”章节，而是在每道题后生成“参考答案与解析”子节。
- 继续按分类、题型和全局连续题号组织试题。

## 非目标

- 不实现 Markdown 表格左右两栏。
- 不拆分为题目文件和答案文件。
- 不修改模型整理 Prompt、JSON schema、题库 Session/Record/Question 数据结构。
- 不新增模型调用、依赖、Wails API 或 bindings。
- 不改写既有导出目录，不覆盖历史文件。
- 不提交、不推送、不创建 Tag 或 Release。

## 允许修改范围

- `pkg/questionbank/export.go`
- `pkg/questionbank/segment_test.go`
- `frontend/src/stores/questionbank.js`
- `frontend/src/components/QuestionBankExportModal.vue`
- `frontend/questionbank_history_regression_test.mjs`
- `_dev/changes/031-inline-question-explanations/**`
- `_dev/changes/INDEX.md`

## 验收标准

1. 每道题的答案与解析位于该题之后、下一道题之前。
2. 不再输出末尾重复的全局“参考答案与解析”章节。
3. 答案、知识点或解析为空时继续安全回退为“待核对”。
4. 来源链接和核对说明保留。
5. `原始记录.md` 的布局、安全围栏和截图引用不变。
6. `RawOnly`、文件命名、新版本目录、ExportResult 字段保持兼容。
7. 不增加模型调用、持久化字段、Wails API 或依赖。
8. 聚焦测试、全量 Go 测试/构建、前端回归/构建及独立 Wails 验证构建如实记录。
