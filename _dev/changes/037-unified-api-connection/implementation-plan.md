# Change 037 Revision B 实施计划

> **执行要求：** 按任务顺序内联执行 TDD；每个任务先写失败测试，再实现最小代码，再运行聚焦测试。项目未提供 `executing-plans` skill；本会话按同等检查点执行。步骤使用 checkbox 跟踪。不得自动 commit/push。

**目标：** 把设置页收敛为 OpenAI 兼容与受控自定义 JSON 模板两种模式，支持安全图片请求、非流式 JSON 提取、旧协议草稿迁移、Keyless 实际解题，并将模型与场景分页且消除按钮/内容裁切。

**架构：** `openai_chat` 保持强类型流式 adapter；新增 `custom_json` adapter，模板先解析为 JSON 值树，再按白名单替换完整值占位符。旧 Responses/Anthropic adapter 继续运行未保存旧配置，设置页只生成模板草稿，保存后才迁移。自定义 adapter 通过单次 `ChunkContent` 兼容现有 Solver 流事件。

**技术栈：** Go、Wails v2.11.0、Vue 3、Pinia、Vite；不新增依赖。

**设计规范：** `_dev/changes/037-unified-api-connection/design.md` Revision B。

## 全局约束

- 不新增依赖，不升级 Go、Wails、Node、OpenAI SDK 或 Anthropic SDK。
- 自动测试只使用 `httptest.Server`，不调用真实 API。
- 不读取、打印或提交 API Key、Token、Header 实值；状态仅写 `SET` / `NOT SET`。
- 模板不得执行脚本、表达式、循环、条件、文件、环境变量或额外网络请求。
- API Key 不得作为 URL 或 request body 占位符。
- 自定义模式首版只支持 POST JSON 与非流式 JSON 响应；不实现通用 SSE/WebSocket。
- 不修改题库、Mobile、快捷键、截图采集、Prompt 或历史上下文语义。
- bindings 和候选必须使用 Wails v2.11.0 与项目保护参数。
- 每次 Git 操作前执行 `git status`、`git branch`、`git remote -v`、`git diff`；不得 `git add .` / `git add -A`。
- 当前未授权 commit、push、稳定版晋升、Tag、Release 或真实收费 API 调用。

---

## 文件结构锁定

### 新建

- `pkg/llm/template.go`：模板语法校验、消息投影、类型化占位符替换、JSON Pointer 提取。
- `pkg/llm/template_test.go`：模板与消息投影安全矩阵。
- `pkg/llm/custom_json.go`：自定义 POST JSON Provider、鉴权、响应上限与错误分类。
- `pkg/llm/custom_json_test.go`：HTTP、Keyless、响应、取消、零目录请求与 stream 兼容测试。

### 修改

- `pkg/llm/protocol/protocol.go`、`protocol_test.go`：新增 `custom_json`。
- `pkg/config/config.go`：新增模板、Pointer、鉴权 Header/前缀字段和默认值。
- `pkg/config/connection.go`：按协议规范化地址和模板字段。
- `pkg/config/connection_test.go`、`manager_test.go`：新 schema、旧配置和原子往返。
- `pkg/llm/factory.go`、`factory_test.go`：构造 `CustomJSONAdapter`。
- `pkg/llm/errors.go`、`errors_test.go`：增加模型目录不支持的安全错误。
- `pkg/llm/service.go`、`service_test.go`：模板模式目录零请求、probe 和统一校验。
- `pkg/solution/solution.go`、`solution_test.go`：移除空 Key 前置拒绝，验证 Keyless 实际解题。
- `app/llm.go`、`app/llm_test.go`、`app/settings_test.go`：DTO、局部保存和安全返回。
- `frontend/src/stores/settings.js`：两模式、legacy 草稿迁移、模板字段、目录禁用和竞态。
- `frontend/src/components/ProviderSelect.vue`：紧凑两模式表单与折叠高级模板。
- `frontend/src/components/SettingsModal.vue`：独立 scene pane、按钮布局、滚动/标签可见性。
- `frontend/src/components/ModelSelect.vue`：仅在页面 active 时保持 dropdown；不新增协议推断。
- `frontend/*_test.mjs`：更新 UI、Store、迁移、布局回归。
- `frontend/wailsjs/go/app/App.js`、`App.d.ts`、`frontend/wailsjs/go/models.ts`：固定版本生成。
- `README.md`、`README_EN.md`：两模式、模板限制、迁移和分页说明。
- Change 037、审计和 INDEX：同步 Revision B 事实。

---

### Task 1：扩展协议和 Config 契约

**文件：**
- 修改：`pkg/llm/protocol/protocol.go`
- 修改：`pkg/llm/protocol/protocol_test.go`
- 修改：`pkg/config/config.go`
- 修改：`pkg/config/connection.go`
- 修改：`pkg/config/connection_test.go`
- 修改：`pkg/config/manager_test.go`

**产出接口：**

```go
const protocol.CustomJSON APIProtocol = "custom_json"

const (
    DefaultCustomAuthHeader = "Authorization"
    DefaultCustomAuthPrefix = "Bearer "
    MaxRequestTemplateBytes = 64 * 1024
)

func NormalizeCustomRequestURL(value string) (string, error)
func NormalizeOpenAIChatAddress(value string) (baseURL string, endpointPath string, err error)
func NormalizeAuthHeader(value string) (string, error)
func NormalizeAuthPrefix(value string) (string, error)
func ValidateRequestTemplateConfig(template, pointer string) error
```

