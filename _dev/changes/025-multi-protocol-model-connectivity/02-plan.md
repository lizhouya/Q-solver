# Change 025 多协议模型连接兼容实施计划

> **For agentic workers:** 按本计划逐任务实施；每个任务先写失败测试，再做最小实现并复核差异。不得跳过治理、依赖审计或安全边界。

**Goal：** 让 Q-Solver 在用户显式选择协议的前提下，完整支持 OpenAI-compatible Chat Completions、OpenAI Responses API 和 Anthropic 原生 Messages API，并提供互相独立的文字与图片实测，修复当前“文字测试成功却无法证明截图可用”和“Responses-only 模型无法连接”的结构性缺陷。

**Architecture：** 保留现有 provider-neutral `llm.Provider` 主接口，新增稳定协议字段和单一 provider 工厂；三个 adapter 分别实现一种 wire protocol，并复用现有 `Message` / `ContentPart`。设置页使用完整连接草稿和结构化诊断 DTO，所有测试只针对当前显式协议发送一次请求，绝不自动换协议、双发或读取用户资料。

**Tech Stack：** Go 1.25.13、Wails v2.11.0、Vue 3、Pinia 3、Vite 4.5.14、`github.com/openai/openai-go v1.12.0`、`github.com/anthropics/anthropic-sdk-go/v2 v2.3.0`。

**Spec：** 本计划获批后先固化到 `_dev/changes/025-multi-protocol-model-connectivity/02-plan.md`；`00-request.md`、`01-current-state.md`、`02-plan.md` 完成前不得修改业务源码。

## Context

用户观察到：修正地址后 DeepSeek 可以使用；Claude 的文字连接测试通过，但截图解题回复称无法看到图片；GPT/Codex 无法连接。只读核查确认，当前所有模型都由 `pkg/llm/openai.go` 的 Chat Completions adapter 处理，连接测试只发送文本 `hi`，本地模型资料中的图片图标和上下文长度也不是当前 endpoint 的实测结果。现有证据不足以确认 Claude 或 GPT/Codex 的具体失败根因，因此本 Change 只修复可确认的协议、诊断和调用链缺陷，不把中转丢图、模型映射、鉴权、额度或 endpoint 猜测写成事实。

用户已选择“同时支持两种接入体系”：保留 OpenAI-compatible，并新增 Anthropic 原生 Messages；OpenAI-compatible 内同时支持 Chat Completions 与 Responses。这个架构选择不等于实施、依赖安装、真实 API 调用或 Git/发布授权。

## Global Constraints

- 固定分支和现有工作树；先只读预检 Change 024 未提交改动，绝不清理或覆盖。
- 固定 `GOTOOLCHAIN=go1.25.13` 和 Wails CLI v2.11.0；不使用 PATH 中其他 Wails 版本。
- Anthropic 原生实现只使用官方 `github.com/anthropics/anthropic-sdk-go/v2 v2.3.0`；不使用 OpenAI shim 或手写 HTTP。
- `openai-go` 保持 v1.12.0；不升级 Wails、Vue、Pinia、Vite 或其他无关直接依赖。
- 自动测试不调用真实模型、不读取真实配置或密钥、不上传用户截图/简历/题库内容、不消费真实额度。
- 不读取、记录或输出 API Key、Token、Authorization、完整 BaseURL、请求/响应体、Prompt、Base64、SDK 原始错误或私人配置。
- 不删除、覆盖或迁移 `_记录`、截图、题库、旧导出、正式 `Q-Solver.exe` 或任何旧验证程序。
- 不执行 `git reset --hard`、`git clean`、`git add .` 或 `git add -A`。
- 未另行授权时不暂存、不 commit、不 push、不创建 Tag、不创建 Release、不发布。
- Windows GUI 和真实 provider 验收始终单列为 `NEEDS_USER`；自动测试不能代替。

---

## 目标契约

### 1. 显式协议

在 `pkg/llm/protocol` 定义：

```go
type APIProtocol string

const (
    OpenAIChat       APIProtocol = "openai_chat"
    OpenAIResponses  APIProtocol = "openai_responses"
    AnthropicMessages APIProtocol = "anthropic_messages"
)

func Normalize(value string) (APIProtocol, error)
```

`config.Config` 新增：

```go
APIProtocol string `json:"apiProtocol,omitempty"`
```

规则：

