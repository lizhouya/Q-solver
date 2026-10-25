# Change 018 总结与注意事项

状态：业务实现与普通自动验证完成；Windows GUI NEEDS_USER；race BLOCKED；未提交、未推送、未发布。

## 已完成

- 最近 20 个 `ended`、`exported`、`paused` 分段可从题库面板真正续接到截图解题主界面。
- 当前 active 在切换时暂停并保留；`discarded` 永久只读；更早分段仅隐藏并保留。
- Solver 使用 replace 而非 append；后续请求使用当前配置，旧 Model/Scene 仅展示。
- 新记录持久化非 system turn 引用；旧记录按 completed Record 重建并显示固定警告。
- 主界面按 Record 显示 1～3 张截图、完整答案和状态；桌面流按 `sessionId + recordId + epoch` 隔离。
- Store 双文件激活具备故障注入、补偿和补偿失败写阻断；只读访问仍可保留现场。
- 题库完成落盘失败会回滚 Solver/refs，并通过“重试保存”原子恢复 Record、ConversationSnapshot、Solver 和 refs。
- 应用重启恢复 persisted/legacy 当前上下文；损坏引用 fail closed。
- exported 分段再次导出保留旧目录并创建后缀版本。
- 正式 bindings 和独立验证 EXE 已生成。
- GUI 截图反馈的按钮挤压已回修：桌面固定右侧操作列、按钮纵向等宽、卡片信息上下排列，窄屏使用全宽单列；业务状态与交互不变。

## 必须保持的语义

- 旧数据是按题库记录重建，不是原始模型请求的无损恢复。
- 历史 Answer 只能作为 assistant，截图只能作为 user image；不得提升为 system。
- cancelled goroutine 真正退出前不得切换，不能只看 Running=false。
- conversation epoch 与 Mobile display epoch 独立。
- 模型元数据仍只是待核对建议，未经人工确认不得成为权威字段。
- 不保存 API Key、Token、完整 Config、Prompt、ResumeContent 或 thinking。
- 写阻断表示跨文件补偿失败；此时不得继续修改题库，应停止操作并保留现场。

## 验证入口与限制

- 最新 GUI：`build/bin/archive/validation/Q-Solver-change018-layout-validation.exe`
- SHA-256：`853e64c3ccd59101317d0fc4e92414a10f1d12695deed9694b0d689ee2e07343`
- 原续接验证程序继续保留，不覆盖。
- 普通自动验证 PASS；race 仍因 Windows `0xc0000139` BLOCKED。
- Windows GUI 尚未人工验收，因此 Change 018 保持未关闭。
- 本 Change 未获得 commit、push、Tag 或 Release 授权。

## 2026-10-25 整合提交状态（最新）

用户明确批准本次将 Change 016～023 累积改动按功能/测试与治理两次提交整合，并连同既有 RC2 提交推送至 origin/dev/qsolver-custom；此授权只适用于本次。

- 功能/测试已提交：`e44bb6163f0cb1b88e2ec4b4da80a1446ddb1990`。
- 本治理档案随第二次整合提交归档；完整边界见 `../../audits/016-023-commit-boundary.md`。
- 原文“未提交/未推送”是当时记录，最新 Git 状态以分支历史和远端分支头为准；本次不创建 Tag 或 Release。
- 不扩大人工验收结论：用户明确确认 Change 016、023；其他未明确项目和既有 race BLOCKED 保留，不因提交自动关闭。
