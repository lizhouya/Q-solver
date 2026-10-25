# Change 021 备注

日期：2026-10-24
状态：普通自动验证、独立构建与归档完成；GUI `NEEDS_USER`，race `BLOCKED`，未提交、未推送、未发布、未关闭。

## 交付结果

- 主界面答案操作栏新增“开始下一套”。
- Windows 默认快捷键为 `Ctrl+Shift+N`，可在“设置 → 常规”重新录制。
- 按钮与快捷键共用同一单例确认层；快捷键本身不执行清理。
- 确认成功后安全封存当前段，清理待发送截图、桌面历史、流式目标和 Solver 上下文。
- 题库 Session/Record、归档截图、完整答案、旧导出与 Mobile 当前显示保持不变。
- 新段不预建；下一次正式解题才懒创建。
- 解题、capture、整理、导出、未落盘、pending completion 或 write blocked 时拒绝且不清理现场。

## 验证入口

推荐使用统一归档程序：

```text
build/bin/archive/validation/Q-Solver-change021-start-next-exam-validation.exe
```

SHA-256：

```text
95f4b3c17b96db0e25fa682ed337bb067458046784a09d3580d1eff4839e3c4c
```

构建输出目录中的同名程序保留；正式 `Q-Solver.exe` 和旧验证程序未被覆盖。

## 操作说明

1. 一套题完成且后台任务全部结束后，点击“开始下一套”或按 `Ctrl+Shift+N`。
2. 阅读确认层中的清理与保留范围，再点击确认。
3. 界面回到待记录状态后，下一次截图并发送会建立新的题库分段。
4. 本操作不会清空同步端；如需同步端空白，另用现有 `Shift+C` 并完成清屏确认。

## 已知边界

- 不根据导出状态自动触发，因为现有异步导出没有可安全依赖的结构化成功终态。
- exported 仅表示旧段已导出；再次开始下一套不会删除或覆盖旧导出。
- 新 action 的默认组合若已被旧 action 占用，不抢占，设置页显示为未绑定。
- Wails bindings 本次正常生成时同步了 Change 016～018 的累计接口；这是当前工作区真实后端接口的生成结果。
- Vite 大 chunk warning 仍存在，但构建成功，本 Change 不扩大为性能重构。
- 独立只读审查受 API 429 限流提前终止，不记为独立审查 PASS。
- race 因 Windows `0xc0000139` 无法启动，记录为 `BLOCKED`，不是 PASS。

## 待完成

- 按 `04-validation.md` 使用归档验证程序完成 Windows GUI 人工验收。
- 在能够运行 race 的环境补做 race 验证。
- 未获得 commit、push、Tag 或 Release 授权，因此没有执行相关操作。

## 关闭边界

任何 GUI `NEEDS_USER`、race `BLOCKED` 或未授权提交状态仍存在时，Change 021 不得标记“已完成”或“已关闭”。

## 2026-10-25 整合提交状态（最新）

用户明确批准本次将 Change 016～023 累积改动按功能/测试与治理两次提交整合，并连同既有 RC2 提交推送至 origin/dev/qsolver-custom；此授权只适用于本次。

- 功能/测试已提交：`e44bb6163f0cb1b88e2ec4b4da80a1446ddb1990`。
- 本治理档案随第二次整合提交归档；完整边界见 `../../audits/016-023-commit-boundary.md`。
- 原文“未提交/未推送”是当时记录，最新 Git 状态以分支历史和远端分支头为准；本次不创建 Tag 或 Release。
- 不扩大人工验收结论：用户明确确认 Change 016、023；其他未明确项目和既有 race BLOCKED 保留，不因提交自动关闭。
