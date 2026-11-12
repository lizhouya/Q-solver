# 验证记录 04

## 当前状态

同步端五项提示实现、普通自动门禁和用户 GUI 验收均已完成。Change 040 原候选在 Change 041 叠加构建前按原哈希归档 transient；用户最终使用包含本功能的 Change 041 叠加候选验收通过，该候选已晋升根稳定版。race 为环境 `BLOCKED` 并持续披露，未发布。

## TDD 与自动验证

| 命令 | 结果 | 备注 |
|---|---|---|
| `node pkg/mobile/web_dom_test.mjs` 红灯 | EXPECTED FAIL | 旧实现固定 `112px`，新断言要求实际高度 146px + 32px = 178px |
| `node pkg/mobile/web_dom_test.mjs` | PASS | 五项顺序、实际绑定、满额键帽、动态高度、resize、重置和隐藏清理通过 |
| `go test ./app ./pkg/mobile -count=1` | PASS | 后端提示契约和真实 SSE→DOM 回归通过 |
| `frontend/*_test.mjs` | PASS | 19 个 Node 回归全部通过 |
| `npm --prefix frontend run build` | PASS | Vite 4.5.14，135 modules；仅既有大 chunk warning |
| `go test ./... -count=1` | PASS | 全仓 Go 测试通过 |
| `go vet ./...` | PASS | 无输出 |
| `go build ./...` | PASS | 退出码 0 |
| `go mod verify` | PASS | `all modules verified` |
| VS Code diagnostics | PASS | Warning 及以上为空 |
| `git diff --check` | PASS | 无空白错误 |
| `go test -race ./app ./pkg/mobile -count=1` | BLOCKED | 启动阶段返回 Windows `exit status 0xc0000139` |

## 候选与晋升

| 项目 | 结果 |
|---|---|
| Change 040 原候选归档 | `archive/transient/Q-Solver-candidate-change040-before-choice-consistency.exe` |
| 原候选 SHA-256 | `51228E51DD744D4E5CD74000FA059203464116DE396F0269CD98BA904178B2C5` |
| 最终叠加候选 / 根稳定版 | `build/bin/Q-Solver.exe` |
| 根稳定版大小 | 42,558,464 字节 |
| 根稳定版 SHA-256 | `2885358371510B827ACE560C354FEE8775BECEB47555D7CE654B0FBE3EB1F15B` |
| 验证归档 | `archive/validation/change-041/Q-Solver-candidate-change041.exe` |
| 被替代稳定版 | `archive/superseded/change-041/Q-Solver-stable-before-change041.exe`，SHA-256 `DC8B…F15B` |
| Wails runtime | v2.11.0 |
| Release | 7 个文件构建前后不变 |
| `testing/current/` | 已清空 |

## 用户验收

用户于 2026-11-12 确认验证通过，包含本 Change 的同步端 F8–F12 五项提示与当前设置保持一致。该确认满足本 Change 的必要 GUI 验收。

## 结论

状态：**实现完成 / 普通自动门禁 PASS / 用户 GUI 验收 PASS / race BLOCKED 已披露 / 功能提交 `09dd485` / 双提交完成 / 本地稳定版已晋升 / 未推送未发布**。
