# Change 016～023 整合提交边界审计

日期：2026-10-25

## 本次明确授权

用户先确认 Change 023 带题号复制验收“通过”，并要求提交和推送。说明工作区包含 Change 016～023 累积修改与既有未推送 RC2 提交后，用户明确选择“整合提交并推送（推荐）”。

仅本次允许将上述 Change 跨项整合为功能/测试与治理两个新提交，并连同 `da273bd`（RC2 本地修复）推送到 `origin/dev/qsolver-custom`。这不是后续默认权限，不修改全局制度，不推送 main/upstream，不创建 Tag 或 Release，不修改 Git 配置。

## 推送目标与副作用

- origin：`https://github.com/lizhouya/Q-solver.git`。
- 分支：`dev/qsolver-custom`。
- 只读核对时远端头为 `fc5cf66a233c10d11413f6c4d06ab4d2b2184eb1`，本地已有后继 `da273bd`。
- 现有 release.yml 仅响应 v* Tag 或 workflow_dispatch，普通分支推送不触发发布；update-models.yml 仅定时/手动触发。
- 正常快进 push；如远端出现新提交则停止，不 force push、不改写已有提交。

## 文件与敏感信息审计

- 精确路径暂存，不使用 git add . / git add -A。
- 排除 build/bin 的 EXE、frontend/dist、node_modules、tmp、日志、生成前备份及私人配置；这些文件本地保留，不为清理工作区而删除。
- 选中文件全部为文本；敏感模式命中经核查均为合成测试夹具（test/fixture/draft key 与固定测试 Token）或 Vue 属性引用，没有发现真实凭据。扫描不是绝对无泄漏保证。
- 只提交脱敏文档与校验和清单，不提交真实截图、题库记录或旧导出。
- go.mod 仅工作树元数据/行尾状态，暂存若无内容差异不会进入提交；不改变依赖版本。

## 验证与验收边界

- Change 023 最终构建与全量普通验证已通过，提交前另复跑 Go test/vet/build、八份前端回归和 Mobile DOM/SSE。
- Change 016、023 有用户明确人工验收确认；不把本次“通过”扩大为 Change 017～022 未明确项目的人工验收。
- Windows race 已知启动阻塞 `0xc0000139` 仍保留，不声称 race 通过；未完成项目不 Close。
- 最新独立程序为 Change 023，哈希 `44fbb9381179b0d739003c53df1abe9c06847255447f87ff786fdbe8a711d339`，不随代码推送二进制。

## 完整提交消息

### 功能与测试

```text
feat: 整合分段续接、同步控制与答案复制（Change 016–023）

整合固定同步链接与清屏、题库历史与真正续接、逐轮浏览、
下一套笔试入口、结构化纯答案及带题号复制。
保留原始截图和完整答案，关闭上下文时同步清理引用。

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

### 治理记录

```text
chore: 归档 Change 016–023 验证与提交边界

记录用户批准的跨 Change 整合范围和 Change 023 人工验收。
保留既有 GUI 待验收、race 环境阻塞与未关闭状态。

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

## 提交一的精确路径

