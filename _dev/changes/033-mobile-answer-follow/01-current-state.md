# Change 033：现状快照

## 涉及文件

| 文件 | 当前职责 | 当前事实 |
|---|---|---|
| `pkg/mobile/web/index.html` | 嵌入式同步端页面、样式、SSE 和滚动控制 | 快捷键是完整卡片；答案渲染无自动滚动；控制步长为视口高度 70% |
| `pkg/mobile/web_dom_test.mjs` | 在 VM 和 DOM stub 中验证同步端行为 | 已验证控制去重、清屏、历史、复制和安全渲染；当前断言 800px 视口向上滚动为 -560px |
| `pkg/mobile/bridge_server_test.go` | 验证 HTTP 返回实际嵌入页面 | `TestIndexHTML` 使用 `viewportHeight * 0.7` 作为页面完整性固定文本哨兵 |

## 布局现状

- `#shortcut-card` 位于答案区域顶部，使用通用 `.card` 的 18px 内边距和 14px 下外边距。
- 卡片包含标题、固定说明段落和 `.hints-grid`；每个 `.hint-item` 还有 8px × 10px 内边距。
- `renderHints` 每次重建列表，并通过 `shortcutCard.hidden` 控制整个卡片显隐，没有折叠控件或折叠状态。

## 答案更新现状

- `showAnswer(html, payload)` 更新 `#answer-text`，显示答案卡片并隐藏等待/错误卡片。
- `applySnapshot` 的 `streaming` 和 `done` 分支均可调用 `showAnswer`。
- 当前没有答案更新后的 `scrollTo`、`scrollIntoView` 或等价逻辑。
- `selectedSeq >= 0` 时，新业务快照只更新历史数据，不替换当前历史浏览内容。

## 键盘滚动现状

- `applyRealtimeControl` 对有效、未处理的 `controlSeq` 安排一次 `requestAnimationFrame`。
- 滚动距离是 `Math.round(viewportHeight * 0.7)`，800px 视口对应 560px。
- 清屏通过 `renderEpoch` 使已排队回调失效；滚动使用 `behavior: 'smooth'`。

## 边界

- 本变更不需要修改后端状态、SSE 帧格式或服务端 hints。
- 页面禁止使用 `localStorage`、`sessionStorage` 和 `console`，现有 DOM 回归已有静态约束。
- 当前分支为 `dev/qsolver-custom`，动码前工作树无未说明改动。
