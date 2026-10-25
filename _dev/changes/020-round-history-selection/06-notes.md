# Change 020 备注

日期：2026-10-23
状态：普通自动验证与验证程序归档完成；GUI NEEDS_USER，race BLOCKED，未提交、未推送、未发布、未关闭。

## 交付结果

- 左侧“一条历史记录”现在代表一个提交/Record，而不是整个 Session。
- 点击条目后右侧只显示该轮；新请求开始时自动选中新轮。
- 从题库续接多 Record Session 时，全部 Records 按轮显示，默认选择最新一轮。
- 用户浏览目标与 typed/legacy 后台流写入目标已经分离；浏览旧轮不会重定向或污染正在生成的新轮。
- 删除、图片导出、暂停、封存、KeepContext、ConversationSnapshot 和续接仍以父 Session 为生命周期边界。

## 验证入口

推荐使用统一归档中的程序：

```text
build/bin/archive/validation/Q-Solver-change020-round-history-selection-validation.exe
```

SHA-256：

```text
3ea25ab19009d373c304f440c126bfeb18ce7c0d4c218d8e9be8204abdaa75bf
```

构建输出目录中的同名源程序保留，正式 `build/bin/Q-Solver.exe` 未被覆盖。

## 已知边界

- 每条历史卡片展示的是单轮内容，但卡片菜单仍删除/导出整个父 Session；文案已明确为“此会话”。
- “第 N 轮”按 Record 在父 Session 中的原始顺序编号；左栏展示顺序为每个 Session 内最新轮优先。
- legacy `localId` 仅用于当前桌面运行期的稳定选择，不持久化。
- Change 019 的截图展开状态仍是组件本地状态；收起不清缓存，切换 round 依靠原有 scopeKey 隔离。
- Vite 大于 500 kB 的 chunk warning 仍存在，但构建成功，本 Change 不扩大为性能重构。
- race 因 Windows `0xc0000139` 无法启动，状态为 `BLOCKED`，不是业务测试失败，也不是 PASS。
- Change 019 的统一归档仍保持原有阻塞记录；Change 020 归档成功不代表 Change 019 已处理。

## 待完成

- 按 `04-validation.md` 执行 Windows GUI 人工验收并记录真实结果。
- 在可运行 race 的环境中补做 race 验证。
- 未获得 commit、push、Tag 或 Release 授权，因此没有执行相关操作。

## 关闭边界

任何 GUI `NEEDS_USER`、race `BLOCKED` 或未授权提交状态仍存在时，Change 020 不得标记“已完成”或“已关闭”。不删除、覆盖或迁移题库记录、截图、旧导出和旧验证程序。

## 2026-10-25 整合提交状态（最新）

用户明确批准本次将 Change 016～023 累积改动按功能/测试与治理两次提交整合，并连同既有 RC2 提交推送至 origin/dev/qsolver-custom；此授权只适用于本次。

- 功能/测试已提交：`e44bb6163f0cb1b88e2ec4b4da80a1446ddb1990`。
- 本治理档案随第二次整合提交归档；完整边界见 `../../audits/016-023-commit-boundary.md`。
- 原文“未提交/未推送”是当时记录，最新 Git 状态以分支历史和远端分支头为准；本次不创建 Tag 或 Release。
- 不扩大人工验收结论：用户明确确认 Change 016、023；其他未明确项目和既有 race BLOCKED 保留，不因提交自动关闭。
