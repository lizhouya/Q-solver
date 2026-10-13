# 现状快照 — 01 · 008-toast-icon-compatibility

## 1. 涉及文件清单

| 文件 | 当前职责摘要 | 关键符号 | 备注 |
|---|---|---|---|
| `frontend/src/App.vue` | 渲染 Toast，并按 type 选择共享图标 | Toast template、`toastIcon(type)` | 当前业务修改目标 |
| `frontend/src/styles/global.css` | Toast 布局、动画和 type 颜色 | `.toast`、`.toast-icon` | 当前业务修改目标 |
| `frontend/src/components/Icon.vue` | 共享内联 SVG 图标组件 | `x-circle`、`check-circle`、`alert-triangle`、`info` | 只读审计，不修改 |
| `frontend/src/stores/ui.js` | Toast 数据与生命周期 | `showToast(text, type = 'error', duration = 2000)` | 只读审计，不修改 |
| `frontend/src/main.js` | 引入生效中的全局样式 | `import './styles/global.css'` | 确认 `global.css` 是运行时样式入口 |

## 2. 关键代码现状

Toast 当前渲染共享 SVG：

```vue
<Icon :name="toastIcon(t.type)" :size="14" class="toast-icon" />
```

类型映射：

```js
function toastIcon(type) {
  if (type === 'error') return 'x-circle'
  if (type === 'success') return 'check-circle'
  if (type === 'warning') return 'alert-triangle'
  return 'info'
}
```

共享 `Icon.vue` 的 `x-circle` 是圆形加两条交叉线的内联 SVG，不依赖外部字体或图标包。其实际视觉语义本身就是 `×`，不是字体缺字 fallback。

`ui.showToast` 的默认类型为 `error`。Wails `toast` 监听器只传消息字符串，因此 Change 007 的开启、关闭和失败反馈都会进入 `error → x-circle` 映射。

## 3. 相关配置

无配置、环境变量或 secret 参与 Toast 图标渲染。

## 4. 依赖与构建现状

- Vue 3 + Vite 4。
- `package.json` 没有独立测试命令。
- 图标为仓库内共享 Vue SVG 组件，没有外部图标依赖。
- 生效的 Toast CSS 来自 `frontend/src/styles/global.css`；旧 `frontend/src/style.css` 未被入口导入，不纳入修改范围。

## 5. 运行行为现状

- `on('toast', msg)` 调用 `ui.showToast(msg)`。
- `showToast` 默认 `type='error'`，2000ms 后淡出。
- Toast 容器不接收指针事件，不抢占鼠标交互。
- Windows WebView2 中用户观察到前置图标显示为 `×`。

## 6. 已知现状问题

- 默认 `error` 类型导致所有只传字符串的后端 Toast 都展示错误型 `x-circle`，即使消息内容是成功的状态通知。
- 本变更受约束不得改变默认类型或业务调用参数，因此只在展示层把 `×` 替换为兼容标记，不修正类型语义。
- Change 007 仍有未提交工作区差异；008 必须与其共存且不得覆盖。
