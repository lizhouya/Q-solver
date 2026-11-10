# Change 037 统一模型与 API 接入设计

## 0. 修订说明

本文件以 **Revision D（2026-11-09，已获用户批准）** 为当前权威设计。Revision D 在 Revision B 两模式连接与分页布局基础上增加多条加密连接 Profiles，并保留 Revision C 中正确的快捷键字段与全显示端同步修复；Revision C 的“模型历史”产品结论被用户澄清取代。

Revision A 已实现并完成普通自动验证，候选 SHA-256 为 `023762359C7FAEF5C00D392ADC2CA6E45DBF00179AF07BC3DFBC92777B48F9B4`，但在 GUI 验收前被本修订取代，不得晋升稳定版。旧实现和验证事实继续保留在 `03-implementation.md`、`04-validation.md`，不倒写为失败。

## 1. Revision B 设计结论

前端 API 模式只展示：

1. `OpenAI 兼容 Chat Completions`
2. `自定义请求模板`

普通用户默认只需处理请求地址、API Key 和模型。高级字段折叠显示。OpenAI 兼容模式继续使用强类型 Chat adapter 和流式响应；自定义模板模式使用新的受控 `custom_json` adapter，只执行 POST JSON 和非流式 JSON 响应提取，不执行脚本、循环、条件、动态网络请求或通用 SSE 解析。

设置弹窗保留“模型与 API”页面，并新增独立“场景”页面。模型刷新、连接测试和保存按钮采用不会被 520px 弹窗裁切的响应式两行布局。

## 2. 用户已确认的 Revision B 决策

| 决策 | 结论 |
|---|---|
| 前端 API 模式 | OpenAI 兼容 Chat Completions + 自定义请求模板 |
| 默认表单 | 请求地址、API Key、模型；其他字段折叠 |
| 模板配置 | 折叠高级编辑，内置可工作的默认模板 |
| 模型目录 | 兼容模式可刷新；模板模式手填模型并禁用刷新 |
| 模板响应 | 首版仅非流式 JSON |
| 图片 | 使用受控类型占位符，支持截图解题 |
| API Key | 默认 `Authorization: Bearer`，高级区可改 Header 名和前缀 |
| 旧 Responses/Anthropic | 设置页自动生成等价模板草稿；用户保存后才迁移 |
| 页面 | “模型与 API”和“场景”分为独立标签页 |
| Git/发布 | 本修订批准不包含 commit、push、稳定版晋升、Tag 或 Release |

## 3. 方案边界

### 3.1 采用：受控 JSON 模板

模板必须是 JSON，程序先解析为值树，再递归替换**完整字符串值**形式的受控占位符。替换值按实际 JSON 类型注入，不做字符串拼接。

### 3.2 不采用：脚本模板

不执行 JavaScript、Go template 函数、JQ、表达式、循环、条件、文件读取、环境变量读取或模板内网络请求。API Key 不可作为正文或 URL 占位符。

### 3.3 不采用：通用 SSE

`custom_json` 首版只接受一次性 JSON 响应。未来若需要 SSE，必须另立 Change 定义 framing、增量字段、终止事件、错误事件和中断语义。

## 4. 权威连接模型

### 4.1 协议枚举

后端保留现有 legacy 协议以读取未迁移配置：

```text
openai_chat
openai_responses       // legacy 可继续运行；新 UI 不直接提供
anthropic_messages     // legacy 可继续运行；新 UI 不直接提供
custom_json            // Revision B 新增
```

UI 只将 `openai_chat` 显示为“OpenAI 兼容 Chat Completions”，将 `custom_json` 显示为“自定义请求模板”。

### 4.2 Config 与 ConnectionDraft 字段

保留 Revision A 字段，并新增：

```text
requestTemplate       string  // JSON 文本，最大 64 KiB
responseJSONPointer   string  // RFC 6901，必须指向非空字符串
authHeader            string  // 默认 Authorization
authPrefix            string  // 默认 "Bearer "
```

现有字段继续保留：

```text
apiKey
baseURL
connectionMode        // 新保存仍规范为 direct
apiProtocol
endpointPath
extraHeaders
responseFormat
model
```

`openai_chat` 中 `requestTemplate/responseJSONPointer/authHeader/authPrefix` 规范化为空或默认安全值，不参与请求。`custom_json` 中 `responseFormat` 固定为 `text`，`endpointPath` 为空。

### 4.3 请求地址语义

#### OpenAI 兼容模式

UI 标签为“请求地址”，允许：

- API root，例如 `https://api.example.com/v1`；或
- 以 `/chat/completions` 结尾的完整地址。

前端只做即时展示，后端负责权威拆分和规范化：完整 Chat 地址拆为 Base URL 与 `/chat/completions`；root 使用默认 Endpoint。不得猜测其他任意路径。

#### 自定义模板模式

`baseURL` 保存完整推理 URL。只允许 HTTP/HTTPS，必须有 host，禁止 userinfo、query、fragment 和 CR/LF。请求固定 POST 到该 URL，不追加 Endpoint。

### 4.4 API Key 与 Headers

- `authHeader` 默认 `Authorization`，按现有 Header token 规则验证。
- `authPrefix` 默认 `Bearer `，禁止 CR/LF，最大 128 字节。
- Key 非空时发送 `authHeader: authPrefix + key`。
- Key 为空时不发送该 Header，继续支持无鉴权本地服务。
- `extraHeaders` 继续使用 Revision A 的名称、值、数量、大小和传输控制头限制。
- `extraHeaders` 不得重复定义规范化后的 `authHeader`。
- 模板、日志、错误 DTO、保存结果和文档不得包含 Key 实值。
- HTTP client 继续禁止重定向，避免凭据跨 origin 传播。

## 5. 安全模板契约

### 5.1 默认模板

新建自定义模板草稿时使用：

