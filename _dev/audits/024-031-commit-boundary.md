# Change 024～031 整合提交边界审计

## 审计结论

用户已明确批准 Change 024～031 的一次性跨 Change 整合例外，并批准总计三个提交：Change 029 设计提交、累计功能/测试提交、治理/规则提交。设计提交 `8047488` 已完成；本审计定义后两个提交的精确边界。 功能/测试提交已创建：`2a8a506636496a1eb6c38662c6ef6eaccc1fe958`。

当前累计差异无法在不伪造归属的情况下按单 Change 拆分：配置连接模式、多协议适配器、CCR 诊断、错误可见化、视觉失败处理、题库失败恢复与逐题解析存在共享类型、bindings、测试和前端状态链。采用功能/测试与治理/规则双提交能保留可审计边界，同时不声称每个历史 Change 是独立原子提交。

## Commit 2：功能与测试

**目的**：提交 Change 024～031 已实现并通过本轮自动验证的业务源码、测试、必要 Wails app bindings 和依赖。

**完整消息**：

```text
feat: integrate question bank recovery and provider diagnostics

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

**文件白名单（84 个）**：

```text
app/error_diagnostics_test.go
app/llm.go
app/llm_test.go
app/mobile.go
app/questionbank.go
app/questionbank_next_exam_test.go
app/questionbank_test.go
frontend/ccr_settings_lifecycle_regression_test.mjs
frontend/ccr_ui_regression_test.mjs
frontend/llm_error_diagnostics_regression_test.mjs
frontend/model_bridge_timeout_regression_test.mjs
frontend/model_connectivity_regression_test.mjs
frontend/model_layout_regression_test.mjs
frontend/questionbank_history_regression_test.mjs
frontend/settings_regression_test.mjs
frontend/src/App.vue
frontend/src/components/ErrorDiagnostics.vue
frontend/src/components/ErrorView.vue
frontend/src/components/ModelSelect.vue
frontend/src/components/ProviderSelect.vue
frontend/src/components/QuestionBankExportModal.vue
frontend/src/components/SettingsModal.vue
frontend/src/components/SolveView.vue
frontend/src/services/api.js
frontend/src/services/modelBridgeTimeout.js
frontend/src/stores/questionbank.js
frontend/src/stores/settings.js
frontend/src/stores/solution.js
frontend/src/stores/ui.js
frontend/src/utils/modelCapabilities.js
frontend/wailsjs/go/app/App.d.ts
frontend/wailsjs/go/app/App.js
frontend/wailsjs/go/models.ts
go.mod
go.sum
pkg/config/config.go
pkg/config/connection.go
pkg/config/connection_test.go
pkg/config/manager.go
pkg/config/manager_test.go
pkg/domain/prompts.go
pkg/domain/prompts_test.go
pkg/llm/anthropic.go
pkg/llm/anthropic_test.go
pkg/llm/diagnostic.go
pkg/llm/diagnostic_test.go
pkg/llm/errors.go
pkg/llm/errors_test.go
pkg/llm/factory.go
pkg/llm/factory_test.go
pkg/llm/http_client.go
pkg/llm/http_client_test.go
pkg/llm/openai.go
pkg/llm/openai_responses.go
pkg/llm/openai_responses_test.go
pkg/llm/openai_test.go
pkg/llm/probe.go
pkg/llm/probe_test.go
pkg/llm/protocol/protocol.go
pkg/llm/protocol/protocol_test.go
pkg/llm/responses_failure_test.go
pkg/llm/service.go
pkg/llm/service_test.go
pkg/llm/types.go
pkg/mobile/bridge.go
pkg/mobile/error_diagnostics_test.go
pkg/mobile/state.go
pkg/mobile/web/index.html
pkg/mobile/web_dom_test.mjs
pkg/questionbank/continuation_test.go
pkg/questionbank/export.go
pkg/questionbank/organize.go
pkg/questionbank/organize_failure.go
pkg/questionbank/organize_failure_test.go
pkg/questionbank/segment_test.go
pkg/questionbank/store_next_exam_test.go
pkg/questionbank/store_test.go
pkg/questionbank/types.go
pkg/resume/service.go
pkg/resume/service_test.go
pkg/solution/conversation_images_test.go
pkg/solution/error_diagnostics_test.go
pkg/solution/solution.go
pkg/solution/solution_test.go
```

边界说明：

- `go.mod`/`go.sum` 中 `go 1.25.0` 与 Anthropic SDK v2.3.0 是实际源码要求，不是 Wails 生成噪音。
- `frontend/wailsjs/go/app/App.js`、`App.d.ts` 和 `models.ts` 仅保留新增 Wails 方法和 DTO；`models.ts` 行尾生成噪音已清除。
- `frontend/wailsjs/runtime/runtime.js` 与 `runtime.d.ts` 的 Wails CLI v2.16 Notification 生成差异已精确排除，不进入白名单。
- README、规则、审计和 Change 档案不进入本提交。

## Commit 3：治理、规则与说明

**目的**：归档 Change 024～031，记录验证与提交边界，建立长期构建产物生命周期规则，并同步中英文用户说明。

**完整消息**：

```text
chore: archive changes 024-031 and define artifact lifecycle

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

