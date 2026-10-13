# 回滚预案 05 · 008-toast-icon-compatibility

## 回滚策略

采用定向还原，仅撤销 Change 008 的两个 Frontend 展示层改动：

1. 在 `frontend/src/App.vue` 中恢复 Toast 的 `<Icon :name="toastIcon(t.type)" ... />`。
2. 恢复 `toastIcon(type)` 函数。
3. 从 `frontend/src/styles/global.css` 移除 Change 008 新增的 `.toast-icon` 圆框和 `::before` 字符规则，保留原有四种 type 颜色规则。
4. 不修改 `Icon.vue`、`ui.js`、Wails 事件链或任何 Go 文件。
5. 不回滚 Change 007 的 App/State/Shortcut 实现和治理记录。

## 回滚触发条件

- 新前置标记在 Windows WebView2 中不显示、错位或遮挡正文。
- CSS 规则影响非 Toast 元素。
- Frontend build 或 Wails build 因该改动失败。
- 可访问性检查发现纯装饰标记被重复朗读。

## 回滚后验证

- `npm --prefix frontend run build`
- `go test ./... -count=1`
- `go build ./...`
- `git diff --check`
- 核对未触碰 Change 007 工作区差异。

## 实际回滚记录

未触发回滚。
