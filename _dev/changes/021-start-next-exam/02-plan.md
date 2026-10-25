# Change 021 实施方案

日期：2026-10-24
状态：用户已批准，待按 TDD 实施。

## 1. 方案概述

新增显式“开始下一套”事务，不监听导出状态。按钮和 `start_next_exam` 快捷键只请求确认；确认后 `App.StartNextExam(true)` 按既有锁顺序检查全部运行态，调用 Store 原子封存当前段，持久化成功后才清 Solver、conversation refs、待发送截图并增加 conversation epoch。前端收到成功结果后清桌面历史并回到待记录状态。Mobile 显示不变，新段仍由下一次正式 Solve 懒创建。

## 2. Store 状态矩阵

| 状态 | 行为 |
|---|---|
| active/paused + Records + 无 running | 保存为 ended，`Sealed=true` |
| ended/exported | 幂等成功，`Sealed=false`；保留导出元数据 |
| discarded | 拒绝 |
| nil/空段 | 拒绝 |
| running/dirty/pending/write-blocked | 拒绝 |
| 保存失败 | 恢复变更前内存状态并返回错误 |

## 3. 实施步骤与白名单

1. Store TDD：`pkg/questionbank/types.go`、`store.go`、新 `store_next_exam_test.go`。
2. App TDD：新 `app/questionbank_next_exam.go`、`questionbank_next_exam_test.go`；复用 `conversationMu`、`HasInFlightTask`、`questionBank.mu`、`Solver.ClearHistory`。
3. 快捷键：`pkg/config/config.go`、`manager.go` 及测试，`pkg/shortcut/service.go` 及测试，`frontend/src/stores/settings.js`、`frontend/settings_regression_test.mjs`。
4. 前端：新 `NextExamConfirm.vue`；修改 `ActionBar.vue`、`questionbank.js`、`api.js`、`App.vue` 及既有回归测试。
5. 用固定 Wails CLI 生成 `App.js`、`App.d.ts`、`models.ts`，审计只含新 API/DTO 且不改 `go.mod`/`go.sum`。
6. 更新 `README.md`、`README_EN.md`，执行全量测试、构建、独立验证程序与治理收尾。

## 4. 原子性与事件

- 锁顺序：`conversationMu → TaskCoordinator 查询 → questionBank.mu → Store.mu → Solver.mu`，不得反转。
- Store 保存是所有运行时清理的提交点；失败路径不得清 Solver/refs/截图、不得加 epoch、不得发 `screenshots-cleared`。
- Store 成功后执行不可失败的内存清理；解锁后发 `screenshots-cleared` 和 `question-bank-updated`。
- 前端只有 API 成功后才调用 `solution.resetLiveHistory()` 和设置 `ui.hasStarted=false`。

## 5. 测试与验证

聚焦测试覆盖状态矩阵、保存失败回滚、busy 拒绝、确认取消、快捷键迁移/冲突/仅发事件、前端单例确认和成功/失败清理。随后运行 `go test ./...`、`go vet ./...`、`go build ./...`、前端回归/Vite 构建、race、diff-check，并使用固定 Wails v2.11.0 构建独立验证程序。

## 6. 回滚与安全边界

定向撤销本 Change 新 API、Store 原语、快捷键 action、UI 接线、bindings、测试和文档；不触碰 Change 016～020、用户配置、题库记录、截图、旧导出和正式程序。不使用 `git reset --hard`、`git clean`、`git add .` 或 `git add -A`。
