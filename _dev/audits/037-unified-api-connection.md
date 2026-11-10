# Change 037 统一模型与 API 接入架构与提交边界审计

## 审计结论

Change 037 当前实现为 **Revision D 多条加密 Connection Profiles**：在 Revision B 的 OpenAI 兼容 Chat 与安全自定义 JSON 模板两模式基础上，最多保存 20 条完整连接记录；列表只返回安全摘要，完整草稿按 ID 单条读取，保存成功后才激活并镜像到根级连接字段。Revision C 的模型历史方向已按用户澄清定向撤销，其快捷键与布局正确修复继续保留。

普通自动门禁、Wails v2.11.0 保护构建、最终候选隔离启动及用户固定 Profile 双启动 GUI 验收均通过。Windows race 在进程启动阶段因 `0xc0000139` 保持环境 `BLOCKED`，真实外部 provider 保持 `NOT_RUN`。候选已按哈希晋升根稳定版，旧稳定版与验证候选均已归档，用户已明确授权按本审计边界执行双提交并推送 `origin/dev/qsolver-custom`。

## 实现后架构事实

- 连接字段为十二个：Mode、Key、Base URL、Protocol、Endpoint、Headers、Response Format、Model，以及 Revision B 新增的 RequestTemplate、ResponseJSONPointer、AuthHeader、AuthPrefix。
- 用户可见模式只有两种：`openai_chat` 与 `custom_json`；`openai_responses`、`anthropic_messages` 仅作为未保存的 legacy 协议继续可加载。
- CCR 无独立 adapter；旧配置规范化后进入 OpenAI Chat。
- `custom_json` 由 `CustomJSONAdapter` 实现：模板渲染为类型化 JSON，POST 完整 URL，按 Pointer 提取非空字符串，非流式，单 chunk 兼容 `GenerateContentStream`。
- 模板占位符只在完整 JSON string value 匹配时替换，保持 JSON 类型；不执行脚本、循环、表达式、文件或环境变量访问。
- 模板最多 64 KiB、64 层嵌套，且必须包含消息/文本/图片占位符；存在图片但模板无图片承载占位符时在 HTTP 前拒绝。
- 模型目录、文字 probe、图片 probe、正式解题和旧兼容入口共享统一连接规范化与 HTTP transport；`custom_json` 目录返回 `catalog_unsupported` 且零网络。
- transport 只重写已知推理请求和必要的模型目录请求；模板模式不经过 SDK 路径重写。
- API Key 可空；空值不发送默认鉴权 Header，Key 只注入可配置 `AuthHeader`。
- Extra Headers 经统一安全校验并随主配置加密保存。
- `ConfigManager.PatchValidated` 保证验证或保存失败不交换内存状态、不通知 subscriber。
- `SaveConnectionSettings` 在单个 `PatchValidated` 中 upsert/激活完整 Profile，返回生成 ID、正式名称和安全状态 DTO，不回显模板/Pointer/Header/Key 值。
- Solver 与 `app/solve.go` 已移除空 Key 前置拒绝，改为校验请求地址非空并在捕获前做无网络配置校验。
- API、模型与场景分页显示；三个连接操作按钮同页提供。
- 前端对目录、probe、保存、Profile 读取和会话切换分别实施迟到结果隔离，模板模式禁用目录刷新。
- Profiles 内嵌在现有加密 Config；正式 ID 为随机 16 字节十六进制，名称限制 64 UTF-8 字节，最多 20 条，同 URL 可按 ID 保存多条。
- `ActiveConnectionProfileID` 指向唯一活动记录；根级十二个连接字段镜像活动 Profile，应用正式记录时 `ConnectionMode=direct`。
- 旧单连接只在内存中显示 `legacy-current / 当前连接`，读取不写盘；首次成功保存才生成正式 ID 和自动名称。
- `GetSettings` 清空 Profiles 和全部根级连接编辑字段；列表只返回主机摘要与 `SET / NOT SET` 布尔，完整 Key 仅通过单条 `GetConnectionProfile(id)` 返回本机设置页。
- `UpdateFromJSON` 从后端旧快照深拷贝 Profiles，按非持久化 ID/name 元数据 upsert 选中项；前端缺少 `connectionProfiles` 不能清空后端记录。
- 选择、新建和删除不调用模型网络；选择只更新草稿，专用保存或全量保存成功后才更新活动记录和 Provider。
- Revision C 的模型历史类型、Store helper、分组 UI、bindings 和回归文件已删除；快捷键 `comboID`、动态提示及语义文案保留。