**文件白名单**：下列 70 个既有候选文件，加上本审计文件本身，共 71 个。

```text
AGENTS.md
CLAUDE.md
README.md
README_EN.md
_dev/README.md
_dev/audits/024-question-bank-organize-failure-recovery.md
_dev/audits/025-multi-protocol-model-connectivity.md
_dev/audits/026-ccr-gateway-validation.md
_dev/audits/027-vision-failure-detection.md
_dev/audits/028-llm-error-diagnostics.md
_dev/build-artifact-policy.md
_dev/changes/024-question-bank-organize-failure-recovery/00-request.md
_dev/changes/024-question-bank-organize-failure-recovery/01-current-state.md
_dev/changes/024-question-bank-organize-failure-recovery/02-plan.md
_dev/changes/024-question-bank-organize-failure-recovery/03-implementation.md
_dev/changes/024-question-bank-organize-failure-recovery/04-validation.md
_dev/changes/024-question-bank-organize-failure-recovery/05-rollback.md
_dev/changes/024-question-bank-organize-failure-recovery/06-notes.md
_dev/changes/025-multi-protocol-model-connectivity/00-request.md
_dev/changes/025-multi-protocol-model-connectivity/01-current-state.md
_dev/changes/025-multi-protocol-model-connectivity/02-plan.md
_dev/changes/025-multi-protocol-model-connectivity/03-implementation.md
_dev/changes/025-multi-protocol-model-connectivity/04-validation.md
_dev/changes/025-multi-protocol-model-connectivity/05-rollback.md
_dev/changes/025-multi-protocol-model-connectivity/06-notes.md
_dev/changes/026-ccr-gateway-validation/00-request.md
_dev/changes/026-ccr-gateway-validation/01-current-state.md
_dev/changes/026-ccr-gateway-validation/02-plan.md
_dev/changes/026-ccr-gateway-validation/03-implementation.md
_dev/changes/026-ccr-gateway-validation/04-validation.md
_dev/changes/026-ccr-gateway-validation/05-rollback.md
_dev/changes/026-ccr-gateway-validation/06-notes.md
_dev/changes/027-vision-failure-detection/00-request.md
_dev/changes/027-vision-failure-detection/01-current-state.md
_dev/changes/027-vision-failure-detection/02-plan.md
_dev/changes/027-vision-failure-detection/03-implementation.md
_dev/changes/027-vision-failure-detection/04-validation.md
_dev/changes/027-vision-failure-detection/05-rollback.md
_dev/changes/027-vision-failure-detection/06-notes.md
_dev/changes/028-llm-error-diagnostics/00-request.md
_dev/changes/028-llm-error-diagnostics/01-current-state.md
_dev/changes/028-llm-error-diagnostics/02-plan.md
_dev/changes/028-llm-error-diagnostics/03-implementation.md
_dev/changes/028-llm-error-diagnostics/04-validation.md
_dev/changes/028-llm-error-diagnostics/05-rollback.md
_dev/changes/028-llm-error-diagnostics/06-notes.md
_dev/changes/028-llm-error-diagnostics/evidence/baseline.json
_dev/changes/028-llm-error-diagnostics/evidence/boundary-audit.json
_dev/changes/028-llm-error-diagnostics/evidence/change028.patch
_dev/changes/028-llm-error-diagnostics/evidence/final-status.txt
_dev/changes/029-build-artifact-lifecycle/02-plan.md
_dev/changes/029-build-artifact-lifecycle/03-implementation.md
_dev/changes/029-build-artifact-lifecycle/04-validation.md
_dev/changes/029-build-artifact-lifecycle/05-rollback.md
_dev/changes/029-build-artifact-lifecycle/06-notes.md
_dev/changes/030-latest-validation-build/00-request.md
_dev/changes/030-latest-validation-build/01-current-state.md
_dev/changes/030-latest-validation-build/02-plan.md
_dev/changes/030-latest-validation-build/03-implementation.md
_dev/changes/030-latest-validation-build/04-validation.md
_dev/changes/030-latest-validation-build/05-rollback.md
_dev/changes/030-latest-validation-build/06-notes.md
_dev/changes/031-inline-question-explanations/00-request.md
_dev/changes/031-inline-question-explanations/01-current-state.md
_dev/changes/031-inline-question-explanations/02-plan.md
_dev/changes/031-inline-question-explanations/03-implementation.md
_dev/changes/031-inline-question-explanations/04-validation.md
_dev/changes/031-inline-question-explanations/05-rollback.md
_dev/changes/031-inline-question-explanations/06-notes.md
_dev/changes/INDEX.md
_dev/audits/024-031-commit-boundary.md
```

