# 风险发现 05 — Change 003 前置审计（Findings）

> 仅记录风险/发现，不修复。编号 F-101 起（沿用审计 001 的 F- 语义，另起 3xx 段避免与 old findings 冲突）。

## F-101 窗口「实体隐藏」现不存在，Show/Hide 能力需从零引入

- **事实**：当前只有 `runtime.WindowShow`（`app.Show()`）+ `SetDisplayAffinity` 防录屏隐身；无 `WindowHide` / `ShowWindow(SW_HIDE)` 调用。
- **影响**：003 若实现「隐藏窗口」，必须新增 platform 层 HideWindow + state 状态 + 绑定方法 + 快捷键 action，改动跨越 4 层。
- **风险等级**：中（内容多，但结构清晰）。

## F-102 「隐藏 vs 隐身（GhostMode 防录屏）」语义易混淆

- 现状快捷键 `toggle`（F9）= 防录屏隐身；003 若新增「实体隐藏」快捷键且 F9 已被占，可能互相踩踏。
- **风险**：用户/开发者可能误以为 F9 就是隐藏；需求若复用需明确区分两个独立能力。
- **仅记录**：需在方案里做「能力矩阵」区分（隐身 / 实体隐藏 / 最小化 / 退出）。

## F-103 Shortcut 全局钩子在窗口隐藏后仍应生效（可行性关键）

- 钩子是 `SetWindowsHookExW(WH_KEYBOARD_LL)` **进程级低层钩子**，不挂到特定窗口消息循环 → 即使主窗口 `ShowWindow(SW_HIDE)` 隐藏，进程依然存活、钩子依然可收到全局按键。
- **结论**：本架构下「隐藏后用快捷键恢复」技术上可行（进程不退出、hook 不自毁）。
- **注意**：`Manager.installHooks()` 在独立 goroutine 跑消息循环，与窗口生命周期无关；但隐藏后若进程被关闭 → 钩子消失。需要「隐藏 ≠ 退出」的设计前提。

## F-104 Mobile Output 与窗口可见性**当前完全解耦**（正向）

- `mobile.Server`（HTTP/SSE）纯 goroutine，不引用 HWND/state/窗口；`setupMobile` 与 `shutdownMobile` 仅在 App 生命周期挂钩。
- `decideRoute` 纯函数，仅依赖 OutputMode + server 可用。
- **结论**：隐藏主窗口**不会**中断 SSE 推送、不会中断 App 生命周期、不会卸载 shortcut（因为是同一进程，隐藏只是 Win32 属性）。
- **风险点**：隐藏后 `WindowShow` 未加回（旧 `app.Show()` 只在单实例触发）——需额外恢复触发源。

## F-105 恢复窗口的触发方式缺失/不确定

- 现唯一恢复 = 第二次启动（SingleInstanceLock → Show）。
- 隐藏后若无「快捷键恢复 / 托盘 / mobile 远程恢复」，用户将无法找回窗口。
- **风险**：方案必须定义恢复路径（至少一个）。

## F-106 快捷键与 Windows 系统组合键冲突

- 默认键（Ctrl+J/D、Alt+方向、F8-10 等）已满；新增 `toggle_window` 若用未占用键位需查表；低层钩子吞键可能影响前台应用。
- 风险：冲突检测 `handleRecordingComplete` 仅检查内部；不与系统级（如 Win 键、Alt+Tab）冲突。

## F-107 多平台差异（Windows 已验证，macOS 未覆盖隐藏）

- Windows：`ShowWindow/SetWindowPos` 等 Win32 可用 → 实体隐藏可行。
- macOS：platform_darwin 是否有等号 `Hide/Show` 未审计（本审计聚焦 Windows）；`ToggleVisibility` 的实现无依存的 import（`SetDisplayAffinity` 只有 Windows 语义）。
- **风险**：若跨平台，macOS 需额外实现（Wails runtime 的 WindowShow 各平台统一，但原生 SW_HIDE 无对应）。

## F-108 单实例与 ML 窗口（双窗口）疑问

- SingleInstanceLock 仅保护「单实例」；未审计是否确实存在多窗口（如 ScreenShotDock 等前端模拟 UI 层）。
- 若隐藏主窗口，需确认是否连带隐藏 webview 子窗口/其他工具窗——当前 `GetWindowHandle` 枚举 PID 首个窗口，未枚举所有。

## F-109 状态事件一致性

- `state.ToggleVisibility` emit `toggle-visibility`；003 若新增实体隐藏，需补充 `window-hidden/show` 事件，前端同步。
- 若藏在 `state` 增加实体 visible 位，需与现有 `windowState.Visible` 去分离防止误用。

## F-110 不影响 001/002 Mobile 已提交

- 003 若立项，改动干净独立：不触碰 `pkg/mobile`、不触碰 `index.html`、不动 SSE 协议；可行性高，与已提交 bcf5ff8/b8acfa3 无回归风险（但需 `go build`/护栏重测）。