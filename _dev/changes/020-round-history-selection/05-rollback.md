# Change 020 回滚预案

日期：2026-10-23
状态：预案已建立，未执行回滚。

## 触发条件

- 逐轮投影导致答案写入错误 Record、流式状态串轮或题库续接回退。
- 删除/reset 后存在悬挂选择或迟到图片回写。
- 无法在白名单范围内满足验收标准，且继续修改会触及未批准的后端/持久化边界。

## 回滚范围

1. 定向撤销 `solution.js` 中本 Change 新增的 entry 投影、浏览选择和 legacy 流引用。
2. 定向恢复 `SolveView.vue`、`HistoryItem.vue` 的本 Change 接线，不影响 Change 019 截图组件。
3. 删除且仅删除 `frontend/history_round_selection_regression_test.mjs` 和 Change 020 新验证程序。
4. 若已更新验证归档哈希，只移除对应 Change 020 条目并重新校验既有条目。
5. 保留本目录和审计，记录失败原因和实际回滚结果。

## 禁止事项

- 不使用 `git reset --hard`、`git clean`、批量 checkout/restore。
- 不回退 Change 016～019、typed 桌面协议、题库续接、Mobile 或模型 Prompt。
- 不删除、覆盖或迁移题库记录、截图、旧导出、正式 EXE 和旧验证程序。
