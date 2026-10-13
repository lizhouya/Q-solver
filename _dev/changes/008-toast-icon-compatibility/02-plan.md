# 实施方案 — 02 · 008-toast-icon-compatibility

## 1. 方案概述

将 Toast 专用的共享 `<Icon>` 替换为无业务逻辑的 `<span>`。由 Toast 自身 type class 配合 CSS `::before` 生成稳定字符：error/warning 使用 `!`，success 使用 `✓`，info 使用 `i`；外圈、尺寸、对齐均由 CSS 绘制。删除仅供 Toast 使用的 `toastIcon(type)` 函数，但保留 `Icon` import，因为同一文件的简历警告弹窗仍使用共享图标。

## 2. 分步实施计划

| 步骤 | 操作 | 涉及文件 | 预期产出 | 完成标志 |
|---|---|---|---|---|
| 1 | 替换 Toast 前置图标节点并移除映射函数 | `frontend/src/App.vue` | Toast 不再实例化共享 SVG | 模板仅含展示 span |
| 2 | 增加固定尺寸、圆框、稳定字符和类型颜色 | `frontend/src/styles/global.css` | WebView2 兼容的前置标记 | 四种 type 均有规则 |
| 3 | 执行 Frontend/Go 构建测试和 Git 边界审计 | 无业务修改 | 可重复验证结果 | 所有自动检查通过或如实记录 |
| 4 | 补齐 03～06 和 INDEX 状态 | `_dev/changes/008-toast-icon-compatibility/`、`INDEX.md` | 完整治理记录 | 停在 commit boundary |

## 3. 涉及业务源码的改动清单

仅允许：

1. `frontend/src/App.vue`
   - Toast 前置节点从共享 `Icon` 改为 `aria-hidden` 的展示 span。
   - 删除只服务于 Toast 的 `toastIcon(type)`。
   - 不移除 `Icon` import，因为 `warn-dialog` 仍使用它。
2. `frontend/src/styles/global.css`
   - 为 `.toast-icon` 增加固定盒模型、圆框、字体和对齐。
   - 用 type class 的 `::before` 内容显示稳定字符。

禁止越界修改其他业务源码。

## 4. 测试与验证计划

- `npm --prefix frontend run build`
- `go test ./... -count=1`
- `go build ./...`
- `git diff --check`
- 静态核对 Toast 不再包含 `toastIcon` / Toast `<Icon>`，并保留 `on('toast') → ui.showToast`。
- `git diff --name-only` 和 status 边界检查：不得出现 `go.mod`、`go.sum`、`frontend/wailsjs/` 或 008 范围外的新修改。
- 当前会话若无法可靠肉眼观察 WebView2，则将视觉验收标记为 `NEEDS_USER`，不伪造 PASS。

## 5. 回滚预案

定向恢复 `App.vue` 的 Toast `<Icon>` 和 `toastIcon(type)`，移除 `global.css` 新增的 Toast 字符规则；不回滚 Change 007 的任何 Go 或治理差异。

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| `✓` 在不同字体下字形略有差异 | success 标记视觉不一致 | 使用 Windows/WebView2 稳定的系统字体栈并固定尺寸与行高 |
| CSS 伪元素被全局样式覆盖 | 标记不显示 | 使用 `.toast.<type> .toast-icon::before` 明确选择器 |
| 误删共享 Icon import | 简历警告图标构建失败 | 保留 `Icon` import并通过 Vite build 验证 |
| 修改业务类型来规避 `×` | 越过用户范围 | 不改 Store、事件监听或 type 参数，只改展示层 |