`Config` 新增：

```go
RequestTemplate     string `json:"requestTemplate,omitempty"`
ResponseJSONPointer string `json:"responseJSONPointer,omitempty"`
AuthHeader          string `json:"authHeader,omitempty"`
AuthPrefix          string `json:"authPrefix,omitempty"`
```

- [x] **Step 1：写 protocol 和 Config 红灯测试**

在 `protocol_test.go` 增加：

```go
func TestNormalizeAcceptsCustomJSON(t *testing.T) {
    got, err := Normalize("custom_json")
    if err != nil || got != CustomJSON { t.Fatalf("got %q, %v", got, err) }
}
```

在 `connection_test.go` 增加：

```go
func TestCustomJSONDefaultsAuthAndRequiresTemplate(t *testing.T)
func TestCustomJSONAcceptsKeylessHTTPSAndHTTP(t *testing.T)
func TestCustomJSONRejectsQueryFragmentUserinfoAndCRLF(t *testing.T)
func TestCustomJSONRejectsInvalidAuthHeaderPrefixAndConflict(t *testing.T)
func TestOpenAIChatAddressAcceptsRootOrFullChatEndpoint(t *testing.T)
func TestLegacyProtocolsRemainValidBeforeMigration(t *testing.T)
```

关键表驱动断言：

```go
cfg := Config{
    ConnectionMode: "direct",
    APIProtocol: "custom_json",
    BaseURL: serverURL + "/generate",
    RequestTemplate: `{"model":"{{model}}","messages":"{{openai_messages}}"}`,
    ResponseJSONPointer: "/answer/text",
    Model: "test-model",
}
normalized, err := cfg.NormalizedConnectionSettings()
// normalized.BaseURL == serverURL + "/generate"
// normalized.EndpointPath == ""
// normalized.ResponseFormat == "text"
// normalized.AuthHeader == "Authorization"
// normalized.AuthPrefix == "Bearer "
```

- [x] **Step 2：运行红灯**

```bash
go test ./pkg/llm/protocol ./pkg/config -run 'Test.*(CustomJSON|OpenAIChatAddress|LegacyProtocols)' -count=1 -v
```

预期：`CustomJSON`、Config 字段和规范化函数不存在而失败。

- [x] **Step 3：实现协议和字段默认值**

`protocol.Normalize` 接受 `CustomJSON`。`NewDefaultConfig` 的默认连接仍为 `openai_chat`，新增字段默认：

```go
RequestTemplate: "",
ResponseJSONPointer: "",
AuthHeader: "",
AuthPrefix: "",
```

只在 `custom_json` 规范化时填默认鉴权值；避免普通 Chat 配置产生无关持久字段。

- [x] **Step 4：实现地址和鉴权校验**

`NormalizeCustomRequestURL` 使用 `url.Parse`，要求 HTTP/HTTPS、host 非空、userinfo/query/fragment 为空；保留完整 path，删除仅末尾 `/` 以外的语义改写。

`NormalizeOpenAIChatAddress`：

```go
clean, err := NormalizeBaseURL(value)
if strings.HasSuffix(strings.TrimRight(clean, "/"), "/chat/completions") {
    return strings.TrimSuffix(strings.TrimRight(clean, "/"), "/chat/completions"), "/chat/completions", nil
}
return clean, "/chat/completions", nil
```

只有标准 `/chat/completions` 被拆分；其他自定义完整 endpoint 要求改用模板模式。

鉴权 Header 复用合法 token 和受禁 Header 校验。Prefix 禁止 CR/LF、最大 128 字节。`extraHeaders` 按 `strings.EqualFold` 拒绝与 `authHeader` 冲突。

- [x] **Step 5：接入 `NormalizedConnectionSettings`**

`custom_json` 分支：

```go
c.BaseURL, err = NormalizeCustomRequestURL(c.BaseURL)
c.EndpointPath = ""
c.ResponseFormat = ResponseFormatText
c.AuthHeader, err = NormalizeAuthHeader(c.AuthHeader)
c.AuthPrefix, err = NormalizeAuthPrefix(c.AuthPrefix)
err = ValidateRequestTemplateConfig(c.RequestTemplate, c.ResponseJSONPointer)
```

`openai_chat` 分支通过 `NormalizeOpenAIChatAddress` 接受 root 或完整 Chat URL。legacy Responses/Anthropic 保持 Revision A 行为，不在 Config Load 时迁移。

- [x] **Step 6：补 Manager 往返测试**

新增：

```go
func TestManagerRoundTripsCustomJSONFields(t *testing.T)
func TestLoadLegacyResponsesDoesNotWriteOrMigrateBeforeSave(t *testing.T)
func TestLoadLegacyAnthropicDoesNotWriteOrMigrateBeforeSave(t *testing.T)
```

比较 Load 前后配置文件 SHA-256，证明读取不写盘；自定义字段通过加密配置往返且 Header map 深拷贝。

- [x] **Step 7：运行聚焦测试并记录检查点**

```bash
go test ./pkg/llm/protocol ./pkg/config -count=1
```

