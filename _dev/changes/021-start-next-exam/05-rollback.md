# Change 021 回滚预案

日期：2026-10-24
状态：未执行回滚；当前实现保留，Windows GUI 待验收。

## 定向回滚范围

仅在确认需要回滚时撤销：

- `SealForNextResult`、`Store.SealCurrentForNext()` 及其专项测试。
- `StartNextExamResult`、`App.StartNextExam()` 及其专项测试。
- `start_next_exam` 默认绑定、安全迁移、快捷键事件和设置页项。
- `NextExamConfirm.vue`、ActionBar 按钮、question-bank store/API/App 事件接线与前端测试。
- `StartNextExam` / `StartNextExamResult` bindings。
- README、Change 021 治理内容及本 Change 独立验证程序。

## 产物回滚

若回滚独立程序：

1. 仅删除 `build/bin/Q-Solver-change021-start-next-exam-validation.exe`。
2. 仅删除 `build/bin/archive/validation/Q-Solver-change021-start-next-exam-validation.exe`。
3. 从 `build/bin/archive/validation/SHA256SUMS` 仅移除对应哈希行。
4. 再次执行全量 `sha256sum -c SHA256SUMS`，确保其余旧程序全部保持 `OK`。

## 禁止范围

- 不回退 Change 016～020 的累计业务实现或 bindings。
- 不删除、覆盖或迁移 `_记录`、题库截图、恢复文件、旧导出目录或用户配置。
- 不覆盖正式 `build/bin/Q-Solver.exe` 或旧验证程序。
- 不使用 `git reset --hard`、`git clean`、`git add .`、`git add -A` 或批量覆盖式恢复。
- 不把“清桌面运行态”改成 discarded 或 Mobile 清屏。
