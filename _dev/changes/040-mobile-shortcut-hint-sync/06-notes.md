# 总结与注意事项 06

## 当前摘要

Change 040 已让同步端底部截图状态卡按截图、发送、快速解题、中断、删除顺序显示五项实际绑定；满三张保留截图键帽，底部空间随卡片高度和窗口变化更新。普通自动门禁和用户 GUI 验收通过；功能已随 Change 041 叠加候选晋升本地稳定版。race 环境 `BLOCKED` 已披露，Git 提交已授权，未发布。

## 长期约束

- 桌面与同步端提示键名必须来自实际配置，不硬编码默认 F 键。
- Mobile Snapshot 只传递 `label/keyName`，不得包含 ComboID、Token、路径或连接配置。
- 同步端保持只读，不因展示“快速解题/中断”而新增控制能力。
- 固定底部卡片增加内容时必须同步核对页面底部安全空间。

## 稳定版与归档

- 根稳定版：`build/bin/Q-Solver.exe`
- 大小：42,558,464 字节
- SHA-256：`2885358371510B827ACE560C354FEE8775BECEB47555D7CE654B0FBE3EB1F15B`
- Change 040 原候选：`archive/transient/Q-Solver-candidate-change040-before-choice-consistency.exe`，SHA-256 `51228E…B2C5`。
- 最终验证归档：`archive/validation/change-041/Q-Solver-candidate-change041.exe`。
- 被替代稳定版：`archive/superseded/change-041/Q-Solver-stable-before-change041.exe`。
- `testing/current/` 已清空，Release 未修改。

## 授权状态

用户已授权本地稳定版更新和 Git 提交。功能提交 `09dd485` 已创建，治理提交随本档案完成；Change 040 与 Change 041 保持独立提交。push、Tag、Release 未授权。

## 参考资源

- `_dev/audits/040-mobile-shortcut-hint-sync.md`
- `pkg/mobile/web/index.html`
- `pkg/mobile/web_dom_test.mjs`