预期：PASS。更新 `03-implementation.md`，不暂存、不提交。

---

### Task 2：实现安全模板和消息投影

**文件：**
- 新建：`pkg/llm/template.go`
- 新建：`pkg/llm/template_test.go`
- 修改：`pkg/config/connection.go`（只调用不依赖 llm 的基础语法校验时不得形成 import cycle）

**依赖约束：** `pkg/config` 不能导入 `pkg/llm`。Config 层只校验 JSON、大小、Pointer 语法和占位符字符串白名单；消息投影与渲染位于 `pkg/llm`。

**产出接口：**

```go
type templatePayloads struct {
    Model             string
    SystemPrompt      string
    UserText          string
    OpenAIMessages    []any
    ResponsesInput    []any
    AnthropicMessages []any
    AnthropicSystem   []any
    ImagesDataURLs    []string
    ImagesBase64      []string
}

func buildTemplatePayloads(model string, messages []Message) (templatePayloads, error)
func renderRequestTemplate(value string, payloads templatePayloads) ([]byte, error)
func extractJSONPointer(body []byte, pointer string) (string, error)
```

Config 层新增无 llm 依赖的 helper：

```go
func ValidateRequestTemplateConfig(value, pointer string) error
func ValidateJSONPointer(pointer string) error
```

- [x] **Step 1：写模板语法红灯测试**

在 `connection_test.go` 增加：

```go
func TestRequestTemplateRejectsOversizeNonObjectAndInvalidJSON(t *testing.T)
func TestRequestTemplateRejectsUnknownEmbeddedAndKeyPlaceholders(t *testing.T)
func TestRequestTemplateRequiresContentPlaceholder(t *testing.T)
func TestJSONPointerValidation(t *testing.T)
```

必须拒绝：

```text
[]
{"model":"prefix-{{model}}"}
{"{{model}}":"x"}
{"body":"{{unknown}}"}
{"model":"{{model}}"}
pointer: choices/0/text
pointer: /bad~2escape
```

必须接受 RFC 6901 的 `~0`、`~1` 和数组索引。

- [x] **Step 2：运行 Config 红灯**

```bash
go test ./pkg/config -run 'Test(RequestTemplate|JSONPointer)' -count=1 -v
```

预期：校验函数缺失或非法模板被接受。

- [x] **Step 3：实现 Config 级模板遍历校验**

用 `json.Decoder` 解码为 `map[string]any`，并确认 decoder 到 EOF，拒绝 trailing JSON。递归遍历 map value/array item：

- 完整字符串等于已知占位符时记录。
- 含 `{{` 或 `}}` 但不完全匹配白名单时拒绝。
- map key 含占位符标记时拒绝。
- 至少出现 `system_prompt`、`user_text`、三种 messages/input 或两种 images 之一。

不得使用正则替换 JSON 文本。

- [x] **Step 4：写消息投影和渲染红灯测试**

在 `template_test.go` 增加：

```go
func TestBuildTemplatePayloadsProjectsFullConversation(t *testing.T)
func TestBuildTemplatePayloadsProjectsImagesAsDataURLAndBase64(t *testing.T)
func TestBuildTemplatePayloadsRejectsInvalidImageAndPDF(t *testing.T)
func TestRenderRequestTemplateInjectsTypedValues(t *testing.T)
func TestRenderRequestTemplateDoesNotMutatePayloads(t *testing.T)
func TestExtractJSONPointerSupportsObjectsArraysAndEscapes(t *testing.T)
func TestExtractJSONPointerRejectsMissingNonStringEmptyAndTrailingJSON(t *testing.T)
```

构造 system、历史 user/assistant、当前 user text + PNG data URL，断言：

- `openai_messages` 保留 role、文本和 `image_url.url`。
- `responses_input` 使用 `input_text` / `input_image`。
- `anthropic_system` 只含 system blocks。
- `anthropic_messages` 不含 system，图片 source 为 base64 + media_type。
- `images_data_urls` 与 `images_base64` 顺序一致。

- [x] **Step 5：运行 llm 模板红灯**

```bash
go test ./pkg/llm -run 'Test(BuildTemplate|RenderRequest|ExtractJSON)' -count=1 -v
```

预期：新文件和函数不存在。

- [x] **Step 6：实现类型化消息投影**

手工构造普通 `map[string]any` / `[]any`，不依赖 SDK 私有 JSON 字段。所有图片/PDF 先调用 `ParseBase64DataURLStrict`；图片 MIME 继续限定 PNG/JPEG/GIF/WebP，PDF 仅 `application/pdf`。

`UserText` 取最后一条 user message中的所有 text part 与 `Content`，以换行连接；`SystemPrompt` 合并全部 system 文本。完整历史只存在 messages/input 投影中。

- [x] **Step 7：实现安全替换和 Pointer 提取**

模板每次重新 decode，递归创建新 map/slice；完整占位符返回 payload 对应类型。结果用 `json.Marshal`。

Pointer：先 `json.Unmarshal` 到 `any`，按 `/` 分段，执行 `~1`→`/`、`~0`→`~`，array segment 必须是规范十进制索引；最终值必须是 trim 后非空 string。

- [x] **Step 8：运行 Task 2 全部测试**

