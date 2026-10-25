# Change 020 现状快照

日期：2026-10-23
状态：业务源码修改前快照。

## 工作区与基线

- 当前分支：`dev/qsolver-custom`，相对 `origin/dev/qsolver-custom` ahead 1。
- 工作区包含 Change 016～019 的大量预期未提交源码、测试、治理和生成文件改动；本 Change 必须叠加实施，禁止 reset、clean、批量 restore/checkout 或覆盖式重写。
- Change 019 的 GUI、真实模型效果、race 和统一验证归档仍有未完成项；Change 020 不改变其状态。

## 当前数据模型

- `frontend/src/stores/solution.js` 中 `history[]` 的每个元素代表一个 Session。
- 每次提交/Record 保存在父 Session 的 `rounds[]` 中。
- typed 桌面事件已经使用 `streamTarget = { epoch, sessionId, recordId }` 定位真实写入目标。
- 题库续接通过 `activateQuestionBankSession()` 将 `detail.records` 映射为同一 history item 的多个 rounds。
- 图片读取通过 `loadRoundImage(round, image)` 反查父 Session，并用 `requestVersion` 和父 Session 存活检查拒绝迟到响应。

## 当前展示行为与缺陷

- `SolveView.vue` 左栏遍历 `solution.history`，所以一个 Session 只显示一条卡片。
- `getSummary(item)` 固定取 Session 最后一轮答案，同一 Session 无法逐轮显示摘要。
- `HistoryItem.vue` 使用 `roundsCount` 显示“N 轮”聚合徽标。
- `currentRounds` 返回当前 Session 的全部 `rounds`，右侧因此连续渲染第 1 轮、第 2 轮等内容。
- `isViewingStreamTarget` 只比较 `sessionId + epoch`，未比较浏览中的 `recordId`；查看同 Session 旧轮时可能错误显示当前新轮的 loading、thinking 或 cursor。
- `selectHistory(index)` 只支持选择 Session，不支持选择 Record。

## 现有操作边界

- `deleteHistory(index)` 删除桌面内存中的整个父 Session，不删除题库磁盘记录。
- `exportImage(index)` 导出父 Session 的全部 rounds。
- `resetLiveHistory()` 清空桌面显示态，不删除题库原始记录、截图或旧导出。
- legacy 事件处理器仍以 `history[0]` 的最后一轮作为写入目标；用户浏览和真实流写入尚未显式解耦。
- `App.vue` 的 legacy 取消追加路径会直接 `current.rounds.pop()`；引入 Record 级选择后必须同步修复浏览目标，否则它会指向已移除 round。

## 不可破坏约束

- Session 继续承担题库续接、ConversationSnapshot、暂停/封存/导出和 KeepContext 生命周期。
- typed 流必须继续严格匹配 `epoch + sessionId + recordId`。
- historical Answer 只能作为 assistant；本 Change 不接触模型上下文重建。
- Change 019 的 `RoundScreenshots.vue` 本地展开状态、`scopeKey`、图片缓存和单次多图请求不变。
- conversation epoch 与 Mobile display epoch 不得混用；本 Change不修改 Mobile。
