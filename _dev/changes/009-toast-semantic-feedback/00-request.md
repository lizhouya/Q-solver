# 变更请求 — 00 · 009-toast-semantic-feedback

| 字段 | 值 |
|---|---|
| 变更编号 | `009-toast-semantic-feedback` |
| 提出人 | 用户 |
| 提出日期 | 2026-10-13 |
| 状态 | `规划中` |

## 1. 背景与问题

Change 008 已将 Toast 前置图标替换为 CSS 圆框与稳定字符，但 Wails `toast` 事件的旧调用只传字符串，Frontend 会调用 `ui.showToast(msg)` 并使用默认 `type='error'`。因此 ClickThrough 成功开启、成功关闭和失败提示都会显示 error 的 `!`，状态语义不正确。

## 2. 目标

为 `toast` 事件增加可选的结构化 payload `{text, type}`，让 ClickThrough 按真实结果传递语义类型：开启为 `success`、关闭为 `info`、失败为 `error`。Frontend 同时兼容结构化 payload 和所有既有字符串 payload。

## 3. 验收标准

- [x] ClickThrough 开启发送 `鼠标穿透：已开启` + `success`。
- [x] ClickThrough 关闭发送 `鼠标穿透：已关闭` + `info`。
- [x] ClickThrough 失败发送 `鼠标穿透切换失败` + `error`。
- [x] 旧字符串 `toast` payload 仍调用 `ui.showToast(text)`，保持默认行为。
- [x] Change 008 的 Toast DOM 和 CSS 图标方案不变。
- [x] 不修改 mobile/window/tray/solution/LLM。
- [x] 自动测试、构建和差异边界检查通过。
- [x] 停在 commit boundary，不执行 staging、commit 或 push。

## 4. 非目标 / 明确不做

- 不修改 Toast template、CSS、动画、字符标记或视觉组件结构。
- 不批量迁移 `app/solve.go` 等旧字符串发送点。
- 不改变 `ui.showToast` 的函数签名、默认 type 或 duration。
- 不修改 ClickThrough 状态切换逻辑、快捷键绑定或平台实现。
- 不新增依赖、事件名、配置、数据库或持久化。

## 5. 影响面预估

- `pkg/shortcut/service.go`：仅 ClickThrough Toast 改为结构化 payload。
- `pkg/shortcut/service_test.go`：验证三种文案与 type，保留动作/真实状态测试。
- `frontend/src/App.vue`：Toast 事件入口兼容字符串和结构化 payload；不动 template。
- `_dev/changes/009-toast-semantic-feedback/`、`_dev/changes/INDEX.md`：治理记录。

## 6. 附带约束

仅在 `dev/qsolver-custom` 工作；保持 Change 007/008 现有未提交差异；不提交 secret；不执行 `git add`、`git commit` 或 `git push`。