```bash
go test ./pkg/config ./pkg/llm -run 'Test(RequestTemplate|JSONPointer|BuildTemplate|RenderRequest|ExtractJSON)' -count=1
```

预期：PASS。更新实现记录，不提交。

---

### Task 3：实现 CustomJSONAdapter

**文件：**
- 新建：`pkg/llm/custom_json.go`
- 新建：`pkg/llm/custom_json_test.go`
- 修改：`pkg/llm/factory.go`
- 修改：`pkg/llm/factory_test.go`
- 修改：`pkg/llm/errors.go`
- 修改：`pkg/llm/errors_test.go`
- 修改：`pkg/llm/service.go`
- 修改：`pkg/llm/service_test.go`

**产出接口：**

```go
const maxCustomJSONResponseBytes int64 = 4 * 1024 * 1024

func NewCustomJSONAdapter(cfg config.Config) *CustomJSONAdapter

type CustomJSONAdapter struct {
    client *http.Client
    config config.Config
}
```

`CustomJSONAdapter` 实现 `Provider` 和现有 `textProbeGenerator`。

- [x] **Step 1：写 HTTP 红灯测试**

在 `custom_json_test.go` 增加：

```go
func TestCustomJSONGenerateContentPostsRenderedJSONAndExtractsPointer(t *testing.T)
func TestCustomJSONGenerateContentStreamEmitsOneContentChunk(t *testing.T)
func TestCustomJSONUsesConfigurableAuthHeaderAndPrefix(t *testing.T)
func TestCustomJSONKeylessOmitsAuthHeader(t *testing.T)
func TestCustomJSONInjectsExtraHeaders(t *testing.T)
func TestCustomJSONRejectsRedirectWithoutForwardingCredentials(t *testing.T)
func TestCustomJSONClassifiesHTTPFailuresWithoutLeakingBody(t *testing.T)
func TestCustomJSONRejectsOversizeInvalidMissingAndEmptyResponses(t *testing.T)
func TestCustomJSONHonorsContextCancellation(t *testing.T)
func TestCustomJSONGetModelsMakesNoRequest(t *testing.T)
```

`httptest.Server` 记录请求次数、method、path、Headers、body。所有测试值使用 `test-only-*`，错误字符串不得包含 sentinel secret 或响应 body。

- [x] **Step 2：运行 adapter 红灯**

```bash
go test ./pkg/llm -run 'TestCustomJSON' -count=1 -v
```

预期：构造函数/类型不存在。

- [x] **Step 3：新增目录不支持错误**

新增：

```go
ErrorCatalogUnsupported ErrorCode = "catalog_unsupported"
```

安全文案：`"自定义请求模板不提供模型目录，请手动填写模型"`。`GetModels` 直接返回该错误，`RequestSent=false` 的语义由 Service/前端测试证明。

- [x] **Step 4：实现 HTTP 执行器**

`execute(ctx, model, messages)`：

1. `buildTemplatePayloads`。
2. `renderRequestTemplate`。
3. `http.NewRequestWithContext(POST, cfg.BaseURL, bytes.NewReader(body))`。
4. 设置 `Content-Type: application/json`、已验证 Extra Headers。
5. Key 非空时设置 `cfg.AuthHeader` 为 `cfg.AuthPrefix + cfg.APIKey`。
6. `http.Client.CheckRedirect` 返回 `http.ErrUseLastResponse`，`Transport=http.DefaultTransport`。
7. 非 2xx 只按 status 调用 `errorCodeForHTTP`，不保留 body 到错误。
8. 用 `io.LimitReader(resp.Body, max+1)`；超限返回 `ErrorRequestTooLarge`。
9. `extractJSONPointer` 得到 content。

- [x] **Step 5：实现 Provider 方法**

```go
func (a *CustomJSONAdapter) GenerateContent(ctx context.Context, model string, messages []Message) (Message, error)
func (a *CustomJSONAdapter) GenerateContentStream(ctx context.Context, messages []Message, onChunk StreamCallback) (Message, error)
func (a *CustomJSONAdapter) TestChat(ctx context.Context) error
func (a *CustomJSONAdapter) GetModels(ctx context.Context) ([]string, error)
func (a *CustomJSONAdapter) generateTextProbe(ctx context.Context, model string, messages []Message) (Message, error)
```

Stream 方法只在完整请求成功后调用一次：

```go
if onChunk != nil {
    onChunk(StreamChunk{Type: ChunkContent, Content: result.Content})
}
```

- [x] **Step 6：接入 Factory 和 Service**

`NewProvider` 对 `protocol.CustomJSON` 返回 `NewCustomJSONAdapter(normalized)`。

`ListModelsForConnection` 对 `custom_json` 必须在 HTTP 前返回 `catalog_unsupported`。新增 Service 测试断言 server request count 为 0。

`ProbeConnection` 继续走 Provider；文字和图片 probe 可真实请求。若模板没有图片相关的 messages/input/images 占位符，图片请求在 build 阶段返回 `invalid_config`，不发网络。

- [x] **Step 7：运行 Task 3 聚焦测试**

```bash
go test ./pkg/llm -run 'Test(CustomJSON|NewProvider.*Custom|Service.*Custom|CatalogUnsupported)' -count=1
```

预期：PASS。

- [x] **Step 8：运行 llm 全包测试**

```bash
go test ./pkg/llm -count=1
```

