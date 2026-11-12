# Change 040 同步端快捷键提示一致性审计

## 结论

同步端提示不一致不是配置更新或 Snapshot 下发失败。`app/mobile.go` 已按截图、发送、快速解题、中断、删除顺序生成提示，并在 Mobile Bridge 初始化及 ConfigManager 更新时调用 `SetHints`。故障位于 `pkg/mobile/web/index.html`：完整快捷键折叠区渲染全部 `hints`，但底部截图状态卡另有旧的三项本地工作流，只筛选截图、删除、发送。

修复后底部卡片仅按标签筛选 Snapshot 数据，严格按截图、发送、快速解题、中断、删除排序；满额只改变截图文案，不移除键帽；页面底部空间取卡片实际高度加 32px，并在窗口尺寸变化时重新计算。Mobile 协议、后端提示实现、配置 schema 和桌面端均未修改。

## 调用链

```text
ConfigManager.Get / Subscribe
  → mobileShortcutHints(cfg.Shortcuts)
  → Bridge.SetHints
  → Snapshot.hints [{label,keyName}]
  → SSE snapshot/message
  → renderHints（完整列表）
  → renderCaptureStatus（五项解题工作流筛选）
  → syncCaptureDockSpacing（实测高度 + 32px）
```

## 安全与架构边界

- 不修改 Mobile 协议、Token、URL、端口或后端控制接口。
- 页面只消费 `label/keyName`，测试证明 JSON 不含 ComboID、Token、路径等内部字段。
- 同步端保持只读，不新增快速解题、中断或删除按钮。
- 键名始终来自 Snapshot，用户重绑定后经既有 ConfigManager 订阅刷新。
- 不新增依赖，不修改 Wails bindings、`go.mod`、`go.sum` 或正式配置。

## 验证结果

- TDD 红灯准确复现固定 `112px` 旧行为。
- DOM 回归覆盖五项严格顺序、非默认键名、满额键帽、动态高度、resize、重置和隐藏清理。
- `app/mobile_test.go` 锁定后端前五项标签和顺序及字段安全。
- 聚焦测试、真实 SSE→DOM、19 个前端回归、Vite、全量 Go、vet、build、mod verify 和 diagnostics 通过。
- race 在 Windows 启动阶段返回 `0xc0000139`，状态 `BLOCKED`。
- 固定 Wails v2.11.0 原候选 SHA-256 为 `51228E51DD744D4E5CD74000FA059203464116DE396F0269CD98BA904178B2C5`；831 个受保护文件、172 个测试区外产物、根稳定版和 Release 均未改变。该候选随后在 Change 041 叠加构建前按原哈希归档到 `archive/transient/Q-Solver-candidate-change040-before-choice-consistency.exe`，最新 Change 041 候选继续包含本 Change 功能。
- 用户于 2026-11-12 使用包含本功能的 Change 041 叠加候选确认同步端 F8–F12 五项提示验证通过；该候选现已晋升本地稳定版，根 SHA-256 为 `2885358371510B827ACE560C354FEE8775BECEB47555D7CE654B0FBE3EB1F15B`。
- race 环境 `BLOCKED` 继续披露；用户已授权本地稳定版更新和 Git 提交，未授权 push、Tag 或 Release。

## 提交边界

功能提交候选范围仅为：

- `pkg/mobile/web/index.html`
- `pkg/mobile/web_dom_test.mjs`
- `app/mobile_test.go`

治理提交范围仅为本审计、Change 040 七份档案和 INDEX 中的 040 状态。功能提交 `09dd485` 已创建，治理提交随本档案完成。用户未授权 push、Tag 或 Release。
