# 实施方案 02

## 实现步骤

1. 在 `pkg/questionbank/segment_test.go` 先更新/补充排版契约测试，使旧的“题答分区”实现失败：
   - 同一题的题干、答案、知识点、解析、来源和核对说明必须连续；
   - 第一题解析必须出现在第二题题干之前；
   - 不得再输出全局 `## 参考答案与解析`；
   - 空字段继续回退为“待核对”。
2. 在 `pkg/questionbank/export.go` 将整理稿改为单次有序渲染：每输出一道题，立即输出该题的答案与解析子节和分隔线。
3. 保持分类、题型、全局题号、来源锚点、未结构化提交列表和原始记录渲染不变。
4. 更新 `frontend/src/stores/questionbank.js` 的预计输出文案，以及 `QuestionBankExportModal.vue` 对整理稿布局的说明。
5. 在 `frontend/questionbank_history_regression_test.mjs` 增加静态 UI 契约断言。
6. 完成聚焦、全量、前端和 Wails 构建验证，回写 03～06 与 INDEX。

## 预期 Markdown 层级

```text
# 题库名称
## 试题与解析
### 分类
#### 题型
##### 第 N 题：标题
题干与选项
###### 参考答案与解析
参考答案 / 知识点 / 解析要点 / 来源 / 核对说明
---
```

为保持现有分类层级与全局编号，题目标题继续使用五级标题，题内答案子节使用六级标题；不使用 HTML、Markdown 表格或折叠标签。

## 风险与处理

- **重复渲染或漏题**：移除第二次答案遍历，测试题号和顺序。
- **空字段产生空白块**：统一输出“待核对”。
- **长代码/公式被表格破坏**：继续使用普通 Markdown 块，不使用左右栏表格。
- **前端误导**：同步“预计输出”和弹窗说明，但不新增选择控件。
- **兼容破坏**：不改文件名、ExportResult、API、数据结构、原始记录或目录策略。
- **生成噪音**：Wails 构建使用唯一文件名及保护参数，构建前后核对关键文件哈希。

## 测试计划

- 先运行聚焦测试并确认旧实现因新契约失败。
- 修改后运行：
  - `go test ./pkg/questionbank -run 'TestNamedPaper|TestInline' -count=1`
  - `go test ./pkg/questionbank ./app -count=1`
  - `go test ./... -count=1`
  - `go build ./...`
  - `node frontend/questionbank_history_regression_test.mjs`
  - `npm --prefix frontend run build`
  - 受控、唯一命名的 Wails 生产验证构建。
- 对比 `wails.json`、Go 依赖、前端锁文件与 Wails bindings/runtime 哈希。

## 回滚方案

只定向还原本 Change 白名单内的代码、测试、UI 文案和治理记录；不使用 `git reset --hard`、`git clean`、`checkout` 或批量恢复，不影响既有未提交修改及历史导出目录。
