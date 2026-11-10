# 实现过程记录 03

## 实现日志

| 日期 | 步骤 | 改动内容 | 主要文件 | 结果 |
|---|---|---|---|---|
| 2026-11-05 | 只读架构审计 | 审计 API/模型前端、Config、Wails 草稿、Provider 工厂、三协议 adapter、目录与 probe | `01-current-state.md`、审计 | 未修改业务源码、配置、依赖或 bindings |
| 2026-11-05 | 需求与设计 | 确认结构化三协议、单配置、最小文字 probe、CCR 快捷预设、Key 可空、Endpoint/Headers/response format 与局部保存事务 | `00`、`02`、`design.md`、`implementation-plan.md` | 用户明确批准设计并要求“实施” |
| 2026-11-05 | Config TDD | 扩展统一连接字段、旧 CCR 迁移、协议默认 Endpoint、Keyless 与 Header/response format 校验 | `pkg/config/**` | 聚焦测试通过 |
| 2026-11-05 | transport TDD | 新增受控路径、Header、鉴权处理；只重写已知推理和模型目录请求 | `pkg/llm/connection_transport*.go`、`http_client*.go` | 聚焦测试通过 |
| 2026-11-05 | adapter TDD | 三协议接入统一 transport；OpenAI Chat 支持强类型 `json_object`；Anthropic Key 可空 | `pkg/llm/openai*.go`、`anthropic*.go`、相关测试 | 三协议本地假服务测试通过 |
| 2026-11-05 | Service 与 probe | 删除 CCR 禁 probe 和 Key 必填分支；统一目录、文字 probe、旧兼容入口；JSON probe 要求 token 唯一 | `pkg/llm/service*.go`、`probe*.go`、`factory_test.go` | 聚焦测试通过 |
| 2026-11-05 | 原子局部保存 | 新增 `ConfigManager.PatchValidated` 与 `App.SaveConnectionSettings`；验证/保存失败不交换内存 Config、Provider 或 subscriber 状态 | `pkg/config/manager*.go`、`app/llm.go`、`app/*_test.go` | 磁盘哈希、通知次数与安全 DTO 测试通过 |
| 2026-11-05 | 前端 Store TDD | 增加八字段连接草稿、Headers JSON 预校验、API 专用保存与 revision 竞态隔离；Keyless 可刷新/测试 | `frontend/src/stores/settings.js`、`services/api.js`、前端回归 | 全部前端回归通过 |
| 2026-11-05 | 统一页面 | 合并“模型与 API”页面；CCR 改为只填草稿的快捷预设；同页提供保存、刷新、测试，图片 probe 放入高级验证 | `SettingsModal.vue`、`ProviderSelect.vue`、`ModelSelect.vue` | UI 静态与状态回归通过 |
| 2026-11-05 | bindings 与说明 | 使用 Wails v2.11.0 生成 `SaveConnectionSettings` bindings，定向清除空白噪音；同步 README 中英文说明 | `frontend/wailsjs/go/**`、`README*.md` | `go.mod`、`go.sum` 和 runtime 未变 |
| 2026-11-06 | 自动门禁 | 运行全量 Go、前端、Vite、模块与差异检查 | 见 `04-validation.md` | 普通门禁 PASS；race 在启动阶段 BLOCKED |
| 2026-11-06 | 保护构建 | 归档上一轮 Change 036 候选；用 Wails v2.11.0 保护参数构建 Change 037 候选 | `build/bin/**`（Git ignored） | 候选与保护哈希通过；根稳定版未变 |
| 2026-11-06 | 独立启动 | 隔离 `APPDATA/LOCALAPPDATA` 启动候选 8 秒并正常关闭 | Change 037 候选 | WebView2、退出码和残留进程检查 PASS |
| 2026-11-06 | 提交边界审计 | 划分功能提交与治理提交，扫描 secret/生成噪音 | 本文、审计、INDEX | Revision A 边界完成；未暂存、未提交、未推送 |
| 2026-11-06 | Revision B 用户反馈 | 用户要求 API 收敛为兼容/模板两模式、默认只填地址和 Key、修复按钮/场景遮挡并将场景独立分页 | 用户参考图与对话决策 | Revision A 候选在 GUI 验收前被取代 |
| 2026-11-06 | Revision B 设计批准 | 确认折叠高级模板、模板模式手填模型、非流式 JSON、安全图片占位符、可配置鉴权头和旧协议草稿迁移 | `00-request.md`、`design.md`、`02-plan.md` | 用户明确回复“批准” |
| 2026-11-06 | Revision B TDD 计划 | 重写逐文件计划，增加 `custom_json` adapter、Keyless Solver、分页和布局测试 | `implementation-plan.md` | 修订代码尚未开始；不含 commit/push/真实 API 授权 |
| 2026-11-06 | Revision B 协议与配置 TDD | 新增 `custom_json` 后端协议；连接配置增加 `requestTemplate`、`responseJSONPointer`、`authHeader`、`authPrefix`；实现 URL/Header/Pointer/模板校验与原子 Patch | `pkg/llm/protocol/protocol.go`、`pkg/config/config.go`、`pkg/config/connection.go`、`pkg/config/manager.go`、相关测试 | 聚焦测试通过 |
| 2026-11-06 | Revision B 模板渲染器 | 新增类型化占位符渲染、三协议消息投影、图片 data URL/Base64 收集、RFC 6901 Pointer 提取与图片承载 guard | `pkg/llm/template.go`、`pkg/llm/template_test.go` | 聚焦测试通过 |
| 2026-11-06 | Revision B CustomJSONAdapter | 非流式 POST、可配置鉴权头、禁重定向、4 MiB 响应上限、目录 `catalog_unsupported`、单 chunk 流式兼容 | `pkg/llm/custom_json.go`、`pkg/llm/custom_json_test.go`、`pkg/llm/factory.go`、`pkg/llm/errors.go` | 聚焦测试通过 |
| 2026-11-06 | Revision B Keyless 解题 | Solver 移除空 Key 前置拒绝并加 nil provider 保护；App 三处改为校验请求地址并在捕获前做无网络配置校验 | `pkg/solution/solution.go`、`app/solve.go`、相关测试 | App `httptest` 端到端通过 |
| 2026-11-06 | Revision B Wails DTO 与保存 | 草稿扩至十二字段；`ConnectionSettingsView` 仅增加 `templateSet`，不回显模板/Pointer/Header 值；专用保存 patch 新字段 | `app/llm.go`、`app/llm_test.go`、`app/settings_test.go` | App/config 全包通过 |
| 2026-11-06 | Revision B 前端 Store | 两模式判定、模板草稿校验、旧协议迁移草稿、模板模式禁目录、专用保存快照扩字段、跨 origin 清理鉴权头、同步 watch | `frontend/src/stores/settings.js` | 前端回归通过 |
| 2026-11-06 | Revision B 设置页面 | API 页收敛为兼容/模板两模式，高级字段折叠；场景独立分页；保存独占一行、刷新与测试并排；窄弹窗单列；消除按钮与内容裁切 | `frontend/src/components/ProviderSelect.vue`、`SettingsModal.vue` | UI/布局回归通过 |
| 2026-11-06 | Revision B bindings 与文档 | Wails v2.11.0 生成新 DTO bindings（仅 `models.ts` 变化）；同步 README 中英文 Revision B 说明 | `frontend/wailsjs/go/models.ts`、`README*.md` | 依赖/runtime 未变 |
| 2026-11-09 | Revision B 边界修复 | 修复 URL 编码/尾斜线、模板深度、重复 Header、Pointer 非规范索引、图片未承载仍请求、Probe 本地失败误报已发送 | `pkg/config/connection.go`、`pkg/llm/template.go`、`pkg/llm/probe.go` | 聚焦测试转绿 |
| 2026-11-09 | Revision B 自动门禁 | 运行全量 Go、前端回归、Vite、vet、build、mod verify、race | 见 `04-validation.md` | 普通门禁 PASS；race 启动阶段 BLOCKED |
| 2026-11-09 | Revision B 保护构建 | 归档 Revision A 候选，冻结构建前保护清单，用 Wails v2.11.0 保护参数构建 Revision B 候选 | `build/bin/**`（Git ignored）、`build/bin/archive/manifests/change-037-revb-prebuild/` | 保护哈希全部 UNCHANGED；根稳定版未变 |
| 2026-11-09 | Revision B 独立启动 | 隔离 `APPDATA/LOCALAPPDATA` 启动候选 9 秒并正常关闭 | Revision B 候选 | WebView2、退出码、无残留 PASS；清理遗留 Vite 服务 |
| 2026-11-09 | GUI 验收布局修订 | “输出”改为“显示”；弹窗 520px→720px；删除标签横向滚动/自动滚动，窄屏自然换行 | `SettingsModal.vue`、`model_layout_regression_test.mjs` | TDD 红灯转绿；全部前端回归、Vite、Go 全量与新候选验证通过 |
| 2026-11-09 | Revision C 模型历史 TDD | 新增按规范化连接隔离的 12×20 有界模型历史，旧配置只读不迁移，快照深拷贝 | `pkg/config/model_history*.go`、Config/Manager | Config 全包通过 |
| 2026-11-09 | Revision C 保存事务 | API 局部保存和全量保存成功前在候选快照记录模型；失败时历史/Provider/subscriber 不变；安全 DTO 只返回当前连接模型 | `app/llm.go`、`app/settings_test.go` | App/Config 聚焦与全包通过 |
| 2026-11-09 | Revision C 前端模型复用 | Store 按连接筛选历史；ModelSelect 分组显示“已保存模型/目录候选”；模板模式历史可用但目录零请求 | `settings.js`、`ModelSelect.vue`、`SettingsModal.vue`、新回归 | 前端历史/布局/连接回归通过 |
| 2026-11-09 | Revision C 快捷键同步 | 修复 `comboID` 字段，录制成功同步桌面运行态；F9 标为隐身模式、实体隐藏标为隐藏/显示窗口；手机端与 README 同步 | `App.vue`、`WelcomeView.vue`、`app/mobile.go`、双语 README | 桌面/手机端回归通过 |
| 2026-11-09 | Revision C 全量验证与保护构建 | 全量 Go、18 份前端、Vite、Wails v2.11.0 候选与隔离启动 | 见 `04-validation.md` | 普通门禁 PASS；race BLOCKED；固定 Profile 人工验收待用户 |
| 2026-11-09 | Revision D 用户澄清与设计 | 用户明确要保存多条请求地址和 API Key，不是模型历史；确认加密内嵌 Profiles、自动命名可修改、列表增删改 | `00-request.md`、`design.md`、`02-plan.md` | Revision C 模型历史候选停止使用并不得晋升；快捷键修复保留；业务代码待新计划批准后定向修订 |
| 2026-11-09 | Revision D 规格自审 | 固定安全结果 DTO 与错误码；列表只返回主机摘要；`GetSettings` 清除 Profiles 和根级连接编辑字段；单条草稿按 ID 定向读取；明确 legacy 名称、自动命名长度和 `ConnectionMode=direct` | `design.md` | 占位符扫描通过；未修改业务源码，书面规格已获用户批准 |
| 2026-11-09 | Revision D TDD 计划 | 按 Profile 领域、Manager 原子事务、Wails 安全接口、Pinia 状态机、UI、bindings、自动门禁、保护候选和双启动验收拆分九个任务 | `revision-d-implementation-plan.md`、`02-plan.md` | 计划已完成自审并获用户批准 |
| 2026-11-10 | Revision D Config/Manager TDD | 新增加密内嵌 Profiles、随机 ID、自动/手动命名、20 条上限、深拷贝、活动根镜像；全量保存按元数据保留并 upsert 后端记录 | `pkg/config/connection_profiles*.go`、`config.go`、`connection.go`、`manager*.go` | 红灯转绿；`pkg/config` 全包 PASS；Revision C 模型历史后端文件定向删除 |
| 2026-11-10 | Revision D Wails 安全接口 TDD | 增加安全摘要、单条读取、原子保存/激活和非活动删除；`GetSettings` 清除连接编辑字段，保存错误固定脱敏 | `app/connection_profiles*.go`、`app/llm.go`、`app/settings*.go` | 聚焦与 `app` 全包 PASS；读取/选择不切换运行 Provider |
| 2026-11-10 | Revision D Pinia/UI TDD | 增加 Profile 初始化、选择、新建、保存、删除和竞态状态机；新增管理组件；恢复单一模型目录；修复 legacy 显示名不持久化 | `settings.js`、`api.js`、`ConnectionProfileManager.vue`、`SettingsModal.vue`、`ModelSelect.vue`、前端回归 | 18 份前端回归与 Vite build PASS；Revision C 模型历史前端文件定向删除 |
| 2026-11-10 | Revision D bindings/README | 固定 Wails v2.11.0 生成 Profile 方法和 DTO，清除纯空白 Tab；双语说明改为最多 20 条加密连接记录 | `frontend/wailsjs/go/**`、`README*.md` | 依赖/runtime/index 保护哈希不变；无模型历史生成类型 |
| 2026-11-10 | Revision D 自动门禁 | Go 全量、vet、build、mod verify、全部前端回归、Vite、diagnostics、diff 与安全边界检查 | 见 `04-validation.md` | 普通门禁 PASS；race 在测试启动前 `0xc0000139`，记为 BLOCKED |
| 2026-11-10 | Revision D 保护候选 | Revision C 与修复前候选归入 transient；固定 Wails v2.11.0 重建最终候选并隔离启动 | `build/bin/**`（Git ignored） | 最终候选启动/正常退出 PASS；根稳定版和 releases 未变；固定 Profile 双启动待用户验收 |
| 2026-11-10 | 用户验收与稳定版晋升 | 用户确认固定隔离 Profile 双启动 GUI 验收通过；归档旧稳定版和验证候选，以同卷原子替换晋升根稳定版 | `build/bin/**`（Git ignored）、治理记录 | 根哈希与候选一致；旧稳定版可回滚；`testing/current/` 清空；releases 7 个文件未变；首次 `$null` 备份参数被 PowerShell 5.1 拒绝且未替换，改用显式备份路径后成功 |
| 2026-11-10 | 功能提交 | 按 Commit Boundary Audit 显式白名单提交功能代码、测试、bindings 与 README | 63 个功能文件 | `0728d8c feat: add unified API connection profiles`；治理记录保持独立 |

