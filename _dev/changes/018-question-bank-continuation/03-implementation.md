# Change 018 实现记录

状态：业务实现与普通自动验证完成；Windows GUI NEEDS_USER；race BLOCKED；未提交、未推送、未发布。

## 时间线

- 2026-10-23：用户确认真正续接、`ended`/`exported`/`paused`、使用当前配置和暂停当前活动段四项决策。
- 2026-10-23：完成三套历史、Store/Solver/前端流、任务取消窗口和旧数据缺口审计。
- 2026-10-23：按 TDD 完成持久快照、可替换 Solver、App 原子编排、桌面事件 envelope、session-aware 前端与主界面历史截图。
- 2026-10-23：补齐发送前简历确认、题库最终落盘失败回滚、启动恢复、Store 三阶段故障注入、补偿失败写阻断与完成态 Retry。
- 2026-10-23：正式生成 Wails bindings，完成全量普通自动验证和独立 Windows 验证构建。
- 2026-10-23：根据 GUI 验收截图修复历史分段操作区；桌面端使用固定操作列和纵向等宽按钮，窄屏使用全宽单列，并补充布局回归与新版验证程序。

## 后端

### Store 与持久化

- `Session.Conversation` 保存最近最多 10 个 `{sessionId, recordId}` 引用，`Record.KeepContext` 保存新请求边界；旧 JSON 字段保持可选。
- `ConversationSnapshotByID` 对旧数据仅取本 Session 内 `completed && Answer != ""` 的最近 10 条并标记 `record-reconstructed`。
- `ResolveConversation` 复用安全图片读取，严格校验引用、状态、答案、PNG/JPEG、单图限制和 80 MiB 总上下文限制；任一引用损坏时整体失败，不部分恢复。
- `ActivateHistoricalSession` 固定执行“源 active→paused，再目标→active”。source、target、rollback 三阶段均有确定性故障注入测试。
- target 写失败且 source 补偿成功时恢复原状态；补偿也失败时，内存 current 对齐已落盘的 paused 源，并设置 `writeBlocked`。此后所有 Store 写 API 统一拒绝并提示停止操作、保留现场；List、Snapshot 与图片读取仍可用。
- `FinishWithConversation` 使用 working clone 同时提交 Record 终态与 ConversationSnapshot。失败时 current 仍显示 running/unsaved，另存 `pendingCompletion`；在 Retry 前禁止其他写操作，`Flush` 恢复后原子提交完成态。
- exported 分段续接保留 `ExportedAt`、`ExportDirectory` 和全部 `Exports`；追加、再次结束和导出产生 `_02` 目录，旧目录内容保持不变。

### Solver、App 与任务

- Solver 历史仅含 user/assistant 对，支持深拷贝 Snapshot/Replace、最多 10 轮和 generation 隔离；历史答案不会成为 system。
- 每次新请求仍按当前 Model、Domain/Scene Prompt、ResumeContent 构造 system。恢复历史只影响非 system 对话。
- `TaskCoordinator.HasInFlightTask()` 从 Start 到 goroutine Complete 全程计数；Cancel 后未退出仍拒绝续接。
- `ContinueQuestionBankSession` 在 `conversationMu` 内同步切换 Solver、Store current、App refs 和 epoch；Store 拒绝时恢复旧 Solver，refs 与 epoch 不推进。
- 应用启动时恢复当前 `active`/`paused` Session：persisted 快照精确恢复；legacy 显示固定警告；图片或引用损坏时 fail closed，Solver 与 refs 清空。
- 模型成功但题库最终落盘失败时，Solver 回滚到请求前快照，App refs 不推进，并发送 typed `solution-error`。用户修复目录后点击“重试保存”，Store 提交 pending completion，App 随即恢复 Solver 与 refs。
- 简历路径存在但正文未解析时，`TriggerSend` 在消费截图和启动任务前发出 `resume-warning-required`；取消不清截图，确认后才真正启动。

## 前端与事件

- `solution` store 使用稳定 `sessionId + recordId + epoch` 路由；A→B 后 A 的迟到 chunk、error、solution 和图片响应均被拒绝。
- 题库 `ended`、`exported`、`paused` 行显示“继续此分段”；`discarded` 永久只读。成功后先激活主界面会话，再刷新并关闭题库面板；失败不改变当前界面。
- 主界面按 Record 展示 1～3 张按需读取的历史截图、完整答案、旧 Model/Scene、KeepContext 与重建警告；查看只读会话时后台流仍只写真实 writable 会话。
- typed `request-start` 到达后才清 ScreenshotDock，并清除上一轮全局 ErrorView。
- desktop conversation epoch 仅进入 `desktop-solve-event`；Mobile 继续使用原事件名、payload 和独立 display epoch。
- 最近 20 分段列表不再按每行按钮内容计算操作列：桌面端固定 `7.5rem` 操作列，继续与重新导出按钮纵向等宽；分段标题/时间与状态上下排列；`560px` 及以下改为卡片在上、操作区在下。按钮状态矩阵、事件与可访问性属性保持不变。

## 生成物

- 使用固定 `_dev/changes/015-rc-release/tmp/tools/wails.exe` v2.11.0 与 `GOTOOLCHAIN=go1.25.13` 生成 bindings。
- 原续接验证程序：`build/bin/archive/validation/Q-Solver-change018-continuation-validation.exe`；SHA-256 `775196521b00528824e4620cc4df1ad80c8bbb78c7820d40c626b528f67fe8e0`。
- 最新布局验证程序：`build/bin/archive/validation/Q-Solver-change018-layout-validation.exe`；SHA-256 `853e64c3ccd59101317d0fc4e92414a10f1d12695deed9694b0d689ee2e07343`。
- validation 目录 9 个 EXE 的 LF-only `SHA256SUMS` 全部校验 `OK`。
- 未覆盖 `build/bin/Q-Solver.exe`，未停止或替换任何正在运行的旧验证程序。

## 实施约束

未调用真实模型，未上传测试截图，未读取或输出 API Key/Mobile Token，未删除、迁移或覆盖题库原始记录和旧导出；未执行 commit、push、Tag 或 Release。