- 空值只为兼容旧配置，运行时解释为 `openai_chat`，不强制写回磁盘。
- 三个稳定值原样通过；未知非空值拒绝，不静默降级。
- 不根据模型名、品牌、BaseURL 或域名推断协议。
- 空 API Key/Model 是合法的“尚未配置”启动状态；只在真实请求、列表或 probe 前失败。
- OpenAI-compatible 网关中的 Claude 仍可显式选择 Chat 或 Responses；Anthropic 官方地址才建议 Messages，但建议不替代用户选择。

### 2. 单一 provider 工厂与不可变快照

新增：

```go
func NewProvider(cfg config.Config) (Provider, error)
```

- 工厂按规范化协议创建 `OpenAIChatAdapter`、`OpenAIResponsesAdapter` 或 `AnthropicAdapter`。
- adapter 保存 `config.Config` 值快照，不保存 `&Service.config` 等共享可变指针。
- 生产业务不再直接调用 `NewOpenAIAdapter`；主解题、简历、题库首次整理、失败重试和兼容入口全部走工厂。
- 未配置启动由安全的 unavailable provider 或延迟校验承接，不能留下会 panic 的 nil provider。
- 工厂错误必须在题库封段、设置 busy/cancel 状态或启动 worker 前返回，避免半变更状态。

保留现有 `llm.Provider` 方法，避免波及全部 fake provider：

```go
type Provider interface {
    GenerateContentStream(context.Context, []Message, StreamCallback) (Message, error)
    GenerateContent(context.Context, string, []Message) (Message, error)
    GetModels(context.Context) ([]string, error)
    TestChat(context.Context) error
}
```

低成本诊断使用独立可选内部接口，不给现有 fake 强加新方法：

```go
type Prober interface {
    Probe(context.Context, ProbeRequest) ProbeResult
}
```

### 3. 安全错误模型

新增内部 `ProviderError`：

```go
type ProviderError struct {
    Code       ErrorCode
    Stage      ErrorStage
    Message    string
    Retryable  bool
    HTTPStatus int
    Protocol   protocol.APIProtocol
    cause      error // 内部使用，不序列化
}
```

`Error()` 只返回白名单中文 `Message`。固定 stage：

- `validate_config`
- `build_request`
- `send_request`
- `read_response`
- `stream_response`
- `validate_response`
- `validate_probe`

固定 code 至少覆盖：

- `invalid_config`
- `cancelled`
- `timeout`
- `network_error`
- `tls_error`
- `authentication_failed`
- `permission_denied`
- `not_found_or_protocol_mismatch`
- `invalid_request`
- `rate_limited`
- `quota_exhausted`
- `request_too_large`
- `provider_unavailable`
- `empty_response`
- `invalid_response`
- `incomplete_response`
- `stream_interrupted`
- `refused`
- `vision_mismatch`
- `vision_inconclusive`

分类顺序为 `errors.Is` 上下文错误 → 类型化 TLS/网络错误 → SDK typed error 的 HTTP status 与白名单 type/code → 安全兜底。禁止从 `err.Error()` 截 JSON、正则猜 provider 能力或向 UI/日志传原文。普通 404 只能提示“端点、协议或模型待检查”；只有稳定结构化 code 才进一步细分。

共享 HTTP client 禁止自动重定向；OpenAI 和 Anthropic SDK 自动重试设为 0。一次业务调用或一次 probe 最多一个网络请求，不得跟随跳转把凭据带到另一主机。

### 4. 结构化 Wails DTO

新增 App DTO：

```go
type ConnectionDraft struct {
    APIKey      string `json:"apiKey"`
    BaseURL     string `json:"baseURL"`
    Model       string `json:"model"`
    APIProtocol string `json:"apiProtocol"`
}

type ProviderErrorDTO struct {
    Code       string `json:"code"`
    Stage      string `json:"stage"`
    Message    string `json:"message"`
    Retryable  bool   `json:"retryable"`
    HTTPStatus int    `json:"httpStatus"`
    Protocol   string `json:"protocol"`
}

type ModelListResult struct {
    Models []string          `json:"models"`
    Error  *ProviderErrorDTO `json:"error,omitempty"`
}

type ModelProbeRequest struct {
    Connection ConnectionDraft `json:"connection"`
    Kind       string          `json:"kind"` // text | vision
}

type ModelProbeResult struct {
    Kind      string            `json:"kind"`
    Outcome   string            `json:"outcome"` // passed | failed | inconclusive
    Protocol  string            `json:"protocol"`
    Message   string            `json:"message"`
    Error     *ProviderErrorDTO `json:"error,omitempty"`
    RequestSent bool            `json:"requestSent"`
}
```

新增：

```go
func (a *App) ListModelsForDraft(ConnectionDraft) ModelListResult
func (a *App) ProbeModelConnection(ModelProbeRequest) ModelProbeResult
```

