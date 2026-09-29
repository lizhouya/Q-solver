# 现状快照 — 01 · 003-desktop-window-toggle

> 动手前对当前相关代码的客观描述。依据 `_dev/audits/003` 与源码取证。

## 1. pkg/shortcut（快捷键体系）

- **统一 Manager**：`shortcut.Service`（对外）持有 `Manager`（Windows 实现）。
- Windows：`SetWindowsHookExW(WH_KEYBOARD_LL)` + 鼠标钩子；全局低层钩子，与窗口生命周期无关（窗口隐藏仍有效，F-103 可选关键）。
- 触发：`onKeysChanged()` → `Service.handleTrigger(action)` → delegate。
- `ServiceDelegate`（`pkg/shortcut/delegate.go`）需 app 实现 9+ 方法：`TriggerSolve/Screenshot/Send/DeleteScreenshot, ToggleVisibility, ToggleClickThrough, MoveWindow, ScrollContent, EmitEvent`。
- 录制：`StartRecording(action)` 松键 `finishRecording`，冲突检查仅内部。
- 默认键（Windows）：F8 截图 / Ctrl+J 发送 / Ctrl+D 删除 / F9 toggle(隐身) / F10 穿透 / Alt+方向移动 / Alt+PgUp/Dn 滚动。**F 区键已占满，`Ctrl+Shift+H` 空闲可作新默认**。

## 2. pkg/state（窗口状态）

- `StateManager`（`pkg/state/manager.go`）：`hwnd platform.WindowHandle`（=HWND）+ `windowState{Visible, ClickThrough, StealthMode}` + `windowMu`。
- 现存能力：`ToggleVisibility`(防录屏/隐身)、`ToggleClickThrough`、`Restore/RemoveFocus`、`MoveWindow`(Wails runtime)、`GetHwnd`。
- `WindowState.Visible=true` 默认；`StealthMode=true`。
- **实体可见性（Show/Hide）不单独存储**——`Visible` 字段其实是隐身状态，不是隐藏状态。003 需新增独立状态。

## 3. app（绑定居中）

- `App{ ctx, configManager, stateManager, shortcutService, mobileBridge, mobileServer }`。
- 绑定：`ToggleVisibility, ToggleClickThrough, MoveWindow, Restore/RemoveFocus, SetWindowAlwaysOnTop, Show`（=`runtime.WindowShow`）、`GetMobileOutputInfo` 等。
- `App.Startup`：load config → `runtime.WindowSetSize` → `state.Startup`→ `shortcut.Start` → 注册 `config.Subscribe` → `setupMobile()`。
- `App.OnShutdown`：`shortcut.Stop` → `shutdownMobile()` → config save。
- `EmitEvent` 分流：白名单 `solution-*` 交 `routeSolutionEvent`（OutputMode router），其余走桌面 runtime。
- **无直接 Hide / 无 tray / 无「mobile 自动隐藏」入口。**

## 4. frontend settings（设置 UI）

- `SettingsModal.vue` 已有「输出模式」单选（desktop/mobile/dual）与二维码信息（001 交付）。
- **无「快捷键 → 窗口隐藏」设置项**；快捷键录制在后端 `shortcut`，前端有启动/设置入口调用 `StartRecordingKey`（`app/shortcut.go`）。003 需在此复用。

## 5. pkg/mobile（SSE）

- `mobile.Bridge`（Publish）+ `mobile.Server`（HTTP/SSE 独立 goroutine），绑定 `App.Startup/OnShutdown`。
- `decideRoute(mode, serverAvail) → desktop/mobile/dual` 纯函数。
- **不依赖窗口可见性** → 隐藏窗口不影响 SSE（F-104）。

## 6. pkg/platform（Windows）

- 已封装：`GetWindowHandle / ApplyGhostMode / SetDisplayAffinity / SetClickThrough / SetWindowPos / Get/SetWindowLong / SetWindowToolWindow` 等。
- **无**：`ShowWindow(SW_HIDE/SW_SHOW)` 实体隐藏/显示、无 `Shell_NotifyIcon` 托盘、无系统托盘/退出菜单。
- `WindowState`：GhostMode 在 `initWindowHandle` 启动时永久调用一次。
  
## 7. 当前无任何窗口「隐藏状态机」

- 现有 `ToggleVisibility` 只做防录屏后态，不做实体隐藏；`windowState.Visible` 与「隐藏/显示」语义混用。**003 必须引入独立的 WindowVisibleState。**

## 8. 版本锁定

- `go.mod`：go 1.24.1 / wails v2.11.0（003 不入依赖升级）。平台仅 implement Windows。