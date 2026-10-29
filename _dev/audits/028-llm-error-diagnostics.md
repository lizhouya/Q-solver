# Change 028 差异边界审计

日期：2026-10-26
状态：已审计；未暂存、未提交、未推送、未发布；不 Close。

## 1. 基线

- 分支 dev/qsolver-custom；HEAD 43022e10cd8fb1b38b72b56d95bc39b8bb4d201d，任务期间未变化。
- 任务入口 42 个 tracked 修改、35 个 untracked 条目，暂存区为空。
- 最终 51 个 tracked 修改、45 个 untracked 条目（目录折叠），暂存区仍为空；差额均为本次白名单内容。
- 入口记录 1286 个文件/旧 EXE 哈希。最终仅 15 个入口文件内容变化（含 INDEX），1271 个完全相同，未删除任何入口文件。
- 其中原有未提交的 errors.go、errors_test.go、openai_responses.go、openai_responses_test.go、solution.go 和 INDEX 与本次重叠；通过任务入口副本与增量补丁审计，不以 HEAD 版本覆盖。

## 2. 本次修改的已有代码与测试（14 个）

- `app/mobile.go`
- `frontend/src/App.vue`
- `frontend/src/components/ErrorView.vue`
- `frontend/src/components/SolveView.vue`
- `frontend/src/stores/solution.js`
- `pkg/llm/errors.go`
- `pkg/llm/errors_test.go`
- `pkg/llm/openai_responses.go`
- `pkg/llm/openai_responses_test.go`
- `pkg/mobile/bridge.go`
- `pkg/mobile/state.go`
- `pkg/mobile/web/index.html`
- `pkg/mobile/web_dom_test.mjs`
- `pkg/solution/solution.go`

## 3. 本次新增代码与测试（8 个）

- `pkg/llm/diagnostic.go`
- `pkg/llm/diagnostic_test.go`
- `pkg/llm/responses_failure_test.go`
- `pkg/solution/error_diagnostics_test.go`
- `app/error_diagnostics_test.go`
- `pkg/mobile/error_diagnostics_test.go`
- `frontend/llm_error_diagnostics_regression_test.mjs`
- `frontend/src/components/ErrorDiagnostics.vue`

## 4. 治理文件

- `_dev/changes/028-llm-error-diagnostics/00-request.md`：授权任务、范围与验收。
- `01-current-state.md`：事前事实与保护基线。
- `02-plan.md`：事前方案与文件白名单。
- `03-implementation.md`：实现、协议边界与分类映射。
- `04-validation.md`：本次命令、结果、失败/阻塞。
- `05-rollback.md`：只撤销本次增量的回滚预案。
- `06-notes.md`：使用方式、产物与下一轮复现。
- `evidence/baseline.json`：入口哈希及本地保护副本位置，无凭据内容。
- `evidence/boundary-audit.json`：文件集合、保护结果及产物哈希。
- `evidence/final-status.txt`：最终工作区状态（目录折叠），仅作为审计证据。
- `evidence/change028.patch`：相对任务入口而非 HEAD 的增量，只用于审查，不自动应用或回滚。
- `evidence/*.log`：本机测试和构建输出；按已有 gitignore 忽略，关键结果已汇总进验证文档。
- `_dev/changes/INDEX.md`：只增加/更新 Change 028 行与摘要，不重写其他 Change。
- 本审计文件：`_dev/audits/028-llm-error-diagnostics.md`。

## 5. 保护结果

- go.mod、go.sum、frontend/package.json、frontend/package-lock.json：与入口哈希一致。
- frontend/wailsjs 全部文件：与入口哈希一致；本次未生成 bindings。
- Provider 工厂、CCR 设置、模型目录、图片、Prompt、上下文代码（除 Solver 报错处）：未改。
- 两个题库捕获器及 schema：未改；新测试确认可选参数透传，不持久化主解题诊断。
- 所有旧 EXE：与入口一致；只新增本次独立验证程序，未启动。
- 原有 Change 024–027 文档和其他未提交文件：除上述显式重叠文件外保持原字节。
- 没有用户配置、凭据、题库、截图、旧导出迁移或写入。

## 6. 空白检查

- 全工作区 git diff --check：FAIL；85 处均位于 frontend/wailsjs/go/models.ts。
- models.ts 与任务入口完全相同，属于原有工作；未清理或覆盖。
- 本次 tracked 文件范围 git diff --check：PASS。
- 所有相对入口新增行（含 untracked 文件）空白检查：PASS。
- git diff --no-index 的退出 1 表示存在内容差异，不是空白错误；最终通过范围 diff-check 与新增行扫描独立确认。

## 7. 验证与产物

普通 Go test/vet/build、14 个前端回归、Mobile DOM、Vite、固定 Wails v2.11.0 与 prod 诊断专项 PASS。prod 全套旧日志测试 FAIL；race 0xc0000139 BLOCKED；GUI/真实 Provider NEEDS_USER。详见 Change 028 的 04-validation.md，不将部分通过冒充全通过。

- 程序：`build/bin/Q-Solver-change028-llm-error-diagnostics.exe`。
- 字节：44012032。
- SHA-256：`370beea9dcff40f4776238f18ef0d4c3c34ac1b711bd3fb3829f3b470644fee7`。
- 构建跳过 bindings、前端安装、mod tidy、go.mod 同步，无依赖污染。

## 8. 隐私与授权

仅 fake/loopback 测试，无真实模型网络请求。摘要仅保留白名单字段和固定文案，不保存/输出 SDK error、Token、Authorization、URL、Prompt 或截图正文。新增测试含显式测试哨兵，不使用真实凭据。

用户禁止 Git 写操作；本次没有 add、commit、push、Tag、Release。任何后续提交仍需按实际重叠工作重新审计和取得授权，不可直接按整文件混合提交。