```json
{
  "model": "{{model}}",
  "messages": "{{openai_messages}}"
}
```

默认响应路径：

```text
/choices/0/message/content
```

因此 OpenAI 风格但使用非标准完整 URL 的接口仍可只填写请求地址、Key 和模型。

### 5.2 允许的占位符

| 占位符 | 注入类型 | 语义 |
|---|---|---|
| `{{model}}` | string | 当前选定模型 |
| `{{system_prompt}}` | string | 当前请求中的 system 文本合并值 |
| `{{user_text}}` | string | 最新 user 消息的文本内容 |
| `{{openai_messages}}` | array | 完整上下文的 OpenAI Chat 多模态消息结构 |
| `{{responses_input}}` | array | 完整上下文的 OpenAI Responses input 结构 |
| `{{anthropic_messages}}` | array | 非 system 上下文的 Anthropic messages 结构 |
| `{{anthropic_system}}` | array | Anthropic system blocks |
| `{{images_data_urls}}` | array[string] | 当前请求中全部图片 data URL |
| `{{images_base64}}` | array[string] | 当前请求中全部图片纯 Base64 数据 |

占位符只在某个 JSON 字符串值**完全等于**表中值时替换。`"prefix-{{model}}"`、未知占位符和对象 key 中的占位符均拒绝保存。

图片和 PDF 继续复用 `ParseBase64DataURLStrict` 与 MIME 白名单；不因模板模式放宽输入校验。

### 5.3 模板校验

- JSON 最大 64 KiB。
- 顶层必须是 JSON object。
- 至少包含一个消息、文字或图片占位符，防止保存与实际请求无关的固定 body。
- 只能使用白名单占位符。
- 解析后再类型替换，禁止文本级 JSON 拼接。
- 保存时完成语法和占位符校验，但不发送网络请求。

### 5.4 响应提取

- `responseJSONPointer` 必须是合法 RFC 6901 pointer，必须以 `/` 开头。
- 响应 body 最大 4 MiB；超限返回 `request_too_large` 类安全错误。
- 成功响应必须是合法 JSON。
- Pointer 结果必须是非空字符串；不存在、类型不符或空字符串均返回 `invalid_response`。
- 非 2xx 状态只按 HTTP status 分类，不把响应 body、Key 或 Header 值写入日志和错误 DTO。

## 6. CustomJSONAdapter

新增 `pkg/llm/custom_json.go`，实现现有 `Provider`：

```text
GenerateContent
  → 渲染受控 JSON body
  → POST 一次
  → JSON Pointer 提取完整文本

GenerateContentStream
  → 调用同一非流式执行器
  → 成功后向 onChunk 发送一个 ChunkContent
  → 返回完整 Message

TestChat
  → 使用最小文字消息执行一次模板请求

GetModels
  → 不发网络请求，返回 catalog_unsupported
```

这样 Solver、题库整理、简历解析和 probe 不需要建立第二套事件协议。模板模式不会产生 thinking chunk。

`probeTextWithToken` 和图片 probe 继续复用现有随机挑战。模板包含对应消息/图片占位符时可以进行真实测试；不含图片占位符时图片 probe 在发送前返回明确配置错误。

## 7. 旧配置迁移

### 7.1 未保存前

后端继续保留 `OpenAIResponsesAdapter` 和 `AnthropicAdapter`，所以旧配置在用户打开设置但没有保存时仍按原 adapter 运行，不发生静默磁盘迁移。

前端加载时将 legacy 配置映射为自定义模板草稿：

#### OpenAI Responses

```json
{
  "model": "{{model}}",
  "input": "{{responses_input}}",
  "store": false
}
```

- 请求地址：原 Base URL + 原 `/responses` Endpoint。
- 响应 Pointer：`/output/0/content/0/text`。
- 鉴权：`Authorization` + `Bearer `。

#### Anthropic Messages

```json
{
  "model": "{{model}}",
  "max_tokens": 8192,
  "system": "{{anthropic_system}}",
  "messages": "{{anthropic_messages}}"
}
```

- 请求地址：原 Base URL + 原 `/v1/messages` Endpoint。
- 响应 Pointer：`/content/0/text`。
- 鉴权 Header：`x-api-key`，前缀为空。
- 自动加入非敏感 `anthropic-version` Header；如旧配置已有值则保留。

UI 显示迁移提示：保存后将改为自定义非流式 JSON；取消设置不会写盘。

### 7.2 保存后

用户点击“保存 API 设置”或“保存全部设置”后写为：

```text
connectionMode=direct
apiProtocol=custom_json
```

旧 URL、Key、Model 和允许的 Extra Headers 保留。保存事务仍遵循 clone → patch → normalize → validate → provider factory → save → swap → notify。

## 8. Keyless 解题一致性

Revision A 已使 adapter、目录和 probe 支持空 Key，但 `pkg/solution.Solver.Solve` 仍有旧的 `APIKey == ""` 前置拒绝。Revision B 必须删除该 Key 专属判断，改为依赖已构造 Provider 和连接校验；用无鉴权本地 `httptest.Server` 证明实际截图解题链路可发送请求。

不得通过“伪造 Key”绕过该问题。

## 9. 前端信息架构

### 9.1 标签

```text
常规 | 模型与 API | 场景 | 截图 | 输出 | 题库 | 简历
```

标签栏保持水平滚动；切换标签后调用 `scrollIntoView({block:'nearest', inline:'nearest'})` 保证活动标签可见。关闭按钮独立固定，不被标签滚动挤出。

### 9.2 模型与 API 页面

顺序：

1. API 模式：两项。
2. 请求地址。
3. API Key。
4. 高级连接设置。
5. 模型名称与候选下拉。
6. 费用提示与错误。
7. 三个主按钮。
8. 文字测试结果。
9. 高级图片实测。

