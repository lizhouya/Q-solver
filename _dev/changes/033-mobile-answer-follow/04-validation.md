# Change 033：验证记录

## TDD 与聚焦验证

| 检查 | 结果 | 说明 |
|---|---|---|
| 红灯 `node pkg/mobile/web_dom_test.mjs` | PASS（预期失败） | 首个失败为“快捷键区域必须使用可折叠 details” |
| 绿灯 `node pkg/mobile/web_dom_test.mjs` | PASS | 折叠、计数、自动滚底、历史保护、清屏失效和 40% 控制断言通过 |
| `go test ./pkg/mobile -count=1` | PASS | 更新旧 70% 嵌入页完整性哨兵后通过 |
| `gofmt -d pkg/mobile/bridge_server_test.go` | PASS | 无格式差异 |

## 全量自动验证

| 命令 | 结果 | 说明 |
|---|---|---|
| 全部 `frontend/*_test.mjs` | PASS | 14 份桌面前端回归通过 |
| `npm --prefix frontend run build` | PASS | Vite v4.5.14，132 modules transformed；仅有既有 chunk size warning |
| `go test ./... -count=1` | PASS | 全部相关包通过 |
| `go vet ./...` | PASS | 无输出 |
| `go build ./...` | PASS | 无输出 |
| `go mod verify` | PASS | `all modules verified` |
| 范围内 `git diff --check` | PASS | 仅有 Git 的 LF→CRLF 工作区提示，无 whitespace error |

## 候选构建

- 原 Change 033 候选已归档到 `build/bin/archive/transient/`；原大小为 42,334,208 字节，SHA-256 为 `19495E67DE3E84781E2E8598EE016EAB4AB4FD5599786817DD3F9EBE757016E9`。
- 当前最新组合验证入口为 `build/bin/testing/current/Q-Solver-candidate-change034.exe`，同时包含 Change 033 与 Change 034。
- 测试前稳定版副本仍为 `build/bin/testing/current/Q-Solver-stable-before-test.exe`。
- 根目录稳定版保持为 `build/bin/Q-Solver.exe`，未覆盖、未晋升。
- `testing/current/SHA256SUMS` 对两个文件均验证通过。
- `go version -m`：候选链接 `github.com/wailsapp/wails/v2 v2.11.0`。

## 构建保护

| 保护项 | 结果 |
|---|---|
| `go.mod` / `go.sum` | 哈希不变 |
| `frontend/wailsjs/go/**` | 哈希不变 |
| `frontend/wailsjs/runtime/**` | 清除 v2.16 Notification 生成噪音后哈希恢复 |
| `frontend/dist` 聚合哈希 | 不变 |
| `build/bin/releases/**` 聚合哈希 | 不变 |
| Git staged diff | 为空 |
| Git index 语义 tree | 与 `HEAD^{tree}` 一致：`3ffa34673924d76f4c3ac5ff56347d2a222861fa` |

## 独立启动与人工验收

- 首次独立启动曾被当时运行中的稳定版固定单实例锁阻塞，未将该次退出冒充启动成功。
- 2026-10-29，用户正常退出旧进程后启动最新组合候选，并明确确认验证通过：快捷键折叠、答案自动滚底、历史浏览保护和 40% 键盘滚动的人工验收记为 `VERIFIED_BY_USER`。
- 根目录稳定版未因本次验收自动晋升。

## 结论

代码、DOM、Mobile Go、全量 Go、桌面前端、候选构建和用户 GUI 人工验收通过。原 Change 033 候选已归档，当前组合候选验证通过；根目录稳定版保持不变。
