# Change 027 备注

日期：2026-10-25

## 待办

- 确认用户是否开启 `KeepContext`（决定历史图片累积机制是否触发）。
- 用户核对 CCR 侧 `APMIX.AI-openai/claude-sonnet-4-6-free` 的实际上游与视觉透传。

## 过程记录

### 治理文档初稿错误（已修正）

`01-current-state.md` 初稿基于会话摘要推测了 `app/questionbank.go` 中的
`buildSystemPrompt`、`SealAndCreateNewSegment` 等符号，并断言“新分段不清空历史”。
读取真实源码后确认该断言**错误**：

- `buildSystemPrompt` 实际位于 `pkg/solution/solution.go:227`。
- `StartNextExam`（`app/questionbank_next_exam.go:74-78`）**确实**调用
  `a.solver.ClearHistory()`、`a.conversationRefs = nil`、`a.conversationEpoch++`。
- `app/solve.go:198-201` 按 epoch 拦截迟到提交。

教训：治理文档不得基于会话摘要推测代码符号，必须实际读取源文件。

## 待补充

### 已知限制：`wails build` 未直接使用

固定 Wails v2.11.0 CLI 在本机 Go 1.25.13 下执行 `wails build` 时报：

```
internal error: package "errors" without types was imported from "Q-Solver/pkg/common"
internal error: package "log" without types was imported from "Q-Solver/pkg/logger"
internal error: package "bytes" without types was imported from "Q-Solver/pkg/ImageUtil"
```

这是该 CLI 版本与本机 Go 版本的类型检查兼容问题，与本次改动无关
（改动前同样会复现）。改用等价的
`go build -tags desktop,production -o build/bin/...` 完成构建，
并在 `03-implementation.md`、`04-validation.md`、审计中如实记录该偏离。

`wails generate module` 可正常工作，bindings 已用固定 v2.11.0 重生。

### 未完成的验证

- race 仍为本机环境阻塞 `0xc0000139`，未获得有效 race 结论。
- 全部人工验收项目（Direct 三协议多图、CCR+DeepSeek 多图、
  KeepContext 连续多轮、`StartNextExam` 分段隔离、上游丢图显式报错）
  均需用户在 GUI 中执行，本会话未代为调用真实服务。

### 根因仍未闭合

本 Change 收敛了一个已确认的放大因子并补齐了视觉失败的可观测性，
但**未**解释「同一请求发往 Claude 与 DeepSeek，为何仅 Claude 异常」。
该问题指向 CCR 路由或该 alias 的实际上游视觉透传，须由用户在 CCR 侧核对。

## 2026-10-29 整合提交状态

- 用户批准 Change 024～031 一次性跨 Change 整合。
- 功能/测试提交：`2a8a506636496a1eb6c38662c6ef6eaccc1fe958`（Change 030 无业务源码，仅归档治理记录）。
- 本 Change 的治理档案纳入 `chore: archive changes 024-031 and define artifact lifecycle`。
- 本轮全量自动验证通过；原有 `NEEDS_USER` 与 race `BLOCKED` 项保持，不因提交自动关闭。
- 不创建 Tag、GitHub Release 或新 RC。
