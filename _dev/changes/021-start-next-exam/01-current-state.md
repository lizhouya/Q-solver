# Change 021 现状快照

日期：2026-10-24
状态：业务源码修改前快照。

## 1. 导出与分段

- `frontend/src/stores/questionbank.js::submitExport()` 在 `ExportQuestionBankSegment` 返回后只表示后台导出已启动；没有结构化 task ID 或成功终态。
- `app/questionbank.go` 的实际成功点位于后台 `store.ExportSession(...)` 返回 nil；`organizing=false` 同时覆盖成功、失败和取消。
- `pkg/questionbank/store.go::ensureWritableLocked()` 在 current 为 ended/exported/discarded 且无 dirty 时，于下一次 `BeginWithContext` 懒创建新 Session。
- `SealCurrent()` 对 active/paused 封存，对 exported/discarded 返回提示；`DiscardCurrent()` 会把记录标记为 discarded，不适合作为“下一套”。

## 2. 三套历史与清理

- Solver 对话位于 `pkg/solution.Solver.chatHistory`，已有线程安全 `ClearHistory()`。
- 桌面历史位于 `frontend/src/stores/solution.js`，`resetLiveHistory()` 会清 history、浏览目标、typed/legacy 流目标和流式状态。
- 题库持久化保持 `Session -> Records -> Questions`，不得因桌面清理而删除。
- `App` 持有 `screenshotBuffer`、`conversationRefs` 和 `conversationEpoch`；Mobile 使用独立状态及 `clearSeq`。

## 3. 并发与锁

- 历史续接使用 `conversationMu → TaskCoordinator 查询 → questionBank.mu → Store.mu → Solver.mu`。
- `TaskCoordinator.HasInFlightTask()` 比 `HasRunningTask()` 更严格，可识别已取消但 goroutine 尚未退出的任务。
- `questionBankService` 使用 `captures`、`organizing`、`operation`、`closing` 表示题库后台活动。

## 4. 快捷键与确认模式

- `pkg/shortcut/service.go` 的 `clear_mobile_display` 只发 `request-clear-mobile-display`，前端再打开共享确认层；快捷键不直接 mutation。
- Windows Combo ID 按 VK 码排序；`Ctrl+Shift+N` 为 `78+160+162`。
- 现有配置迁移对 `clear_mobile_display` 有专用逻辑；新 action 需要更保守地保留显式空绑定、自定义绑定及被其他 action 占用的默认组合。

## 5. UI 与 bindings

- `ActionBar.vue` 已含“导出题库”和“结束本段”，适合新增可发现的“开始下一套”。
- `MobileActionConfirm.vue` 提供 Teleport 单例确认层样式参考，但本 Change 不复用 Mobile store 或 Mobile mutation。
- `frontend/src/services/api.js` 的 `callApp` 支持 bindings 分批生成；新增 App 方法最终仍需用 Wails v2.11.0 正式生成 bindings。

## 6. 构建基线与已知问题

- 固定 `GOTOOLCHAIN=go1.25.13`、Wails CLI v2.11.0。
- 工作区叠加 Change 016～020 的未提交修改；不得 reset、clean、批量 restore 或覆盖。
- Windows race 此前在启动阶段返回 `0xc0000139`，只能记为 BLOCKED，普通测试不能替代。