移除占高的 overview card 和默认展开的 Endpoint/response format 字段。兼容模式高级区可保留 Extra Headers；模板模式高级区显示模板、响应 Pointer、鉴权 Header/前缀和 Extra Headers。

### 9.3 场景页面

`DomainSelector` 从 API pane 移到独立 `scene` pane，使用该 pane 自身的纵向滚动，不再与模型 dropdown scroll lock 共享状态。

### 9.4 按钮布局

宽度充足时：

```text
[        保存 API 设置         ]
[   刷新候选模型   ][ 测试 API 连接 ]
```

CSS 使用两列 grid，保存按钮跨两列。按钮设置 `min-width:0`、允许正常文字显示。窄于 420px 时三按钮单列。

模板模式下刷新按钮禁用，文案和状态明确为“自定义模板模式请手动填写模型”，不调用 bridge。

### 9.5 滚动和 Footer

- 每个 `.tab-pane` 是唯一纵向滚动容器。
- `.modal-body` 继续 `min-height:0`，Footer 继续在 flex 布局底部，不覆盖内容。
- 只有模型下拉打开时锁定 API pane；切到场景或关闭弹窗时立即释放。
- 不使用场景列表内部嵌套滚动或裁切。

## 10. 状态与保存

连接 identity 新增模板、Pointer、鉴权 Header 和前缀。任一连接字段变化：

- 清除目录/probe 状态。
- 递增 connection revision。
- 迟到 list/probe/save 结果不得覆盖新草稿。

模板模式不允许目录请求，因此 `refreshModels()` 在调用 bridge 前返回明确状态。API 专用保存继续只 patch 连接字段；场景草稿只由“保存全部设置”提交。

安全 DTO 只新增非敏感字段或 `TemplateSet` 布尔状态，不回显 RequestTemplate、Key、Header 值或 AuthPrefix。

## 11. 测试策略

### Go

- `custom_json` protocol 与 Config 默认/校验。
- 模板大小、顶层类型、未知/嵌入式占位符、内容占位符要求。
- 各允许占位符的类型化渲染。
- 图片 data URL/Base64、PDF 与历史消息转换。
- RFC 6901 Pointer 转义、数组索引、缺失、类型错误和空值。
- 非 2xx 分类、响应上限、无重定向、context cancel/timeout。
- Keyless、可配置 auth Header/prefix、Header 冲突和脱敏。
- `GenerateContentStream` 只产生一个 content chunk。
- `GetModels` 零网络请求。
- legacy Responses/Anthropic 保存迁移字段完整。
- `SaveConnectionSettings` 仍为原子局部 patch。
- Solver 无 Key 本地截图解题回归。

所有网络测试只使用 `httptest.Server`。

### 前端

- API 模式只有两项。
- 默认只显示请求地址、Key、模型；高级字段折叠。
- legacy Responses/Anthropic 生成正确模板草稿且不自动保存。
- 模板模式刷新在 bridge 前停止。
- 模板字段进入连接 identity、保存 payload 和迟到结果隔离。
- 独立 `scene` 标签与 DomainSelector。
- 按钮两行布局、保存跨列、窄屏单列。
- Footer 不遮挡、API pane 独立滚动、模型下拉锁正确释放。
- 全局保存仍包含 scene，API 专用保存不包含 scene。

### 全量

- 聚焦测试。
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `go mod verify`
- 全部 `frontend/*_test.mjs`
- `npm --prefix frontend run build`
- 固定 Wails v2.11.0 bindings 与保护构建。
- 新候选独立启动、WebView2、无 Node/Vite、正常退出和哈希保护。
- `go test -race ./pkg/config ./pkg/llm ./pkg/solution ./app -count=1`；若仍为 `0xc0000139`，如实记为 BLOCKED。

## 12. 文件范围

新增：

- `pkg/llm/custom_json.go`
- `pkg/llm/custom_json_test.go`
- 可选的 `pkg/llm/template.go` / `template_test.go`；仅当拆分能保持单一职责时创建。

修改：

- `pkg/llm/protocol/protocol.go` 及测试。
- `pkg/config/config.go`、`connection.go`、`manager.go` 及测试。
- `pkg/llm/factory.go`、`service.go`、`probe.go`、必要的消息转换 helper 及测试。
- `pkg/solution/solution.go`、`solution_test.go`。
- `app/llm.go`、`app/settings_test.go`、`app/llm_test.go`。
- `frontend/src/stores/settings.js`、`services/api.js`（仅 DTO 需要时）。
- `ProviderSelect.vue`、`SettingsModal.vue`、必要的 `ModelSelect.vue`。
- 相关 `frontend/*_test.mjs`。
- `frontend/wailsjs/go/**`，使用固定 Wails v2.11.0 生成。
- `README.md`、`README_EN.md`。
- Change 037 和审计记录。

不新增依赖，不修改 Go/Node/Wails/SDK 版本，不修改题库、Mobile、快捷键、截图采集或 Prompt 语义。

## 13. 回滚与候选

- Revision A 候选在重建前按原哈希移入 `build/bin/archive/transient/Q-Solver-candidate-change037-revision-a.exe`，不得标为正式通过或稳定版。
- 测试前稳定副本继续与根稳定版哈希一致。
- Revision B 构建仍使用 `Q-Solver-candidate-change037.exe`，更新 `testing/current/SHA256SUMS`。
- 候选失败、BLOCKED 或 GUI `NEEDS_USER` 时根稳定版不变。
- 代码回滚采用定向文件回滚或经授权提交后的 `git revert`，禁止 `git reset --hard`。

## 14. 非目标

- 多连接 profile。
- 模板脚本、循环、条件、函数或变量拼接。
- API Key 正文/URL占位符。
- 通用 SSE/WebSocket 响应。
- 自定义模型目录；模板模式只手填模型。
- 自动协议探测、fallback 或重试其他协议。
- 模型目录分页。
- CCR 进程管理。
- 新依赖或 runtime 升级。

