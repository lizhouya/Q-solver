# Change 018 实施方案

日期：2026-10-23
状态：用户已批准。

## 方案

1. Session 新增可选 ConversationSnapshot、Record KeepContext 和 Exports 历史；旧 JSON 继续兼容，不批量迁移。
2. 旧 Session 无快照时，仅从自身 completed+非空 Answer Records 重建最近 10 轮并标记 `record-reconstructed`；新成功轮次保存实际 retained Record 引用，KeepContext=false 保存准确空快照。
3. Solver 历史规范为不含 system 的 user/assistant 对，增加 mutex、generation、Snapshot/Replace；每次请求始终用当前配置新建 system。
4. Store 新增可补偿的 ActivateHistoricalSession：源 active→paused 后目标 ended/exported/paused→active；失败恢复两份文件和内存 current；discarded/running/dirty 拒绝。
5. App 用 conversationMu 原子编排任务安全检查、上下文解析、Solver replace、Store activate、失败补偿和 epoch 更新。TaskCoordinator 新增 goroutine 完成前始终为真的 in-flight 门。
6. 桌面 solve 事件携带 epoch/sessionID/recordID；Mobile 保持现有协议与 display epoch。
7. solution store 升级为稳定 session/record ID 的单一会话模型；流按目标定位。主界面按需安全加载历史截图，题库 panel 成功切换后关闭。
8. exported 激活保留旧目录；每次成功导出追加 Exports 并继续使用新目录后缀。

## 允许文件范围

- `pkg/questionbank/**`、`pkg/solution/**`、`pkg/task/**`
- `app/app.go`、`app/solve.go`、`app/mobile.go`、`app/questionbank*.go` 及对应测试
- `frontend/src/App.vue`、`frontend/src/services/**`、`frontend/src/stores/solution.js`、`questionbank.js`、相关 components 与专项测试
- Wails 生成 bindings、README/README_EN、Change 018 与 INDEX

## GUI 布局回修方案

1. 在 `frontend/questionbank_history_regression_test.mjs` 先增加布局约束，锁定固定操作列、纵向等宽按钮、卡片内部纵向信息和窄屏单列规则。
2. `QuestionBankPanel.vue` 桌面端使用 `minmax(0, 1fr) 7.5rem`，操作按钮纵向排列并占满操作列；0、1、2 个按钮均不改变左侧卡片宽度。
3. 分段标题/时间与状态 meta 改为上下排列，标题保持省略，meta 可安全换行；删除 `flex-shrink: 0` 和失效的 `justify-self`。
4. `560px` 及以下使用卡片在上、操作区在下的单列布局，按钮全宽。
5. 保留浏览按钮与操作按钮的兄弟关系以及既有状态矩阵、`aria-*`、禁用条件、`@click.stop` 和焦点样式；不修改后端、store 或 bindings。

## 验证

专项 Go/Node 回归后执行全量 `go test ./...`、`go vet ./...`、`go build ./...`、前端 build、Wails 独立验证构建和 `git diff --check`。race 单独尝试；`0xc0000139` 继续记 BLOCKED。Windows GUI 按 00-request 验收，完成前不关闭 Change。布局回修使用新的独立验证文件名，不覆盖正式 `Q-Solver.exe` 或此前验证程序。

## 回滚

只回退 Change 018 代码入口，不删除新可选 JSON 字段、题库数据、截图或导出。跨组件失败优先避免两个 active；不得使用破坏性 Git 命令。
