# Change 022 回滚预案与记录

状态：未执行回滚。

## 定向回滚范围

1. 先核对当前差异与 Change 022 实施记录，不批量还原累计 Change 016～021 的改动。
2. 撤销 `pkg/answersummary` 及本 Change 对 Prompt、Solver Request、题库 AnswerItems/完成事务/编号查询、App capture/历史 DTO、KeepContext 清理与恢复守卫的增量。
3. 撤销 Desktop 复制入口、clipboard helper、round 结构化字段，以及 Mobile Payload/Bridge/复制按钮/实际快捷键提示的增量；保留之前的清屏、滚动与确认链。
4. 根据回滚后的 Go DTO 用固定 Wails v2.11.0 重新生成 bindings，不回退或升级 runtime、依赖。
5. 如需移除本 Change 验证程序，先核对目标和是否运行，只移除 Change 022 专用文件及对应 SHA256SUMS 条目；不动正式程序和旧验证程序。
6. 回归测试、前端构建和独立 Wails 构建后再记录实际结果；未经授权不提交或发布。

## 数据保护

- 保留所有 Session、Record、原始 Answer、截图和旧导出。
- `answerItems` 是 schema version 2 的可选字段，旧代码可忽略；不需要删除字段、迁移旧题库或重写数据。
- 已清空的对话引用不能在回滚时擅自从原始记录重新构建为当前上下文。
- 不使用 `git reset --hard`、`git clean`、`git add .`、`git add -A` 或覆盖整个工作区。

## 实际执行

无。此次只交付增量实现与验证程序，未进行任何回滚、数据清理、Git 提交、推送或发布。
