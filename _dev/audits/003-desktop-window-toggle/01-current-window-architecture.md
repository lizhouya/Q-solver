# 现状窗口架构 01 — Change 003 前置审计

> 描述「窗口是如何创建、样式如何施加、可见性如何管理」的当前事实。全部来自源码取证。

## 1. 窗口创建（`main.go`）

Wails v2 `wails.Run(&options.App{...})` 一处创建主窗口，初值：

| 项 | 值 | 说明 |
|---|---|---|
| Title | `""`（空） | 无标题栏（配合 GhostMode） |
| Width/Height | 1024 × 768 | 桌面主窗口尺寸 |
| Min/Max | 840×700 / 1024×768 | 固定尺寸范围（Max=Min 同为 1024×768 → **不可缩放**） |
| AlwaysOnTop | `true` | 总置顶 |
| BackgroundColour | RGBA 全 0 | 透明背景配合 translucency |
| SingleInstanceLock | 有（UniqueId） | `OnSecondInstanceLaunch → app.Show()` |
| Windows.Options | WebviewIsTransparent / WindowIsTranslucent | 透明窗口，BackdropType.None |

关键点：`AlwaysOnTop: true` 让窗口**长期置顶**；`SingleInstanceLock` 的第二实例回调直接调用 `app.Show()`（Wails runtime `WindowShow`）。

## 2. 窗口状态层（`pkg/state/manager.go`）

`StateManager` 持有窗口状态并调用 platform：

| 方法 | 作用 | 底层 |
|---|---|---|
| `Startup(ctx, emitEvent)` | 初始化 `hwnd` | `platform.GetWindowHandle()`（枚举 PID 匹配 HWND） |
| `initWindowHandle()` | 获取 HWND 后即时 `ApplyGhostMode` | `platform.ApplyGhostMode(hwnd)` |
| `ToggleVisibility()` | **防录屏隐身**：`SetDisplayAffinity(false/true)` | 非「显示/隐藏实体」，仅让录屏不可见 |
| `ToggleClickThrough()` | 鼠标穿透 | `SetClickThrough` |
| `RestoreFocus()` / `RemoveFocus()` | 焦点允许/禁止 | `platform.RestoreFocus/RemoveFocus` |
| `MoveWindow(dx,dy)` | 移动窗口 | `runtime.WindowGetPosition/SetPosition` |
| `GetHwnd()` | 取 `platform.WindowHandle` | — |

> **现状要点**：`ToggleVisibility` 当前语义 = **防录屏（DisplayAffinity 隐身）**，窗口仍在桌面可见可交互；**不是**「消失/隐藏」。

## 3. 可见性能力现状（Wails runtime + platform）

| 能力 | 现状 | 位置 |
|---|---|---|
| `WindowShow` | 可用（`app.Show()` 包装） | `app/app.go` + 单实例回调 |
| `WindowHide` | **未使用**（runtime 支持但当前不调用） | — |
| Minimize | 未封装（无 `WindowMinimise` 绑定） | — |
| AlwaysOnTop | `platform.SetWindowLevel`（Windows 为空实现，靠 style） | `app/window.go` |
| 实体隐藏/恢复 | **未存在**（仅隐身） | — |

## 4. 窗口句柄与生命周期

- `StateManager.hwnd`（`platform.WindowHandle`，Windows = HWND）在 `Startup` 早于业务启动时获取，此后可被 `GetHwnd`/`SetWindowAlwaysOnTop` 复用。
- 主窗口无独立 `Hide`；关闭窗口 = `runtime.Quit`?（**未发现** `WindowHide`/`WindowMinimise` 调用）。
- 窗口移动/尺寸经 Wails runtime（非原生），样式经 Win32（GhostMode）。

## 5. 小结

当前「窗口控制」聚焦 **GhostMode + 置顶 + 防录屏 + 焦点/穿透**，属于「常显透明悬浮工具」定位；**没有「整体隐藏/恢复桌面窗口」的入口**（既无快捷键、也无 API、也无 mobile 联动）。这正是 Change 003 的空白点。