## 实现结果

### 1. 统一连接契约

连接身份统一为以下八个字段：

- `ConnectionMode`
- `APIKey`
- `BaseURL`
- `APIProtocol`
- `EndpointPath`
- `ExtraHeaders`
- `ResponseFormat`
- `Model`

旧 `connectionMode=ccr` 在规范化时迁移为 `direct + openai_chat`，保留 URL、Key 和模型；加载过程不主动写盘。默认 Endpoint 分别为：

- OpenAI Chat：`/chat/completions`
- OpenAI Responses：`/responses`
- Anthropic Messages：`/v1/messages`

### 2. 安全连接层

- Endpoint 必须是以 `/` 开头的相对路径，拒绝查询串、片段和换行。
- Extra Headers 限制名称、值、数量和总大小，拒绝传输控制 Header。
- API Key 非空时禁止 Headers 重复设置 `Authorization` 或 `x-api-key`。
- API Key 为空时删除 SDK 默认空鉴权头，并允许用户显式提供自定义鉴权 Header。
- 推理 Endpoint 与模型目录分开处理；自定义推理路径不会改写标准模型目录。
- 重定向保持禁用，避免凭据跨 origin 泄露。

### 3. 保存事务与安全 DTO

`ConfigManager.PatchValidated` 按“克隆 → patch → normalize → Config 校验 → Provider 工厂校验 → 保存 → 交换状态 → 通知”执行。任一验证或保存失败均不会改变磁盘、内存 Config、`oldConfig`、活动 Provider 或 subscriber 状态。

