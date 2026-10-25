# Change 021 实现后范围与安全审计

日期：2026-10-24
状态：实现后本地审计完成；独立代理审查受 API 429 阻塞；GUI `NEEDS_USER`，race `BLOCKED`，Change 未关闭。

## 结论

实现采用显式确认事务，没有依赖不可靠的异步导出完成推断。Store 持久化成功是唯一提交点；此前的错误路径不清 Solver、refs、截图或桌面状态，此后的内存清理不可失败。普通测试、前端回归、构建、依赖/正式程序哈希和统一归档校验通过。

## 实现范围

后端与测试：

- `pkg/questionbank/types.go`
- `pkg/questionbank/store.go`
- `pkg/questionbank/store_next_exam_test.go`
- `app/questionbank_next_exam.go`
- `app/questionbank_next_exam_test.go`

快捷键与配置：

- `pkg/config/config.go`
- `pkg/config/manager.go`
- `pkg/config/config_test.go`
- `pkg/config/manager_test.go`
- `pkg/shortcut/service.go`
- `pkg/shortcut/service_test.go`
- `frontend/src/stores/settings.js`
- `frontend/settings_regression_test.mjs`

前端与 bindings：

- `frontend/src/components/NextExamConfirm.vue`
- `frontend/src/components/ActionBar.vue`
- `frontend/src/stores/questionbank.js`
- `frontend/src/services/api.js`
- `frontend/src/App.vue`
- `frontend/questionbank_history_regression_test.mjs`
- `frontend/wailsjs/go/app/App.js`
- `frontend/wailsjs/go/app/App.d.ts`
- `frontend/wailsjs/go/models.ts`

文档与产物：

- `README.md`、`README_EN.md`
- `_dev/changes/021-start-next-exam/00-request.md`～`06-notes.md`
- `_dev/audits/021-start-next-exam.md`
- `_dev/changes/INDEX.md`
- Change 021 构建目录与统一归档验证程序、`SHA256SUMS`

## 原子性与并发审计

- 锁序与既有续接路径一致：`conversationMu → TaskCoordinator → questionBank.mu → Store.mu → Solver.mu`。
- 使用 `HasInFlightTask()`，已 Cancel 但 goroutine 未退出的任务仍拒绝。
- `captures`、`organizing`、`operation=organizing/exporting`、`closing` 在 Store 之前拒绝。
- Store 以候选 Session 写盘，失败时不替换 current；App 仅在 Store 成功后清运行态。
- `screenshots-cleared` 和题库更新事件在 App 锁外发送，避免事件链重入死锁。
- ended/exported 的 Store 幂等调用不改时间戳与导出快照；下一段仍由既有 Begin 链懒创建。

## 数据与 Mobile 边界

- 不删除、覆盖或迁移题库 Session/Record、归档截图、恢复文件或旧导出目录。
- 不把正常“下一套”标记为 discarded。
- 不调用 Mobile clear/reset/stable-link API，不改变 Mobile bridge snapshot、clearSeq/display epoch、固定链接或当前显示。
- 不读取、修改或输出 API Key、Mobile Token、密码或用户私人配置内容。
- 未调用真实模型、未上传截图、未消费真实 API 额度。

## 快捷键迁移审计

- 新 action 缺失与显式空绑定通过原始持久化 key 存在性区分。
- 缺失且默认组合被占用时不抢占原 action；显式空和自定义值不改写。
- 现有 `clear_mobile_display` 特殊迁移语义保留并通过回归。
- 快捷键触发只发 `request-start-next-exam`，由前端统一进入确认链。

## bindings 与依赖审计

- 固定 Wails CLI v2.11.0 正常生成当前工作区全部导出 App API/DTO；新增 Change 021 的 `StartNextExam` / `StartNextExamResult`，并保留 Change 016～018 的累计接口。
- 事前只记录 bindings 哈希、没有文件副本；逆向移除新增接口后未匹配事前哈希，无法作出“仅一个接口的逐字节差异”结论。
- 已确认的边界是当前 Go API、bindings、Frontend、Wails 构建一致，累计公开 API 均存在；没有伪写更强的哈希证明。
- `go.mod`、`go.sum` 和 Wails runtime 未升级。
- 正式 `build/bin/Q-Solver.exe` 哈希不变。
- 最终 `git diff --check` 无空白错误；仅保留行尾转换 warning。

## 验证审计

- Store/App/config/shortcut 专项测试：PASS。
- 六组前端回归与 Mobile DOM：PASS。
- Go 全量 test/vet/build：PASS。
- Vite、bindings 生成构建及两次 `-skipbindings` Wails 构建：PASS；最终重建哈希不变。
- 统一归档 11 个哈希：全部 `OK`。
- race：Windows 启动阶段 `0xc0000139`，`BLOCKED`。
- Windows GUI：`NEEDS_USER`。
- 独立只读审查：API 429 提前终止，不能记为 PASS；已返回的局部结论未确认阻断缺陷。

## 授权与 Git 边界

- 未执行 commit、push、Tag、Release、`git reset --hard`、`git clean`、`git add .` 或 `git add -A`。
- 工作区继续叠加 Change 016～020 和本 Change，明确不是 clean。
- 本 Change 不含 commit、push、Tag 或 Release 授权。

## 最终状态

Change 021 的业务实现、普通自动验证、bindings、独立构建与统一归档已经完成。GUI 人工验收和 race 环境问题未完成，因此状态保持“进行中 / 未关闭”。
