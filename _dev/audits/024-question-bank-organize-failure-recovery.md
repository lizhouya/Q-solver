# Change 024 提交边界审计

日期：2026-10-25
状态：实施后边界已审计；未暂存、未提交、未推送、未发布。

## 1. 审计结论

当前业务与治理变更均属于 Change 024 白名单。未发现依赖升级、用户配置、密钥、快捷键、正式程序、旧验证程序或其他 Change 的新增修改。TCL `_02` 位于仓库外，不进入 Git。

用户明确授权后，固定 Wails v2.11.0 已核验并完成 bindings 与独立生产构建；PATH 中的 v2.16.0 未使用。新验证程序使用独立文件名，既有 EXE 哈希均未变化。

## 2. 计划纳入的已跟踪文件

### 用户文档

- `README.md`
- `README_EN.md`

### 治理索引

- `_dev/changes/INDEX.md`

### App

- `app/questionbank.go`
- `app/questionbank_test.go`
- `app/questionbank_next_exam_test.go`

### 前端

- `frontend/questionbank_history_regression_test.mjs`
- `frontend/src/components/QuestionBankExportModal.vue`
- `frontend/src/services/api.js`
- `frontend/src/stores/questionbank.js`
- `frontend/wailsjs/go/app/App.js`
- `frontend/wailsjs/go/app/App.d.ts`
- `frontend/wailsjs/go/models.ts`

### questionbank

- `pkg/questionbank/types.go`
- `pkg/questionbank/organize.go`
- `pkg/questionbank/export.go`
- `pkg/questionbank/store_test.go`
- `pkg/questionbank/segment_test.go`
- `pkg/questionbank/continuation_test.go`
- `pkg/questionbank/store_next_exam_test.go`

## 3. 计划纳入的新增文件

- `pkg/questionbank/organize_failure.go`
- `pkg/questionbank/organize_failure_test.go`
- `_dev/changes/024-question-bank-organize-failure-recovery/00-request.md`
- `_dev/changes/024-question-bank-organize-failure-recovery/01-current-state.md`
- `_dev/changes/024-question-bank-organize-failure-recovery/02-plan.md`
- `_dev/changes/024-question-bank-organize-failure-recovery/03-implementation.md`
- `_dev/changes/024-question-bank-organize-failure-recovery/04-validation.md`
- `_dev/changes/024-question-bank-organize-failure-recovery/05-rollback.md`
- `_dev/changes/024-question-bank-organize-failure-recovery/06-notes.md`
- `_dev/audits/024-question-bank-organize-failure-recovery.md`

## 4. 仓库外数据交付

以下目录是本次离线恢复交付，不纳入 Git，也不得在未来提交边界操作中删除或移动：

`E:\DATA\求职_AI\输出\求职笔试\题库\2026-10-25_095652_TCL_综合笔试_02`

包含：

- `TCL_综合笔试.md`
- 与旧目录逐字节一致的 `原始记录.md`
- 与旧目录逐文件一致的 25 张 `assets/` 截图

旧 TCL 目录和 `_记录/a6ef78f5c602e1950b991e3d/session.json` 哈希未变化。

## 5. 生成文件边界

固定 Wails v2.11.0 仅产生三份必要 binding 差异：新 App 方法声明/实现以及 draft/request DTO 字段。Wails runtime 文件无实际差异；生成器引入的 `models.ts` 尾随空白已定向清理。

独立验证程序位于 `build/bin/Q-Solver-change024-organize-failure-recovery-validation.exe`，不进入 Git 提交；SHA-256 为 `1953484b5fb42576f2db013691bdc6190f7a60ce7e1aabbdadad539b3f35d274`。

## 6. 明确排除

- `go.mod`、`go.sum`、`frontend/package.json`、`frontend/package-lock.json`；
- 用户真实配置、API Key、Token、Authorization、完整 BaseURL；
- 正式 `Q-Solver.exe` 和全部旧验证程序；
- `frontend/dist/` 构建产物；
- 其他 Change 文件；
- 旧题库目录、`_记录`、截图和旧导出。

## 7. 依赖与生成物审计

四个依赖文件 SHA-256 与实施前一致。PATH 中 Wails v2.16.0 未使用；固定 v2.11.0 通过精确 `go run` 执行。Change 024 EXE 已使用独立名称生成，所有既有 EXE 构建前后逐文件哈希一致。

## 8. 验证摘要

- 聚焦 Go 测试：PASS；
- 全量 Go test/vet/build：PASS；
- 八份前端回归：PASS；
- Vite production build：PASS（既有 chunk size warning）；
- `git diff --check`：PASS；
- TCL 数据保护后检：PASS；
- race：Windows `0xc0000139`，BLOCKED；
- Wails v2.11.0 bindings/集成构建/验证 EXE：PASS；
- Windows GUI 与真实 provider：NEEDS_USER。

## 9. Git 授权

用户尚未授权 Change 024 的暂存、commit、push、Tag 或 Release。审计完成不构成这些操作的授权。
