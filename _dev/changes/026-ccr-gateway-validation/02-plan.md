# Change 026 实施方案

日期：2026-10-25
状态：方案已批准，待按 TDD 实施。

## 1. 方案概述

新增显式 `direct | ccr` 连接模式。Direct 保留 Change 025 三协议；CCR 固定复用现有 OpenAI Chat adapter，只从 `/v1/models` 同步目录，禁止 probe，由 CCR 承担上游转换与路由。先修复 App typed-nil mapper，再加入配置规范化和后端零调用保证，随后拆分前端目录/probe 生命周期、增加 bridge deadline，最后修复 UI 与模型页滚动。

## 2. 稳定契约

### 2.1 配置

```go
const (
    ConnectionModeDirect = "direct"
    ConnectionModeCCR    = "ccr"
)

func NormalizeConnectionMode(value string) (string, error)
func NormalizeCCRBaseURL(value string) (string, error)
func (c Config) NormalizedConnectionSettings() (Config, error)
```

- 空模式仅为旧配置兼容解释为 direct；未知非空值拒绝。
- CCR 固定有效协议为 `openai_chat`。
- CCR URL 只接受 http/https root、`/`、`/v1`、`/v1/`，统一为 `/v1`；拒绝 userinfo、query、fragment、完整 endpoint 和额外 path。
- Load 只规范化内存，不主动写回；Update/Patch 在既有候选快照事务中保存 canonical 值。

### 2.2 后端网络

- CCR 目录：现有 `OpenAIAdapter.GetModels`，一次 `GET /v1/models`。
- CCR 生成：现有 Chat Completions 路径。
- CCR probe：Service 在 provider 构造前返回 `probe_disabled / validate_probe / requestSent=false`，零网络。
- 不新增 adapter、fallback、重试或协议推断。

### 2.3 前端状态

- catalog identity：模式、URL、Key、有效目录协议；不含 model。
- probe identity：catalog identity + model。
- CCR 在可见设置弹窗进入模型 tab 时自动同步一次；CTA 只切 tab。
- 同 revision 在途 Promise 复用，成功结果缓存；失败/空/timeout 可手动刷新。
- 仅目录/probe bridge 等待 20 秒；timeout 只停止 UI 等待，不表示后端取消。

## 3. 分步实施计划

| 步骤 | 操作 | 主要文件 | 完成标志 |
|---|---|---|---|
| 1 | typed-nil RED/GREEN | `app/llm.go`、`app/llm_test.go` | 成功 App probe 不 panic |
| 2 | 连接模式与 CCR URL RED/GREEN | `pkg/config/connection*`、manager/config tests | 旧配置兼容、保存 canonical、原子性保持 |
| 3 | CCR 工厂/目录/零 probe RED/GREEN | `pkg/llm/factory.go`、`service.go`、`errors.go` 及测试 | 目录单 GET、probe 零请求 |
| 4 | bridge deadline 与目录状态 | `frontend/src/services`、`stores/ui.js`、Node tests | pending 最多等待 20 秒 |
| 5 | revision 与同步生命周期 | `stores/settings.js`、Node tests | model 不清目录、自动同步单发 |
| 6 | direct/CCR UI 与本地过滤 | Provider/Model/Settings 组件 | CCR 不显示 probe/推测能力 |
| 7 | 模型页滚动修复 | `SettingsModal.vue`，必要时 `DomainSelector.vue` | 场景可滚动访问 |
| 8 | 固定 bindings、全量验证与独立 EXE | Wails 生成文件、治理文档 | 自动验证和产物记录完成 |

## 4. 允许修改的业务文件

