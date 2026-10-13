# 决策记录 — 05 · 007-clickthrough-toggle-feedback

> **日期**：2026-10-13

## D-007-001：提示绑定 action，不绑定 F10

`clickthrough` 是业务 action，F10 只是 Windows 默认 KeyBinding。提示逻辑放在 `handleTrigger("clickthrough")`，因此自定义快捷键无需额外分支。

## D-007-002：成功文案必须使用 State 成功返回值

不在 Shortcut 或 App 中自行取反状态。只有 State 确认平台调用成功、更新内部状态后，才返回真实 `enabled` 值用于文案。

## D-007-003：错误成为返回契约的一部分

原 `bool` 无法区分失败保持 false 与成功切换到 false，也无法区分失败保持 true 与成功切换到 true。采用 `(bool, error)` 是防止成功误报所需的最小契约变化。

## D-007-004：失败只显示通用失败 Toast

失败时不显示任何开启/关闭成功文案。复用现有 `toast` 事件显示 `鼠标穿透切换失败`，不新建复杂错误 UI，也不向用户暴露平台错误细节。

## D-007-005：保留既有 `click-through-state`

该事件继续只在平台成功后广播，用于 Frontend 同步 `ui.isClickThrough` 和 pointer-events。Toast 是附加反馈，不替代状态事件，也不合并 ClickThroughState 与其他窗口状态。

## D-007-006：不修改 Frontend

现有 Toast 容器和 `ui.showToast` 已满足自动消失、不阻塞和不抢焦点要求。修改 Frontend 会增加无必要范围与回归面。

## D-007-007：不为显示提示临时改变穿透状态

Toast 只是渲染反馈。实现不调用额外平台 API，不关闭穿透，不恢复焦点，不触碰 WindowVisibility 或 GhostMode。

## D-007-008：既有 vet 警告不在本变更修复

`shortcut_windows.go` 的 `unsafe.Pointer` vet 警告位于未修改代码，修复会扩展到 Windows hook 机制。Change 007 只记录它，并用排除 `unsafeptr` 分析器的补充命令验证其余 Shortcut 代码。
