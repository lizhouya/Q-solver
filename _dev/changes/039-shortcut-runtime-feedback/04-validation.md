# 验证记录 04

## 当前状态

历史配置迁移、F8–F12 Dock 顺序提示、普通自动门禁、revision C 用户 GUI 复验和本地稳定版晋升均已完成。首版候选用户 GUI 验收为 FAIL 并归档 failed；revision B 因新增 Dock 验收项归档 transient；revision C 验收通过并晋升。race 为环境 `BLOCKED` 并持续披露，未发布。

## 自动验证

| 命令 | 结果 | 说明 |
|---|---|---|
| `go test ./pkg/shortcut ./pkg/config -count=1` | PASS | 单 F 键解析、五项历史迁移、F10 释放/接管、冲突拒绝和磁盘不写通过 |
| `node frontend/solve_retry_cancel_fast_regression_test.mjs` | PASS | 新 `vkCode` 契约通过 |
| `frontend/*_test.mjs` | PASS | 19 个 Node 回归全部通过；Dock 五项实际绑定顺序严格为 F8→F12，满三张保留 F8 |
| `go test ./... -count=1` | PASS | 全仓 Go 测试通过 |
| `go vet ./...` | PASS | 无输出，退出码 0 |
| `go build ./...` | PASS | 退出码 0 |
| `go mod verify` | PASS | `all modules verified` |
| `npm --prefix frontend run build` | PASS | Vite 4.5.14，135 modules；仅既有大 chunk warning |
| VS Code diagnostics | PASS | Warning 及以上为空 |
| `go test -race ./pkg/shortcut ./pkg/config -count=1` | BLOCKED | 两包启动阶段均为 `exit status 0xc0000139` |

## 正式配置只读迁移验证

- 修复前：16 项快捷键中 5 项 `keyName` / `vkCode` 不一致。
- 受影响 action：`cancel_solve`、`clickthrough`、`delete`、`send_fast`、`toggle_window`。
- 使用当前源码只读加载后：`mismatched=0`。
- 加载前后加密配置文件字节一致：`diskUnchanged=true`。
- 未输出按键编码、连接配置或凭据；仓库外临时探针已删除。

## 候选验证

| 项目 | 结果 |
|---|---|
| 首版候选 | `E4C5…D5C2`，用户 GUI FAIL，已归档 failed |
| revision B | `C9F7…AC8E`，因新增 Dock 验收项归档 transient |
| revision C 根稳定版 | `build/bin/Q-Solver.exe` |
| revision C 大小 | 42,556,928 字节 |
| revision C SHA-256 | `DC8B0E248B82972F92A6EC45152F0D9B4F93450DC335A61185476129D0D357EC` |
| 被替代稳定版归档 | `archive/superseded/change-039/Q-Solver-stable-before-change039.exe`，SHA-256 `8A86…D220` |
| revision C 验证归档 | `archive/validation/change-039/Q-Solver-candidate-change039.exe`，哈希与根稳定版一致 |
| Wails runtime | v2.11.0 |
| 依赖/bindings/dist | 构建前后保护哈希不变 |
| Release | rc.1、rc.2、rc.3 哈希不变 |
| revision C 隔离启动 | PASS：创建窗口、退出码 0、无 Q-Solver 残留 |
| revision C 正式 Profile GUI | PASS：用户于 2026-11-12 确认快捷键行为、设置一致性和 Dock F8–F12 顺序通过 |
| 晋升结果 | 根稳定版与 revision C 哈希一致；`testing/current/` 已清空；Release 未变 |

## 用户复验结果

用户于 2026-11-12 确认 revision C 验证通过，覆盖快捷键设置与实际行为一致、F10 快速解题、F12 删除截图，以及截图 Dock 按 F8、F9、F10、F11、F12 顺序完整显示。该确认满足本 Change 的必要 GUI 验收。

## 结论

状态：**revision C 实现完成 / 普通自动验证 PASS / 正式配置内存迁移验证 PASS / Dock F8–F12 顺序回归 PASS / 用户 GUI 复验 PASS / race BLOCKED 已披露 / 本地稳定版已晋升 / 功能提交 `fbae933` / 治理提交与推送已授权 / 未发布**。