- `app/llm.go`、`app/llm_test.go`
- `pkg/config/config.go`、`pkg/config/connection.go`、`pkg/config/connection_test.go`、`pkg/config/manager.go`、`pkg/config/manager_test.go`
- `pkg/llm/errors.go`、`pkg/llm/errors_test.go`、`pkg/llm/factory.go`、`pkg/llm/factory_test.go`、`pkg/llm/service.go`、`pkg/llm/service_test.go`
- `frontend/src/services/api.js`、`frontend/src/services/modelBridgeTimeout.js`
- `frontend/src/stores/settings.js`、`frontend/src/stores/ui.js`
- `frontend/src/components/ProviderSelect.vue`、`ModelSelect.vue`、`SettingsModal.vue`；仅测试证明必要时修改 `DomainSelector.vue`
- `frontend/settings_regression_test.mjs`、`frontend/model_connectivity_regression_test.mjs`、新增 deadline/layout 回归
- 固定 Wails v2.11.0 生成的 `frontend/wailsjs/go/app/App.js`、`App.d.ts`、`models.ts`

越过此白名单前必须先在 `03-implementation.md` 说明原因和影响；依赖文件不在允许修改范围。

## 5. 测试与验证计划

### Go 聚焦

```text
GOTOOLCHAIN=go1.25.13 go test ./pkg/config ./pkg/llm/... ./app -count=1
```

覆盖 typed nil、成功 probe、旧配置不写回、CCR URL、工厂路由、目录单 GET、probe 零网络和 direct 回归。

### 前端聚焦

```text
node frontend/model_bridge_timeout_regression_test.mjs
node frontend/model_connectivity_regression_test.mjs
node frontend/settings_regression_test.mjs
node frontend/model_layout_regression_test.mjs
npm --prefix frontend run build
```

覆盖 timeout 语义、revision、自动同步单发、无隐式推理、过滤、文案和 CSS 目标类。

### 全量与集成

```text
GOTOOLCHAIN=go1.25.13 go test ./... -count=1
GOTOOLCHAIN=go1.25.13 go vet ./...
GOTOOLCHAIN=go1.25.13 go build ./...
GOTOOLCHAIN=go1.25.13 go mod verify
全部 frontend/*_regression_test.mjs
npm --prefix frontend run build
git diff --check
```

用固定 Wails v2.11.0 重生 bindings 并构建 `build/bin/Q-Solver-change026-ccr-gateway-validation.exe`。构建前后比较依赖和受保护 EXE 哈希。

Race：

```text
GOTOOLCHAIN=go1.25.13 go test -race ./pkg/config ./pkg/llm ./app -count=1
```

若为 `0xc0000139`，记为 `BLOCKED`。Windows GUI 与真实 CCR 记为 `NEEDS_USER`。

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| typed nil 再次穿过 interface | Wails Promise 永久 pending | mapper 与调用点双层 nil 防护 + App 成功测试 + 前端 deadline |
| CCR URL 重复 `/v1` 或接受 endpoint | 404 或错误路径 | 纯函数表驱动规范化，严格 path 白名单 |
| UI 隐藏但旧前端仍可 probe | 产生费用 | Service 在网络前权威拒绝，测试服务器请求数为 0 |
| model 变化清空目录 | 无法稳定切换 | catalog/probe identity 分离 |
| CTA 与 watcher 双请求 | 目录请求重复 | CTA 只切 tab，watcher 唯一自动触发 |
| 迟到结果覆盖新配置 | 展示错误目录/状态 | session + revision + request version 三重校验 |
| 前端 timeout 被误解为取消 | 错误费用判断 | 固定文案明确后台不保证取消，不伪造 requestSent |
| 修复滚动形成嵌套滚动 | 场景仍难操作 | 模型页唯一纵向滚动，子列表 overflow visible |
| bindings/依赖漂移 | 引入无关变化 | 固定 Wails 2.11.0，构建前后哈希与 diff 审计 |

## 7. 回滚预案

只做 Change 026 白名单定向回滚，不使用 destructive Git。先回退前端 CCR UI/生命周期，再回退后端模式和规范化，固定 Wails 2.11.0 重生 bindings。typed-nil 修复是独立确认缺陷，若保留须在回滚记录中明确。绝不自动改写用户已保存的 CCR 配置，不删除题库、截图、导出、治理记录或验证程序。