- `README.md`
- `README_EN.md`
- `app/app.go`
- `app/mobile.go`
- `app/mobile_commands_test.go`
- `app/mobile_test.go`
- `app/phase4_test.go`
- `app/questionbank.go`
- `app/questionbank_continuation.go`
- `app/questionbank_continuation_test.go`
- `app/questionbank_history.go`
- `app/questionbank_history_test.go`
- `app/questionbank_next_exam.go`
- `app/questionbank_next_exam_test.go`
- `app/questionbank_solve_capture.go`
- `app/questionbank_test.go`
- `app/solve.go`
- `app/solve_persistence_test.go`
- `app/solve_preflight_test.go`
- `frontend/clipboard_regression_test.mjs`
- `frontend/history_round_selection_regression_test.mjs`
- `frontend/mobile_store_regression_test.mjs`
- `frontend/numbered_copy_regression_test.mjs`
- `frontend/questionbank_continuation_regression_test.mjs`
- `frontend/questionbank_history_regression_test.mjs`
- `frontend/round_screenshots_regression_test.mjs`
- `frontend/settings_regression_test.mjs`
- `frontend/src/App.vue`
- `frontend/src/components/ActionBar.vue`
- `frontend/src/components/HistoryItem.vue`
- `frontend/src/components/MobileActionConfirm.vue`
- `frontend/src/components/NextExamConfirm.vue`
- `frontend/src/components/QuestionBankExportModal.vue`
- `frontend/src/components/QuestionBankPanel.vue`
- `frontend/src/components/RoundScreenshots.vue`
- `frontend/src/components/ScreenshotDock.vue`
- `frontend/src/components/SettingsModal.vue`
- `frontend/src/components/SolveView.vue`
- `frontend/src/components/TopBar.vue`
- `frontend/src/services/api.js`
- `frontend/src/stores/mobile.js`
- `frontend/src/stores/questionbank.js`
- `frontend/src/stores/settings.js`
- `frontend/src/stores/solution.js`
- `frontend/src/utils/clipboard.js`
- `frontend/wailsjs/go/app/App.d.ts`
- `frontend/wailsjs/go/app/App.js`
- `frontend/wailsjs/go/models.ts`
- `go.mod`
- `pkg/answersummary/numbering.go`
- `pkg/answersummary/numbering_test.go`
- `pkg/answersummary/parser.go`
- `pkg/answersummary/parser_test.go`
- `pkg/answersummary/types.go`
- `pkg/config/config.go`
- `pkg/config/config_test.go`
- `pkg/config/manager.go`
- `pkg/config/manager_test.go`
- `pkg/config/mobile_state.go`
- `pkg/config/mobile_state_test.go`
- `pkg/domain/prompts.go`
- `pkg/domain/prompts_test.go`
- `pkg/mobile/bridge.go`
- `pkg/mobile/bridge_server_test.go`
- `pkg/mobile/server.go`
- `pkg/mobile/state.go`
- `pkg/mobile/web/index.html`
- `pkg/mobile/web_dom_test.mjs`
- `pkg/questionbank/continuation.go`
- `pkg/questionbank/continuation_test.go`
- `pkg/questionbank/export.go`
- `pkg/questionbank/history.go`
- `pkg/questionbank/history_test.go`
- `pkg/questionbank/organize.go`
- `pkg/questionbank/segment_test.go`
- `pkg/questionbank/store.go`
- `pkg/questionbank/store_next_exam_test.go`
- `pkg/questionbank/store_test.go`
- `pkg/questionbank/types.go`
- `pkg/shortcut/service.go`
- `pkg/shortcut/service_test.go`
- `pkg/solution/conversation_test.go`
- `pkg/solution/solution.go`
- `pkg/solution/solution_test.go`
- `pkg/task/coordinator.go`
- `pkg/task/coordinator_test.go`

## 提交二的精确路径

