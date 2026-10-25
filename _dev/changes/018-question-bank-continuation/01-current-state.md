# Change 018 现状快照

日期：2026-10-23
状态：事前审计完成。

## 三套独立历史

- `pkg/solution.Solver.chatHistory`：仅内存保存模型消息，无会话 ID、磁盘恢复和并发替换接口。
- `frontend/src/stores/solution.js`：仅进程内展示历史，流回调固定写 `history[0]`，没有稳定 sessionID/recordID。
- `pkg/questionbank.Session → Records`：持久化截图、答案、Model、Scene 和状态，但不保存原始 system、KeepContext 边界、实际上下文裁剪或消息序列。

## 关键风险

- `Store.Open` 会切换 current 并改状态，却不能把 ended/exported 恢复为可写，也不恢复 Solver；不能用于真正续接。
- `CancelCurrentTask` 当前会在 goroutine 真正退出前让 `HasRunningTask` 返回 false，存在切换后迟到回调污染新会话的窗口。
- 直接把 Records 灌入前端 history 会被后续流式输出固定写 `history[0]` 覆盖。
- 旧 Record 只能以 completed+非空 Answer 的截图/答案近似重建 user/assistant 轮次，不能声称无损。
- exported 只有一组 ExportedAt/ExportDirectory；再次导出需要保留完整历史。

## 可复用能力

- `Store.SnapshotByID` 无副作用读取；`Store.ReadRecordImage` 已有路径、格式、大小和像素安全边界。
- 最近 20 API 和 8 MiB detail 限制已在 `app/questionbank_history.go` 实现。
- `Store.saveSessionLocked` 使用图片与 JSON 原子写；导出目录已有 `_02/_03` 冲突策略。
- Change 016 已有 Mobile display epoch，只处理同步端清屏，不应与新 conversation epoch 混用。

## 数据边界

不得持久化 API Key、Token、完整 Config、system prompt、ResumeContent 或 thinking。历史 Answer 只可作为 assistant；截图只可作为 user。
