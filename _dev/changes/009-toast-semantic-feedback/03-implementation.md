# 实现过程记录 03 · 009-toast-semantic-feedback

## 实现日志

| 日期 | 步骤 | 改动内容 | 涉及文件 | 说明 |
|---|---|---|---|---|
| 2026-10-13 | 只读审计 | 搜索所有 Go Toast 发送点、Frontend handler 和 Store 默认值 | Shortcut、App.vue、ui.js、events.js | 确认旧字符串调用必须继续兼容 |
| 2026-10-13 | 事前归档 | 完成 00/01/02 并登记 009 | Change 009 文档、INDEX | 在业务代码修改前完成 |
| 2026-10-13 | 后端语义 | ClickThrough 三种结果发送 `{text,type}` | `pkg/shortcut/service.go` | 开启 success、关闭 info、失败 error |
| 2026-10-13 | 测试 | 精确断言结构化 payload 的文案、类型和字段数 | `pkg/shortcut/service_test.go` | 保留单次调用与自定义绑定覆盖 |
| 2026-10-13 | 前端兼容 | Toast handler 同时接受对象与旧字符串 | `frontend/src/App.vue` | 不改 Store、template 或 CSS |
| 2026-10-13 | 验证 | 聚焦/重复/全量测试、构建、vet、Wails 打包和 diff-check | 无额外业务文件 | 已知生成噪音精确恢复 |

## 与计划的偏差

无。只修改计划中的三个业务/测试文件。

## 关键决策纪要

1. 继续使用现有 `toast` 事件名，不创建第二套事件或通知系统。
2. 结构化 payload 使用 `map[string]string`，字段固定为 `text` 和 `type`，Wails 可直接序列化为 JavaScript 对象。
3. Frontend 只在 payload 是对象且 `text` 为字符串时采用结构化路径；其他 payload（包括旧字符串）走原兼容路径。
4. 不更改 `ui.showToast` 的默认参数。结构化对象若 `type` 缺失，JavaScript 的 `undefined` 仍会触发其默认 `error`。
5. 不迁移 `app/solve.go` 和 macOS 快捷键提示等旧字符串发送点，以直接验证兼容性边界仍存在。

## 遗留未完成项

Windows WebView2 中肉眼确认三种颜色/字符组合仍为 `NEEDS_USER`。类型 payload 和生产构建已经自动验证。
