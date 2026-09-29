# 调用链 04 — Change 003 前置审计

## 1. 总体调用链图

```
[用户按键]
   │
   ▼
pkg/shortcut (Windows)
  keyboardHookProc → onKeysChanged() → OnTrigger(action)
   │
   ▼
shortcut.Service.handleTrigger(action)
   │
   ▼ (delegate = App)
app:(App 实现 ServiceDelegate)
   ├─ TriggerScreenshot / TriggerSend / TriggerDeleteScreenshot / ToggleVisibility / ToggleClickThrough / MoveWindow / ScrollContent / EmitEvent
   │
   ├─ 窗口相关：→ state.StateManager
   │     ToggleVisibility → platform.SetDisplayAffinity
   │     ToggleClickThrough → platform.SetClickThrough
   │     MoveWindow → runtime.WindowGet/SetPosition
   │     RestoreFocus/RemoveFocus → platform.RestoreFocus/RemoveFocus
   │
   └─ 业务：→ screen.Service / solution.Solver / mobile.Bridge.Publish(needMobile)
```

## 2. 快捷键 → 动作 → 最终实现的映射（调用链明细）

| action | shortcut.Service | App 委托 | 最终实现 |
|---|---|---|---|
| `screenshot` | handleTrigger | TriggerScreenshot | screen 截图 |
| `send` | handleTrigger | TriggerSend | 发送解题（触发 solve） |
| `delete` | handleTrigger | TriggerDeleteScreenshot | 删截图 |
| `toggle` | handleTrigger | ToggleVisibility | state.ToggleVisibility → SetDisplayAffinity |
| `clickthrough` | handleTrigger | ToggleClickThrough | state.ToggleClickThrough → SetClickThrough |
| `move_*` | handleTrigger | MoveWindow(dx,dy) | state.MoveWindow → runtime.WindowSetPosition |
| `scroll_up/down` | handleTrigger | ScrollContent | EmitEvent("scroll-content") → 前端 |

## 3. Wails runtime 侧窗口 API（main.go / app / state 引用）

| API | 使用点 |
|---|---|
| `runtime.WindowShow` | `app.Show()`（单实例回调） |
| `runtime.WindowSetSize/GetPosition/SetPosition` | state.MoveWindow、App.Startup |
| `runtime.WindowSetAlwaysOnTop` | 无（改由 platform level 处理） |
| `runtime.WindowHide` / `WindowMinimise` | **未使用**（不存在） |

> `win32` 原生层（platform）已能**改样式/置顶/防录屏/焦点**，但**没有 `ShowWindow(SW_HIDE / SW_SHOW)` 级别的实体隐藏**封装。

## 4. Change 003 需要的改动面（潜在，仅记录不改）

若实现「桌面窗口显示/隐藏切换」，最小改动面（按调用链）：

| 层 | 建议改动点 | 现状 |
|---|---|---|
| `pkg/platform` | 新增 `HideWindow/ShowWindow`（封装 `ShowWindow(hwnd, SW_HIDE/SW_SHOW)` 或 Win32 API） | 无 |
| `pkg/state` | `StateManager` 增加实体可见状态 + hide/show 方法 | 只有防录屏 Toggle |
| `pkg/shortcut` | 新增 action（如 `toggle_window`）→ delegate | 需新增 |
| `app/app.go` | 新绑定方法 `HideWindow/ShowWindow/IsWindowHidden` | 仅 `app.Show()` |
| `frontend` | 设置面板加开关/快捷键录制项 | 无 |
| `main.go` | 无改动（除非初始态要隐藏） | — |

> 实际改动层数与范围取决于用户决策（见 06），此处仅据现状盘点。

## 5. 生命周期顺序（Startup → OnShutdown）

```
App.Startup
 ├─ config.load
 ├─ state.Startup（取 HWND，ApplyGhostMode）
 ├─ screen.Startup
 ├─ llm/solver/resume
 ├─ shortcut.Start（钩子装入）
 ├─ config.subscribe
 └─ setupMobile（bridge + server.Start）  ← 与窗口状态无关
App.OnShutdown
 ├─ shortcut.Stop（卸钩子）
 ├─ mobile server.Stop（关端口）
 └─ config.Save
```
→ Mobile server 生命周期从 Startup 到 Shutdown，**与窗口可见性完全解耦**（见 05）。