## 2026-11-08 实施补充：Keyless 入口一致性

实际审计发现 app/solve.go 的 TriggerScreenshot、triggerSend、solveInternal 仍以空 APIKey 拒绝；仅修 Solver 无法满足已批准的无鉴权实际解题目标。范围增加 `app/solve.go` 和 `app/solve_preflight_test.go`，仅把 Key 必填判断改为地址/模型连接就绪判断及无效 Provider 安全拒绝，并将旧 model 设置路由改为 api；不改截图采集、任务与题库行为。

## 2026-11-09 GUI 验收补充：标签栏完整显示

用户在 Revision B 候选人工验收时发现 520px 设置弹窗无法同时容纳七个标签，现有 `overflow-x: auto` 与 `scrollIntoView()` 形成不必要的横向滑动。该反馈按有界修订处理并已获用户批准：

- 标签“输出”改为“显示”，仅改变展示文案，不改变 `activeTab='output'` 和业务语义。
- 弹窗桌面宽度改为 720px，最大宽度保留 32px 视口边距。
- 标签容器取消横向滚动并允许自然换行；桌面宽度下一行完整显示，窄窗口才换行。
- 删除仅为横向滚动服务的 `tabsElement`、`nextTick` 与 `scrollIntoView()`。
- 变更仅涉及 `SettingsModal.vue`、布局回归测试和治理记录，不扩展连接、模板、场景或移动端业务范围。

# Revision C：按连接模型历史与快捷键提示同步

## 15. 修订背景与已确认事实

用户完成 Revision B 新布局候选的人工验收，确认七个设置标签完整显示、无横向滚动条、“显示”文案正确且弹窗宽度合适。后续测试提出三项反馈：模型希望保留已配置记录、题库目录在新候选中未显示、快捷键提示与正式版设置不同。

只读审计确认：

1. 当前模型 `Config.Model` 已在“保存 API 设置”和“保存全部设置”成功后持久化；用户新需求实际是同一连接可复用多个历史模型。
2. 题库目录独立保存在用户配置目录的 `questionbank-settings.json`；此前每次候选使用不同的临时 `APPDATA/LOCALAPPDATA`，因此看不到上一测试 Profile 的记录。这不是题库持久化代码丢失数据。
3. 顶部栏、欢迎页、截图 Dock 等桌面提示读取 Settings Store，手机端提示读取后端 Snapshot，并通过 Config subscriber 更新。
4. 快捷键录制前端把后端 `comboID` 错写到 `vkCode`，可能造成显示名已变但重启后实际组合键仍旧。
5. F9 `toggle` 调用 GhostMode/显示亲和性切换，与 `toggle_window` 的实体窗口隐藏正交；欢迎页却把 F9 标为“隐藏窗口”，语义错误。

## 16. 范围与非目标

### 16.1 目标

- 按连接保存最近使用的模型名称，下次打开设置可直接选择。
- 修复快捷键组合键字段，确保运行态、持久配置和提示一致。
- 统一 GhostMode 与实体窗口隐藏的展示文案。
- 覆盖桌面顶部、欢迎页、截图 Dock、空/错误状态、手机同步端和 README 的快捷键提示。
- 用固定隔离测试 Profile 验证题库目录、模型历史和快捷键跨重启保留。

### 16.2 非目标

- 不实现完整连接 Profiles，不保存多个 API Key 或整套连接草稿。
- 不复制或暴露 API Key、Headers、请求模板、响应正文或题库内容。
- 不增加模型历史删除/重命名/置顶管理界面。
- 不改变题库目录文件格式、题库存储结构或正式用户配置。
- 不合并 GhostMode、WindowVisibility、ClickThrough 或 OutputMode 语义。
- 不新增依赖，不升级 Wails、Go、Vue、Node 或 SDK。

## 17. 模型历史数据模型

### 17.1 持久结构

在 `config.Config` 增加：

```go
type ModelHistoryEntry struct {
    APIProtocol  string   `json:"apiProtocol"`
    BaseURL      string   `json:"baseURL"`
    EndpointPath string   `json:"endpointPath,omitempty"`
    Models       []string `json:"models"`
}

ModelHistory []ModelHistoryEntry `json:"modelHistory,omitempty"`
```

历史随现有主配置加密保存。记录不包含 Key、Header、模板、Pointer、响应或 probe 结果。

### 17.2 连接身份

连接身份由规范化后的三元组确定：

```text
APIProtocol + BaseURL + EndpointPath
```

- `openai_chat` 使用规范化 API root 与 `/chat/completions`。
- `custom_json` 使用完整推理 URL，`EndpointPath` 为空。
- legacy Responses/Anthropic 未保存时继续原 adapter；设置页迁移草稿保存为 `custom_json` 后，历史归入迁移后的规范化身份。
- API Key 变化不拆分模型历史，避免密钥轮换导致模型记录丢失。

### 17.3 更新规则

仅以下事务成功后记录当前非空模型：

1. `SaveConnectionSettings` 成功。
2. “保存全部设置”的 `UpdateFromJSON` 成功。

刷新目录、文字/图片 probe、修改草稿、取消设置或保存失败均不更新历史。

每次记录执行：

- Trim 模型名称；空值和超过 256 字节的名称不加入历史，当前连接保存仍按既有 Config 规则处理。
- 当前模型移动到列表首位，大小写按原值保留，精确字符串去重。
- 每个连接最多 20 个模型。
- 当前连接移动到连接历史首位；最多保留 12 个连接。
- 所有限制在后端规范化和验证，不依赖前端自律。

旧配置缺少 `modelHistory` 时按空列表加载，不主动保存或迁移文件。

### 17.4 克隆与安全 DTO

`cloneConfigSnapshot` 深拷贝历史条目和模型切片，避免事务候选污染当前配置。`ConnectionSettingsView` 只返回当前连接的 `SavedModels []string`，不新增 Key/Header/模板回显。完整 `GetSettings` 仍通过现有加密配置读取路径向本机设置页提供历史，用于连接切换时过滤。