## 本轮验证证据

| 项目 | 结果 |
|---|---|
| 聚焦 Go 测试（7 个包） | PASS |
| `go test ./... -count=1` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `go mod verify` | PASS |
| 14 份前端回归脚本 | PASS |
| Vite production build | PASS |
| Wails 独立候选构建 | PASS；CLI v2.16.0，实际链接 runtime v2.11.0，保护参数启用 |
| 候选与晋升后根目录启动验证 | PASS；WebView2、无 Node/Vite、正常退出码 0、无残留进程 |
| 构建保护 | `go.mod`、`go.sum`、`frontend/dist`、Git index 不变；runtime 生成噪音已清除并恢复到构建前哈希 |
| 差异空白检查 | 业务源码与治理 Markdown PASS；`evidence/change028.patch` 保留原始 patch 空白提示 |
| 产物迁移 | 44 个迁移前文件均可按原哈希恢复；25 份当前目录级清单通过 |
| Release 保护 | 7 个文件哈希逐项不变 |
| Windows race | 既有 `0xc0000139` 环境 BLOCKED，不伪报 PASS |
| Change 031 GUI 真实导出 | 没有新人工证据，继续 `NEEDS_USER` |

候选与最终根目录 EXE：

- 路径：`build/bin/Q-Solver.exe`
- 大小：42,331,136 字节
- SHA-256：`82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46`

## 明确排除

以下内容不进入任何提交：

- `build/bin/**` 的 EXE、`SHA256SUMS` 和归档目录；
- Q-Solver 用户配置；
- 本机或远程 CCR 配置、数据库、日志、管理 URL 或认证字段；
- 仓库外 re-export helper、CCR probe、迁移/验证脚本和审计日志；
- API Key、Token、Cookie、Authorization 或完整认证 URL；
- 与本轮无关的本机文件；
- Tag、GitHub Release、RC 或发布资产变更。

## 推送边界

完成两个实施提交并确认工作树与暂存区符合预期后，仅允许普通 push 到 `origin/dev/qsolver-custom`。禁止推送 `upstream`、修改 `main`、force push、创建或推送 Tag、创建 GitHub Release 或发布新 RC。