`SaveConnectionSettings` 只 patch 八个连接字段，返回值仅暴露非敏感连接字段以及 `APIKeySet`、`ExtraHeadersSet`，不回显 Key 或 Header 值。

### 4. 前端统一页面与竞态

- API、模型、场景和三个主操作集中在“模型与 API”页面。
- CCR 仅为 OpenAI Chat 快捷预设，选择时不联网、不测试、不保存。
- `extraHeadersText` 使用稳定两空格 JSON；非法 JSON、非对象、数组或非字符串值在调用 Wails bridge 前被拒绝。
- 连接、目录、probe 和保存分别使用 revision/version；迟到响应不能覆盖当前草稿。
- 跨 origin 变化清除 Key、模型及鉴权 Header，保留其他非鉴权 Header。
- API 专用保存不关闭设置页，也不提交其他标签草稿；底部全局保存仍保留。

## 与计划的偏差及修正

1. OpenAI SDK 在空 Endpoint 下曾被 transport 重写到 API root。修正为 transport 自行补协议默认 Endpoint，并增加空 Endpoint 回归测试。
2. `ConfigManager.UpdateFromJSON` 从默认 Config 反序列化旧 JSON 时，缺失 `endpointPath` 曾继承 Chat 默认值。修正为检测字段是否实际存在：缺失时按协议补默认，显式路径始终保留。
3. OpenAI Chat 的 `json_object` 初版未进入所有调用路径。新增统一 `newChatParams`，流式、非流式、测试和文字 probe 共同使用。
4. JSON probe 初版只接受纯文本 token。修正为同时接受顶层 JSON 对象中恰好一个等于随机 token 的字符串值；数组、非法 JSON、尾随文本、零次或多次匹配均失败。
5. Wails v2.11.0 生成 `models.ts` 时产生空白 Tab 噪音。只定向删除空白噪音，未修改 runtime、依赖或无关 bindings。
6. Windows race runtime 继续在测试进程启动阶段返回 `0xc0000139`。按事前规则记录为 `BLOCKED`，没有改写或禁用 race 检查。

