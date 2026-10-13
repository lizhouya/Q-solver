# 验证记录 — 04 · 005-keepcontext-backend-wiring

> **状态**：验证完成（2026-10-13）
> **阶段**：Implementation 自动验证完成

## 测试矩阵（T-KC 系列）

| ID | 场景 | 真实结果 |
|---|---|---|
| T-KC-01 | `KeepContext=false` 连续两次 Solve | ✅ 第二次仅含 `system,current user`；旧问答未进入请求，history 为空 |
| T-KC-02 | `KeepContext=true` 第二次 Solve | ✅ Provider 收到 `system,U1,A1,U2` |
| T-KC-03 | 成功完成两轮 | ✅ `chatHistory` 为 `system,U1,A1,U2,A2` |
| T-KC-04 | 超过 `MaxConversationRounds` | ✅ 丢弃最旧轮，精确保留最近 N 轮；system 保持首位 |
| T-KC-05 | `ClearHistory()` 后再次 Solve | ✅ 下一次请求仅含 `system,current user` |
| T-KC-06 | Provider error / 空响应 | ✅ Solve 返回 false，失败轮未写入 history，旧成功历史保持不变 |
| T-KC-07 | System Prompt 从 A 更新为 B | ✅ 第二轮 Provider 输入仅含唯一 system，包含 B、不含 A，同时保留旧问答 |
| T-KC-08 | KeepContext true → false | ✅ false 请求不携带旧 history，并清空残留 |
| T-KC-09 | KeepContext false → true | ✅ 从新的当前轮开始建立 history，不恢复旧轮次 |
| T-KC-10 | multipart user message | ✅ history 保留实际 `Parts`、image 类型和原始 base64 |

## 自动验证（真实结果）

| 命令 | 结果 | 备注 |
|---|---|---|
| `go test ./pkg/solution/... -count=1` | ✅ PASS | `ok Q-Solver/pkg/solution 0.607s` |
| `go test ./pkg/llm/... -count=1` | ✅ PASS | 无测试文件，包加载成功 |
| `go test ./app/... -count=1` | ✅ PASS | 最终复验：`ok Q-Solver/app 0.826s` |
| `go test ./pkg/... -count=1` | ✅ PASS | 最终复验所有 `pkg` 包通过；`pkg/solution` 0.618s |
| `go build ./...` | ✅ PASS | exit 0，无输出 |
| `go vet ./pkg/solution/...` | ✅ PASS | exit 0，无输出 |
| `npm --prefix frontend run build` | ✅ PASS | Vite 118 modules，8.60s；仅既有 chunk > 500 kB 警告 |
| `git diff --check` | ✅ PASS | exit 0；仅 LF→CRLF 工作区兼容性提示，无空白错误 |

## 边界核查

允许范围：

- `pkg/solution/solution.go`
- `pkg/solution/solution_test.go`
- `_dev/changes/005-keepcontext-backend-wiring/`

确认未修改：

- `frontend/`
- `app/`
- `pkg/mobile/`
- `pkg/config/`
- `pkg/window/`
- `pkg/tray/`
- `pkg/shortcut/`
- LLM Provider 实现
- `go.mod` / `go.sum`
- `frontend/wailsjs/`

## 验证结论

Change 005 的 KeepContext 后端接线符合 D-005-001～D-005-007：OFF 完全独立且不保存，ON 携带并保存完整成功轮，失败轮不写入，当前 System Prompt 唯一，历史裁剪复用既有实现，multipart 截图保持原结构。
