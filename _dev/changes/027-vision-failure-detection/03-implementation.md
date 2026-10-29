# Change 027 实施记录

日期：2026-10-25
状态：已实施（未提交）

## 基线哈希（Task 1 记录）

| 文件 | SHA-256 |
|---|---|
| `go.mod` | `ead5d49037577845346967f41fab9dc04138f75f500d09387b306ccce5738595` |
| `go.sum` | `22174ab9fc52ddc6ac87f6862961a0c5dddcb254bb05b0e30db300eaefbb7014` |
| `frontend/package.json` | `0595e7f0957dd72c62d42dc1a310498d4ae2a3ab6a5e1efb198ac6a2f06b846a` |
| `frontend/package-lock.json` | `827a54f1049d6d143ead365c00c6c0870d6f9eabde57f77ec0926c061f63d725` |

## 受保护 EXE 基线（构建后已比对一致）

| 文件 | 大小 | SHA-256 |
|---|---|---|
| `Q-Solver.exe` | 23,126,528 | `5dcabb17933bfa9ab4dc15f289b4b8e50af99879c316b58831b09bbb4e39aa11` |
| `Q-Solver-change025-multi-protocol-validation.exe` | 44,017,664 | `589443f464f0e3f0c8adf5c6100beed81d5380d1c2ac98eb54b7f0a0bf6efa37` |
| `Q-Solver-change026-ccr-gateway-validation.exe` | 44,032,512 | `5584d5448e6808a230a4d01e35e098938a92690beb3b1f3d149129b0d9771fe1` |

## Task 2：测量结果（诊断）

新增 `pkg/solution/conversation_images_test.go`，用既有 `fakeProvider`
捕获 `GenerateContentStream` 收到的 `[]llm.Message`。

实测输出：

| 场景 | 请求画像 | 图片总数 |
|---|---|---|
| KeepContext=false，单轮 2 图 | `[system:0 user:2]` | 2 |
| KeepContext=true，第 1 轮 | `[system:0 user:2]` | 2 |
| KeepContext=true，第 2 轮 | `[system:0 user:2 assistant:0 user:2]` | 4 |
| KeepContext=true，第 3 轮 | `… user:2 assistant:0 user:2` | 6 |
| KeepContext=true，第 13 轮 | 10 轮历史 + 当前轮 | **22** |

**确认**：KeepContext=true 时旧截图随历史逐张重发，图片数线性增长；
上限为 10 轮历史 + 当前轮 = 11 轮（`MaxScreenshots=3` 时理论上限 33 张）。
**当前轮截图始终完整**，不存在当前轮丢图。

## Task 3：发送侧历史预算（本次唯一的行为变更）

**改动**：`pkg/solution/solution.go`

```go
messagesToSend = append(messagesToSend, trimHistoryForSend(history)...)
```

新增发送侧纯函数：

```go
func trimHistoryForSend(history []llm.Message) []llm.Message {
	maxMessages := (MaxConversationRounds - 1) * 2
	...
}
```

理由：`trimChatHistoryLocked` 在发送前把 history 裁到
`MaxConversationRounds*2` 条消息，但当前轮随后才追加，导致请求实际携带
**11 轮**图片。发送侧再预留 1 轮，使请求总轮数恰为 `MaxConversationRounds`。

未改动：`trimChatHistoryLocked`、`MaxConversationRounds` 常量、
`SnapshotConversation` / `ReplaceConversation` 契约、KeepContext 开关语义、
当前轮截图、user/assistant 配对不变式。

RED → GREEN 证据：

```
RED  : 历史图片应不超过 18 张（9 轮），实际 20 张；请求总图片 22 张
GREEN: 最后一轮请求：图片总数=20，其中历史 18 张、当前轮 2 张
```

## Task 4：视觉失败显式指令

**改动**：`pkg/domain/prompts.go`，在 `<BehaviorRules>` 中紧跟既有防编造规则新增：

> 如果无法读取或看不到当前用户消息中的图片，必须明确说明无法识别图片，
> 不得依据题号摘要、历史对话或任何文本提示生成、猜测或补全题目。

RED → GREEN 证据：新增 `TestSystemBehaviorPromptRequiresExplicitVisionFailure`
先因三个片段均缺失而 FAIL，加规则后 PASS。

## 文件清单（Change 027 白名单）

| 文件 | 变更 |
|---|---|
| `pkg/solution/solution.go` | 新增 `trimHistoryForSend`，发送侧调用 |
| `pkg/solution/conversation_images_test.go` | 新增（测量 + 预算断言） |
| `pkg/domain/prompts.go` | 新增一条 `<Rule>` |
| `pkg/domain/prompts_test.go` | 新增视觉失败测试 |
| `_dev/changes/027-*/**` | 治理文档 |
| `_dev/audits/027-*.md` | 审计 |
| `_dev/changes/INDEX.md` | 登记 |

**未改动**：`go.mod`、`go.sum`、前端依赖、前端源码、`app/`、`pkg/llm/`、
`pkg/questionbank/`、题库数据、用户配置。

## 观察：工作树中的非本次改动

`git diff` 显示 `pkg/solution/solution.go` 还包含**非本次**的日志改动
（`system 提示词` 与 `模型返回内容` 不再打印全文）。这些来自 Change 024/025/026
的未提交工作，本 Change 未触碰、未回退。

## 构建

- 使用从本地模块缓存编译的固定 Wails **v2.11.0** CLI（未使用 PATH 中的 v2.16.0）。
- `wails build` 在 v2.11.0 下报
  `internal error: package "errors" without types was imported from ...`，
  属该 CLI 与本机 Go 1.25.13 的兼容问题；改用等价的
  `go build -tags desktop,production` 完成构建。
- 产物：`build/bin/Q-Solver-change027-vision-failure-detection.exe`
  60,950,016 字节
  SHA-256 `f75e27398dfe3562ae7ac140bf33ec15d4ce6736ae50210d7981a1bb90c9c90a`
- 未覆盖 `Q-Solver.exe`、Change 025/026 验证程序或 `archive/`、`releases/`。

## bindings 审计

用固定 v2.11.0 CLI 重生 `frontend/wailsjs/go/*`：

- `App.js` / `App.d.ts`：无实质变化（仅缩进空白）。
- `models.ts`：新增 `ConnectionDraft`、`ProviderErrorDTO`、`ModelListResult`、
  `ModelProbeRequest`、`ModelProbeResult`。
- 相对 **HEAD** 的差异（新增 `ListModelsForDraft`、`ProbeModelConnection`、
  `RetryFailedQuestionBankRecords`）来自 Change 024/025/026，**不是**本 Change 引入。
- **本 Change 未新增任何 App 方法**，故 bindings 无本 Change 贡献。
