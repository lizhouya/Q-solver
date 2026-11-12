# 验证记录 04

## 当前状态

选择题映射优先与一致性约束、普通自动门禁和用户实际模型复验均已完成。固定 Wails v2.11.0 叠加候选通过保护核对并已晋升根稳定版。race 为环境 `BLOCKED` 并持续披露，未发布。

## TDD 红灯

| 命令 | 结果 | 证据 |
|---|---|---|
| `go test ./pkg/domain -run 'TestSystemBehaviorPromptChoiceConsistencyContract|TestSpeedPromptPreservesChoiceMappingCheck' -count=1` | EXPECTED FAIL | 全局八个目标片段和快速模式六个目标片段均缺失 |

## 自动验证

| 命令 | 结果 | 备注 |
|---|---|---|
| 提示词聚焦契约 | PASS | 映射优先、四方一致、待核对和快速模式约束通过 |
| `go test ./pkg/domain -count=1` | PASS | domain 全包通过 |
| `frontend/*_test.mjs` | PASS | 19 个 Node 回归全部通过 |
| `npm --prefix frontend run build` | PASS | Vite 4.5.14，135 modules；仅既有大 chunk warning |
| `go test ./... -count=1` | PASS | 全仓 Go 测试通过，包含 Change 040 回归 |
| `go vet ./...` | PASS | 无输出 |
| `go build ./...` | PASS | 退出码 0 |
| `go mod verify` | PASS | `all modules verified` |
| VS Code diagnostics | PASS | Warning 及以上为空 |
| `git diff --check` | PASS | 无空白错误 |
| `go test -race ./pkg/domain -count=1` | BLOCKED | 启动阶段返回 Windows `exit status 0xc0000139` |

## 候选与晋升

| 项目 | 结果 |
|---|---|
| 根稳定版 | `build/bin/Q-Solver.exe` |
| 大小 | 42,558,464 字节 |
| SHA-256 | `2885358371510B827ACE560C354FEE8775BECEB47555D7CE654B0FBE3EB1F15B` |
| 验证归档 | `build/bin/archive/validation/change-041/Q-Solver-candidate-change041.exe`，哈希与根稳定版一致 |
| 被替代稳定版 | `build/bin/archive/superseded/change-041/Q-Solver-stable-before-change041.exe` |
| 被替代稳定版 SHA-256 | `DC8B0E248B82972F92A6EC45152F0D9B4F93450DC335A61185476129D0D357EC` |
| Wails runtime | v2.11.0 |
| 保护边界 | 831 个保护文件、175 个构建前产物不变 |
| Release | 7 个文件不变 |
| `testing/current/` | 已清空 |
| Q-Solver 残留进程 | 0 |

## 用户验收

用户于 2026-11-12 确认验证通过，覆盖实际选择题中选项字母、选项文本、解析结论与最终汇总一致性，以及叠加的 Change 040 同步端提示。该确认满足必要人工验收。

## 结论

状态：**实现完成 / 普通自动门禁 PASS / 用户模型复验 PASS / race BLOCKED 已披露 / 功能提交 `ac8af04` / 双提交完成 / 本地稳定版已晋升 / 未推送未发布**。
