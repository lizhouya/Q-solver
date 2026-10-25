# Change 019 回滚预案

状态：未触发回滚。

## 策略

采用定向还原：只撤销 Change 019 在截图组件、SolveView、统一 Prompt 和对应测试中的局部改动；如验证程序已归档，只删除 Change 019 新文件并从 SHA256SUMS 移除对应条目后重新校验旧程序。

当前验证程序尚未复制到统一归档目录；若此时回滚，只处理 `build/bin/Q-Solver-change019-answer-summary-preview-validation.exe`，不得触碰正式程序和旧验证程序。

## 定向范围

- `frontend/src/components/RoundScreenshots.vue`：仅移除本地展开状态、切换按钮和 `scopeKey` 接口，恢复 Change 018 既有显示方式。
- `frontend/src/components/SolveView.vue`：仅移除 Change 019 的 `scopeKey` 接线，保留 Change 018 的 Session/Record 历史展示和流路由。
- `pkg/domain/prompts.go`：仅移除 `<QuestionAnsweringRules>`，保留其他系统行为和全部 Scene Prompt。
- 删除 Change 019 新增的专项测试，撤销续接测试中仅用于缓存复用的断言及 Solver 三图保护测试。
- 保留本 Change 的治理档案，记录回滚原因和实际结果。

## 触发条件

- 展开状态跨 Session/Record 泄漏。
- 收起导致图片缓存、导出兼容字段或题库数据丢失。
- 新增重复图片请求或迟到响应自动展开。
- OCR、翻译、秒杀、举一反三等场景发生明显回退。
- 构建产生意外 bindings、依赖或正式程序改动。

## 禁止操作

不得使用 `git reset --hard`、`git clean`、批量 checkout 或其他会覆盖 Change 016～018 叠加工作区的命令；不得删除题库、截图、旧导出或旧验证程序。
