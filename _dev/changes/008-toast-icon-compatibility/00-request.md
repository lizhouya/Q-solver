# 变更请求 — 00 · 008-toast-icon-compatibility

| 字段 | 值 |
|---|---|
| 变更编号 | `008-toast-icon-compatibility` |
| 用户命名 | `Change 007.1 Toast Icon Compatibility` |
| 提出人 | 用户 |
| 提出日期 | 2026-10-13 |
| 状态 | `规划中` |

## 1. 背景与问题

Windows WebView2 中 Toast 前置图标被观察为 `×`。当前后端 `toast` 事件只传字符串，前端 `ui.showToast` 的默认类型是 `error`，Toast 展示层因此将消息映射为共享 `Icon.vue` 的 `x-circle` 内联 SVG。对“鼠标穿透：已开启/已关闭”这类状态反馈而言，前置 `×` 容易被理解为失败标记，也未满足 Windows WebView2 下的视觉兼容性要求。

## 2. 目标

仅优化 Frontend Toast 展示层：以 CSS 绘制容器和稳定字符表示状态，不再让 Toast 依赖共享 SVG 图标渲染；保持 Toast 事件、Store、消息文案、类型、时序及业务调用链不变。

## 3. 验收标准

- [x] Toast 前置标记不再使用 `Icon.vue` 的 `x-circle`，默认类型不再显示 `×`。
- [x] error、success、warning、info 四种类型均有稳定且可辨识的前置标记。
- [x] `on('toast') → ui.showToast`、`showToast` 默认参数和生命周期完全不变。
- [x] 不修改 Shortcut、State、Window、Mobile 或其他 Go 业务代码。
- [x] 不新增依赖；Frontend build、Go test/build 与 diff-check 通过。
- [x] 停在 commit boundary，不执行 staging、commit 或 push。

## 4. 非目标 / 明确不做

- 不改变 Toast 业务调用链、事件协议、文案、默认 type 或 duration。
- 不修改共享 `Icon.vue`，避免影响 ErrorView、ResumeWarning 等其他图标使用点。
- 不修改 `pkg/shortcut/`、`pkg/state/`、`app/`、`pkg/mobile/`。
- 不新增前端组件、图标库或字体依赖。
- 不处理 Change 007 以外的 UX 或样式问题。

## 5. 影响面预估

- 业务源码：`frontend/src/App.vue`、`frontend/src/styles/global.css`。
- 治理文档：`_dev/changes/008-toast-icon-compatibility/`、`_dev/changes/INDEX.md`。
- 风险：CSS 字符尺寸或对齐偏差；通过固定盒模型、系统字体栈和 Frontend build 检查控制。

## 6. 附带约束

仅在 `dev/qsolver-custom` 工作；不得提交 secret；不得制造 `go.mod`、`go.sum` 或 `frontend/wailsjs/` 生成污染；不得执行 `git add`、`git commit` 或 `git push`。