## 核心风险复核

| 风险 | 控制措施 | 证据 | 结论 |
|---|---|---|---|
| 旧 CCR 配置丢字段 | 读取兼容、首次保存规范化、manager/factory/service 测试 | `pkg/config/*_test.go`、`pkg/llm/*_test.go` | 通过 |
| Endpoint 误改模型目录 | 受控 RoundTripper 分离 default inference 与 models path | `connection_transport_test.go`、三协议 adapter 测试 | 通过 |
| Keyless 仍发空鉴权 | Key 为空时移除 SDK 默认 Header | 三协议 `httptest` | 通过 |
| 自定义 Header 注入或泄密 | token/CRLF/控制头/冲突/大小限制；安全 DTO | Config、App、transport 测试 | 通过 |
| API/Profile 保存夹带其他草稿 | 后端从已保存 Config 深拷贝 Profiles，仅 upsert 当前草稿并在事务成功后激活 | `manager_test.go`、`app/settings_test.go`、前端回归 | 通过 |
| 保存失败替换 Provider | save-before-swap 事务 | App/ConfigManager 失败测试 | 通过 |
| 迟到结果覆盖新草稿 | revision/version 校验 | `unified_api_settings_regression_test.mjs` | 通过 |
| 生成器升级依赖或 runtime | 固定 Wails v2.11.0，保护参数与逐文件哈希 | `04-validation.md` | 通过 |
| race 隐患未执行 | Windows runtime 启动错误 | `0xc0000139` 原始结果 | BLOCKED |
| 模板注入任意 URL/body | 受控 JSON 占位符白名单、类型化注入、无脚本执行 | `template_test.go`、`connection_test.go` | 通过 |
| 模板未承载图片却发送请求 | 渲染前解析模板树，缺少图片占位符时 HTTP 前拒绝 | `custom_json_test.go`（零网络） | 通过 |
| Pointer 提取非规范索引 | 数组索引逐字符 0-9，拒绝前导 0 与符号 | `template_test.go` | 通过 |
| Probe 把本地构建失败报为已发送 | 按 `ProviderError.Stage` 判定 `RequestSent` | `custom_json_test.go` | 通过 |
| Profiles 批量泄露 Key/模板 | GetSettings 清空连接字段；列表只返回安全摘要；单条按 ID读取 | Config/App 测试、前端回归 | 通过 |
| 前端全量保存清空后端 Profiles | Manager 从旧快照深拷贝并忽略 payload 中的 Profiles | `manager_test.go` | 通过 |
| 选择记录误切换 Provider或联网 | 前端只单条读取草稿；后端读取无 patch/network | App/Pinia 回归 | 通过 |
| legacy 显示名被持久化 | legacy name 始终发送空值，由后端自动命名 | `connection_profiles_regression_test.mjs` | 通过 |
| Profile 明文落盘 | 继续复用加密主配置并检查原始磁盘 sentinel | `manager_test.go` | 通过 |
| 固定 Profile 跨重启 | 用户确认两次 GUI 启动人工验收通过 | `04-validation.md` | 通过 |
| 真实服务差异 | 未调用外部 provider，不伪造调用结果 | `04-validation.md` | NOT_RUN |

## Secret 与生成噪音审计

- 未读取、输出或写入用户真实 API Key、Token、Header 值、认证 URL 或配置明文。
- 关键字扫描命中均为明确测试 fixture，例如 `test-only-key`、`custom-secret`，不是真实凭据。
- README、Change 文档、审计和提交信息不包含配置实值。
- `go.mod`、`go.sum` 未改变，未新增依赖。
- `frontend/wailsjs/runtime/**` 未改变。
- `frontend/dist/**` 为 ignored 构建产物，不进入提交。
- `build/bin/**` 为 ignored 构建产物，不进入提交，也不使用 `git add -f`。
- Git 暂存区为空。

## Commit 1：功能代码、测试、bindings 与 README

**提交目的：** 完成 Change 037 的统一模型与 API 接入：两模式（OpenAI 兼容 Chat 与安全自定义 JSON 模板）、Keyless 截图解题、模型/场景布局、多条加密 Connection Profiles，以及快捷键提示一致性修复。

**建议完整 commit message：**