保留 `GetModels`、`TestConnection`、`GetModelsForConnection`、`TestConnectionForConnection` 的签名；旧入口固定保持 `openai_chat` 语义并委托新实现。新 UI 不再丢弃后端安全分类。

### 5. 文字与图片实测

- 文字 probe：非流式低成本请求，要求返回随机 token；trim 后精确匹配才通过。
- 图片 probe：使用 Go 标准库 `image/png` 生成高对比度随机两位七段数字；文本 prompt 不含答案；通过当前 adapter 的**流式多模态路径**发一个请求，以覆盖真实截图解题的图片转换和流终态。
- 图片结果只允许 trim、固定前缀和全角数字的确定性规范化；不从长段落猜答案。
- 错数字为 `vision_mismatch`，无法唯一解析为 `vision_inconclusive`；网络/鉴权/额度错误保持原分类，不能显示“不支持图片”。
- probe 不读取 screenshot buffer、简历、题库、历史会话或角色 Prompt。
- 文字和图片状态独立、只在当前设置弹窗草稿内存中保存；四项连接字段任一变化或关闭弹窗即失效。
- 设置页直接说明“每次点击会调用一次模型，可能产生少量费用”；按钮点击本身是显式触发，不再增加隐式探测或组合双请求。

---

## Task 1：建立 Change 025 事前治理与依赖基线

**Files：**
- Create: `_dev/changes/025-multi-protocol-model-connectivity/00-request.md`～`06-notes.md`
- Create: `_dev/audits/025-multi-protocol-model-connectivity.md`
- Modify: `_dev/changes/INDEX.md`

- [ ] **Step 1：只读预检。** 记录仓库分支、remote、工作树、Change 024 未提交文件和 Change 编号；出现非预期冲突只报告，不清理。
- [ ] **Step 2：建立受保护基线。** 记录 `go.mod`、`go.sum`、`frontend/package*.json`、正式及全部验证 EXE 的名称、大小、SHA-256。
- [ ] **Step 3：完成 `00-request.md`。** 固定用户现象、三协议目标、验收、非目标、隐私、真实 API 和 Git 授权边界。
- [ ] **Step 4：完成 `01-current-state.md`。** 记录已确认调用链、8 个直接 adapter 构造点、文字测试局限、静态能力误导和“具体 provider 根因未知”。
- [ ] **Step 5：完成 `02-plan.md`。** 固化本计划的接口、TDD 顺序、依赖门槛、回滚和人工验收。
- [ ] **Step 6：初始化 `03`～`06`、审计和 INDEX。** 标为未实施/未验证，不预写 PASS；前三份文档完成后才进入业务代码。

**验收：** 业务源码、依赖和 bindings 零变化；旧输出、配置和程序零变化。

## Task 2：以 TDD 建立协议字段和原子配置保存