## 18. 前端模型复用

Settings Store 保存完整 `modelHistory` 基线，并根据当前草稿的规范化连接身份计算 `savedModelsForCurrentConnection`。

模型建议顺序：

1. 当前连接“已保存模型”，按最近使用排序。
2. 本次目录刷新返回的候选。
3. 两组精确去重，已保存项优先。

`ModelSelect` 增加两组展示标签“已保存模型”和“目录候选”；仍允许自由输入。模板模式不调用目录，但可展示“已保存模型”。连接身份变化时立即切换对应历史，迟到目录结果仍由现有 revision/version 规则丢弃。

API 专用保存成功后，以后端返回的 `SavedModels` 更新当前连接历史基线；全量保存成功后使用提交后的完整历史更新基线。失败时不改变历史。

## 19. 题库目录与测试 Profile

产品代码不修改题库目录持久化。后续候选人工测试固定复用：

```text
%TEMP%\qsolver-change037-manual-profile\
```

该目录只存 Change 037 测试配置，候选重建时不删除；不读取、复制或修改正式用户 Profile。验证流程：

1. 在固定 Profile 中选择题库目录并保存。
2. 保存 API 模型和快捷键。
3. 正常关闭候选。
4. 使用同一 Profile 重启。
5. 核对题库目录、当前模型、按连接模型历史、快捷键显示名和实际绑定。

测试完成后是否清理 Profile 由用户决定；日志和治理文档只记录 `SET/NOT SET` 状态，不记录实际路径或配置值。

## 20. 快捷键一致性

### 20.1 录制字段

`key-recorded` 事件继续返回 `action`、`keyName`、`comboID`。前端必须写入：

```js
tempShortcuts[action] = { keyName: data.keyName, comboID: data.comboID }
```

禁止继续写 `vkCode`。已有绑定更新时同时覆盖 `keyName` 和 `comboID`。

### 20.2 运行态与保存时机

后端 `shortcut.Service.handleRecordingComplete` 现有行为保持：冲突检查通过后立即更新当前运行态并发送 `shortcut-saved`。

- `shortcut-saved` 后，桌面 Settings Store 的运行态 `shortcuts[action]` 同步为 `tempShortcuts[action]`，使顶部栏、欢迎页、截图 Dock 和其他桌面提示立即更新。
- 关闭设置但未“保存全部设置”时，本次运行态维持新绑定；应用重启后恢复上次已保存配置。这与现有快捷键服务行为一致。
- “保存全部设置”成功后写入加密配置，并经 Config subscriber 更新快捷键 Service 与手机端 Snapshot；手机端只展示已持久化配置，不展示未保存草稿。
- 保存失败时持久配置与手机端保持旧值，桌面运行态仍反映本次已生效的临时绑定，并提示用户重新保存或重启恢复。

### 20.3 语义文案

| action | 语义 | 统一文案 |
|---|---|---|
| `toggle` | GhostMode / 防录屏显示亲和性 | 隐身模式 |
| `toggle_window` | 主窗口实体显示/隐藏 | 隐藏/显示窗口 |

同步修改：

- 设置页快捷键名称。
- 欢迎页 F9 提示。
- 手机端快捷键定义。
- README / README_EN 快捷键表。
- 相关静态和行为回归测试。

顶部栏继续保持三个紧凑按键值，不新增长标签；按键值读取响应式 Store。截图 Dock 继续显示截图、删除、发送三个动态绑定。空状态和错误状态继续显示其实际使用的截图/发送绑定。

## 21. 错误处理与边界

- 加载或接收历史时跳过空值、超长名称和非法条目，再执行连接/模型数量上限；不回显配置内容。
- 连接保存失败时，历史、当前配置、活动 Provider 和 subscriber 均不改变。
- 全量保存失败时，模型历史和快捷键持久配置均不改变。
- 模型目录不可用时仍展示已保存模型；历史为空时继续允许手填。
- 固定测试 Profile 不存在时自动创建空 Profile；存在时复用，不清空。
- 题库目录不可用时沿用现有安全提示，不猜测或替换路径。

## 22. 测试与验收

### 22.1 Go

- Config：旧配置兼容、身份规范化、去重/LRU、12×20 上限、256 字节上限、深拷贝。
- ConfigManager：API 局部保存和全量保存成功后记录；失败原子性；通知次数不变。
- App：`SavedModels` 安全 DTO；模板/Key/Header 不回显。
- Mobile：自定义快捷键值和统一文案；Config subscriber 更新 Snapshot。
- Shortcut：录制运行态、冲突与重启持久化输入保持现有行为。
- QuestionBank：运行现有偏好保存/重载测试，不修改实现。

### 22.2 前端

- 连接切换只显示对应保存历史。
- 保存成功更新历史；失败不更新。
- 历史与目录候选分组、去重；模板模式目录零 bridge。
- `comboID` 正确写入，禁止 `vkCode`。
- 录制成功后顶部栏、欢迎页、截图 Dock 的响应式值更新。
- F9 显示“隐身模式”，实体隐藏显示“隐藏/显示窗口”。

### 22.3 完整门禁

