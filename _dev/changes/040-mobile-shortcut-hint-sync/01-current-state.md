# 现状快照 — 01

## 1. 涉及文件清单

| 文件 | 当前职责 | 当前事实 |
|---|---|---|
| `app/mobile.go` | 从 Config 生成 Mobile 快捷键提示，并在配置更新时刷新 Bridge | `mobileShortcutHintDefinitions` 前五项已按截图、发送、快速解题、中断、删除排列；无需修改业务实现 |
| `pkg/mobile/web/index.html` | 渲染同步端页面、完整快捷键列表和底部截图状态卡 | `renderHints` 能显示全部下发提示；`renderCaptureStatus` 只筛选截图、删除、发送三项 |
| `pkg/mobile/web_dom_test.mjs` | 在隔离 DOM Stub 中验证同步端页面行为 | 当前明确断言截图 Dock 只有三项，锁定了旧行为 |
| `app/mobile_test.go` | 验证 Mobile 提示顺序、字段安全和可见标签 | 已验证 fast/cancel 存在，但未用五项同形输入严格断言前五项顺序 |

## 2. 关键代码现状

后端定义当前为：

```text
截图 → 发送解题 → 快速解题 → 中断解题 → 删除截图
```

Mobile Bridge 初始化和配置订阅均调用 `mobileShortcutHints(cfg.Shortcuts)`，因此新配置会进入 Snapshot `hints`。

同步端底部卡片当前本地定义为：

```text
截图 → 删除截图 → 发送解题
```

它没有筛选 `快速解题` 和 `中断解题`。当截图数达到上限时，当前实现追加纯文本 `已满 N 张` 并提前返回，因此截图 `kbd` 不再显示。

## 3. 相关配置

不读取或修改正式用户配置。提示只消费 Snapshot 中脱敏的 `label` 与 `keyName`；ComboID、Token、配置路径和连接信息不进入 DOM。

## 4. 依赖与构建现状

- 不需要新增 Go 或前端依赖。
- Mobile 页面是 `pkg/mobile/web/index.html` 的嵌入资产。
- 现有聚焦测试入口：`node pkg/mobile/web_dom_test.mjs`、`go test ./app ./pkg/mobile -count=1`。
- 业务门禁仍要求全量 Go 测试、vet、build、前端构建；候选构建按固定 Wails v2.11.0 和产物政策执行。

## 5. 运行行为现状

- 完整“答题端快捷键”折叠区域能够显示快速解题和中断解题。
- 有待发送截图时，页面底部固定状态卡只显示旧三项。
- 底部预留空间固定为 `112px`；五项提示在窄屏换行后可能超过该高度。

## 6. 已知现状问题

- Change 039 的 race 验证仍因 Windows `0xc0000139` 为环境 `BLOCKED`；本 Change 不把它误记为通过。
- 本 Change 不处理同步端执行控制、主题或其他布局优化。
