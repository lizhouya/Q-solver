# 快捷键体系分析 02 — Change 003 前置审计

## 1. 快捷键架构总览

```
用户按键 → [Windows 低级键盘钩子]  (pkg/shortcut/shortcut_windows.go: keyboardHookProc)
        → onKeysChanged() 匹配 ComboID
        → shortcut.Service.handleTrigger(action)
        → App（app/shortcut.go 委托）→ 各业务触发
```

- **统一管理器**：`pkg/shortcut/Service`（对外）+ `Manager`（Windows 实现）。
- **钩子**：`SetWindowsHookExW(WH_KEYBOARD_LL)` + `WH_MOUSE_LL`（侧键），全局低层钩子。
- **匹配键**：`GetComboID(heldKeys)` 生成组合 ID → 查 `Smartcuts` map（`ComboID ↔ action`）。
- **吞键**：命中返回 1 吞掉按键；未命中 `CallNextHookEx` 放行。
- **录制**：`StartRecording(action)` → 松键时 `finishRecording` 存键。

## 2. 事件触发链（App 侧）

`shortcut.Service.handleTrigger(action)` 分发到 delegate（`app` 实现 `ServiceDelegate`）：

| action | delegate 调用 | 业务 |
|---|---|---|
| `screenshot` | `TriggerScreenshot()` | 截图 |
| `send` | `TriggerSend()` | 发送解题 |
| `delete` | `TriggerDeleteScreenshot()` | 删截图 |
| `toggle` | `ToggleVisibility()` | 隐身切换（防录屏） |
| `clickthrough` | `ToggleClickThrough()` | 鼠标穿透 |
| `move_up/down/left/right` | `MoveWindow(dx,dy)` | 移动窗口 |
| `scroll_up/down` | `ScrollContent(...)` | 滚动内容 |

> `app/app.go` 未在 `handleTrigger` 中发现其它 action；当前 action 全集如上述 9+2。

## 3. 当前快捷键列表（默认值，`pkg/config/config.go getDefaultShortcuts`）

| action | Windows 默认 | macOS 默认 |
|---|---|---|
| `screenshot` | `F8`（VK 119） | — |
| `solve` | — | `Cmd+1` |
| `send` | `Ctrl+J`（74+162） | `Cmd+J` |
| `delete` | `Ctrl+D`（68+162） | `Cmd+D` |
| `toggle`（隐身） | `F9`（V 120） | `Cmd+2` |
| `clickthrough`（穿透） | `F10`（VK 121） | `Cmd+3` |
| `move_up/down/left/right` | `Alt+↑↓←→` | `Cmd+Option+Arrow` |
| `scroll_up/down` | `Alt+PgUp/PgDn` | `Cmd+Option+Shift+Arrow` |

## 4. 是否存在统一 shortcut manager

**是**。`pkg/shortcut` 即统一管理器：
- 一个 `Service`（进程内单例），持有 `Manager`（全局 key/mouse hook）。
- 配置驱动（订阅 config 变更 → `maps.Copy` 刷新 `Shortcuts`）。
- 支持录制（Windows 限定，macOS 仅预设）。

## 5. 对 Change 003 的含义

- 新增「隐藏/显示窗口」动作，可**直接复用**现有 action 分发：加一个 action 如 `"hide_toggle"` → delegate 加方法 → 前端设置区出现新 key 录制项。
- 现有钩子为**进程内全局钩子**：窗口隐藏后**钩子仍生效**（钩子不是窗口消息），这是可行性的关键点。
- 注意：`toggle`（对齐）已占位 — 若 003 用「隐藏/恢复」同名会歧义；须新 action 或用按键复用（见 F-003/D-003-002）。