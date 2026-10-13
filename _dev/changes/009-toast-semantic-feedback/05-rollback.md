# 回滚预案 05 · 009-toast-semantic-feedback

## 回滚策略

定向回滚 Change 009，不回滚 Change 007/008：

1. 将 `pkg/shortcut/service.go` 的三个 ClickThrough 结构化 Toast 恢复为字符串 payload。
2. 将 `pkg/shortcut/service_test.go` 的结构化 payload 断言恢复为文案字符串断言。
3. 将 `frontend/src/App.vue` 的 Toast handler 恢复为 `ui.showToast(msg)`。
4. 保留 Change 008 的 Toast span 与 CSS 字符图标。
5. 从 INDEX 移除 009 登记并按治理要求保留或标记归档状态。

## 回滚触发条件

- Wails 事件无法把 `map[string]string` 序列化为预期对象。
- 旧字符串 Toast 不再显示。
- ClickThrough 类型与真实状态相反。
- Frontend handler 因 payload 类型产生运行时异常。

## 回滚后验证

- `go test ./pkg/shortcut -run ClickThrough -count=1`
- `go test ./... -count=1`
- `go build ./...`
- `npm --prefix frontend run build`
- `git diff --check`
- 确认 Change 008 template/CSS 仍存在。

## 实际回滚记录

未触发回滚。