预期：Revision A 三 adapter 回归仍 PASS。记录检查点，不提交。

---

### Task 4：修复 Keyless Solver 实际解题

**文件：**
- 修改：`pkg/solution/solution.go`
- 修改：`pkg/solution/solution_test.go`
- 可修改：`pkg/solution/error_diagnostics_test.go`（仅旧 require-key 断言需要更新）

**产出语义：** Solver 不根据 Key 是否为空决定连接可用性；Provider 构造和请求错误是权威判断。

- [x] **Step 1：写 Keyless Solver 红灯测试**

新增：

```go
func TestSolverAllowsKeylessProviderAndEmitsCompletedAnswer(t *testing.T)
func TestSolverReportsProviderErrorInsteadOfRequireAPIKeyForKeylessFailure(t *testing.T)
```

使用 fake Provider 或 `CustomJSONAdapter + httptest.Server`。成功测试使用一张最小合法 PNG data URL，断言：

- Provider 被调用一次。
- 事件包含 `solution-stream-start`、单个 `solution-stream-chunk` 和 `solution`。
- 不出现 `require-api-key`。

- [x] **Step 2：运行红灯**

```bash
go test ./pkg/solution -run 'TestSolver.*Keyless' -count=1 -v
```

预期：当前 `if req.Config.APIKey == ""` 提前返回导致失败。

- [x] **Step 3：删除 Key 专属前置拒绝**

删除 `Solve` 开头的 `require-api-key` 分支，不添加伪造 Key。保持 provider 错误通过 `solution-error` 安全诊断输出。

如 App 其他路径仍发 `require-api-key`，只在其确实代表“没有可构造 Provider”时保留，不以空 Key 为判断条件。

- [x] **Step 4：运行 solution 聚焦与全包测试**

```bash
go test ./pkg/solution -run 'TestSolver' -count=1
go test ./pkg/solution -count=1
```

预期：PASS，KeepContext、图片历史和错误诊断不回归。

---

### Task 5：扩展 Wails 草稿和原子局部保存

**文件：**
- 修改：`app/llm.go`
- 修改：`app/llm_test.go`
- 修改：`app/settings_test.go`
- 修改：`pkg/config/manager.go`（仅 clone 新字段为值类型时无需修改；不得重写事务）
- 修改：`pkg/config/manager_test.go`

**DTO 变更：**

```go
type ConnectionDraft struct {
    // Revision A 字段保留
    RequestTemplate     string `json:"requestTemplate"`
    ResponseJSONPointer string `json:"responseJSONPointer"`
    AuthHeader          string `json:"authHeader"`
    AuthPrefix          string `json:"authPrefix"`
}

type ConnectionSettingsView struct {
    // 非敏感字段保留
    TemplateSet bool `json:"templateSet"`
}
```

安全 View 不返回 RequestTemplate、ResponseJSONPointer、AuthHeader、AuthPrefix、Key 或 Header 值；前端成功后以当前已验证草稿作为本地基线。

- [x] **Step 1：写 DTO 和保存红灯测试**

新增/扩展：

```go
func TestConnectionDraftMapsCustomTemplateFields(t *testing.T)
func TestSaveConnectionSettingsPersistsCustomTemplateFieldsOnly(t *testing.T)
func TestSaveConnectionSettingsRejectsInvalidTemplateAtomically(t *testing.T)
func TestSaveConnectionResultDoesNotExposeTemplateOrAuthValues(t *testing.T)
```

失败原子性比较：配置文件 SHA-256、`ConfigManager.Get()`、subscriber 次数、活动 Provider 均不变。

返回 JSON 使用 sentinel：`test-only-template-marker`、`test-only-auth-prefix`，断言结果不包含。

- [x] **Step 2：运行红灯**

```bash
go test ./app ./pkg/config -run 'Test(ConnectionDraft|SaveConnection|PatchValidated).*Template' -count=1 -v
```

预期：字段缺失或未保存。

- [x] **Step 3：扩展映射和 patch 白名单**

`ConnectionDraft.config()` 复制四个新字段。`SaveConnectionSettings` patch 只增加：

```go
cfg.RequestTemplate = connection.RequestTemplate
cfg.ResponseJSONPointer = connection.ResponseJSONPointer
cfg.AuthHeader = connection.AuthHeader
cfg.AuthPrefix = connection.AuthPrefix
```

不接纳 transparency、domainId、outputMode、resume、shortcuts 等字段。validateFn 继续只构造 Provider，不发送请求。

- [x] **Step 4：更新安全 View**

`TemplateSet` 只依据 trim 后 RequestTemplate 是否非空。不得返回 Pointer/Auth 字段；错误仍使用安全 DTO。

- [x] **Step 5：运行 app/config 测试**

```bash
go test ./pkg/config ./app -count=1
```

预期：PASS。

---

### Task 6：前端 Store 两模式和旧配置草稿迁移

**文件：**
- 修改：`frontend/src/stores/settings.js`
- 修改：`frontend/unified_api_settings_regression_test.mjs`
- 修改：`frontend/ccr_settings_lifecycle_regression_test.mjs`
- 修改：`frontend/model_connectivity_regression_test.mjs`
- 修改：`frontend/settings_regression_test.mjs`

**新增 Store 字段：**

