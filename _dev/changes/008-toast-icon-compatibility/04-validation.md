# 验证记录 04 · 008-toast-icon-compatibility

> **日期**：2026-10-13  
> **分支**：`dev/qsolver-custom`  
> **基线 HEAD**：`aa35b55 chore: add change 006 governance records`  
> **提交状态**：未暂存、未提交、未推送

## 构建验证

| 命令 | 结果 | 备注 |
|---|---|---|
| `npm --prefix frontend run build` | ✅ PASS | Vite 4.5.14，118 modules；仅既有 chunk > 500 kB 警告 |
| `go test ./... -count=1` | ✅ PASS | 全部 Go package 通过 |
| `go build ./...` | ✅ PASS | exit 0 |
| `wails version` | ✅ PASS | v2.16.0 |
| `wails build -debug -windowsconsole` | ✅ PASS | 生成 Windows WebView2 debug EXE；项目 Wails module 仍为 2.11.0 |
| `git diff --check` | ✅ PASS | 无 whitespace error；仅既有 LF→CRLF 提示 |

## 静态行为验证

| 检查项 | 预期 | 实际 | 结果 |
|---|---|---|---|
| Toast template | 不再渲染共享 SVG | `<span class="toast-icon" aria-hidden="true"></span>` | ✅ PASS |
| Toast mapping | 删除 `toastIcon(type)` | App.vue 中已不存在该函数 | ✅ PASS |
| 业务事件链 | 保持不变 | `on('toast', msg) { ui.showToast(msg) }` 原样保留 | ✅ PASS |
| Store 语义 | 保持不变 | `ui.js` 未修改；type/duration/lifecycle 未改变 | ✅ PASS |
| 四种标记 | error、warning、success、info 可区分 | `!`、`!`、`✓`、`i` + 各自颜色 | ✅ PASS |
| 共享图标 | 不影响其他 UI | `Icon.vue` 未修改；App.vue import 保留 | ✅ PASS |
| 辅助技术 | 纯装饰不重复朗读 | `aria-hidden="true"` | ✅ PASS |

## Windows WebView2 验证

- Wails debug build 已通过，说明 Frontend 可被打包进 Windows WebView2 应用。
- 原 `×` 来源已从 Toast DOM 中移除，生产构建不再为 Toast 实例化 `x-circle` SVG。
- **最终肉眼视觉：NEEDS_USER**。当前应用启用了 GhostMode 防录屏，当前会话无法可靠观察实际 Toast 像素，因此不把视觉项伪报为 PASS。

## Wails v2.16 生成污染

构建生成了以下与本变更无关的噪音：

- `go.mod`
- `frontend/wailsjs/runtime/*`
- `frontend/wailsjs/go/app/*`

审计确认其来自 CLI v2.16 与项目 module v2.11.0 的版本差异和 bindings 重生成。已仅恢复上述文件；最终工作区没有 Wails 生成污染。

## 验收标准对照

- [x] 默认 Toast 不再使用 `x-circle` 或显示 `×` 型 SVG。
- [x] 四种 type 均有 CSS/稳定字符标记。
- [x] Toast 业务调用链、消息、type 与时序未修改。
- [x] 未因 008 修改 Shortcut、State、Window、Mobile 或 Go 业务代码。
- [x] 未新增依赖，Frontend/Go/Wails build 通过。
- [x] 暂存区为空，未执行 commit 或 push。
- [ ] Windows WebView2 肉眼确认新图标外观：`NEEDS_USER`。

## 回归与副作用检查

Change 007 原有 Go 差异仍在工作区，008 没有覆盖或扩展它们。`go.mod`、`go.sum` 和 `frontend/wailsjs/` 最终无修改。未运行 `wails dev`。

## 结论

**自动化 Commit Boundary：PASS。** 实现、构建、打包和边界检查均完成，且没有确定性缺陷。产品视觉验收仍为 `NEEDS_USER`；因此停在 commit boundary，不提交。