## 遗留项（Revision A 历史）

- Windows GUI 与真实接口验收当时仍为 `NEEDS_USER`，见 `04-validation.md`。
- Revision A 候选当时保留在 `build/bin/testing/current/`，未晋升根稳定版。
- Revision A 阶段没有 commit、push、Tag 或 Release 授权；当时未执行 Git 写操作。
---

# Revision B 实现结果（2026-11-09）

## 1. 两模式连接契约

设置页只暴露两种模式，后端仍保留 legacy 协议用于未保存配置：

| 模式 | 后端协议 | 目录刷新 | 流式 | 模型来源 |
|---|---|---|---|---|
| OpenAI 兼容 Chat Completions | `openai_chat` | 支持 | 支持 | 目录或手填 |
| 自定义请求模板 | `custom_json` | 禁用 | 非流式 | 手填 |

新增持久字段：`RequestTemplate`、`ResponseJSONPointer`、`AuthHeader`、`AuthPrefix`。旧 `openai_responses`、`anthropic_messages` 在未保存时继续走原 adapter；打开设置页只生成内存迁移草稿。

## 2. 模板渲染与提取

- 占位符仅在完整 JSON string value 匹配时替换，保持 JSON 类型：`{{model}}`、`{{system_prompt}}`、`{{user_text}}`、`{{openai_messages}}`、`{{responses_input}}`、`{{anthropic_messages}}`、`{{anthropic_system}}`、`{{images_data_urls}}`、`{{images_base64}}`。
- 模板必须为顶层 JSON object，最多 64 KiB、64 层嵌套，且至少包含一个消息/文本/图片占位符。
- 拒绝未知、嵌入式占位符与 object key 占位符；不执行脚本、循环、表达式、文件或环境变量访问。
- 存在图片但模板缺少图片承载占位符时，在 HTTP 请求前以 `ErrorInvalidConfig` 拒绝。
- 响应经 RFC 6901 Pointer 提取，拒绝非法 `~` 转义与非规范数组索引（`+0`、`-0`、`00`），结果必须为非空字符串；响应体上限 4 MiB。

