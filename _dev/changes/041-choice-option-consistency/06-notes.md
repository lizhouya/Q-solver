# 总结与注意事项 06

## 当前摘要

Change 041 已修复生成约束：选择题先识别原始字母与选项文本映射，再判断正确文本并回填字母；正文结论、解析和最终汇总必须一致，冲突或无法辨认时标记“待核对”。快速模式保持简洁但不跳过映射复核。普通自动门禁与用户模型复验通过，叠加候选已晋升本地稳定版。race 环境 `BLOCKED` 已披露，Git 提交已授权，未发布。

## 长期约束

- 选择题字母不能脱离当前截图中的选项文本独立猜测。
- 解析器没有原始图像语义，不得用关键词擅自修正字母。
- 映射不可靠或字母、文本、解析、汇总冲突时标记“待核对”。
- 快速模式可以简短，但不能跳过内部映射和一致性检查。
- 提示词约束降低错误概率，但不等于绝对保证；字符串测试不能冒充真实模型正确率证明。

## 稳定版与归档

- 根稳定版：`build/bin/Q-Solver.exe`
- 大小：42,558,464 字节
- SHA-256：`2885358371510B827ACE560C354FEE8775BECEB47555D7CE654B0FBE3EB1F15B`
- 验证归档：`build/bin/archive/validation/change-041/Q-Solver-candidate-change041.exe`
- 被替代稳定版：`build/bin/archive/superseded/change-041/Q-Solver-stable-before-change041.exe`
- 被替代稳定版 SHA-256：`DC8B0E248B82972F92A6EC45152F0D9B4F93450DC335A61185476129D0D357EC`
- `testing/current/` 已清空；7 个 Release 文件未改变。

## 授权状态

用户已授权本地稳定版更新和 Git 提交。功能提交 `ac8af04` 已创建，治理提交随本档案完成；Change 041 与 Change 040 保持独立提交。push、Tag、Release 未授权。

## 参考资源

- `_dev/audits/041-choice-option-consistency.md`
- `pkg/domain/prompts.go`
- `pkg/domain/prompts_test.go`
- `pkg/answersummary/parser.go`