```js
requestTemplate: ''
responseJSONPointer: ''
authHeader: ''
authPrefix: ''
legacyMigrationNotice: ''
```

**新增 helper：**

```js
function defaultCustomTemplate()
function migrateLegacyConnectionDraft(config)
function isCustomTemplateMode()
function canRefreshModelCatalog()
```

- [x] **Step 1：写 Store 红灯测试**

覆盖：

```js
assert.equal(store.tempSettings.apiProtocol, 'custom_json')
assert.equal(store.canRefreshModelCatalog, false)
```

并断言：

- 新默认模式仍为 `openai_chat`。
- Responses load 生成 `{{responses_input}}` 模板、完整 `/responses` URL 和 Pointer。
- Anthropic load 生成 `{{anthropic_system}}` / `{{anthropic_messages}}`、`x-api-key`、空 prefix、`anthropic-version`。
- `settings.apiProtocol` 在未保存时仍保留原 legacy 值；只迁移 `tempSettings`。
- 打开/取消设置零 save bridge 调用。
- 模板模式 `refreshModels()` 零 list bridge 调用并设置明确状态。
- `connectionDraft()` 包含全部新字段，不含 domainId/scene。
- 模板/Pointer/Auth 变化清目录与 probe 并增加 revision。
- 迟到 save/probe 不能覆盖新模板草稿。
- API 专用保存成功只更新连接基线；全局保存才包含 domainId。

- [x] **Step 2：运行前端红灯**

```bash
node frontend/unified_api_settings_regression_test.mjs
node frontend/model_connectivity_regression_test.mjs
node frontend/settings_regression_test.mjs
```

预期：新字段/helper 和迁移行为缺失。

- [x] **Step 3：实现默认模板与 legacy 迁移**

默认模板文本固定为稳定两空格 JSON：

```js
JSON.stringify({ model: '{{model}}', messages: '{{openai_messages}}' }, null, 2)
```

`applyConfig` 保持 `settings` 为后端真实值，复制到 `tempSettings` 后再调用 `migrateLegacyConnectionDraft(config)` 覆盖草稿连接字段。迁移只发生在内存，不调用 API。

URL 拼接使用 URL 解析或安全的 slash join，不拼 query/token，不打印结果。

- [x] **Step 4：扩展连接 identity 和 payload**

`connectionDraft()` 固定返回 Revision A 八字段 + 四个新字段。`custom_json` 时：

- `endpointPath=''`
- `responseFormat='text'`
- BaseURL 为完整请求地址
- 模板/Pointer/Auth 必填并原样交后端权威校验

watch identity 加入四字段；模型变化仍只清 probe。

- [x] **Step 5：实现目录禁用和状态恢复**

`refreshModels()` 首行检查 `custom_json`，在 bridge 前：

```js
setModelCatalog('unsupported', '自定义请求模板不提供模型目录，请手动填写模型', 'catalog_unsupported', 0)
return false
```

切回 Chat 后允许再次刷新。手填模型永不被清空。

- [x] **Step 6：扩展局部保存成功基线**

成功时不从安全 DTO 读取模板或鉴权值，而是复制本次已验证 request snapshot 到 `settings`；canonical BaseURL/协议/模型可从 DTO 更新。revision 不一致时丢弃结果。

- [x] **Step 7：运行 Store 回归**

```bash
node frontend/unified_api_settings_regression_test.mjs
node frontend/ccr_settings_lifecycle_regression_test.mjs
node frontend/model_connectivity_regression_test.mjs
node frontend/settings_regression_test.mjs
```

预期：PASS。

---

### Task 7：重构设置 UI 与响应式布局

**文件：**
- 修改：`frontend/src/components/ProviderSelect.vue`
- 修改：`frontend/src/components/SettingsModal.vue`
- 修改：`frontend/src/components/ModelSelect.vue`（仅必要 active/锁行为）
- 修改：`frontend/ccr_ui_regression_test.mjs`
- 修改：`frontend/model_layout_regression_test.mjs`
- 修改：`frontend/unified_api_settings_regression_test.mjs`

- [x] **Step 1：写 UI 结构红灯测试**

`ccr_ui_regression_test.mjs` 断言：

- API select 只有 `openai_chat` 与 `custom_json` 两个 option。
- 页面包含“请求地址”“API Key”“高级设置”。
- 默认模板高级字段存在但由 `v-if="apiProtocol === 'custom_json'"` 控制。
- 不再直接展示 Responses/Anthropic option。
- CCR 可保留为 Chat 地址 preset，但不是第三模式。

`model_layout_regression_test.mjs` 断言：

- 存在 `ui.activeTab === 'scene'` 和独立 scene pane。
- API pane 内不含 `DomainSelector`。
- `.connection-actions` 为两列。
- 保存按钮 class 含跨列规则。
- `@media (max-width: 420px)` 改为单列。
- button `min-width:0`。
- 每个 pane 纵向滚动，Footer 不 absolute/fixed 覆盖 body。
- model dropdown active 只绑定 API tab。

- [x] **Step 2：运行 UI 红灯**

```bash
node frontend/ccr_ui_regression_test.mjs
node frontend/model_layout_regression_test.mjs
node frontend/unified_api_settings_regression_test.mjs
```

预期：旧三模式和同页 DomainSelector 导致失败。

