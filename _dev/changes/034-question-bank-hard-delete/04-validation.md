# Change 034：验证记录

## TDD 红灯

| 测试 | 预期失败 |
|---|---|
| `go test ./pkg/questionbank -run 'TestDeleteCurrent' -count=1` | `Store.DeleteCurrent undefined` |
| `go test ./app -run 'TestDiscardQuestionBankSegment' -count=1` | 目录仍存在；确认文案仍为软清空 |
| `node frontend/questionbank_history_regression_test.mjs` | 缺少永久删除警告 |
| in-flight/exporting 补强测试 | 旧实现错误地允许删除 |
| 部分删除失败补强测试 | 旧实现错误地声称原记录已恢复 |

## 聚焦绿灯

- `go test ./pkg/questionbank -run 'TestDeleteCurrent' -count=1`：PASS。
- `go test ./app -run 'TestDiscardQuestionBankSegment' -count=1`：PASS。
- `node frontend/questionbank_history_regression_test.mjs`：PASS。

覆盖：

- active、ended、legacy paused 物理删除；
- `session.json`、多张截图 assets、分段目录和历史列表移除；
- 下一题建立不同新分段；
- running、dirty、pending completion、exported、旧 discarded 拒绝；
- 删除失败恢复目录和 current；
- 部分删除无法完整恢复时阻止后续写入；
- 未确认、in-flight、exporting 和 Store 拒绝时 App 运行时零清理；
- 成功后 Solver/refs 清空、epoch 增加，未发送截图和 Mobile 不变；
- 清空/结束成功后前端清实时历史，失败不清；
- 永久删除文案和导出原调用链。

## 全量自动验证

| 命令 | 结果 |
|---|---|
| `go test ./... -count=1` | PASS，全部相关包通过 |
| `go vet ./...` | PASS，无输出 |
| `go build ./...` | PASS，无输出 |
| `go mod verify` | PASS，`all modules verified` |
| 全部 `frontend/*_test.mjs` | PASS，14 份回归通过 |
| `npm --prefix frontend run build` | PASS，Vite v4.5.14，132 modules transformed；仅既有 chunk size warning |
| 范围内 `git diff --check` | PASS，仅 Git LF→CRLF 提示，无 whitespace error |

## 数据与人工验收边界

- 所有物理删除自动测试只使用 `t.TempDir()`，未读取或删除用户现有题库。
- 已生成组合候选，包含 Change 033 与 Change 034；不包含尚未实施的 ScreenshotDock 快捷键显示修复。候选现归档于 `build/bin/archive/validation/change-034/`。
- 候选大小：42,345,472 字节；SHA-256：`C033EAB4844721EB233841915C82583D213095965357E77091738D9FBC21B0B8`。
- 候选实际链接 Wails runtime v2.11.0；依赖、bindings/runtime、Frontend dist、Release、测试区和归档 SHA-256 保护检查通过。
- 构建时本机 Wails CLI v2.16.0 注入的 Notification runtime 噪音已按差异精确清除，受保护文件哈希恢复。
- 原 Change 033 候选已双副本归档到 `build/bin/archive/transient/`；当时运行中的进程未被中断。
- 2026-10-29，用户正常退出旧 Change 033 进程后启动 Change 034 组合候选，并明确确认验证通过；测试分段永久删除警告、删除结果、“结束本段”保留/清理行为和既有导出流程的 GUI 验收记为 `VERIFIED_BY_USER`。
- 人工验收使用用户明确选择的可删除测试分段；未把自动测试扩展到用户现有题库。

## 稳定版晋升

- 2026-10-29，用户明确要求更新版本。
- 旧稳定版（42,331,648 字节，SHA-256 `C1F1873F054520F0F45FADFCCBCC087056D1556381DD1EFC66D4CD3748085F7B`）已归档到 `build/bin/archive/superseded/change-034/`。
- 通过候选已归档到 `build/bin/archive/validation/change-034/`，两个目录的 `SHA256SUMS` 全部验证通过。
- 根目录 `build/bin/Q-Solver.exe` 已更新为 42,345,472 字节，SHA-256 `C033EAB4844721EB233841915C82583D213095965357E77091738D9FBC21B0B8`；根目录只有该一个 EXE。
- 晋升后程序持续运行 8 秒，检测到 WebView2，Node/Vite 子进程为 0；正常窗口关闭成功，退出码 0，无残留 Q-Solver 进程。
- `testing/current/` 已清空；`build/bin/releases/` 7 个文件聚合哈希前后均为 `D7B12A61D23F575086F08280FEE8F0D85CB0D94B4413BE074D5D2F6AC517DC4B`。

## 结论

代码、自动验证、候选构建、用户 GUI 人工验收和稳定版晋升验证通过。Change 034 已成为根目录最新稳定版。
