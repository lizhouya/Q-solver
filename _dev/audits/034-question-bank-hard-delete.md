# Change 034 提交边界审计

## 功能与测试候选

- `pkg/questionbank/store.go`
- `pkg/questionbank/store_delete_test.go`
- `app/questionbank.go`
- `app/questionbank_delete_test.go`
- `frontend/src/components/QuestionBankPanel.vue`
- `frontend/src/stores/questionBank.js`
- `frontend/questionbank_history_regression_test.mjs`

建议功能提交消息：

```text
feat: permanently delete cleared question bank segments

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

## 治理候选

- `_dev/audits/034-question-bank-hard-delete.md`
- `_dev/changes/034-question-bank-hard-delete/00-request.md`
- `_dev/changes/034-question-bank-hard-delete/01-current-state.md`
- `_dev/changes/034-question-bank-hard-delete/02-plan.md`
- `_dev/changes/034-question-bank-hard-delete/03-implementation.md`
- `_dev/changes/034-question-bank-hard-delete/04-validation.md`
- `_dev/changes/034-question-bank-hard-delete/05-rollback.md`
- `_dev/changes/034-question-bank-hard-delete/06-notes.md`
- `_dev/changes/INDEX.md` 中仅 Change 034 对应行和摘要；该文件同时含未提交 Change 033 更新，提交时必须构造仅含 034 的精确 patch，不能整文件直接暂存。

建议治理提交消息：

```text
chore: add change 034 governance records

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

## 验证

- Store、App、Frontend TDD 红绿过程完整。
- 全量 Go test/vet/build/mod verify 通过。
- 14 份前端回归和 Vite build 通过。
- 删除自动测试仅使用临时目录，未触碰用户题库。

## 人工验收与发布门禁

- Change 034 组合候选已归档到 `build/bin/archive/validation/change-034/`，42,345,472 字节，SHA-256 `C033EAB4844721EB233841915C82583D213095965357E77091738D9FBC21B0B8`。
- 候选链接 Wails runtime v2.11.0，依赖、bindings/runtime、Frontend dist、Release 和目录哈希保护通过。
- 2026-10-29，用户启动组合候选并明确确认 Change 034 验证通过，GUI 人工验收记为 `VERIFIED_BY_USER`。
- 2026-10-29，用户随后明确要求更新版本；组合候选已晋升为根目录 `build/bin/Q-Solver.exe`，晋升后启动冒烟验证通过。
- 旧稳定版已归档到 `build/bin/archive/superseded/change-034/`，测试区已清空；本轮未创建 Tag、Release 或 RC。

## 排除项

- Change 033 的 `pkg/mobile/**` 功能/测试与 Change 033 档案；
- `build/bin/**`、`frontend/dist/**`；
- 用户现有题库、导出文件、截图和配置；
- CCR、Provider、模型调用、日志和 secret；
- Wails bindings、依赖、Tag、Release、RC、upstream、main 和 force push。

## Git 授权

用户已明确授权并完成本轮 Change 033/034 提交、推送和稳定版晋升；不授权 upstream、main、force push、Tag、Release 或 RC。
