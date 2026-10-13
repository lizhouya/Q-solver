# 实施记录 — 03 · 007-clickthrough-toggle-feedback

> **状态**：实现完成  
> **日期**：2026-10-13

## 实施时间线

1. 在 `pkg/shortcut/`、`pkg/state/`、`app/`、`frontend/src/` 完成只读审计。
2. 确认真实状态仅在 `platform.SetClickThrough` 成功后写入 `StateManager.windowState.ClickThrough`。
3. 确认现有统一通知链为 `EmitEvent("toast", ...)` → Frontend `on('toast')` → `ui.showToast()`，无需新增 Frontend 代码。
4. 在任何业务代码修改前完成 `00-request.md`、`01-current-state.md`、`02-plan.md` 并登记 INDEX。
5. 把 `ToggleClickThrough` 的 State/App/Delegate 返回契约调整为 `(bool, error)`。
6. 在 `clickthrough` action 中基于真实返回状态发出成功 Toast，error 分支只发失败 Toast。
7. 增加并执行 T-CT-01～T-CT-05，以及 HWND 缺失失败语义测试。
8. 清理 `gofmt` 对既有 Phase 4 测试注释产生的范围外格式噪音。

## 代码改动

### `pkg/state/manager.go`

- HWND 未初始化时返回当前状态和 error。
- `platform.SetClickThrough` 失败时返回当前状态和包装后的 error。
- 成功路径保持原顺序：平台调用成功 → 更新状态 → 日志 → `click-through-state` → 返回真实状态。
- 失败路径不更新状态、不发 `click-through-state`。

### `app/window.go`

- `App.ToggleClickThrough()` 原样转发 `(bool, error)`，不增加状态推断。

### `pkg/shortcut/delegate.go`

- `ServiceDelegate.ToggleClickThrough()` 同步为 `(bool, error)`。

### `pkg/shortcut/service.go`

- `clickthrough` action 仍只调用一次 delegate。
- 成功返回 `true`：发 `EmitEvent("toast", "鼠标穿透：已开启")`。
- 成功返回 `false`：发 `EmitEvent("toast", "鼠标穿透：已关闭")`。
- 返回 error：记录日志并发 `EmitEvent("toast", "鼠标穿透切换失败")`，不发成功文案。

### 测试

- `pkg/shortcut/service_test.go`：记录 delegate 返回值、错误和 Event payload，覆盖成功开启、成功关闭、失败、自定义绑定与单次调用。
- `pkg/state/manager_test.go`：覆盖 HWND 未初始化时返回 error、状态不变且不发成功状态事件。
- `app/phase4_test.go`：仅适配既有 fake delegate 的接口签名。

## Frontend

未修改。现有 `App.vue` 和 `stores/ui.js` 已提供 2000ms 自动消失 Toast；开启穿透不会触发任何临时关闭或恢复穿透的逻辑。

## 范围确认

没有修改 `toggle_window`、GhostMode、WindowVisibility、Tray、Mobile Output、OutputMode、SSE、Solver、KeepContext、LLM、Prompt、Config 或依赖文件。
