# Change 027 回滚方案

日期：2026-10-25
状态：未使用

## 回滚触发条件

- 新测试或人工验收证明修复破坏 KeepContext 或分段续接语义。
- 修复导致既有 Change 024/025/026 回归失败。
- 用户判定修复方向错误。

## 白名单（仅可回滚以下文件）

| 文件 | 回滚方式 |
|---|---|
| `pkg/solution/solution.go` | 定向还原本次改动 |
| `pkg/solution/solution_test.go` / 新增测试文件 | 删除新增测试，还原改动 |
| `pkg/domain/prompts.go` | 定向还原新增 Rule |
| `pkg/domain/prompts_test.go` | 删除新增测试 |
| `frontend/wailsjs/**`（仅当有漂移） | 用固定 Wails v2.11.0 重生 |

## 禁止

- 禁止 `git reset --hard`、`git clean`、`git checkout .`、`git restore .`、
  `git add .`、`git add -A`。
- 不得触碰 Change 024/025/026 的未提交改动。
- 不得删除或迁移题库 `_记录`、截图、旧导出、用户配置。
- 不得覆盖正式 `Q-Solver.exe` 或既有验证程序。
- 不得删除 Change 027 文档与验证程序（标注为已回滚即可）。
- 未获授权不提交回滚。

## 回滚后核对

- [ ] `go test ./... -count=1` 恢复基线结果。
- [ ] 依赖文件哈希与 §基线一致。
- [ ] 受保护 EXE 哈希与基线一致。
- [ ] bindings 无漂移。
