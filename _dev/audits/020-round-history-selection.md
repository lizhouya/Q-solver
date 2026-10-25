# Change 020 实现后范围与安全审计

日期：2026-10-23
状态：实现后审计完成；GUI NEEDS_USER，race BLOCKED，Change 未关闭。

## 结论

实现遵循事前架构：保留 Session/Record 权威结构，仅在桌面前端增加逐轮派生投影和独立浏览目标。自动回归、普通构建、Wails 独立构建、保护哈希及统一归档校验通过；未发现需要修改 Go 后端、Wails bindings、Mobile、题库 schema 或依赖的情况。

## 实现范围

业务与测试文件：

- `frontend/src/stores/solution.js`
- `frontend/src/components/SolveView.vue`
- `frontend/src/components/HistoryItem.vue`
- `frontend/src/App.vue`（仅 legacy 取消后的选择修复）
- `frontend/history_round_selection_regression_test.mjs`（新增）
- `frontend/questionbank_continuation_regression_test.mjs`
- `frontend/round_screenshots_regression_test.mjs`（仅稳定 key 静态契约）

治理与产物：

- `_dev/changes/020-round-history-selection/00-request.md`～`06-notes.md`
- `_dev/audits/020-round-history-selection.md`
- `_dev/changes/INDEX.md`
- `build/bin/Q-Solver-change020-round-history-selection-validation.exe`
- `build/bin/archive/validation/Q-Solver-change020-round-history-selection-validation.exe`
- `build/bin/archive/validation/SHA256SUMS`

`round_screenshots_regression_test.mjs` 是实施中发现的必要测试白名单补充：只把静态 key 断言从 `recordId || idx` 更新为 `recordId || localId || idx`，没有改变 Change 019 图片行为。

## 正确性审计

- `historyEntries` 是只读派生，不复制或替代 Session/round 权威状态。
- typed 身份和写入继续使用 `epoch + sessionId + recordId`；Vue key 使用稳定 `sessionId + recordId`，避免重排和重新激活造成无意义重建。
- `selectedRoundTarget` 只控制浏览；typed `streamTarget` 和 legacy `legacyStreamTarget` 只控制后台写入。
- 查看同 Session 旧 Record 时，`isViewingStreamTarget` 为 false，后台 chunk 仍写真实新 Record。
- 题库续接恢复全部 Records 并选中最新 Record；历史 Answer 的模型上下文语义未改动。
- 删除仍以父 Session 为单位；删除其他 Session 保持选中 Record，删除流目标 Session 清理流状态，删除当前 Session 修复邻近选择。
- legacy 取消 `pop()` 后重新选择剩余最新 round，不留下悬挂浏览目标。
- reset 清空桌面展示、浏览目标和流目标，不触碰题库磁盘记录、截图或旧导出。

## 数据与安全边界

- 未新增持久化字段、单 Record 删除、单 Record 题库导出或从任意 Record 恢复上下文能力。
- 未读取、修改或输出 API Key、Mobile Token、密码或用户私人快捷键配置。
- 未调用真实模型、未上传截图、未消费真实 API 额度。
- 未删除、覆盖或迁移 `_记录`、截图、旧导出、正式 EXE 或旧验证程序。
- 未停止占用端口 38511 的进程。
- 未执行 commit、push、Tag、Release、`git reset --hard`、`git clean`、`git add .` 或 `git add -A`。

## 生成文件与依赖审计

Wails 构建前后以下 SHA-256 均一致：

- `frontend/wailsjs/go/app/App.d.ts`
- `frontend/wailsjs/go/app/App.js`
- `frontend/wailsjs/go/models.ts`
- `go.mod`
- `go.sum`
- `build/bin/Q-Solver.exe`

因此 Change 020 没有引入 bindings、模块依赖或正式程序漂移。独立验证程序使用 Wails v2.11.0 和 Go 1.25.13 构建，并已通过统一归档完整哈希校验。

## 验证审计

- 专项和五组兼容 JavaScript 回归：PASS。
- Mobile DOM：PASS。
- Frontend Vite build：PASS，存在既有 chunk-size warning。
- `go test ./... -count=1`、`go vet ./...`、`go build ./...`：PASS。
- race：Windows 启动阶段 `0xc0000139`，`BLOCKED`。
- Windows GUI：`NEEDS_USER`。

普通测试没有替代 race，自动测试没有替代 GUI 肉眼与真实交互验收。

## 最终状态

Change 020 的实现、普通自动验证、独立构建和统一归档已经完成；由于 GUI 仍待人工验收且 race 环境仍阻塞，状态保持“进行中 / 未关闭”。工作区继续叠加 Change 016～019，不得声称 clean；本 Change 没有提交、推送或发布授权。
