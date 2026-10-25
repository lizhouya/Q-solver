# Change 020 实施方案

日期：2026-10-23
状态：用户已批准；完成本文件后允许进入测试与实现。

## 方案概述

保留 `history[] -> rounds[]` 作为唯一事实来源，在 Store 中新增逐轮只读投影 `historyEntries` 和独立浏览目标 `selectedRoundTarget`。左栏遍历 entry，右栏只解析并显示所选 round；typed/legacy 流继续写各自真实目标，不以用户当前浏览位置作为路由依据。

## 业务与测试白名单

- `frontend/src/stores/solution.js`
- `frontend/src/components/SolveView.vue`
- `frontend/src/components/HistoryItem.vue`
- `frontend/src/App.vue`（仅修复 legacy 取消后选择目标）
- `frontend/history_round_selection_regression_test.mjs`（新建）
- `frontend/questionbank_continuation_regression_test.mjs`

除治理档案和验证产物外，不修改 Go 后端、Wails bindings、Mobile、题库 schema、模型请求、依赖、`go.mod` 或 `go.sum`。如测试证明必须越界，先暂停并更新方案。

## 状态与身份设计

1. `historyEntries` 依照 Session 新到旧顺序展开；每个 Session 内 rounds 最新优先。
2. typed entry key 为 `typed:${sessionId}:${recordId}`，不含数组 index 或 epoch；epoch 仍用于代际匹配。
3. legacy Session/round 在创建时分配仅内存使用的稳定本地 ID，不进入题库或 API。
4. `selectedRoundTarget` 只表示用户浏览目标；`streamTarget` 继续表示 typed 后台流真实写入目标；legacy 流保存独立 round 引用。
5. `getRoundSummary(round)` 只读取当前 round；`currentRounds` 只返回匹配浏览目标的单元素数组，无有效目标时返回空数组。
6. `currentHistoryItem` 继续返回所选 round 的父 Session，供标题、续接警告、会话级菜单和截图 scope 使用。

## 状态转换

- 激活题库 Session：恢复全部 Records，默认选中最后一个 Record，清理旧流目标。
- typed `request-start`：创建/定位 Record、把父 Session 移至首位、自动选中新 Record、设置三元组 stream target。
- 用户点击旧轮：只改变 selected target，不改变 stream target。
- `isViewingStreamTarget`：同时比较 selected 的 `sessionId + epoch + recordId` 与 stream target。
- legacy start：保存实际 round 引用并自动选中；后续 chunk/thinking/final/error 写该引用，不依赖当前浏览位置。
- 删除 Session：全部 entries 一起消失；修复选择；如流目标属于被删 Session，同步清理流状态。
- legacy 取消追加轮：`App.vue` 弹出尚未完成的最新轮后调用既有 `selectHistory(0)`，把浏览目标修复到父 Session 的上一轮。
- reset：清空 history、selected target、typed/legacy 流目标和流式状态。

## UI 设计

- `SolveView.vue` 左栏遍历 `historyEntries`，显示本轮摘要、时间、轮次标签和父 Session 的 writable 状态。
- active 样式通过稳定目标比较，不再使用单纯 Session index。
- 右侧不再显示 `RoundDivider`，因为 `currentRounds` 最多一项。
- `HistoryItem.vue` 移除“N 轮”徽标，显示“第 N 轮”。
- 菜单文案为“导出此会话为图片”“删除此会话”，明确仍作用于父 Session。

## TDD 与验证

先新增失败回归，覆盖逐轮顺序、稳定 key、自身摘要、单轮选择、typed/legacy 流隔离、删除/reset 修复、题库续接默认选择和静态 UI 契约；确认 RED 后实现最小改动并转 GREEN。

随后运行既有 continuation/history/screenshot/settings/mobile 回归、Mobile DOM、Vite build、`go test ./...`、`go vet ./...`、`go build ./...`、相关 race 和 `git diff --check`。race 若仍为 Windows `0xc0000139`，记 `BLOCKED`。

使用固定 Wails v2.11.0 和 Go 1.25.13 构建唯一名验证程序；构建前后保护正式 EXE、bindings、`go.mod` 和 `go.sum` 哈希，不覆盖旧产物。

## 风险与对策

| 风险 | 对策 |
|---|---|
| 直接扁平化破坏题库续接和上下文 | 仅增加派生投影，父 Session/rounds 保持不变。 |
| 查看旧轮时流式状态串轮 | 浏览目标与 stream target 分离，展示状态精确比较 recordId。 |
| 数组重排导致 key/选择漂移 | typed 使用 Session+Record，legacy 使用创建时稳定本地 ID。 |
| 每轮菜单让用户误以为单轮删除 | 文案明确会话级，操作仍传父 Session。 |
| 删除或 reset 留下悬挂目标 | 统一修复选择并清理 typed/legacy 流目标。 |
| 叠加工作区误覆盖旧 Change | 只编辑白名单，不使用破坏性 Git 命令，前后做范围审计。 |

## 回滚边界

只定向撤销本 Change 在白名单中的逐轮投影、浏览选择、UI 接线和测试；删除且仅删除 Change 020 新验证程序。保留治理档案并记录原因，不回退 typed 桌面协议、题库续接、Mobile、Change 019 截图折叠/答案汇总，也不删除任何题库、截图或旧导出。