**Files：**
- Create: `pkg/llm/protocol/protocol.go`
- Create: `pkg/llm/protocol/protocol_test.go`
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/manager.go`
- Modify: `pkg/config/manager_test.go`
- Modify: `app/settings_test.go`

- [ ] **Step 1：写 RED 协议测试。** 覆盖空值→Chat、三个稳定值、未知非空值拒绝，以及不接受品牌/模型/URL 别名。
- [ ] **Step 2：写 RED 兼容测试。** 旧加密 JSON 无字段时 Load 成功且解释为 Chat；不自动写回；空 Key/Model 仍可启动。
- [ ] **Step 3：写 RED 事务测试。** `UpdateFromJSON` 和 `Patch` 保存失败时内存配置、`oldConfig`、subscriber 和 active provider 全不变；成功时必须先持久化、再原子替换、最后锁外通知一次；并发更新保持磁盘与发布顺序一致。
- [ ] **Step 4：实现协议规范化与字段校验。** `Config.Validate` 只增加协议枚举检查，不把“尚未配置”变成启动错误。
- [ ] **Step 5：最小重构 ConfigManager。** 引入更新串行锁和 `saveSnapshot(candidate)`；候选副本先校验并落盘，成功后才替换内存并通知。保留当前 typed config 字段和前端 `backendSnapshot` 兼容方式，不扩大为通用配置框架重写。
- [ ] **Step 6：运行聚焦测试转 GREEN。** `GOTOOLCHAIN=go1.25.13 go test ./pkg/llm/protocol ./pkg/config ./app -count=1`。

**验收：** 保存失败不会让运行 provider 与磁盘/界面状态分裂；不记录配置内容。

## Task 3：以 TDD 建立安全错误、HTTP 策略和工厂

**Files：**
- Create: `pkg/llm/errors.go`
- Create: `pkg/llm/errors_test.go`
- Create: `pkg/llm/http_client.go`
- Create: `pkg/llm/http_client_test.go`
- Create: `pkg/llm/factory.go`
- Create: `pkg/llm/factory_test.go`

- [ ] **Step 1：写 RED 分类测试。** 用伪 Key、私有 URL、Authorization、响应 JSON、Base64、Windows 路径作为 sentinel，覆盖 context、TLS、DNS、401/403/404/413/429/5xx 和 unknown。
- [ ] **Step 2：写 RED 防泄漏测试。** `ProviderError.Error()`、DTO、日志和序列化结果均不得出现 sentinel；404 文案不得假定唯一根因。
- [ ] **Step 3：写 RED HTTP 测试。** 301/302/307/308 不跟随，目标服务零请求；adapter 重试关闭后服务端仅收到一次请求。
- [ ] **Step 4：写 RED 工厂测试。** 三协议路由、旧空值 Chat、未知协议拒绝、空配置安全构造、值快照不可变、模型名/域名不参与路由。
- [ ] **Step 5：实现安全类型和共享 HTTP client。** 只保存内部 cause；任何用户可见文本来自固定映射。
- [ ] **Step 6：实现工厂骨架。** 每个协议必须命中独立 adapter；尚未实现的分支返回明确安全错误，绝不暂时回落 Chat。
- [ ] **Step 7：运行聚焦测试转 GREEN。** 不读取真实配置或环境密钥。

**验收：** 单请求、零重定向、零原文泄漏、零协议猜测。

## Task 4：修正 OpenAI Chat 基线

**Files：**
- Modify: `pkg/llm/openai.go`
- Modify/Create: `pkg/llm/openai_test.go`
- Modify: `pkg/llm/types.go`

- [ ] **Step 1：写 RED 请求形状测试。** system/user/assistant 与多图顺序不变；图片为 `image_url`；PDF 使用 SDK file content part；非法 data URL/Base64 在发请求前失败。
- [ ] **Step 2：写 RED token 与次数测试。** 正常业务不固定输出 token；低成本 probe 使用 `max_completion_tokens=32`；字段不兼容也只失败一次，不改字段重发。
- [ ] **Step 3：写 RED 非流式终态测试。** 空 choices/空正文、refusal、`length`、正常 `stop`、工具或未知 finish reason 分别分类。
- [ ] **Step 4：写 RED 流式终态测试。** 正常 stop、reasoning 扩展、partial 后失败、普通 EOF 无 finish reason、取消；缺终态不得持久化为成功。
- [ ] **Step 5：实现值快照 client 和类型化错误。** 删除从 `err.Error()` 截 JSON 及记录 SDK 原始错误的逻辑；显式 Key/BaseURL/HTTP client/零重试。
- [ ] **Step 6：实现 PDF、终态和正文校验。** 保留兼容 reasoning 扩展，但解析失败只记录安全 code/长度。
- [ ] **Step 7：聚焦测试转 GREEN。** 同时跑 `pkg/solution` 现有多图、上下文、取消回归。

**验收：** Chat 现有 DeepSeek/兼容端点语义不回归；不再把 PDF 当图片或把异常 EOF 当成功。

## Task 5：以 TDD 新增 OpenAI Responses adapter

**Files：**
- Create: `pkg/llm/openai_responses.go`
- Create: `pkg/llm/openai_responses_test.go`
- Modify: `pkg/llm/factory.go`
- Modify: `pkg/llm/factory_test.go`

- [ ] **Step 1：写 RED endpoint 和输入测试。** 显式协议访问 Responses endpoint；完整本地历史转 input；图片为 `input_image`；PDF 为 `input_file`；顺序不变。
- [ ] **Step 2：写 RED 状态测试。** `Store=false`，不用 `previous_response_id`；`OutputText()`、empty、refusal、failed、incomplete 各有稳定结果。
- [ ] **Step 3：写 RED SSE 测试。** 处理 output-text delta、公开 reasoning-summary delta、completed/failed/incomplete/error；completed 不重复追加已发 delta；EOF 无 terminal event 为 `stream_interrupted`。
- [ ] **Step 4：写 RED probe/次数测试。** `max_output_tokens=32`；任意失败都不改走 Chat，服务端只收到一次请求。
- [ ] **Step 5：按 `openai-go v1.12.0` 现有类型实现。** 不升级 SDK，不开启 background/store/server conversation。
- [ ] **Step 6：接入工厂并转 GREEN。** 验证 Chat/Responses 请求类型完全隔离。

**验收：** Responses-only 模型具备明确通道；失败不触发 Chat fallback；本地 KeepContext 语义不变。

## Task 6：审计并接入 Anthropic 官方 SDK v2.3.0

**Files：**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `pkg/llm/anthropic.go`
- Create: `pkg/llm/anthropic_test.go`
- Modify: `pkg/llm/factory.go`
- Modify: `pkg/llm/factory_test.go`

- [ ] **Step 1：冻结依赖基线。** 记录 `go.mod/go.sum` 哈希、`go list -m all`、当前 `go 1.24.1` 和固定工具链。
- [ ] **Step 2：核验 v2.3.0 API 再写 RED。** 只依据官方 v2.3.0 源码确认 `WithoutEnvironmentDefaults`、`WithAPIKey`、`WithBaseURL`、`WithMaxRetries(0)`、`WithHTTPClient`、Messages/Models、图片/文档块、流事件和 `anthropic.Error` 类型，不凭旧版本记忆编码。
- [ ] **Step 3：写 RED 请求测试。** `/v1/messages`、system 顶层 blocks、user/assistant 顺序、图片 MIME+裸 Base64、PDF document、多文本块汇总。
- [ ] **Step 4：写 RED 终态测试。** 正常 stop、`max_tokens`、context window、refusal、tool/pause/unknown、空正文；流式 TextDelta/ThinkingDelta/ErrorResponse/message_stop/异常 EOF。
- [ ] **Step 5：写 RED 环境与安全测试。** 设置伪 `ANTHROPIC_*` 环境变量后仍只使用草稿显式值；302 不跟随；typed error 的 URL/JSON/Key 不泄漏；请求次数为 1。
- [ ] **Step 6：精确 pin 依赖并实现。** 唯一新增直接依赖为 `github.com/anthropics/anthropic-sdk-go/v2 v2.3.0`；为满足 SDK 将主模块 `go` directive 从 `1.24.1` 调整到 `1.25.0`，不新增 `toolchain` directive。
- [ ] **Step 7：实现原生 adapter。** `WithoutEnvironmentDefaults` 后显式设置 Key/BaseURL/HTTP client/零重试；业务 `max_tokens` 使用命名常量 8192，probe 32；不启用 tools、thinking、beta、fallback、prompt caching 或服务端会话。
- [ ] **Step 8：接入工厂并转 GREEN。** `errors.As(*anthropic.Error)` 只读取 `StatusCode` 和 `Type()`；不调用 `Error()`、`RawJSON()` 或 dump。
- [ ] **Step 9：依赖差异门禁。** `openai-go`、Wails 和其他既有直接依赖版本不变；所有新增 indirect 均可追溯到 SDK 模块图；无法解释的升级立即停止。

**验收：** 原生 Claude 不经过 OpenAI shim；环境默认值不污染用户草稿；异常终止不冒充成功。

## Task 7：模型列表和显式文字/图片 probe

**Files：**
- Create: `pkg/llm/probe.go`
- Create: `pkg/llm/probe_test.go`
- Modify: `pkg/llm/service.go`
- Modify: `pkg/llm/service_test.go`
- Modify: `app/llm.go`
- Create/Modify: `app/llm_test.go`

- [ ] **Step 1：写 RED 列表测试。** 三协议均通过 `Provider.GetModels`；结果去重并保序；列表失败返回结构化错误但不修改保存配置、active provider 或手工模型；删除对 `provider.client` 的访问。
- [ ] **Step 2：写 RED 文字 probe。** 随机 token 精确匹配；空/错误返回安全分类；本地校验失败 `requestSent=false`；进入 SDK 调用后为 true；每次只有一请求。
- [ ] **Step 3：写 RED 图片 probe。** 标准库生成随机两位七段数字 PNG；prompt 不含答案；通过 adapter 流式多模态路径；匹配、错值、不确定、网络/API 错误和异常终态分别断言。
- [ ] **Step 4：实现 Prober 和共享编排。** probe 只携带测试自身消息，不带用户 Prompt/历史/截图；文字和图片分别一个入口。
- [ ] **Step 5：实现完整草稿 Wails API。** DTO mapper 只复制白名单字段；15 秒超时沿用；草稿 client 不触发订阅、不替换 Solver provider。
- [ ] **Step 6：保留旧 API。** 旧方法固定 Chat 并委托新实现；原签名与 Wails 方法名不变。
- [ ] **Step 7：聚焦测试转 GREEN。** 断言测试服务器请求计数、路径、headers、body 和敏感字段隔离。

**验收：** 模型列表失败不阻止手填模型；文字通过不宣称图片通过；图片网络失败不冒充“不支持图片”。

## Task 8：所有生产业务入口统一走工厂

**Files：**
- Modify: `pkg/llm/service.go`
- Modify: `pkg/resume/service.go`
- Modify: `app/questionbank.go`
- Modify: `pkg/questionbank/organize.go`
- Modify: `pkg/questionbank/organize_failure.go`
- Modify: relevant tests in `pkg/solution`, `pkg/resume`, `pkg/questionbank`, `app`

- [ ] **Step 1：写 RED 工厂贯通测试。** 覆盖 Service 初始化/旧 API、Resume PDF、题库首次整理、只重试失败、自动整理导出和旧兼容入口；三个协议选择正确且不依赖 model/URL。
- [ ] **Step 2：写 RED 业务不变式。** Solver 多图顺序、KeepContext、取消；Resume file/document；题库一条 Record 一次调用、成功项不重试、RawOnly 零 provider 调用。
- [ ] **Step 3：写 RED 失败时序测试。** 工厂/配置错误发生在题库封段和 busy/cancel 变更前；失败不自动换协议、不创建第二请求。
- [ ] **Step 4：替换全部生产 `NewOpenAIAdapter`。** 业务层只依赖 `Provider`；仓库搜索确认生产代码无绕过工厂构造。
- [ ] **Step 5：对接 Change 024。** typed provider 错误继续映射到既有 `OrganizeFailure` 字段和已有安全 code；只根据安全类型调整 `Retryable`，不保存 SDK 原文、不改 Session/Record schema。
- [ ] **Step 6：运行聚焦测试转 GREEN。** 同时验证 Change 024 全失败/部分失败/只重试失败/RawOnly 回归。

**验收：** 主截图流式解题、简历、题库和设置诊断使用同一协议事实源；真实生成零 fallback、零双发。

## Task 9：设置页协议、手工模型与实测状态

**Files：**
- Modify: `frontend/src/stores/settings.js`
- Modify: `frontend/src/stores/ui.js`
- Modify: `frontend/src/components/ProviderSelect.vue`
- Modify: `frontend/src/components/ModelSelect.vue`
- Modify: `frontend/src/components/SettingsModal.vue`
- Modify: `frontend/src/services/api.js`
- Modify: `frontend/src/utils/modelCapabilities.js`
- Modify: `frontend/settings_regression_test.mjs`
- Create: `frontend/model_connectivity_regression_test.mjs`

- [ ] **Step 1：写 RED 配置生命周期测试。** 缺字段显示 Chat；`settings/tempSettings/applyConfig/buildConfigToSave` 携带协议；取消/重开正确；probe 状态不进保存 payload；既有 typed 隐藏字段继续保留。
- [ ] **Step 2：写 RED 零隐式调用测试。** 打开、加载、保存、切品牌、切协议均不自动刷新模型或 probe。
- [ ] **Step 3：写 RED 品牌/协议分离测试。** 品牌预设只在用户点击时给出推荐 BaseURL+协议；同 origin 允许保留 Key，切换到不同 origin 清空草稿 Key、model、列表和 probe 状态，避免把旧凭据带到新主机；用户之后可单独改协议。
- [ ] **Step 4：写 RED 手工模型测试。** `ModelSelect` 改为可输入 combobox；列表只是建议；刷新失败或列表不含手填值时不得替换；保存和 probe 允许手填值。
- [ ] **Step 5：写 RED 竞态测试。** 每次打开/关闭递增 session revision；四项连接字段变化递增 connection revision；文字/图片各自 request version；A→B→A 的旧 A、关闭后和重开后的迟到结果均无效。
- [ ] **Step 6：写 RED 文案测试。** 有 Key 只显示“已配置 / 未验证”；文字和图片各自显示未测试/通过/失败/不确定；固定费用提示；静态图标和 context length 明示“模型目录参考，非当前通道实测”。
- [ ] **Step 7：实现 UI 和 store。** ProviderSelect 显示实际协议标签，不再写死 `OPENAI-COMPATIBLE`；Anthropic 官方默认根地址使用 SDK 要求的 API root；自定义地址字段说明填写 API root 而非完整 endpoint。
- [ ] **Step 8：接入结构化 API。** 安全 message 原样展示；不把异常对象 `console.error`；改变连接身份立即清空旧结果。
- [ ] **Step 9：Node 回归转 GREEN。** 保持现有设置、题库和 Mobile 回归通过。

**验收：** 用户可明确选择三协议和手填模型；静态资料、文字实测、图片实测不再混淆；无自动费用调用。

## Task 10：定向清理敏感日志并生成固定版本 bindings

**Files：**
- Modify: `pkg/solution/solution.go`
- Audit/Modify only relevant logs in `pkg/llm`, `pkg/resume`, `pkg/questionbank`, `app`
- Regenerate: `frontend/wailsjs/go/app/App.js`
- Regenerate: `frontend/wailsjs/go/app/App.d.ts`
- Regenerate: `frontend/wailsjs/go/models.ts`

- [ ] **Step 1：写 RED 日志捕获测试。** system prompt、答案、伪 Key、URL、Base64、SDK body sentinel 不得出现；允许协议、stage、code、长度和安全计数。
- [ ] **Step 2：删除完整 Prompt/答案和 `%v` 原始 SDK error 日志。** 只做本次调用链的定向修复，不扩大为全仓 logger 重构。
- [ ] **Step 3：使用固定 Wails v2.11.0 生成 bindings。** 审计新 DTO/方法齐全、旧方法未丢失、runtime 无无关漂移。
- [ ] **Step 4：运行 Node 回归和 Vite build。** 生成文件不得包含真实密钥或本地私人配置。

**验收：** 日志和 bindings 安全、可解释；Wails runtime 版本不漂移。

## Task 11：全量验证、独立验证程序和治理收尾

- [ ] **Step 1：运行聚焦测试。**

```text
GOTOOLCHAIN=go1.25.13 go test ./pkg/llm/... ./pkg/config ./pkg/resume ./pkg/questionbank ./pkg/solution ./app -count=1
node frontend/settings_regression_test.mjs
node frontend/model_connectivity_regression_test.mjs
```

- [ ] **Step 2：运行全量自动验证。**

```text
GOTOOLCHAIN=go1.25.13 go test ./... -count=1
GOTOOLCHAIN=go1.25.13 go vet ./...
GOTOOLCHAIN=go1.25.13 go build ./...
node frontend/clipboard_regression_test.mjs
node frontend/history_round_selection_regression_test.mjs
node frontend/mobile_store_regression_test.mjs
node frontend/numbered_copy_regression_test.mjs
node frontend/questionbank_continuation_regression_test.mjs
node frontend/questionbank_history_regression_test.mjs
node frontend/round_screenshots_regression_test.mjs
node frontend/settings_regression_test.mjs
node frontend/model_connectivity_regression_test.mjs
npm --prefix frontend run build
git diff --check
```

- [ ] **Step 3：按环境条件运行 race。** 若仍出现 Windows `0xc0000139`，记录 `BLOCKED`，不得写 PASS 或归因于 Change 025。
- [ ] **Step 4：复核依赖图。** 直接依赖只新增 Anthropic v2.3.0；`go` directive 仅改到 1.25.0；无 `toolchain`；OpenAI/Wails/前端依赖不变；逐项解释 indirect 变化。
- [ ] **Step 5：固定版本 Wails 集成构建。**

```text
GOTOOLCHAIN=go1.25.13 go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -o Q-Solver-change025-multi-protocol-validation.exe
```

不使用清理 build 目录的参数；构建前后比对全部既有 EXE 哈希，新程序不得覆盖正式或 Change 024 程序。

- [ ] **Step 6：完成数据与隐私复核。** 测试只命中 loopback fake endpoint；无真实配置读取、无真实 provider 调用、无用户文件变化。
- [ ] **Step 7：回写治理。** `03`～`06`、审计和 INDEX 如实记录 PASS/FAIL/BLOCKED/NEEDS_USER/未执行；未经授权保持未提交、未推送、未发布。

---

## 依赖审计门槛

只有同时满足以下条件才能继续交付：

1. 唯一新增直接模块是 `github.com/anthropics/anthropic-sdk-go/v2 v2.3.0`。
2. 主模块 `go` directive 只从 `1.24.1` 调整到 `1.25.0`；固定执行工具链仍为 1.25.13。
3. 不新增 `toolchain` directive。
4. `github.com/openai/openai-go` 保持 v1.12.0，Wails 保持 v2.11.0。
5. 其他既有直接依赖和所有 npm 依赖版本不变。
6. `go.sum` 和 indirect 变化全部能由 Anthropic SDK v2.3.0 模块图解释；不能以“tidy 自动生成”为理由接受未知漂移。
7. SDK 选项已由 v2.3.0 官方源码核实；编译通过不足以替代环境默认、重试、重定向和错误泄漏测试。
8. 依赖门禁失败时停止，不通过放宽版本、删除测试或升级无关包绕过。

## 人工验收矩阵（均为 NEEDS_USER）

### Windows WebView2

1. 品牌和协议是独立控件，协议有 Chat / Responses / Anthropic Messages 三项。
2. OpenRouter/自定义地址不会自动切协议；Claude 模型名也不会触发原生协议。
3. 切换到不同 provider origin 会清空旧草稿凭据，未保存配置不受影响。
4. 模型可手填；列表失败、为空或不含该模型时仍能保存和测试。
5. 有 Key 只显示“已配置 / 未验证”。
6. 文字与图片实测状态独立，每个按钮明确提示一次请求和可能费用。
7. 打开、保存、切品牌/协议不会自动调用模型。
8. 快速 A→B→A、关闭和重开不会出现迟到结果覆盖。
9. 图片/context 图标明确为目录参考，不是通道实测。

### 用户自愿的真实 provider

每次仅对当前显式协议测试：

1. OpenAI Chat：文字、图片、单图、多图顺序、流式终态。
2. OpenAI Responses：文字、图片、单图、多图顺序、流式终态。
3. Anthropic Messages：文字、图片、单图、多图顺序、流式终态。
4. 用户实际自定义网关：明确选其支持协议，确认应用不 fallback。
5. 错 Key、无权限、404、429、5xx、超时和断网显示对应安全类别。
6. 404 只提示端点/协议/模型待检查；断网不显示成不支持图片。
7. 模型列表失败后手填模型仍可工作。
8. Resume PDF 和题库整理/只重试失败项按所选协议运行。

真实验收只允许用户自行配置并点击；自动化不得读取真实配置或代为消费额度。

## 回滚

1. 只按 Change 025 白名单定向回滚；禁止 destructive Git 命令。
2. 先移除前端新 UI/DTO，恢复旧调用并用 Wails v2.11.0 重生 bindings。
3. 再恢复 Service/Resume/Question Bank wiring；不删除任何题库 Session、Record、Questions、截图、导出或 Change 024 失败记录。
4. 移除 Responses/Anthropic adapter 与工厂分支；保留可单独验证的 Chat 安全修复只能另行记录，不能形成未审计混合状态。
5. 移除所有 Anthropic imports 后，才移除 v2.3.0 requirement，并把 `go` directive 恢复到实施前值；运行 tidy、全量 test/vet/build 并比较依赖图。
6. `apiProtocol` 为可选字段，旧二进制会忽略；若用户已经保存原生协议，回退旧程序前必须人工改成旧程序可用的 OpenAI-compatible 配置，不自动改写私人配置。
7. 保留 Change 025 文档与审计并标记“已回滚”；不删除验证程序或用户数据。
8. 未获授权不提交回滚。

## 明确排除范围

- 不确认当前 Claude/GPT/Codex 失败的真实 provider 根因。
- 不做自动协议检测、失败后换协议、换 endpoint、换模型、双发或多模型投票。
- 不启用 OpenAI Assistants/Realtime/Batch、Responses 服务端会话或 background。
- 不启用 Anthropic tools、MCP、thinking、fallback、prompt caching、beta 或服务端会话。
- 不实现业务级 provider 自动重试；SDK retry 明确关闭。
- 不按静态模型目录宣称当前 endpoint 支持图片/PDF。
- 不调用真实 API、真实截图、简历或题库进行自动验收。
- 不引入 Vitest/Jest 或其他前端测试框架。
- 不全面重构设置页、LLM Message schema、题库 schema、KeepContext 或日志框架。
- 不修改 Change 024 状态机和导出语义，只消费其既有安全失败结构。
- 不批量迁移或重写旧配置。
- 不覆盖正式 EXE、旧验证程序或发布版本。
- 不进行未授权 Git/远程操作。

## Sources

- [Anthropic Go SDK v2.3.0 releases](https://github.com/anthropics/anthropic-sdk-go/releases/tag/v2.3.0)
- [Anthropic Go SDK repository](https://github.com/anthropics/anthropic-sdk-go/tree/v2.3.0)
- [Anthropic Go SDK request options](https://github.com/anthropics/anthropic-sdk-go/blob/v2.3.0/option/requestoption.go)
- [Anthropic Go SDK client defaults](https://github.com/anthropics/anthropic-sdk-go/blob/v2.3.0/client.go)
- [Anthropic Go SDK typed API error](https://github.com/anthropics/anthropic-sdk-go/blob/v2.3.0/internal/apierror/apierror.go)
