# 窗口控制能力分析 03 — Change 003 前置审计

## 1. 现有关（窗口）控制能力清单

| 能力 | 现状 | 实现文件 / 函数 | 备注 |
|---|---|---|---|
| 创建/初始尺寸 | 有 | `main.go` `wails.Run(Width=1024, Height=768)` | 不可缩放（Max=Min） |
| 置顶 | 有 | `main.go AlwaysOnTop:true`；`app.SetWindowAlwaysOnTop`→`platform.SetWindowLevel` | Windows 侧 SetWindowLevel 为空实现，实际靠 GhostMode EXTOPMOST |
| 隐身（防录屏） | 有 | `state.ToggleVisibility`→`platform.SetDisplayAffinity` | **非实体隐藏**，仅影响录屏可见性 |
| 鼠标穿透 | 有 | `state.ToggleClickThrough`→`platform.SetClickThrough` | |
| 焦点获取/释放 | 有 | `state.RestoreFocus/RemoveFocus` | 改 EXSTYLE + SWP |
| 移动窗口 | 有 | `state.MoveWindow`→runtime.WindowSetPosition | |
| `Show()` | 有 | `app.Show()` → `runtime.WindowShow` | 单实例重启窗口时用 |
| **`Hide()` / 实体隐藏** | **无** | 无绑定、无调用 | 003 需新增 |
| `Minimize` | **无** | 无封装 | 003 可选 |
| 全屏/普通切换 | 无 | — | — |
| 尺寸持久化 | 有 | config.WindowWidth/Height → `runtime.WindowSetSize` | Startup 恢复 |

## 2. `ToggleVisibility` 语义澄清（关键）

`pkg/state/manager.go: ToggleVisibility()` 实际只做：
```
SetDisplayAffinity(hwnd, false)   // 变成可被录屏检测
SetDisplayAffinity(hwnd, true)    // 变成录屏不可见
```
并 `windowState.Visible` 翻转，emit `toggle-visibility` 事件。

**结论**：当前 `toggle` 快捷键 = **GhostMode 隐身开关**，不是「窗口消失/显示」。Change 003 的「隐藏窗口」是独立能力，不要与之混淆。

## 3. GhostMode 关系分析（用户问题的 A/B/C/D 归类）

`platform.applyGhostMode(hwnd)`（Windows）组合：
- 移除 `WS_CAPTION|WS_THICKFRAME|WS_MINIMIZEBOX|WS_MAXIMIZEBOX|WS_SYSMENU` → 无边框
- `&^WS_EX_APPWINDOW | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE | WS_EX_TOPMOST` → 工具窗/不抢焦点/置顶
- `SetDisplayAffinity(WDA_EXCLUDEFROMCAPTURE)` → 防录屏（Win11 隐身，失败降级 WDA_MONITOR 黑屏）
- `SetWindowPos(SWP_FRAMECHANGED|SWP_NOACTIVATE)` 刷新样式
- Win10 下 `enableTransparentGradient` 消除 DWM 模糊

**归类：B（含显示逻辑，但不含实体隐藏，也不含快捷键）**
- A「只是窗口样式」？ → 部分（样式 + 防录屏）
- **B（含隐藏逻辑）** → 是的：`SetDisplayAffinity` 让窗口对录屏/捕捉「隐身」
- C（含快捷键） → **否**：GhostMode 是样式/防录屏逻辑，快捷键在 shortcut 体系
- D（其他） → 进程启动时永久引用（initWindowHandle 调用一次），**不是** toggle 的另一端

> 因此 Change 003 若做「桌面窗口整体隐藏/恢复」，要**新增实体 Show/Hide**（`runtime.WindowHide/Show` 或 `ShowWindow`），与 GhostMode 的「隐身」正交但需共存（隐藏期间防录屏仍有意义？见 F-001）。

## 5. 单实例 / 恢复路径

- `main.go SingleInstanceLock.OnSecondInstanceLaunch → app.Show()`。
- 若实现实体隐藏，可复用这一路径作为「恢复」触发之一（第二实例/再次启动），需纳入设计。
- 当前唯一的 `Hide` 级别关闭 = 进程退出（`OnShutdown`），无最小化到托盘。