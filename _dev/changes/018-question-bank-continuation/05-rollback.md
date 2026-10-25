# Change 018 回滚预案

状态：未执行回滚。

## 原则

- 只撤销 Change 018 的续接 API、会话快照、epoch 路由和主界面入口。
- 不删除或重写任何 Session、Record、截图、ConversationSnapshot、Exports 或旧导出目录。
- 新 JSON 字段均为可选；旧版本忽略，不做反向批量迁移。
- 已成功切换产生的 paused/active 状态以磁盘事实保留，后续通过正常选择流程处理，不用脚本强改。
- 跨文件切换失败时优先保证不会出现两个 active；保留现场并明确报告补偿结果。
- 禁止 `git reset --hard`、`git clean` 和其他破坏性回滚。
