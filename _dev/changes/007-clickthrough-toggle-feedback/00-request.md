# 变更请求 — 00 · 007-clickthrough-toggle-feedback

> **状态**：已批准实施  
> **日期**：2026-10-13  
> **英文目标**：Add visible feedback when mouse click-through is enabled or disabled.

## 背景

现有 `clickthrough` 快捷键能够切换主窗口鼠标穿透状态，但成功后只广播 `click-through-state` 供前端同步交互状态，没有向用户显示明确、短暂的状态提示。用户无法直接确认本次动作实际开启还是关闭了穿透。

## 目标

在现有鼠标穿透动作成功后，根据底层确认的真实新状态显示以下提示：

- `鼠标穿透：已开启`
- `鼠标穿透：已关闭`

仅增强反馈，不改变鼠标穿透本身的 Win32/macOS 平台行为、状态语义或快捷键绑定机制。

## 验收标准

1. `clickthrough` action 仍只调用一次 `ToggleClickThrough`。
2. 成功提示根据 `StateManager` 在平台调用成功后返回的真实新状态生成，不根据旧状态或物理按键推断。
3. 底层失败时不显示“已开启”或“已关闭”；允许复用现有 Toast 显示“鼠标穿透切换失败”。
4. 提示复用现有 `EmitEvent("toast", ...)` → Frontend `ui.showToast()` 链路，自动消失、不抢焦点、不阻塞快捷键。
5. 提示与 `clickthrough` action 绑定；用户将默认 F10 改为其他快捷键后行为不变。
6. 不新增前端组件、通知系统、第三方依赖或配置字段。
7. 完成 T-CT-01～T-CT-05、Go 全量测试/构建、修改包 vet、Frontend build 和 Git 边界检查。

## 允许范围

- `pkg/state/manager.go`
- `pkg/state/*_test.go`
- `pkg/shortcut/delegate.go`
- `pkg/shortcut/service.go`
- `pkg/shortcut/*_test.go`
- `app/window.go`
- 为适配 `ServiceDelegate` 接口而必须修改的 `app/*_test.go`
- `_dev/changes/007-clickthrough-toggle-feedback/`
- `_dev/changes/INDEX.md`

## 禁止范围

不修改 `toggle_window`、GhostMode、WindowVisibility、System Tray、Mobile Output、OutputMode、SSE、Solver、KeepContext、LLM、Prompt、Config、`go.mod`、`go.sum` 或 `frontend/wailsjs/runtime/*`；不合并 ClickThroughState、WindowVisibleState 与 GhostModeState；不新增依赖或持久化；不运行 `wails dev`；本阶段不执行 `git add`、`git commit` 或 `git push`。
