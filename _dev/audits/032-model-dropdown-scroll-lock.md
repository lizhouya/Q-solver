# Change 032 提交边界审计

## 功能/测试候选

- `frontend/src/components/ModelSelect.vue`
- `frontend/src/components/SettingsModal.vue`
- `frontend/model_layout_regression_test.mjs`

建议功能提交消息：

```text
fix: lock settings scroll while model dropdown is open

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

## 治理候选

- `_dev/audits/032-model-dropdown-scroll-lock.md`
- `_dev/changes/032-model-dropdown-scroll-lock/00-request.md`
- `_dev/changes/032-model-dropdown-scroll-lock/01-current-state.md`
- `_dev/changes/032-model-dropdown-scroll-lock/02-plan.md`
- `_dev/changes/032-model-dropdown-scroll-lock/03-implementation.md`
- `_dev/changes/032-model-dropdown-scroll-lock/04-validation.md`
- `_dev/changes/032-model-dropdown-scroll-lock/05-rollback.md`
- `_dev/changes/032-model-dropdown-scroll-lock/06-notes.md`
- `_dev/changes/INDEX.md`

建议治理提交消息：

```text
chore: add change 032 governance records

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

## 排除项

- `build/bin/**` 二进制与哈希清单；
- 用户配置、CCR 配置/数据库/日志；
- 仓库外测试日志和审计目录；
- API Key、Token、Cookie 或认证 URL；
- Tag、Release 和 RC。

## 当前结论

自动验证、用户 GUI 人工验收、候选晋升和晋升后独立启动验证均完成。根目录最终版 SHA-256 为 `C1F1873F054520F0F45FADFCCBCC087056D1556381DD1EFC66D4CD3748085F7B`。功能/测试提交已创建：`0d8b61b08ed3bd56708771ab0cbc1840151637ff`；治理记录纳入本轮第二提交。用户已授权普通 push 到 `origin/dev/qsolver-custom`；不创建 Tag、Release 或 RC。
