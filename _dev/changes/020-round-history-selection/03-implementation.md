# Change 020 实现记录

日期：2026-10-23
状态：业务实现与普通自动验证完成；Windows GUI 待人工验收，race BLOCKED，未提交、未推送、未发布、未关闭。

## 事前状态

- 用户已批准逐轮派生投影方案。
- `00-request.md`、`01-current-state.md`、`02-plan.md` 已在业务源码修改前建立。
- 当前工作区叠加 Change 016～019 的未提交修改；本 Change 未执行 reset、clean、批量 restore/checkout 或覆盖式重写。

## RED

1. 新建 `frontend/history_round_selection_regression_test.mjs`，先锁定一个 Session 的多个 Record 必须投影为多条历史记录、单轮选择、稳定 key、typed/legacy 流隔离、删除/reset 修复和 UI 文案。
2. 旧实现首次运行因不存在 `historyEntries` 报错：`TypeError: Cannot read properties of undefined (reading 'map')`，确认测试能够捕获原始聚合显示缺陷。
3. Node VM 跨 realm 数组不适合直接做 prototype 敏感的 `deepStrictEqual`；测试改用 `Array.from(...)` 转换到当前 realm，只修正测试表达，不降低断言范围。

## GREEN：Store 状态与投影

修改 `frontend/src/stores/solution.js`：

- 保留 `history[] -> rounds[]` 作为唯一事实来源，没有物理扁平化 Session/Record。
- 新增 `historyEntries` 只读投影：Session 保持新到旧，每个 Session 内从最新 Record 到最早 Record 生成条目；每条持有父 Session、真实 round、原始索引、轮次标签、轮级时间和自身答案摘要。
- typed key 使用 `typed:${sessionId}:${recordId}`；legacy Session/round 在内存中分配稳定 `localId`，不进入题库、API 或磁盘。
- 新增 `selectedRoundTarget` 作为浏览目标；typed `streamTarget` 和 legacy `legacyStreamTarget` 分别保留真实后台写入目标。
- `currentRounds` 只返回所选 round 的单元素数组；`currentHistoryItem` 仍返回父 Session，保留会话级标题、截图 scope、导出和删除语义。
- `isViewingStreamTarget` 对 typed 目标精确比较 `sessionId + epoch + recordId`，对 legacy 目标比较稳定本地 ID；查看旧轮时不显示后台新轮的 loading、thinking 或 cursor。
- `activateQuestionBankSession()` 恢复全部 Records 后默认选择最新 Record；`beginDesktopRequest()` 自动选择新 Record，但后续 chunk/final/error 仍只按真实流目标写入。
- legacy stream start 固定写入 round；用户切换历史轮不会重定向后续 chunk。
- `deleteHistory()` 仍删除整个桌面 Session，并修复选择、清理属于被删 Session 的流目标；`resetLiveHistory()` 同步清理浏览和两类流目标。
- `normalizeArchivedRound()` 映射 `createdAt` 为轮级时间，运行时新轮记录本地提交时间。

## GREEN：界面接线

修改 `frontend/src/components/SolveView.vue`：

- 左栏由 `history` 改为遍历 `historyEntries`，每个 Record 单独一条，使用稳定 entry key、自己的摘要、时间和轮次标签。
- 点击条目调用 `selectHistoryEntry(entry)`；active 状态按稳定浏览目标判断。
- 右栏继续复用 `currentRounds`，但最多渲染一轮；移除 `RoundDivider` 和聚合轮次接线。
- 删除和图片导出仍传父 Session index；Change 019 的 `RoundScreenshots`、scopeKey、图片懒加载与缓存保持不变。

修改 `frontend/src/components/HistoryItem.vue`：

- 移除 `roundsCount`、`N 轮` badge 及样式，新增“第 N 轮”标签。
- 菜单明确为“导出此会话为图片”“删除此会话”，避免误解为单 Record 持久化操作。
- 保留 Teleport 菜单、点击隔离和删除禁用逻辑。

修改 `frontend/src/App.vue`：

- legacy 取消追加轮在 `current.rounds.pop()` 后调用 `solution.selectHistory(0)`，避免浏览目标悬挂到已移除 round 而导致右侧空白。

## 测试兼容调整

- 扩展 `frontend/questionbank_continuation_regression_test.mjs`，断言题库续接后的逐轮 entry、选择 API、stable key 和父 Session 删除规则。
- 更新 `frontend/round_screenshots_regression_test.mjs` 的 round key 契约为 `recordId || localId || idx`，兼容 typed 与 legacy 稳定身份。
- 专项测试覆盖跨 Session 后台流、同 Session 旧轮浏览、删除其他/当前/流目标 Session、legacy 浏览与写入隔离、reset 及静态 UI 契约。

## 构建产物

- 使用 Wails CLI v2.11.0、`GOTOOLCHAIN=go1.25.13`、`-clean=false -skipbindings -nosyncgomod -m` 构建：`build/bin/Q-Solver-change020-round-history-selection-validation.exe`。
- 未覆盖正式 `build/bin/Q-Solver.exe` 或旧验证程序；构建前后六项保护哈希全部一致。
- 验证程序已复制到 `build/bin/archive/validation/`，并追加到 LF-only `SHA256SUMS`；完整清单校验通过。
