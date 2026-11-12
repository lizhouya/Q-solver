# Change 041 选择题选项一致性审计

## 结论

示例错误由模型响应自身产生：解释正确指出 Java 8 使用 Metaspace，但模型把该文本错误映射为 C，并在末尾汇总继续填 C。前端、Mobile 和 QuestionBank 只是显示结构化结果；`pkg/answersummary` 只校验单选值为合法字母，既没有原始截图，也无法知道“元空间”在本题对应 A。

本次不在解析层按关键词改字母，而是在生成约束中强制“映射优先”：先识别截图原始字母与选项文本，再判断正确文本，最后回填该字母；正文结论同时显示字母和文本，汇总前复核字母、文本、解析和表格一致。映射不可辨认或冲突时必须标记“待核对”。

## 调用链

```text
截图
  → solution.buildSystemPrompt
  → 全局 QuestionAnsweringRules + ScenePrompt
  → Provider 生成正文和最终答案汇总
  → questionSolveCapture.Emit(solution)
  → answersummary.Parse（只做结构/规范校验）
  → Desktop / Mobile / QuestionBank 展示
```

## 实施边界

- 全局三条规则覆盖普通、逐步和其他解题场景。
- `gen-speed` 内部建立选项映射并复核一致性；对外保持简洁的字母＋文本，不输出隐藏推理过程。
- 表格 schema 继续只保存纯字母或“待核对”。
- 不修改解析器、数据 schema、前端、Provider 请求次数或正式配置。
- Change 041 功能文件仅为 `pkg/domain/prompts.go` 与 `pkg/domain/prompts_test.go`。

## 验证结果

- TDD 红灯证明旧提示缺少全部目标规则，最小实现后聚焦契约转绿。
- `pkg/domain`、19 个前端回归、Vite、全量 Go、vet、build、mod verify 和 diagnostics 通过。
- race 在 Windows 启动阶段返回 `0xc0000139`，状态 `BLOCKED`。
- 固定 Wails v2.11.0 叠加候选 SHA-256 为 `2885358371510B827ACE560C354FEE8775BECEB47555D7CE654B0FBE3EB1F15B`；831 个保护文件、175 个构建前产物、根稳定版和 Release 均未改变。
- 用户于 2026-11-12 确认实际模型复验通过；叠加候选已晋升本地稳定版，根 SHA-256 为 `2885358371510B827ACE560C354FEE8775BECEB47555D7CE654B0FBE3EB1F15B`，旧稳定版与验证候选完成双归档，Release 未改变。
- race 环境 `BLOCKED` 继续披露；用户已授权本地稳定版更新和 Git 提交，未授权 push、Tag 或 Release。

## 提交边界

功能提交候选范围仅为：

- `pkg/domain/prompts.go`
- `pkg/domain/prompts_test.go`

治理提交范围仅为本审计、Change 041 七份档案和 INDEX 中的 041 状态。功能提交 `ac8af04` 已创建，治理提交随本档案完成。Change 040 的业务与治理文件未混入本 Change 提交。用户未授权 push、Tag 或 Release。
