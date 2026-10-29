# Change 027 实施后审计

日期：2026-10-25
范围：`027-vision-failure-detection`

## 1. 变更边界

本 Change 对业务源码的改动**恰好两处**：

| 文件 | 变更 | 行数 |
|---|---|---|
| `pkg/solution/solution.go` | 新增 `trimHistoryForSend` 并在发送侧调用 | +17 / -1 |
| `pkg/domain/prompts.go` | `<BehaviorRules>` 新增一条 `<Rule>` | +1 |

新增测试：`pkg/solution/conversation_images_test.go`、
`pkg/domain/prompts_test.go` 增补一个用例。

**未改动**：`app/`、`pkg/llm/`、`pkg/config/`、`pkg/questionbank/`、
任何前端源码、`go.mod`、`go.sum`、前端依赖。

## 2. 与 Change 024/025/026 的隔离

`git diff` 中 `pkg/solution/solution.go` 另含两处**非本次**日志改动
（不再打印 system 提示词全文与模型返回内容全文）。经核对，它们属于
Change 024/025/026 的未提交工作，本 Change 未触碰、未回退、未清理。

工作树中其余已修改文件（`app/llm.go`、`frontend/src/**`、
`frontend/*_regression_test.mjs` 等）均为前序 Change 的未提交内容。

## 3. 语义正确性

### 3.1 发送侧裁剪不污染存储

`trimHistoryForSend` 返回 `history` 的切片视图，**不修改** `s.chatHistory`。
既有 `TestTKC04TrimKeepsRecentRounds` 直接断言
`len(s.chatHistory) == MaxConversationRounds*2`，在修复后仍 PASS，
证明存储层契约未被破坏。

### 3.2 配对不变式

裁剪按「从尾部取偶数条」并保留 `start%2==0` 修正，
与既有 `trimChatHistoryLocked` 使用同一不变式，
不会产生孤立的 user 或 assistant 消息。

### 3.3 当前轮不受影响

当前轮消息在裁剪**之后**追加，且断言
`TestKeepContextRequestDoesNotExceedHistoryRoundBudget` 明确校验
当前轮图片数等于 `shotsPerRound`。

### 3.4 KeepContext 语义保持

请求总轮数由 11 收敛到 10，仍保留多轮上下文；
`KeepContext=false` 路径完全未改动（实测仍为 1 轮）。

## 4. 未宣称根因

同一请求同时发往 Claude 与 DeepSeek，仅 Claude 异常。
因此本 Change 修复的是**已确认的放大因子与可观测性缺口**，
不是已证实的唯一根因。CCR 侧因子保持 `NEEDS_USER`。

## 5. 构建与产物

- 使用从本地模块缓存编译的固定 Wails **v2.11.0** CLI。
- `wails build` 在 v2.11.0 + Go 1.25.13 下报
  `internal error: package "errors" without types was imported from ...`，
  为该 CLI 与本机 Go 版本的兼容问题；改用等价
  `go build -tags desktop,production` 构建成功。
  **此偏离已在 `03-implementation.md` 与 `04-validation.md` 中记录，不隐藏。**
- 产物 60,950,016 字节，SHA-256
  `f75e27398dfe3562ae7ac140bf33ec15d4ce6736ae50210d7981a1bb90c9c90a`。
- 未覆盖 `Q-Solver.exe`、Change 025/026 验证程序、`archive/`、`releases/`。

## 6. 保护哈希核对

`go.mod`、`go.sum`、`frontend/package.json`、`frontend/package-lock.json`、
`build/bin/Q-Solver.exe`、`build/bin/Q-Solver-change026-ccr-gateway-validation.exe`
全部与基线一致。

## 7. 审计结论

| 维度 | 结论 |
|---|---|
| 变更范围 | 收敛，仅 2 处业务改动 |
| 前序 Change 隔离 | 保持，未清理未回退 |
| 语义正确性 | 存储层契约与配对不变式保持 |
| 数据安全 | 未触碰题库、截图、导出、用户配置 |
| 网络与隐私 | 零真实调用，零密钥输出 |
| 根因声明 | 未过度宣称，CCR 因子留 `NEEDS_USER` |
| race | `BLOCKED`（环境，非本次引入） |
| Git 授权 | 未暂存、未提交、未推送、未 Tag、未 Release |