- `go test ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `go mod verify`
- 全部 `frontend/*_test.mjs`
- `npm --prefix frontend run build`
- Windows race 按既有环境结果如实记录。
- Wails v2.11.0 保护构建与 `go version -m`。
- 固定隔离 Profile 两次启动人工验收。

## 23. 回滚

- 模型历史字段可被旧版本忽略；定向回滚时停止写入但不删除配置中的历史。
- 快捷键字段修复可独立回滚，但不得把 `vkCode` 重新作为持久字段。
- 文案回滚不得重新混淆 GhostMode 与 WindowVisibility。
- 候选失败时根稳定版不变；不使用正式配置做回滚验证。

# Revision D：多条加密连接 Profiles

## 24. 修订背景与结论

用户澄清需要保存的是“请求地址和密钥”，目标是输入一次后下次无需重新输入，并确认需要多条可选择、可新增、可修改、可删除的连接记录。Revision C 的按连接模型历史不是所需功能，其未提交实现和候选不得晋升；快捷键 `comboID`、动态提示和语义修复方向正确，继续保留。

采用**现有加密主配置内嵌 Profiles**：不新增文件、不新增依赖，复用 ConfigManager 的加密、原子写盘、回滚和 subscriber。根级连接字段继续镜像活动 Profile，Solver、LLM Service、题库模型调用和旧版本兼容读取不需要改为 Profile-aware。

不采用：

- 独立连接记录文件：会产生主配置与记录文件的跨文件事务分叉。
- 系统凭据库：需要平台专用实现或新依赖，超出本轮跨平台范围。
- 浏览器存储：不可靠且无法与后端 Provider 原子切换。

## 25. 持久数据模型

```go
type ConnectionProfile struct {
    ID                  string            `json:"id"`
    Name                string            `json:"name"`
    APIKey              string            `json:"apiKey,omitempty"`
    BaseURL             string            `json:"baseURL"`
    APIProtocol         string            `json:"apiProtocol"`
    EndpointPath        string            `json:"endpointPath,omitempty"`
    ExtraHeaders        map[string]string `json:"extraHeaders,omitempty"`
    ResponseFormat      string            `json:"responseFormat,omitempty"`
    RequestTemplate     string            `json:"requestTemplate,omitempty"`
    ResponseJSONPointer string            `json:"responseJSONPointer,omitempty"`
    AuthHeader          string            `json:"authHeader,omitempty"`
    AuthPrefix          string            `json:"authPrefix,omitempty"`
    Model               string            `json:"model,omitempty"`
}
```

`config.Config` 增加：

```go
ActiveConnectionProfileID string              `json:"activeConnectionProfileId,omitempty"`
ConnectionProfiles        []ConnectionProfile `json:"connectionProfiles,omitempty"`
```

根级 `APIKey`、`BaseURL`、`APIProtocol`、`EndpointPath`、`ExtraHeaders`、`ResponseFormat`、`RequestTemplate`、`ResponseJSONPointer`、`AuthHeader`、`AuthPrefix`、`Model` 继续存在，并始终镜像活动 Profile；历史字段 `ConnectionMode` 不进入 Profile，应用任一正式 Profile 时固定规范化为 `direct`，避免旧 CCR 模式残留。

### 25.1 有界规则

- 最多 20 条 Profile。
- Profile ID 使用 `crypto/rand` 生成 16 字节随机值并十六进制编码，不增加依赖。
- 持久化后的 Profile 名称 Trim 后必须非空，最多 64 个 UTF-8 字节；仅新建请求允许名称为空并在后端触发自动命名。
- ID、名称、连接字段均由后端验证；前端校验只改善交互，不能代替后端。
- `ExtraHeaders` 为每条 Profile 深拷贝，Profile slice 和配置快照均不得共享可变 map。
- Profile 保存完整连接字段，但不保存快捷键、题库、场景、截图、输出模式或其他全局设置。

### 25.2 自动命名

当新建 Profile 名称为空时，根据规范化连接生成：

```text
<hostname> · OpenAI Chat
<hostname> · 自定义模板
```

无法提取 hostname 时使用“未命名连接”。自动名称与已有名称大小写不敏感冲突时，依次追加 ` 2`、` 3`。后端在 UTF-8 字符边界截短 hostname 部分，为协议后缀和冲突编号预留空间，确保最终名称不超过 64 字节；不得截断协议后缀或编号。用户修改的手动名称若重复或超过上限则拒绝保存，不静默重命名或截断。

同一 URL 允许保存不同名称、Key、模板或模型的多条记录；身份以 Profile ID 为准，不按 URL 覆盖。

## 26. 旧配置兼容与活动镜像

旧配置没有 Profiles 时：

1. 加载和规范化根级单连接，不自动写盘。
2. 设置页列表返回一条内存临时摘要，固定临时 ID 为 `legacy-current`，列表显示名称“当前连接”并标记为活动；该显示名不作为持久 Profile 名称。
3. 选择临时记录可加载原地址、Key、协议、模板和模型；返回的草稿名称为空并显示自动名称预览，用户未手动命名时由首次保存按 §25.2 生成正式名称，不要求重新输入连接字段。
4. 首次成功保存时生成随机正式 ID，持久化 Profile，并设置 `ActiveConnectionProfileID`；`legacy-current` 和“当前连接”均不写入配置。

当 Profiles 已存在：

- `ActiveConnectionProfileID` 必须指向现有 Profile。
- 活动 Profile 被规范化后复制到根级连接字段。
- 如果活动 ID 缺失或非法，加载失败并保留原配置，不猜测活动记录。
- Profile 列表顺序保持用户保存顺序；新 Profile 追加到列表尾部。
- 旧版本忽略新增字段时仍从根级镜像读取当前连接。

Revision C 已写入但从未正式发布的 `modelHistory` 字段在 Revision D Config 中移除。若固定测试 Profile 中存在该未知 JSON 字段，Go 解码自然忽略；不迁移为 Profile，不污染正式配置。

## 27. ConfigManager 原子事务

新增 Profile helper：

```go
func NormalizeConnectionProfiles(cfg Config) (Config, error)
func FindConnectionProfile(cfg Config, id string) (ConnectionProfile, bool)
func UpsertConnectionProfile(cfg *Config, draft ConnectionProfile) (ConnectionProfile, error)
func DeleteConnectionProfile(cfg *Config, id string) error
func ApplyProfileToActiveConnection(cfg *Config, profile ConnectionProfile)
```

保存 Profile 在一个 `PatchValidated` 候选快照中完成：

1. 解析并规范化完整连接草稿。
2. 新建时生成 ID和自动名称；更新时按 ID 查找，禁止修改 ID。
3. 校验名称和 20 条上限。
4. 深拷贝并 upsert Profile。
5. 将 Profile 复制为根级活动连接镜像。
6. 设置活动 ID。
7. 运行 Config.Validate 和 Provider 工厂验证。
8. 加密写盘成功后交换 Config、Provider 和 subscriber。

任何一步失败，磁盘、内存、Profiles、根级连接、活动 Provider、`oldConfig` 和 subscriber 均不改变。

“保存全部设置”继续使用单次 ConfigManager 全量事务。`GetSettings` 只作为普通全局设置快照：返回前清空 `ConnectionProfiles`，并清空根级 `APIKey`、`BaseURL`、`ConnectionMode`、`APIProtocol`、`EndpointPath`、`ExtraHeaders`、`ResponseFormat`、`RequestTemplate`、`ResponseJSONPointer`、`AuthHeader`、`AuthPrefix`、`Model`，仅保留 `ActiveConnectionProfileID` 供连接列表定位。连接表单不得从 `GetSettings` 初始化；前端先调用安全列表接口，再以选中的活动 ID（旧配置使用 `legacy-current`）定向调用 `GetConnectionProfile`，每次只取得一条完整草稿。这样页面初始化不会通过全量设置快照取得任何 Key，也不会批量下发 Profiles。

若列表或选中草稿加载失败，前端保留后端当前运行连接但不展示旧连接字段，并禁用连接保存、目录刷新、probe 和“保存全部设置”，直到单条草稿加载成功；不得用空字段覆盖活动 Profile。

全量保存 payload 在现有普通设置字段之外携带两个仅供更新入口解析、不直接持久化的元数据字段：

```json
{
  "connectionProfileId": "selected-id-or-legacy-current",
  "connectionProfileName": "用户名称或空字符串"
}
```

`ConfigManager.UpdateFromJSON` 必须先加载旧 Config 并深拷贝现有 Profiles，再解析普通设置和上述元数据；它不能用前端 payload 中缺失的 Profiles 覆盖后端记录。随后按选中 ID 在候选快照中创建/更新 Profile、激活并更新根级镜像，再与其他普通设置一起完成一次验证和加密写盘。新建草稿 ID 为空；`legacy-current` 视为首次创建，不作为正式 ID 持久化。题库偏好保持现有独立结果，不并入主配置事务。

## 28. Wails 接口与安全 DTO

```go
type ConnectionProfileSummary struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    AddressLabel string `json:"addressLabel"`
    APIProtocol  string `json:"apiProtocol"`
    Model        string `json:"model"`
    Active       bool   `json:"active"`
    APIKeySet    bool   `json:"apiKeySet"`
    TemplateSet  bool   `json:"templateSet"`
}

type ConnectionProfileDraft struct {
    ID         string          `json:"id"`
    Name       string          `json:"name"`
    Connection ConnectionDraft `json:"connection"`
}

type ConnectionProfileErrorDTO struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type ConnectionProfileListResult struct {
    Profiles []ConnectionProfileSummary `json:"profiles"`
    Error    *ConnectionProfileErrorDTO  `json:"error,omitempty"`
}

type GetConnectionProfileResult struct {
    Success bool                       `json:"success"`
    Profile *ConnectionProfileDraft    `json:"profile,omitempty"`
    Error   *ConnectionProfileErrorDTO `json:"error,omitempty"`
}

type DeleteConnectionProfileResult struct {
    Success   bool                       `json:"success"`
    DeletedID string                     `json:"deletedId,omitempty"`
    Error     *ConnectionProfileErrorDTO `json:"error,omitempty"`
}
```

接口：

```go
ListConnectionProfiles() ConnectionProfileListResult
GetConnectionProfile(id string) GetConnectionProfileResult
SaveConnectionSettings(draft ConnectionProfileDraft) SaveConnectionResult
DeleteConnectionProfile(id string) DeleteConnectionProfileResult
```

- `ListConnectionProfiles` 只返回安全摘要：`AddressLabel` 由后端从规范化 URL 提取为不含 scheme、userinfo、path、query、fragment 的 hostname；URL 显式使用非默认端口时保留 `host:port`（IPv6 使用方括号），并以布尔值表达 `SET / NOT SET`。不得返回完整 URL、Key、Header 值、模板或 AuthPrefix。
- 列表、读取、保存和删除错误只使用以下稳定码：`profile_load_failed`、`profile_not_found`、`profile_invalid`、`profile_limit_reached`、`profile_name_duplicate`、`profile_active_delete_forbidden`、`profile_legacy_delete_forbidden`、`profile_save_failed`、`profile_delete_failed`。结果 DTO 只携带对应固定安全文案，不把底层 Go `error`、配置内容或路径直接交给前端；既有 `SaveConnectionResult` 的错误字段承载同一错误码和安全文案。
- `GetSettings` 按 §27 清空 `ConnectionProfiles` 和全部根级连接编辑字段，只保留活动 ID 与普通全局设置；连接草稿必须通过按 ID 定向读取取得。
- `GetConnectionProfile` 仅对安全列表中选中的一个 ID 返回该条完整草稿，复用当前本机 Wails 设置页可读取活动 Key 的信任边界；前端初始化活动项和用户后续切换均使用这一单条定向请求。
- API Key 输入保持 `type=password`，不新增明文显示按钮。
- 保存/删除错误使用固定安全文案，不包含 Profile 内容。
- 连接测试和目录刷新使用已加载草稿；选择记录本身不联网。

## 29. 前端交互

### 29.1 记录区

“模型与 API”页顶部新增：

- “已配置连接”下拉列表。
- “新建连接”按钮。
- “删除连接”按钮。
- 可编辑“记录名称”输入框。

每条列表项显示名称、协议、主机/本地地址、模型、活动标记和 Key `SET / NOT SET`，不显示 Key 实值、Header 值或模板。

### 29.2 选择

选择记录时：

1. 递增现有 connection/catalog/probe/save revision，使迟到结果失效。
2. 调用 `GetConnectionProfile(id)` 加载完整草稿。
3. 填充现有连接表单和密码输入框。
4. 清空目录/probe 状态。
5. 不发送模型目录、probe、保存或 Provider 切换请求。

关闭设置且未保存时恢复当前活动 Profile 草稿；不会改变后端运行态。

### 29.3 新建与保存

“新建连接”建立本地空草稿：

- ID 为空。
- 名称输入框为空，旁边随地址显示自动名称预览；未编辑名称时保存请求继续发送空名称，由后端生成最终名称，避免前后端编号规则分叉。用户一旦修改名称则发送手动名称。
- 默认 `openai_chat`、`/chat/completions`、`text`，地址/Key/模型为空。
- 不复制当前 Profile 的 Key、Headers 或模板。

“保存 API 设置”和“保存全部设置”成功后更新列表、活动标记和当前草稿；失败时保留未保存草稿，列表和活动 Profile 不变。

### 29.4 删除

- 只允许删除非活动 Profile。
- 使用设置页内确认区域，禁止浏览器 `confirm()`。
- 活动 Profile 删除按钮禁用并提示先保存激活另一条记录。
- 删除不联网；成功后选择下一条可用记录。
- `legacy-current` 临时记录在正式保存前不能删除。

## 30. 定向撤销 Revision C 模型历史

Revision D 实施时先写 Profiles 红灯测试，再定向删除：

- `pkg/config/model_history.go`
- `pkg/config/model_history_test.go`
- `Config.ModelHistory`、`ModelHistoryEntry`
- ConfigManager 的 `RememberCurrentModel` 调用和历史深拷贝
- `ConnectionSettingsView.SavedModels`
- Settings Store 的 `modelHistory`、`savedModelsForCurrentConnection`、history helpers
- ModelSelect 的“已保存模型 / 目录候选”分组，恢复单一目录候选 prop
- `frontend/model_history_regression_test.mjs`
- Wails `ModelHistoryEntry` / `modelHistory` / `savedModels` 类型
- README 中模型历史说明

保留：

- 720px 弹窗、七标签完整显示、“输出”→“显示”。
- `comboID` 修复和录制成功后的桌面运行态同步。
- F9“隐身模式”、实体“隐藏/显示窗口”文案。
- 手机端和双语 README 快捷键修正。

禁止使用 `git reset`、`git restore` 或批量文件恢复。

## 31. 题库与固定隔离 Profile

题库产品代码和文件格式不修改。Revision D 候选复用固定隔离 Profile，并只报告：

- 题库目录：`SET / NOT SET`
- Profile 数量
- 每条 Key：`SET / NOT SET`
- 快捷键：`SET / NOT SET`

第一次启动：保存题库目录、至少两条连接 Profile 和一个自定义快捷键。正常关闭后，用相同 Profile 重启，确认全部保留。不读取、复制或修改正式用户 Profile，不把实际目录、地址或凭据写入日志和治理文档。

## 32. 快捷键修复保留规则

- `key-recorded` 使用 `comboID`，禁止恢复错误的 `vkCode`。
- 录制成功立即更新桌面运行态提示；“保存全部设置”后持久化并由 Config subscriber 更新手机 Snapshot。
- `toggle` 文案为“隐身模式”；`toggle_window` 文案为“隐藏/显示窗口”。
- 顶部栏、欢迎页、截图 Dock、空/错误状态继续读取动态 KeyName。
- Connection Profile 不包含快捷键；快捷键为应用全局设置。

## 33. 测试矩阵

### 33.1 Config / Manager

- 旧单连接无损加载、不自动写盘、首次保存创建 Profile。
- ID 随机且稳定；自动名称与编号；手动名称重复/空/超长拒绝。
- 20 条上限；同 URL 不同 Profile 共存。
- ExtraHeaders 和 Profile slice 深拷贝。
- 活动 ID 与根级镜像一致；非法活动 ID 安全失败。
- 保存失败的磁盘/内存/Provider/subscriber 原子性。
- 非活动删除成功；活动/legacy-current 删除拒绝。

### 33.2 App / Wails

- 列表摘要无 Key/Header/模板泄露。
- 显式选择返回正确完整草稿。
- 选择不联网、不激活。
- 保存新建/更新并激活；Keyless 与有 Key Profile 独立。
- 删除结果安全且不联网。

### 33.3 前端

- 列表、新建、自动名称预览、修改、删除确认。
- 选择只加载草稿；revision 使迟到目录/probe/save 失效。
- 保存成功更新活动标记；失败保持列表和草稿。
- 活动/legacy 删除禁用。
- 模型历史分组彻底移除，普通模型目录和模板模式手填继续工作。
- 快捷键正确修复继续通过。

### 33.4 完整门禁

- `go test ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `go mod verify`
- 全部前端回归与 Vite build
- Windows race 如仍为 `0xc0000139` 则记录 `BLOCKED`
- 固定 Wails v2.11.0 bindings、保护构建、`go version -m`
- 一次性空 Profile 启动烟测和固定 Profile 两次人工启动

## 34. 回滚与候选边界

- 当前包含错误模型历史的 Revision C 候选只归入 transient，不晋升。
- Revision D 失败时根稳定版保持不变。
- Profiles 字段可被旧版本忽略；根级活动连接镜像保障回退读取。
- 定向回滚 Profiles 时不删除加密配置中的记录，先停止写入并保留根镜像。
- 快捷键正确修复不随 Profiles 回滚。
- 未经授权不 commit、push、晋升、Tag 或 Release。