- `_dev/audits/016-023-commit-boundary.md`
- `_dev/audits/017-question-bank-history.md`
- `_dev/audits/018-question-bank-continuation.md`
- `_dev/audits/019-screenshot-fold-answer-summary.md`
- `_dev/audits/020-round-history-selection.md`
- `_dev/audits/021-start-next-exam.md`
- `_dev/audits/022-answer-copy-numbering-sync.md`
- `_dev/audits/023-numbered-answer-copy.md`
- `_dev/changes/016-mobile-stable-link/00-request.md`
- `_dev/changes/016-mobile-stable-link/01-current-state.md`
- `_dev/changes/016-mobile-stable-link/02-plan.md`
- `_dev/changes/016-mobile-stable-link/03-implementation.md`
- `_dev/changes/016-mobile-stable-link/04-validation.md`
- `_dev/changes/016-mobile-stable-link/05-rollback.md`
- `_dev/changes/016-mobile-stable-link/06-notes.md`
- `_dev/changes/017-question-bank-history/00-request.md`
- `_dev/changes/017-question-bank-history/01-current-state.md`
- `_dev/changes/017-question-bank-history/02-plan.md`
- `_dev/changes/017-question-bank-history/03-implementation.md`
- `_dev/changes/017-question-bank-history/04-validation.md`
- `_dev/changes/017-question-bank-history/05-rollback.md`
- `_dev/changes/017-question-bank-history/06-notes.md`
- `_dev/changes/018-question-bank-continuation/00-request.md`
- `_dev/changes/018-question-bank-continuation/01-current-state.md`
- `_dev/changes/018-question-bank-continuation/02-plan.md`
- `_dev/changes/018-question-bank-continuation/03-implementation.md`
- `_dev/changes/018-question-bank-continuation/04-validation.md`
- `_dev/changes/018-question-bank-continuation/05-rollback.md`
- `_dev/changes/018-question-bank-continuation/06-notes.md`
- `_dev/changes/019-screenshot-fold-answer-summary/00-request.md`
- `_dev/changes/019-screenshot-fold-answer-summary/01-current-state.md`
- `_dev/changes/019-screenshot-fold-answer-summary/02-plan.md`
- `_dev/changes/019-screenshot-fold-answer-summary/03-implementation.md`
- `_dev/changes/019-screenshot-fold-answer-summary/04-validation.md`
- `_dev/changes/019-screenshot-fold-answer-summary/05-rollback.md`
- `_dev/changes/019-screenshot-fold-answer-summary/06-notes.md`
- `_dev/changes/019-screenshot-fold-answer-summary/pre-wails-formal-exe.sha256`
- `_dev/changes/019-screenshot-fold-answer-summary/pre-wails-protected.sha256`
- `_dev/changes/020-round-history-selection/00-request.md`
- `_dev/changes/020-round-history-selection/01-current-state.md`
- `_dev/changes/020-round-history-selection/02-plan.md`
- `_dev/changes/020-round-history-selection/03-implementation.md`
- `_dev/changes/020-round-history-selection/04-validation.md`
- `_dev/changes/020-round-history-selection/05-rollback.md`
- `_dev/changes/020-round-history-selection/06-notes.md`
- `_dev/changes/020-round-history-selection/pre-wails-protected.sha256`
- `_dev/changes/021-start-next-exam/00-request.md`
- `_dev/changes/021-start-next-exam/01-current-state.md`
- `_dev/changes/021-start-next-exam/02-plan.md`
- `_dev/changes/021-start-next-exam/03-implementation.md`
- `_dev/changes/021-start-next-exam/04-validation.md`
- `_dev/changes/021-start-next-exam/05-rollback.md`
- `_dev/changes/021-start-next-exam/06-notes.md`
- `_dev/changes/021-start-next-exam/pre-bindings-protected.sha256`
- `_dev/changes/022-answer-copy-numbering-sync/00-request.md`
- `_dev/changes/022-answer-copy-numbering-sync/01-current-state.md`
- `_dev/changes/022-answer-copy-numbering-sync/02-plan.md`
- `_dev/changes/022-answer-copy-numbering-sync/03-implementation.md`
- `_dev/changes/022-answer-copy-numbering-sync/04-validation.md`
- `_dev/changes/022-answer-copy-numbering-sync/05-rollback.md`
- `_dev/changes/022-answer-copy-numbering-sync/06-notes.md`
- `_dev/changes/022-answer-copy-numbering-sync/pre-build-protected.json`
- `_dev/changes/023-numbered-answer-copy/00-request.md`
- `_dev/changes/023-numbered-answer-copy/01-current-state.md`
- `_dev/changes/023-numbered-answer-copy/02-plan.md`
- `_dev/changes/023-numbered-answer-copy/03-implementation.md`
- `_dev/changes/023-numbered-answer-copy/04-validation.md`
- `_dev/changes/023-numbered-answer-copy/05-rollback.md`
- `_dev/changes/023-numbered-answer-copy/06-notes.md`
- `_dev/changes/023-numbered-answer-copy/protected.json`
- `_dev/changes/INDEX.md`

## 执行状态

已获授权，暂存、提交及远端核验按上述范围依次执行；最终提交和推送结果以 Git 历史及远端分支头为准。本文随治理提交归档，不记录自身提交哈希，不以提交替代验收。

## 功能提交与提交前复验

- 功能提交：`e44bb6163f0cb1b88e2ec4b4da80a1446ddb1990`，85 个文件；go.mod 无内容差异，未进入提交。
- 提交前复跑 Go test/vet/build、八份前端回归、Mobile DOM/SSE 全部通过；证据保留在本地忽略的 tmp/precommit-validation.log。
- 功能暂存区检查通过，没有二进制、配置、日志或治理文件混入。
- 本文件属于第二次治理提交；不记录自身提交哈希，不提前宣称远端推送成功。

治理暂存检查发现旧 Change 019/020 文档日期行含 Markdown 尾随双空格；本次仅清理这些尾随空白，不改验收结论或业务代码。
