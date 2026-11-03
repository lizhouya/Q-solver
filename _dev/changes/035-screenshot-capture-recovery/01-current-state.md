# 现状快照 — 01

## 1. 涉及文件清单

| 文件 | 当前职责摘要 | 关键行/符号 | 备注 |
|---|---|---|---|
| `pkg/shortcut/shortcut_windows.go` | Windows 低级键盘/鼠标 Hook 与组合键匹配 | `keyboardHookProc`、`onKeysChanged` | 命中后同步调用 `globalManager.OnTrigger(action)` |
| `pkg/shortcut/service.go` | 将快捷键 action 路由到 App delegate | `NewService`、`Start`、`Stop`、`handleTrigger` | `manager.OnTrigger` 当前直接绑定 `handleTrigger` |
| `app/solve.go` | 截图缓冲、最多 3 张、发送与解题任务启动 | `MaxScreenshots`、`TriggerScreenshot`、`triggerSend` | 截图捕获与压缩同步执行；三图缓冲逻辑仍存在 |
| `pkg/screen/screen.go` | 屏幕捕获、压缩和 Base64 预览 | `CapturePreview` | 调用 `screenshot.Capture` 和图片编码 |
| `frontend/src/components/ScreenshotDock.vue` | 待发送截图缩略图、计数和快捷键提示 | `screenshots`、`screenshot-taken` | 计数固定展示 `/3`；快捷键在 setup 时一次性读取 |
| `frontend/src/stores/settings.js` | 快捷键状态和派生显示名 | `solveShortcut`、`sendShortcut`、`deleteShortcut` | 三者均为 computed，可通过 `storeToRefs` 保持响应性 |
| `pkg/solution/solution_test.go` | Solver 多图请求回归 | `TestSolveThreeScreenshotsSingleCallInOrderAndPassesFullAnswer` | 已证明 3 张图按顺序进入一次 Provider 调用 |

## 2. 关键代码现状

### Windows Hook 调用链

`keyboardHookProc` 在 Hook 回调线程内调用 `onKeysChanged`；命中配置后，`onKeysChanged` 直接同步调用 `globalManager.OnTrigger(action)`。`OnTrigger` 指向 `Service.handleTrigger`，截图 action 最终同步进入 `App.TriggerScreenshot`。

### 截图耗时链

`App.TriggerScreenshot` 在持有 `conversationMu` 时调用 `GetScreenshotPreview`。后者经 `screen.Service.CapturePreview` 执行屏幕捕获、可选 Lanczos 缩放、灰度、锐化、JPEG/PNG 编码和 Base64 转换，完成后才向前端发送 `screenshot-taken`。

### 三图能力

- `MaxScreenshots` 当前为 3。
- `TriggerScreenshot` 在缓冲未满时 append。
- `triggerSend` 复制整个 `screenshotBuffer`，清空缓冲后启动一次 Solver。
- Solver 已有三图单调用顺序测试。

### ScreenshotDock

- 组件始终由 `App.vue` 挂载。
- 收到 `screenshot-taken` 后 push 图片并显示 `screenshots.length/3`。
- 当前只展示删除、发送提示，不展示继续截图提示。
- `const sendShortcut = settingsStore.sendShortcut` 与 `const deleteShortcut = settingsStore.deleteShortcut` 在 setup 时从 Pinia store proxy 取值，不能保证后续配置加载或重绑后同步更新。

## 3. 相关配置

- 本变更不读取或修改用户 `config.json`。
- Windows 默认快捷键仍为：截图 `F8`、发送 `Ctrl+J`、删除 `Ctrl+D`。
- 自定义快捷键继续由现有 `Config.Shortcuts` 管理，本变更不改变持久化格式。

## 4. 依赖与构建现状

- 不需要新增 Go 或 npm 依赖。
- 项目 Wails Go 模块为 v2.11.0。
- 前端使用 Vue 3 + Pinia + Vite，已有 `storeToRefs` 可直接使用。
- 标准验证要求包括 Go test/vet/build、前端回归、Vite build 和 Wails build。

## 5. 运行行为现状

1. 按下截图快捷键后，Hook 回调线程同步等待截图和压缩完成。
2. 截图动作完成后才返回 Hook；期间其他键盘事件处理可能被阻塞。
3. 若 Hook 因耗时被 Windows 静默移除，后续快捷键不再进入应用。
4. 只有成功发送 `screenshot-taken` 时 Dock 才会出现。
5. Dock 出现后可显示图片数量，但快捷键提示可能停留在组件初始化时的默认值。

## 6. 已知现状问题

- 本次尚无 Windows 运行时 Hook 超时日志；Windows 静默移除低级 Hook 通常不会向应用返回明确错误，因此当前根因结论来自确定的同步耗时调用链与用户症状匹配，最终仍需 GUI 连续截图验证。
- 既有 `TriggerScreenshot` 持有 `conversationMu` 完成捕获和压缩。本 Change 优先解除 Hook 线程阻塞并保持动作顺序，不在没有证据时扩大为截图锁架构重写。
- Change 034 稳定版记录明确注明未包含 ScreenshotDock 快捷键显示修复。
