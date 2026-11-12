# 变更请求 — 00

| 字段 | 值 |
|---|---|
| 变更编号 | `040-mobile-shortcut-hint-sync` |
| 提出人 | 用户 |
| 提出日期 | 2026-11-12 |
| 关联 Change | Change 039 |
| 状态 | 已完成 / 普通自动验证与用户 GUI 验收通过 / race BLOCKED 已披露 / 功能提交 `09dd485` / 双提交完成 / 本地稳定版已晋升 / 未推送未发布 |

## 1. 背景与问题

Change 039 已让桌面 Screenshot Dock 按截图、发送、快速解题、中断、删除的顺序展示五项实际快捷键。用户复验发现同步端底部截图状态卡仍显示旧的三项提示，没有同步快速解题与中断。

只读审计确认：后端 Snapshot 已下发五项有序实际绑定，配置更新也会刷新 Mobile Bridge；遗漏位于 `pkg/mobile/web/index.html` 的 `renderCaptureStatus`，其工作流仍硬编码为截图、删除、发送三项。满三张时同步端还会用纯文本替换整个截图提示，丢失截图键帽。

## 2. 目标

1. 同步端底部截图状态卡与桌面 Dock 使用相同的五项业务顺序：截图、发送、快速解题、中断、删除。
2. 每项都使用 Snapshot 下发的实际 `keyName`，用户修改快捷键后自动同步，不硬编码 F8–F12。
3. 满三张时保留截图快捷键键帽，只把文案改为“已满 3 张”。
4. 五项提示在窄屏换行后不遮挡页面末尾内容。

## 3. 验收标准

- [x] DOM 回归证明待发送截图存在时，底部卡片严格按截图、发送、快速解题、中断、删除顺序显示五项提示。
- [x] 提示键名来自 Snapshot `hints`；非默认键名测试按原值显示。
- [x] 满三张时截图键帽仍存在，文案为“已满 3 张”。
- [x] 配置提示数据和同一业务代际重置后仍保留五项提示。
- [x] 固定卡片按实际渲染高度为页面预留底部空间；窗口尺寸变化重新计算，卡片隐藏时清除预留。
- [x] Mobile 继续只读，没有新增控制端点，也未向提示 JSON 暴露 ComboID、Token、路径或连接配置。
- [x] 聚焦 DOM 回归、Mobile/App 测试、19 个前端回归、全量 Go 测试、vet、build、依赖校验和 Vite build 通过。
- [x] 用户使用最终叠加候选确认同步端 F8–F12 五项提示、实际绑定和布局验证通过；候选已晋升本地稳定版。

## 4. 非目标 / 明确不做

- 不修改快捷键配置 schema、默认快捷键或历史迁移逻辑。
- 不修改桌面 Screenshot Dock 已通过验收的行为。
- 不在同步端增加快速解题、中断等操作按钮；同步端仍只展示提示。
- 不新增依赖，不修改 Mobile 网络协议、Token 或 URL。
- 不晋升稳定版，不创建 Tag 或 Release；除非用户后续单独授权。

## 5. 实际影响面

- 业务源码：`pkg/mobile/web/index.html`。
- 测试：`pkg/mobile/web_dom_test.mjs`、`app/mobile_test.go`。
- 治理：Change 040 七份档案、对应审计和 `_dev/changes/INDEX.md`。
- 构建产物：Change 040 原候选按哈希归档 transient；包含本功能的 Change 041 叠加候选已晋升 `build/bin/Q-Solver.exe`，验证候选与旧稳定版完成双归档，`testing/current/` 已清空；Release 未修改。

## 6. 附带约束

- 遵循 `AGENTS.md > CLAUDE.md > Change 040`。
- 只使用服务端下发的 `label/keyName`，不得向 Mobile Snapshot 增加 ComboID 或其他配置实值。
- 用户已确认设计、实施和最终验证，并授权本地稳定版更新与 Git 提交；push、Tag、Release 未授权。