```text
feat: add unified API connection profiles

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

**文件范围：**

```text
README.md
README_EN.md
app/connection_profiles.go
app/connection_profiles_test.go
app/llm.go
app/llm_test.go
app/mobile.go
app/mobile_test.go
app/settings.go
app/settings_test.go
app/solve.go
app/solve_preflight_test.go
frontend/ccr_settings_lifecycle_regression_test.mjs
frontend/ccr_ui_regression_test.mjs
frontend/model_connectivity_regression_test.mjs
frontend/model_layout_regression_test.mjs
frontend/connection_profiles_regression_test.mjs
frontend/shortcut_hint_sync_regression_test.mjs
frontend/settings_regression_test.mjs
frontend/unified_api_settings_regression_test.mjs
frontend/src/App.vue
frontend/src/components/ConnectionProfileManager.vue
frontend/src/components/ModelSelect.vue
frontend/src/components/ProviderSelect.vue
frontend/src/components/SettingsModal.vue
frontend/src/components/WelcomeView.vue
frontend/src/services/api.js
frontend/src/stores/settings.js
frontend/wailsjs/go/app/App.d.ts
frontend/wailsjs/go/app/App.js
frontend/wailsjs/go/models.ts
pkg/config/config.go
pkg/config/connection.go
pkg/config/connection_profiles.go
pkg/config/connection_profiles_test.go
pkg/config/connection_test.go
pkg/config/manager.go
pkg/config/manager_test.go
pkg/llm/anthropic.go
pkg/llm/anthropic_test.go
pkg/llm/connection_transport.go
pkg/llm/connection_transport_test.go
pkg/llm/custom_json.go
pkg/llm/custom_json_test.go
pkg/llm/errors.go
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
pkg/llm/service.go
pkg/llm/service_test.go
pkg/llm/template.go
pkg/llm/template_test.go
pkg/solution/solution.go
pkg/solution/solution_test.go
```

**明确排除：** `_dev/**`、`build/bin/**`、`frontend/dist/**`、用户配置、日志、临时探针、`go.mod`、`go.sum`、`frontend/wailsjs/runtime/**`。

## Commit 2：治理记录

**提交目的：** 归档 Change 037 的请求、现状、设计、TDD 计划、实现、验证、回滚、总结、架构审计和 INDEX 状态。

**建议完整 commit message：**

```text
chore: add change 037 governance records

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

**文件范围：**

```text
_dev/audits/037-unified-api-connection.md
_dev/changes/037-unified-api-connection/00-request.md
_dev/changes/037-unified-api-connection/01-current-state.md
_dev/changes/037-unified-api-connection/02-plan.md
_dev/changes/037-unified-api-connection/03-implementation.md
_dev/changes/037-unified-api-connection/04-validation.md
_dev/changes/037-unified-api-connection/05-rollback.md
_dev/changes/037-unified-api-connection/06-notes.md
_dev/changes/037-unified-api-connection/design.md
_dev/changes/037-unified-api-connection/implementation-plan.md
_dev/changes/037-unified-api-connection/revision-c-implementation-plan.md
_dev/changes/037-unified-api-connection/revision-d-implementation-plan.md
_dev/changes/INDEX.md
```

## 提交前门禁

1. 用户已确认固定 Profile 双启动 GUI 验收通过；真实外部 provider 保持 `NOT_RUN` 并明确披露，不改写为 PASS。
2. 若业务代码或测试再变化，重新运行相关测试、全量普通门禁并复核稳定版来源。
3. 每次 Git 操作前重新执行 `git status`、`git branch`、`git remote -v`、`git diff`；提交前额外执行 `git diff --cached`。
4. 只按上述显式路径分别暂存，禁止 `git add .` 或 `git add -A`。
5. Commit 1 后再暂存 Commit 2，避免代码与治理资产混合。
6. 用户已授权本次双提交和推送 `origin/dev/qsolver-custom`；不得修改 `main`、推送 `upstream`、强推、创建 Tag 或 Release。

## 当前结论

提交边界审计、必要自动验证、用户 GUI 验收、产物归档和稳定版晋升均已完成。根稳定版为 Change 037 候选哈希，旧稳定版具备经清单验证的回滚副本，`releases/**` 未变。Commit 1 已按功能白名单完成为 `0728d8c`；用户已明确授权独立治理提交并推送 `origin/dev/qsolver-custom`，本次不创建 Tag 或 Release。
