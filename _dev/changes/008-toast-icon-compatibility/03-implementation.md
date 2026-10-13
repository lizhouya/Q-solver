# 实现过程记录 03 · 008-toast-icon-compatibility

## 实现日志

| 日期 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-10-13 | 只读审计 | 追踪 Toast template、type 映射、Store 默认值、共享 Icon 和生效 CSS | `App.vue`、`Icon.vue`、`ui.js`、`main.js`、`global.css` | 确认 `×` 是 `error → x-circle` 的预期 SVG 视觉，不是字体缺字 |
| 2026-10-13 | 事前归档 | 完成 00/01/02 并分配编号 008 | Change 008 文档、INDEX | 用户命名 Change 007.1；仓库治理编号使用连续整数 |
| 2026-10-13 | Template | Toast 前置 `<Icon>` 替换为 `aria-hidden` span，删除 Toast 专用映射函数 | `frontend/src/App.vue` | 保留 `Icon` import，避免影响简历警告弹窗 |
| 2026-10-13 | CSS | 用固定圆框与 `!`、`✓`、`i` 稳定字符渲染四种 type | `frontend/src/styles/global.css` | 不依赖共享 SVG、外部字体或新增包 |
| 2026-10-13 | 验证 | Frontend、Go、Wails build 与差异边界检查 | 无额外业务文件 | Wails v2.16 生成噪音已审计并精确恢复 |

## 与计划的偏差

无。仅修改计划列出的两个 Frontend 展示文件。

## 关键决策纪要

1. 不修改 `ui.showToast` 的默认 `error` 类型，因为用户明确禁止改变 Toast 业务调用链。
2. 不修改共享 `Icon.vue`，避免影响 ErrorView、ResumeWarning 等非 Toast UI。
3. 使用 CSS `::before` 与普通字符，不引入图标字体、第三方库或新的 Vue 组件。
4. error 和 warning 均以 `!` 表达需注意状态；success 使用 `✓`；info 使用 `i`。
5. 字符节点标记 `aria-hidden="true"`，避免辅助技术重复朗读纯装饰内容。

## 遗留未完成项

Windows WebView2 中的最终肉眼视觉验收为 `NEEDS_USER`。编译、打包及源码路径已验证，但当前会话无法在 GhostMode 防录屏条件下可靠确认实际像素呈现。