## 3. 鉴权与传输安全

- Key 只注入 `AuthHeader`（默认 `Authorization` + `Bearer `），可配置为 `x-api-key` 等；显式空前缀被保留。
- Key 为空时不发送默认鉴权头，支持无鉴权本地服务。
- Extra Headers 拒绝重复名称（含大小写）、控制字节、传输控制头及与 API Key 的鉴权冲突。
- 自定义请求地址仅允许 HTTP/HTTPS，禁止内嵌凭据、query 与 fragment；HTTP 客户端禁用重定向，超时 120 秒。
- 保存结果与错误不回显模板、Key 或 Header 值。

## 4. Keyless 解题链路

`pkg/solution` 移除空 API Key 前置拒绝并增加 nil provider 安全事件；`app/solve.go` 的 `TriggerScreenshot`、`triggerSend`、`solveInternal` 改为校验请求地址非空，并在截图捕获前调用 `llm.NewProvider` 做无网络配置校验。App 端到端测试证明无 Key 的 `custom_json` 配置可实际到达本地模板服务且不发送 `Authorization`。

## 5. 前端两模式与分页

- API 页默认只显示请求地址、API Key 与模型；高级区折叠展示兼容连接字段或模板配置。
- 模板模式禁用目录刷新并提示手填模型；测试与保存不受影响。
- 场景移入独立标签页；标签改为可键盘操作的 button 并自动滚动到可见位置。
- 保存按钮独占一行，刷新与测试并排；弹窗窄至约 420px 时改为单列，使用容器查询而非仅视口查询。
- 覆盖 `min-width: 0`、`overflow-wrap`、`flex-shrink` 等，消除按钮与场景文字裁切。

