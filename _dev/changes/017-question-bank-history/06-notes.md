# 总结与注意事项 06

状态：业务实现与普通自动验证完成；Windows GUI 人工验收待执行；race BLOCKED；未提交、未推送、未发布。

## 已达成结论

- “清空本段”成功后，桌面实时答案历史回到干净待记录状态；题库原始记录、截图、旧导出和模型上下文继续保留。
- 最近 20 只是 UI/API 视图，不是数据保留策略。第 21 段及更早段仍保留在磁盘，不触发模型、导出或删除。
- 历史详情是独立只读状态，未写入 `solution.history`，未调用 `Store.Open`，不会切换后台 current。
- 模型输出只称为“模型建议/待核对”；只有最终人工确认才写入 Session 权威字段。
- 已有 Questions 的旧记录不会为补元数据再次调用模型；最终确认导出也不会再次调用模型。
- bindings 由固定 Wails v2.11.0 生成，Change 016 的 `ClearMobileDisplay`、`ResetMobileOutputLink`、`SetMobileStableLink` 仍保留。

## 验证程序

- Windows GUI 验收入口：`build/bin/archive/validation/Q-Solver-change017-history-view-fix-validation.exe`；构建原件同时保留在 `build/bin/Q-Solver-change017-validation.exe`。
- 该文件独立于 `build/bin/Q-Solver.exe`，构建和归档过程均未覆盖原程序；统一验证目录的 `SHA256SUMS` 已更新并完整校验通过。
- 最终大小和 SHA-256 记录在 `04-validation.md`，以最后一次源码复验后的构建结果为准。

## 尚未关闭的验证项

- Windows GUI 人工验收尚未执行，因此 Change 017 不能标记为关闭或发布完成。
- Windows race 启动阶段仍返回 `0xc0000139`，状态保持 BLOCKED；普通测试、`go vet` 和普通构建不能替代 race。
- 符号链接拒绝测试在当前 Windows 账户下因缺少创建符号链接权限而 SKIP；生产代码的符号链接拒绝逻辑未放宽，其他路径、文件类型、大小和像素边界均已执行。

## GUI 验收重点

1. 清空成功后实时历史和答案区归零，取消或后端失败时保持不变。
2. 清空后的下一题懒创建新段，不产生零记录段。
3. ended、exported、discarded 的详情与按需截图正确，浏览不切换后台 current。
4. A→B 快速切换不串答案或截图，关闭面板后迟到响应不重现。
5. 第 20 段可见，第 21 段只留磁盘且不触发额外操作。
6. RawOnly 只生成原稿；模型模式先生成建议，人工核对后才纯本地导出。
7. 整理过程中此前输入的人工字段和手工名称不被刷新覆盖。
8. Change 016 的固定同步链接、清屏确认、左 `Shift+C` 和同步端行为不回退。

## 发布边界

Change 017 没有 commit、push、Tag 或 Release 授权；这些动作必须另行获得用户明确授权。

## 2026-10-25 整合提交状态（最新）

用户明确批准本次将 Change 016～023 累积改动按功能/测试与治理两次提交整合，并连同既有 RC2 提交推送至 origin/dev/qsolver-custom；此授权只适用于本次。

- 功能/测试已提交：`e44bb6163f0cb1b88e2ec4b4da80a1446ddb1990`。
- 本治理档案随第二次整合提交归档；完整边界见 `../../audits/016-023-commit-boundary.md`。
- 原文“未提交/未推送”是当时记录，最新 Git 状态以分支历史和远端分支头为准；本次不创建 Tag 或 Release。
- 不扩大人工验收结论：用户明确确认 Change 016、023；其他未明确项目和既有 race BLOCKED 保留，不因提交自动关闭。