- [x] **Step 3：重写 `ProviderSelect.vue` 紧凑表单**

默认可见：

```text
API 模式
请求地址
API Key
```

删除 overview card。`<details>` 内：

- Chat：Extra Headers。
- Custom：Request Template textarea、Response JSON Pointer、Auth Header、Auth Prefix、Extra Headers。

props/emits 增加四个模板字段。切换模式只更新草稿：

- Chat：Endpoint `/chat/completions`，ResponseFormat `text`。
- Custom：空模板时填默认模板/Pointer/Auth；不发网络、不保存。

密码输入不回显已保存 Key 的掩码为真实值；继续使用当前 session 草稿。

- [x] **Step 4：增加独立场景标签页**

header 新增：

```vue
<div class="tab" :class="{ active: ui.activeTab === 'scene' }" @click="ui.activeTab = 'scene'">场景</div>
```

API pane 删除 DomainSelector，新增：

```vue
<div v-show="ui.activeTab === 'scene'" class="tab-pane scene-tab-pane">
  <DomainSelector v-model="settingsStore.tempSettings.domainId" :categories="settingsStore.domainCategories" />
</div>
```

scene pane 只负责页面布局，不改变保存语义。

- [x] **Step 5：修复主按钮布局**

模板：保存按钮加 `connection-save-action` class。CSS：

```css
.connection-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sp-2);
}
.connection-actions .connection-save-action { grid-column: 1 / -1; }
.connection-actions button { min-width: 0; min-height: 40px; white-space: normal; }
@media (max-width: 420px) {
  .connection-actions { grid-template-columns: 1fr; }
  .connection-actions .connection-save-action { grid-column: auto; }
}
```

模板模式刷新按钮 disabled，并显示“模板模式手填模型”或旁侧状态；不得隐藏按钮造成布局跳动。

- [x] **Step 6：修复标签、滚动和 Footer**

- `.tabs` 保持 `overflow-x:auto`，`.tab{flex:0 0 auto}`。
- tab click 后 `nextTick` + active element `scrollIntoView({inline:'nearest', block:'nearest'})`。
- `.modal-body{flex:1;min-height:0;overflow:hidden}`。
- `.tab-pane{height:100%;overflow-y:auto}`。
- `.scene-tab-pane :deep(.domain-list){max-height:none;overflow:visible}`。
- Footer 保持普通 flex 子元素，不使用 absolute/fixed。
- watch activeTab/close 时 `modelDropdownOpen=false`。

- [x] **Step 7：运行 UI 和全部前端回归**

```bash
for test_file in frontend/*_test.mjs; do node "$test_file" || exit 1; done
```

预期：全部 PASS。

---

### Task 8：生成 bindings 与更新 README

**文件：**
- 修改：`frontend/wailsjs/go/app/App.js`
- 修改：`frontend/wailsjs/go/app/App.d.ts`
- 修改：`frontend/wailsjs/go/models.ts`
- 修改：`README.md`
- 修改：`README_EN.md`

- [x] **Step 1：先运行聚焦测试**

```bash
go test ./pkg/config ./pkg/llm ./pkg/solution ./app -count=1
for test_file in frontend/*_test.mjs; do node "$test_file" || exit 1; done
```

预期：PASS 后才生成 bindings。

- [x] **Step 2：冻结生成保护哈希**

盘点 `go.mod`、`go.sum`、`frontend/wailsjs/**`、`frontend/dist/**`、`build/bin/releases/**` 和 Git index。只记录路径、大小、SHA-256，不读取用户配置。

- [x] **Step 3：使用 Wails v2.11.0 生成 bindings**

使用同版本 CLI；只允许 `ConnectionDraft`、`ConnectionSettingsView` 和 `config.Config` 四个新字段及必要 constructor 变化。`runtime/**`、`go.mod`、`go.sum` 必须不变；删除仅空白生成噪音。

- [x] **Step 4：更新双语 README**

写明：

- 两种前端模式。
- 普通用户只需请求地址、Key、模型。
- 模板高级字段、安全占位符、非流式 JSON 与 Pointer。
- 模板模式不支持刷新模型目录，需手填模型。
- 旧 Responses/Anthropic 保存时迁移且变为非流式。
- 模型与场景分页。
- 文字/图片测试可能产生费用。
- 不支持脚本、通用 SSE、自动协议探测或 fallback。

不得写真实凭据、私有 URL 或 Header 值。

- [x] **Step 5：构建验证 bindings**

```bash
npm --prefix frontend run build
go build ./...
```

预期：PASS；Vite chunk warning 如复现单独记录。

---

### Task 9：Revision B 全量验证和候选构建

**文件：**
- 修改：`_dev/changes/037-unified-api-connection/03-implementation.md`
- 修改：`04-validation.md`
- 修改：`06-notes.md`
- 修改：`_dev/audits/037-unified-api-connection.md`
- 修改：`_dev/changes/INDEX.md`
- ignored 产物：`build/bin/**`

- [x] **Step 1：运行完整自动门禁**

```bash
go test ./... -count=1
go vet ./...
go build ./...
go mod verify
for test_file in frontend/*_test.mjs; do node "$test_file" || exit 1; done
npm --prefix frontend run build
git diff --check
```

