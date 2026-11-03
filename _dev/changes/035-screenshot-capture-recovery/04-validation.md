# 验证记录 04

## TDD 红灯

| 命令 | 结果 | 证据 |
|---|---|---|
| `go test ./pkg/shortcut -run 'TestSerialActionDispatcher' -count=1` | EXPECTED FAIL | `undefined: newSerialActionDispatcher` |
| `node frontend/screenshot_dock_regression_test.mjs` | EXPECTED FAIL | 缺少 `storeToRefs` import，后续继续截图提示断言尚未执行 |

## 聚焦与压力验证

| 命令 | 结果 | 备注 |
|---|---|---|
| `go test ./pkg/shortcut -run 'TestSerialActionDispatcher' -count=1` | PASS | 阻塞 handler 期间快速入队、FIFO、单 handler、幂等 Stop |
| `go test ./pkg/shortcut -count=1` | PASS | Shortcut 全包通过 |
| `go test ./pkg/shortcut -run 'TestSerialActionDispatcher' -count=50` | PASS | 50 次压力回归，2.430s |
| `node frontend/screenshot_dock_regression_test.mjs` | PASS | 响应式快捷键、张数、继续截图/满额和事件契约通过 |
| `go test ./pkg/solution -run '^TestSolveThreeScreenshotsSingleCallInOrderAndPassesFullAnswer$' -count=1 -v` | PASS | 3 张图片按顺序进入一次 Provider 调用 |
| `go test -race ./pkg/shortcut -count=1` | BLOCKED | 测试程序启动阶段退出 `0xc0000139`；补充项未执行，未记为通过 |

## 全量构建验证

| 命令 | 结果 | 备注 |
|---|---|---|
| `go test ./... -count=1` | PASS | 全部相关包通过 |
| `go vet ./...` | PASS | 无输出 |
| `go build ./...` | PASS | 无输出 |
| `go mod verify` | PASS | `all modules verified` |
| 全部 `frontend/*_test.mjs` | PASS | 15 份回归通过 |
| `npm --prefix frontend run build` | PASS | Vite v4.5.14，132 modules transformed；仅既有大 chunk warning |
| Wails v2.11.0 保护构建 | PASS | Go 1.25.13；保护参数完整；42.215s |
| `git diff --check` | PASS | 仅 LF→CRLF 提示，无 whitespace error |
| VS Code diagnostics | PASS | 本次 Go 文件无诊断 |

## GUI 与运行验证

| 检查项 | 结果 | 证据 |
|---|---|---|
| 快速连续截图 3 次与 Dock `1/3～3/3` | PASS | 用户确认候选测试通过 |
| 第 4 次截图上限 | PASS | 用户确认候选测试通过 |
| 删除后补拍 | PASS | 用户确认候选测试通过 |
| 三图发送与一次解题 | PASS | 用户确认候选测试通过；Solver 自动测试同时验证一次请求三图顺序 |
| 实际快捷键提示 | PASS | 用户确认候选测试通过；静态回归验证 `storeToRefs` |
| 晋升后根稳定版启动 | PASS | 运行 8 秒持续存活，检测到 WebView2，Node/Vite 子进程 0 |
| 晋升后正常退出 | PASS | `CloseMainWindow` 成功，退出码 0，无强制终止，无残留 Q-Solver 进程 |

## 稳定版晋升与归档

- 当前稳定版：`build/bin/Q-Solver.exe`
- 大小：44,029,440 字节
- SHA-256：`29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE`
- 候选归档：`build/bin/archive/validation/change-035/Q-Solver-candidate-change035.exe`
- 旧稳定版归档：`build/bin/archive/superseded/change-035/Q-Solver-before-change035.exe`
- 旧稳定版 SHA-256：`C033EAB4844721EB233841915C82583D213095965357E77091738D9FBC21B0B8`
- 两个归档目录的 `SHA256SUMS` 与文件逐项一致。
- `testing/current/` 已清空；根目录唯一 EXE 为 `Q-Solver.exe`。
- `go version -m`：Go 1.25.13，Wails runtime v2.11.0。
- 构建和晋升前后 `go.mod`、`go.sum`、`frontend/wailsjs/**`、`frontend/dist/**`、Git index、`build/bin/releases/**` 保护通过。

## 验收标准对照

- [x] Hook 命中后只轻量入队，不直接执行业务 handler。
- [x] 阻塞首个 handler 时后续动作快速入队，最终 FIFO。
- [x] 单 worker 保证动作不并发。
- [x] 连续三图和第四张上限通过用户 GUI 验收。
- [x] 三图按顺序进入一次模型请求。
- [x] Dock 显示张数、继续截图、满额、删除和发送提示。
- [x] Dock 响应实际快捷键。
- [x] 发送、删除和清空后的 Dock 状态通过验收。
- [x] 必要自动测试、构建和晋升后运行验证通过。
- [x] race 补充检查的环境阻塞已如实记录，未伪造通过。

## 回归与副作用检查

- 未修改 `app/solve.go`、`pkg/screen`、模型、Prompt、题库、Mobile、配置、依赖或 Wails bindings。
- Vite 与 Wails 构建未留下 tracked 生成噪音。
- `build/bin/**` 保持 Git ignored；未使用 `git add -f`。
- Release 资产未修改，未创建 Tag 或 Release。

## 结论

Change 035 的必要自动验证、用户 GUI 验收、稳定版晋升和晋升后独立启动验证均通过。race 补充项因既有 Windows 环境错误保持 BLOCKED，不作为本 Change 必要关闭门禁。功能提交 `e23f86a` 已完成；本验证记录与其余治理资产随 `chore: add change 035 governance records` 提交落地后完成双提交并关闭 Change。