## 6. 与计划的偏差及修正（Revision B）

1. 自定义 URL 初版复用 `NormalizeBaseURL`，丢失路径编码与末尾 `/`；改为独立校验并保留 `EscapedPath`。
2. 模板初版无嵌套上限；增加 64 层深度限制。
3. Extra Headers 初版仅检查规范化后字符串相等；补充大小写重复与控制字节拒绝。
4. JSON Pointer 初版接受 `+0`/`-0`/`00`；改为数组索引逐字符 0-9。
5. 图片模板未承载占位符时仍会发起请求；改为渲染前解析模板树并在 HTTP 前拒绝。
6. Probe 初版把所有错误硬编码为 `RequestSent=true`；改为按规范化 `ProviderError.Stage` 判定，本地构建/校验阶段为 false。
7. 追加测试时曾两次用 replace 覆盖旧测试函数（Manager、Connection）；均已立即恢复，未丢失既有覆盖。
8. 两个并行子代理因限流（HTTP 429）终止，仅产出未完成草稿；由主线程接手完成，未宣称代理完成。

## 7. 遗留项（Revision B）

- 旧 Responses/Anthropic 迁移使用固定 Pointer，只提取单个文本块，不等价于旧 SDK 聚合；须人工核对。
- Keyless 仅覆盖模型测试与截图解题；题库模型整理入口仍保留 Key 前置要求。
- 真实 WebView2 GUI 布局与真实 provider 验收为 `NEEDS_USER`；浏览器自动化 `BLOCKED`。
- 未 commit、未 push、未晋升稳定版。

