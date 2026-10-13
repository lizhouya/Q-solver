# 验证记录 04 · 009-toast-semantic-feedback

> **日期**：2026-10-13  
> **分支**：`dev/qsolver-custom`  
> **基线 HEAD**：`aa35b55 chore: add change 006 governance records`  
> **提交状态**：未暂存、未提交、未推送

## 语义矩阵

| ClickThrough 结果 | 文案 | type | Change 008 图标 |
|---|---|---|---|
| 成功开启 | `鼠标穿透：已开启` | `success` | `✓` |
| 成功关闭 | `鼠标穿透：已关闭` | `info` | `i` |
| 失败 | `鼠标穿透切换失败` | `error` | `!` |

## 自动测试与构建

| 命令 | 结果 | 备注 |
|---|---|---|
| `go test ./pkg/shortcut -run ClickThrough -count=1` | ✅ PASS | 三类 payload、真实返回状态、自定义绑定 |
| `go test ./pkg/shortcut -run ClickThrough -count=50` | ✅ PASS | 重复稳定性 |
| `go test ./... -count=1` | ✅ PASS | 全部 Go package |
| `go build ./...` | ✅ PASS | exit 0 |
| `npm --prefix frontend run build` | ✅ PASS | Vite 4.5.14，118 modules；仅既有 chunk > 500 kB 警告 |
| `go vet ./pkg/shortcut/...` | ⚠️ 既有警告 | 未修改的 `shortcut_windows.go:122,155` 报 possible misuse of unsafe.Pointer |
| `go vet -unsafeptr=false ./pkg/shortcut/...` | ✅ PASS | 排除既有 unsafeptr 项后通过 |
| `wails build -debug -windowsconsole` | ✅ PASS | Windows WebView2 debug build |
| `git diff --check` | ✅ PASS | 无 whitespace error；仅既有 LF→CRLF 提示 |

## 兼容性验证

- Go 测试精确验证结构化 payload 仅含 `text/type`。
- `app/solve.go` 和 macOS 快捷键提示仍保留字符串 `EmitEvent("toast", "...")`，未批量迁移。
- Frontend handler 对合法结构化对象调用 `ui.showToast(payload.text, payload.type)`。
- Frontend handler 对字符串调用 `ui.showToast(payload)`，与 Change 009 前行为一致。
- `ui.showToast` 本身未修改，默认 `error` 和 duration 仍保持原值。

## Change 008 保护检查

- Toast template 未因 Change 009 修改，仍使用 `<span class="toast-icon" aria-hidden="true"></span>`。
- `frontend/src/styles/global.css` 未因 Change 009 修改。
- success/info/error 分别复用 Change 008 已有 `✓`/`i`/`!` CSS 标记。
- 未修改共享 `Icon.vue`。

## Wails v2.16 生成污染

Wails CLI v2.16.0 构建后重生成 `go.mod`、`frontend/wailsjs/runtime/*` 和 `frontend/wailsjs/go/app/*`。审计确认属于版本差异和 bindings 噪音后已精确恢复；最终无生成污染。

## 禁止范围检查

Change 009 未修改：

- `pkg/mobile/`
- Window 实现
- `pkg/tray/`
- `pkg/solution/`
- LLM Provider 或 Prompt
- 配置、依赖、数据库、持久化

工作区中已有的 `app/window.go` 等差异属于未提交的 Change 007，不是 Change 009 新增修改。

## 验收标准对照

- [x] 开启为 success。
- [x] 关闭为 info。
- [x] 失败为 error。
- [x] 旧字符串 Toast 兼容。
- [x] Change 008 视觉组件结构和 CSS 方案保留。
- [x] 禁止范围未因 009 修改。
- [x] 自动测试、构建和边界审计完成。
- [x] 暂存区为空；未 commit、未 push。
- [ ] Windows WebView2 肉眼观察三种语义图标：`NEEDS_USER`。

## 结论

**自动化 Commit Boundary：PASS。** 没有发现确定性 Change 009 缺陷。产品视觉确认仍为 `NEEDS_USER`，当前按要求停止，不提交。
