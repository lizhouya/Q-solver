# 总结与注意事项 06 · 009-toast-semantic-feedback

## 变更摘要

Change 009 在不改变 Toast 视觉结构的前提下，为 ClickThrough 反馈增加 `text/type` 语义 payload：开启 success、关闭 info、失败 error。Frontend 同一事件入口兼容对象与旧字符串，因此现有发送点不需要一次性迁移。

## 经验与踩坑

| 现象/坑 | 原因 | 结论/建议 |
|---|---|---|
| Change 008 后成功提示仍显示 `!` | 所有后端字符串 Toast 使用 Store 默认 error | 图标兼容与事件语义需分层处理 |
| 不能破坏旧发送点 | 多个 App/Shortcut 路径仍发字符串 | 在接收端支持联合格式，逐步迁移 |
| Wails build 再次生成 bindings/runtime | CLI v2.16 高于 module v2.11.0 | 审计后精确恢复，不将生成物混入变更 |
| 标准 Shortcut vet 非零 | 未修改 Windows hook 中既有 unsafe.Pointer 用法 | 如实记录，并用 `-unsafeptr=false` 验证其余项 |

## 对本项目后续的影响

- `toast` 事件现在支持两种 payload：旧字符串，或 `{text,type}` 对象。
- 新增需要明确语义的后端 Toast 可采用结构化对象；旧发送点继续可用。
- 视觉仍由 Change 008 的 type class 控制，没有新增组件或样式系统。

## 遗留项与 TODO

- `NEEDS_USER`：Windows WebView2 肉眼触发 ClickThrough 开启、关闭和失败路径，确认分别显示 `✓`、`i`、`!`。
- Change 007～009 都尚未提交；任何 staging/commit 仍需用户明确授权。

## 参考资源

- `pkg/shortcut/service.go`
- `pkg/shortcut/service_test.go`
- `frontend/src/App.vue`
- `_dev/changes/008-toast-icon-compatibility/`