## 8. GUI 验收布局修订（2026-11-09）

人工验收截图证明七个设置标签在 520px 弹窗中依赖横向滑动，和“直接完整显示”的产品目标不一致。按批准短设计实施：

1. 先修改 `model_layout_regression_test.mjs`，要求 720px 弹窗、标签 `flex-wrap: wrap` + `overflow: visible`、无 `overflow-x: auto`、无 `scrollIntoView`/`tabsElement`/`nextTick`，并要求展示“显示”。测试按预期红灯。
2. 修改 `SettingsModal.vue`：展示文案“输出”→“显示”，保留 `activeTab='output'`；弹窗宽度 720px、`max-width: calc(100vw - 32px)`；标签容器取消横向滚动并允许窄屏换行；删除滚动专用逻辑。
3. 布局聚焦测试、16 份前端回归、Vite、`go test ./...`、`go build ./...` 全部通过。
4. 旧 Revision B 候选以 `Q-Solver-candidate-change037-revision-b-before-tab-layout.exe` 归档；新候选使用 Wails v2.11.0 保护参数重建，根稳定版不变。

## 9. Revision C 实现结果（2026-11-09）

- `ModelHistoryEntry` 只保存协议、规范化地址、Endpoint 与模型名，不保存 Key/Header/模板/响应；每连接 20 个模型、全局 12 个连接。
- `NormalizedConnectionSettings` 只清洗历史，不因读取旧配置而自动记录模型；`RememberCurrentModel` 仅在局部/全量保存候选中调用。
- ConfigManager 和 DTO 对历史切片深拷贝；失败事务不交换内存、磁盘或 subscriber 状态。
- API 专用返回只暴露当前连接的 `savedModels`，不暴露其他连接地址或模型。
- 前端按当前草稿身份筛选历史，Key 轮换不拆分；模板模式可选历史但刷新目录仍在 bridge 前停止。
- 模型下拉分为“已保存模型”和“目录候选”，精确去重并保留自由输入。
- 快捷键录制改用 `comboID`；`shortcut-saved` 后立即同步桌面运行态提示，保存全部设置后由现有 Config subscriber 同步手机端与持久配置。
- `toggle` 统一为“隐身模式”，`toggle_window` 统一为“隐藏/显示窗口”；顶部、欢迎页、截图 Dock、空/错误状态与手机端均使用动态 KeyName。
- 题库产品代码未修改；固定隔离 Profile 的跨重启持久化仍待用户人工验证。

## 10. Revision D 最终结果（2026-11-10）

- 在现有加密主配置中实现最多 20 条完整 Connection Profiles；安全列表只返回摘要，完整草稿按 ID 单条读取，保存事务成功后才激活并镜像根级连接字段。
- Revision C 模型历史后端、前端、Wails 类型和专用回归已定向删除；720px 无滑动标签栏、“显示”文案、`comboID` 修复、桌面/手机动态提示和 GhostMode/WindowVisibility 文案继续保留。
- 普通自动门禁、Wails v2.11.0 保护构建、空隔离启动均通过；Windows race 在测试进程启动前因 `0xc0000139` 保持环境 `BLOCKED`，真实外部 provider 保持 `NOT_RUN`。
- 用户于 2026-11-10 明确确认固定隔离 Profile 双启动 GUI 验收通过，并授权版本晋升、双提交和推送。
- 最终候选 SHA-256 `8B19B242F7917D4BC0267E45BB6248077E2BA5BD2D3CE428AC9E1B836C8B4994` 已晋升为 `build/bin/Q-Solver.exe`；旧稳定版归档到 `archive/superseded/change-037/`，通过候选归档到 `archive/validation/change-037/`。
- 晋升后根目录唯一 EXE，`testing/current/` 为空，`releases/**` 7 个文件哈希保持不变。