逐项记录 PASS/FAIL/BLOCKED；不得用聚焦结果替代全量结果。

- [x] **Step 2：运行 race**

```bash
go test -race ./pkg/config ./pkg/llm ./pkg/solution ./app -count=1
```

若启动阶段返回 `0xc0000139`，记录 `BLOCKED` 和原始包结果，不改写为 PASS。

- [x] **Step 3：归档 Revision A 候选**

先验证当前候选哈希必须为：

```text
023762359C7FAEF5C00D392ADC2CA6E45DBF00179AF07BC3DFBC92777B48F9B4
```

复制到：

```text
build/bin/archive/transient/Q-Solver-candidate-change037-revision-a.exe
```

验证复制哈希后才移除 current 中旧候选；稳定副本必须仍等于根稳定版 `29F0100...80FE`。不得修改 Releases。

- [x] **Step 4：构建 Revision B 候选**

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build \
  -s \
  -o testing/current/Q-Solver-candidate-change037.exe \
  -skipbindings \
  -skipembedcreate \
  -nosyncgomod \
  -m \
  -nocolour \
  -v 1 \
  -ldflags "-s -w" \
  -tags prod
```

生成并验证新的 `testing/current/SHA256SUMS`。`go version -m` 必须显示 Wails v2.11.0。逐一复核依赖、bindings/runtime、dist、Release、Git index。

- [x] **Step 5：独立启动验证**

使用隔离临时 `APPDATA/LOCALAPPDATA`：运行至少 8 秒；WebView2 direct child ≥1；Node/Vite direct child=0；`CloseMainWindow` 正常退出；ExitCode=0；ForcedCleanup=false；残留 direct child=0。清理隔离临时目录。

- [x] **Step 6：更新治理记录和提交边界**

记录 Revision A 与 B 两套候选身份，不能用 B 覆盖 A 历史。Commit 1 文件范围增加 custom/template/solution 与前端修订；Commit 2 仍为 Change/审计/INDEX。保持未暂存、未提交、未推送。

---

### Task 10：GUI 与真实接口验收

**候选：** Task 9 生成的 Revision B `Q-Solver-candidate-change037.exe`。

- [ ] **Step 1：布局验收**

检查 520px 默认弹窗及可达到的窄窗口：

1. 两种 API 模式完整显示。
2. 三个按钮文字完整且无横向裁切。
3. 模型 dropdown 打开/关闭和切页后滚动正常。
4. 场景页独立、角色卡完整可滚动。
5. Footer 不遮挡最后一项内容。

- [ ] **Step 2：OpenAI 兼容模式验收**

经用户主动提供测试配置后：root/完整 Chat 地址、刷新、文字测试、图片测试、保存和实际解题。记录费用授权，不记录 Key/URL 实值。

- [ ] **Step 3：自定义模板模式验收**

使用经授权本地或真实接口：默认模板、手填模型、禁用刷新、文字测试、图片测试、保存和完整非流式答案。非法模板/Pointer/Auth Header 必须阻止 bridge 或由后端安全拒绝。

- [ ] **Step 4：legacy 迁移验收**

用保护副本验证旧 Responses/Anthropic 配置：打开时只生成草稿、不写盘；取消不迁移；保存后变为 `custom_json` 且连接字段保留。不得在未保护用户配置时进行破坏性试验。

- [ ] **Step 5：最终状态**

必要 GUI 项全部 PASS 后才能讨论候选晋升和提交；race `BLOCKED` 继续披露。commit、push、晋升、Tag、Release 分别等待明确授权。

---

## 计划自审

- 设计 §4 Config/协议：Task 1、5、6 覆盖。
- 设计 §5 模板和 Pointer：Task 2 覆盖全部占位符、安全语法与提取。
- 设计 §6 CustomJSONAdapter：Task 3 覆盖 Provider 全接口、HTTP 和错误。
- 设计 §7 legacy 迁移：Task 6、10 覆盖未保存/保存边界。
- 设计 §8 Keyless：Task 4 覆盖实际 Solver，不以 probe 冒充。
- 设计 §9 UI：Task 7 覆盖两模式、分页、按钮、滚动和 Footer。
- 设计 §10 状态/保存：Task 5、6 覆盖 DTO、局部 patch、identity 和迟到隔离。
- 设计 §11 全量：Task 8～10 覆盖 bindings、README、门禁、候选和 GUI。
- 设计 §12 文件范围：计划未扩展到题库、Mobile、快捷键或截图采集。
- 接口名称在任务间一致：`custom_json`、`RequestTemplate`、`ResponseJSONPointer`、`AuthHeader`、`AuthPrefix`。
- 占位项扫描通过，所有步骤都包含具体接口、断言或命令。
- 项目未授权 Git 写操作；所有默认 commit 步骤均替换为治理检查点。

## 2026-11-08 实施补充：Keyless 入口一致性

实际审计发现 app/solve.go 的 TriggerScreenshot、triggerSend、solveInternal 仍以空 APIKey 拒绝；仅修 Solver 无法满足已批准的无鉴权实际解题目标。范围增加 `app/solve.go` 和 `app/solve_preflight_test.go`，仅把 Key 必填判断改为地址/模型连接就绪判断及无效 Provider 安全拒绝，并将旧 model 设置路由改为 api；不改截图采集、任务与题库行为。
