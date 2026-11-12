# 实现过程记录 03

## 当前状态

映射优先、四方一致性、冲突待核对和快速模式约束已实现；普通自动门禁与用户实际模型复验通过。固定 Wails v2.11.0 叠加候选已晋升根稳定版，旧稳定版与验证候选完成双归档。race 环境 `BLOCKED` 已披露；功能提交 `ac8af04` 已创建，治理提交随本档案完成。未推送、未发布。

## 实现日志

| 时间 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-11-12 | 只读根因审计 | 确认错误字母来自模型响应本身，前端与解析器未篡改 | prompts、solution、answersummary、capture | 解析器缺少原始选项映射，不能安全语义修复 |
| 2026-11-12 | 方案确认与事前归档 | 用户批准映射优先、四方一致性、待核对和快速模式约束 | Change 041 | 不增加二次 API 调用 |
| 2026-11-12 | TDD 红灯 | 新增全局和快速模式契约，旧提示缺少全部目标片段 | `pkg/domain/prompts_test.go` | 失败只指向缺失规则 |
| 2026-11-12 | 全局约束 | 增加先映射选项文本、再判断、后回填字母；正文、解析与汇总一致；冲突待核对 | `pkg/domain/prompts.go` | 所有解题场景继承 |
| 2026-11-12 | 快速模式 | 内部建立映射和一致性复核，外部输出简洁的字母＋选项文本 | `pkg/domain/prompts.go` | 不暴露隐藏推理过程 |
| 2026-11-12 | 聚焦与全量门禁 | 聚焦、`pkg/domain`、19 个前端回归、Vite、全量 Go、vet、build、mod verify、diagnostics 通过 | 全仓相关范围 | 无依赖或 schema 变化 |
| 2026-11-12 | race | `pkg/domain` 启动阶段返回 Windows `0xc0000139` | race | `BLOCKED`，未进入测试体 |
| 2026-11-12 | 候选轮换 | Change 040 候选按原哈希归档 transient，保留测试前稳定版 | ignored 产物 | current 只保留唯一最新候选 |
| 2026-11-12 | 保护构建 | 固定 Wails v2.11.0 构建 Change 041 叠加候选；831 个保护文件和 175 个构建前产物不变 | `build/bin/testing/current/` | 根稳定版与 Release 未变 |
| 2026-11-12 | 启动前检查 | 检测到用户正在运行 Change 040 候选，未停止用户进程 | 候选运行 | Change 041 独立启动 `BLOCKED` |
| 2026-11-12 | 用户验收 | 用户确认实际选择题字母、选项文本、解析和汇总一致性验证通过 | Change 041 候选 | 模型复验 PASS |
| 2026-11-12 | 稳定版晋升 | 候选替换根 EXE，旧稳定版和验证候选双归档，测试区清空 | `build/bin/**` | 根 SHA-256 `288535…F15B`；Release 7 个文件不变 |

## 与计划的偏差

- 为保持 `testing/current/` 唯一候选，Change 040 候选在构建 Change 041 前归档 transient；Change 041 候选包含两项未提交工作树改动，但 Git 功能和治理边界保持独立。
- PowerShell 5.1 对 JSON 顶层数组按嵌套对象返回，前两次保护比较误报全部文件。未恢复或删除任何受保护文件；改用 Path/Size/SHA256 按索引展开后，证明 831 个保护文件和 175 个构建前产物均未变化。
- 独立启动未执行，因为用户当前正在运行 Change 040 候选，不能擅自终止。

## 关键决策纪要

- 不按知识关键词自动改字母。
- 不修改结构化答案 schema、解析器或显示层。
- 全局规则覆盖所有解题场景，快速模式单独避免“只猜字母”。
- 提示只要求输出可见结论和必要简短逻辑，不要求暴露隐藏推理过程。

## 稳定版与归档

- 根稳定版：`build/bin/Q-Solver.exe`
- 大小：42,558,464 字节
- SHA-256：`2885358371510B827ACE560C354FEE8775BECEB47555D7CE654B0FBE3EB1F15B`
- 验证归档：`build/bin/archive/validation/change-041/Q-Solver-candidate-change041.exe`，哈希与根稳定版一致。
- 被替代稳定版：`build/bin/archive/superseded/change-041/Q-Solver-stable-before-change041.exe`，SHA-256 `DC8B0E248B82972F92A6EC45152F0D9B4F93450DC335A61185476129D0D357EC`。
- Wails runtime：v2.11.0；`testing/current/` 已清空，7 个 Release 文件未改变。
- 稳定版同时包含 Change 040 Mobile 提示修复；Git 提交仍按各自白名单拆分。

## 遗留未完成项

- Git 功能与治理提交已授权，尚未执行；push、Tag、Release 未授权。
