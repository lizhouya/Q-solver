# Change 024 回滚预案

状态：未执行回滚。以下为本 Change 的定向回滚边界。

## 1. 代码回滚

只定向撤销 Change 024 修改或新增的文件，不使用 `git reset --hard`、`git clean`、`git add .` 或 `git add -A`：

- `pkg/questionbank/types.go`
- `pkg/questionbank/organize.go`
- `pkg/questionbank/organize_failure.go`
- `pkg/questionbank/export.go`
- 对应 questionbank 测试文件
- `app/questionbank.go`
- `app/questionbank_test.go`
- `app/questionbank_next_exam_test.go`
- `frontend/src/services/api.js`
- `frontend/src/stores/questionbank.js`
- `frontend/src/components/QuestionBankExportModal.vue`
- `frontend/questionbank_history_regression_test.mjs`
- `README.md`
- `README_EN.md`

新增 JSON 字段为可选字段，且 Session version 未提升；旧代码会忽略该字段，因此不需要批量迁移或重写历史 Session。

## 2. 数据保护

不得为了回滚代码而删除、清空或改写：

- 任意 Session、Record、Answer、AnswerItems 或 Questions；
- `_记录`；
- 任意截图或 `assets/`；
- 旧导出目录；
- 用户后续通过失败重试成功保存的结构化题目。

若新代码已在真实使用中保存 `OrganizeFailure`，回滚后保留该字段；不得重写整个 Session 来删除它。

## 3. TCL `_02`

旧 TCL 目录和 Session 从未被修改，无需数据回滚。`2026-10-25_095652_TCL_综合笔试_02` 默认保留为独立、待人工核对的恢复副本。

未来若确需删除 `_02`，必须先确认其中没有用户后续人工修改，并取得单独删除授权；代码回滚不自动删除该目录。

## 4. Bindings 与程序

Bindings 已由固定 Wails v2.11.0 生成。回滚业务 API 时，必须同时定向恢复 `frontend/wailsjs/go/app/App.js`、`App.d.ts` 和 `frontend/wailsjs/go/models.ts`，避免生成契约与 Go API 不一致。

独立验证程序为 `build/bin/Q-Solver-change024-organize-failure-recovery-validation.exe`。它没有覆盖任何既有 EXE；只有收到单独删除授权时才移除，不把删除验证程序作为代码回滚的隐式步骤。

## 5. 治理记录

即使回滚业务代码，也保留 Change 024 的 `00`～`06` 与审计文件，并在本文件追加实际回滚时间、原因、文件边界和验证结果。不得删除治理证据来伪装本 Change 从未发生。
