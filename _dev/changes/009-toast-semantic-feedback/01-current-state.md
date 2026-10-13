# 现状快照 — 01 · 009-toast-semantic-feedback

## 1. 涉及文件清单

| 文件 | 当前职责 | 现状 |
|---|---|---|
| `pkg/shortcut/service.go` | ClickThrough action 与 Toast 发送 | 开启、关闭、失败均只发送字符串 |
| `pkg/shortcut/service_test.go` | Shortcut 行为与 Toast 断言 | `assertSingleToast` 只接受字符串 payload |
| `frontend/src/App.vue` | Wails Toast 事件入口 | `on('toast', msg) => ui.showToast(msg)` |
| `frontend/src/stores/ui.js` | Toast 数据与生命周期 | `showToast(text, type='error', duration=2000)` |
| `frontend/src/styles/global.css` | Change 008 CSS 图标 | error/warning=`!`、success=`✓`、info=`i` |

## 2. 关键代码现状

ClickThrough 当前发送：

```go
s.delegate.EmitEvent("toast", "鼠标穿透切换失败")
s.delegate.EmitEvent("toast", "鼠标穿透：已开启")
s.delegate.EmitEvent("toast", "鼠标穿透：已关闭")
```

Frontend 当前入口：

```js
on('toast', (msg) => {
  ui.showToast(msg)
})
```

Store 默认值：

```js
function showToast(text, type = 'error', duration = 2000)
```

因此上述三个字符串均成为 error Toast。

## 3. 旧调用兼容面

只读搜索发现 `app/solve.go` 与 macOS 快捷键提示仍使用字符串 payload。Change 009 不修改这些发送点；Frontend 必须保留字符串分支，使其继续走原有默认 error 行为。

`frontend/src/services/events.js` 会把 Wails 事件参数原样转发给 handler，因此 Go 的 `map[string]string` 会作为单个结构化对象到达 App.vue。

## 4. Change 008 视觉结构

当前 Toast template 为：

```vue
<span class="toast-icon" aria-hidden="true"></span>
<span>{{ t.text }}</span>
```

图标由 `.toast.<type> .toast-icon::before` 控制。本变更不修改 template 或 CSS。

## 5. 依赖与构建现状

- Vue 3 + Vite 4；没有独立 Frontend test script。
- Go 测试可直接验证 Shortcut 发送 payload。
- 不需要新增依赖或生成 bindings，因为事件名和 Wails 暴露方法均不变。

## 6. 已知边界

Change 007/008 尚未提交，工作区已有其 Go、Frontend 与治理差异。Change 009 必须叠加实施但保持可审计，不得清理或覆盖前序变更。
