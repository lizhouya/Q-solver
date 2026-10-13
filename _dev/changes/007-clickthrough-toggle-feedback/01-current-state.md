# 现状记录 — 01 · 007-clickthrough-toggle-feedback

> **阶段**：业务代码修改前只读快照  
> **基线分支**：`dev/qsolver-custom`  
> **基线提交**：`aa35b55 chore: add change 006 governance records`  
> **工作树**：审计开始时 clean

## 当前调用链

默认 Windows F10 只是配置中的 `clickthrough` action 绑定，真实调用链不依赖 F10：

```text
Shortcut Manager 触发 action="clickthrough"
  → shortcut.Service.handleTrigger("clickthrough")
  → ServiceDelegate.ToggleClickThrough()
  → app.App.ToggleClickThrough()
  → state.StateManager.ToggleClickThrough()
  → platform.SetClickThrough(hwnd, newState)
  → 成功后更新 WindowState.ClickThrough
  → EmitEvent("click-through-state", newState)
  → Frontend App.vue 同步 ui.isClickThrough / pointer-events
```

设置页允许把 `clickthrough` action 绑定到自定义快捷键；`handleTrigger` 只接收 action 名，不读取或写死 F10。

## 真实状态与失败语义

- 真实状态保存在 `StateManager.windowState.ClickThrough`，由 `windowMu` 保护。
- `newState` 在锁内由当前真实状态取反。
- 只有 `platform.SetClickThrough` 成功后才写入 `windowState.ClickThrough` 并发出 `click-through-state`。
- HWND 未初始化或平台调用失败时，当前实现保持原状态，也不发 `click-through-state`。
- 当前 `StateManager.ToggleClickThrough()` 只返回 `bool`；失败时返回旧状态，因此上层无法区分“成功切换到该布尔状态”与“失败后保持该布尔状态”。
- `App.ToggleClickThrough()` 和 `ServiceDelegate.ToggleClickThrough()` 当前不返回状态或错误，Shortcut 层无法可靠生成成功提示。

## 现有提示机制

仓库已有统一 Toast/Event 链路：

1. Go 后端调用 `EmitEvent("toast", message)`。
2. `App.EmitEvent` 对非解题事件直接调用 Wails runtime event emitter。
3. Frontend `App.vue` 监听 `toast` 并调用 `ui.showToast(msg)`。
4. `ui.showToast` 默认显示 2000ms，随后淡出并清理；Toast 是页面覆盖层，不触发聚焦或阻塞。

现有隐藏/显示相关反馈：

- GhostMode 的 `toggle-visibility` 事件由 Frontend 显示“隐身模式已开启/关闭”。
- WindowVisibility 实体显示/隐藏当前只维护状态并记录日志，没有建立另一套通知系统。
- `click-through-state` 当前只同步 UI 状态，没有提示文案。

## 缺口

缺少一条把“平台切换结果 + 真实新状态”送回 Shortcut 层的错误感知路径。若只在 Shortcut 层对旧状态取反，或仅依据触发次数显示文案，会在 HWND/平台失败时误报成功。

## 审计结论

最小方案是保留现有 action 调用链，把 `ToggleClickThrough` 的返回契约改为 `(bool, error)`：State 在成功后返回真实新状态，App 原样转发，Shortcut 成功时复用 `toast`，失败时只显示失败提示。Frontend 已完整支持该事件，无需修改。
