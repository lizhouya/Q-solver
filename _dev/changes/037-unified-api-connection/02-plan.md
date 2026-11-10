# 实施方案 — 02

## 当前状态

完整权威设计见 [design.md](design.md)。Revision B 两模式连接、分页布局和 720px 无滑动标签栏已实现并完成自动/人工验证。Revision C 的快捷键字段与全显示端同步修复保留；其“模型历史”方向被用户澄清取代。Revision D 当前目标是多条加密连接 Profiles。

当前状态：**Revision D 已按批准计划实现 / 普通自动门禁与用户固定隔离 Profile 双启动 GUI 验收通过 / race 环境 BLOCKED 已披露 / 稳定版已晋升 / 双提交与推送已获授权**。

## 1. Revision B 目标

1. 前端 API 模式只展示 OpenAI 兼容 Chat Completions 和自定义请求模板。
2. 默认表单只显示请求地址、API Key、模型；高级连接/模板字段折叠。
3. 新增受控 `custom_json` Provider：安全 JSON 占位符、可配置鉴权 Header、非流式 JSON 和 RFC 6901 Pointer。
4. 旧 Responses/Anthropic 配置在设置页生成模板草稿，保存后才迁移。
5. 修复 Solver 对空 API Key 的旧前置拒绝，使无鉴权本地服务可真正截图解题。
6. 把模型与场景拆成不同标签页，并修复按钮、滚动和 Footer 裁切。

## 2. 架构方案

- 现有 `openai_chat` adapter 保持默认兼容模式和流式行为。
- 现有 `openai_responses`、`anthropic_messages` adapter 保留用于未迁移旧配置，不再作为新 UI 选项。
- 新增 `custom_json` protocol 与 `CustomJSONAdapter`。
- 模板先解析 JSON，再按白名单替换完整值占位符；不执行脚本或字符串模板。
- `GenerateContentStream` 在完整 JSON 响应成功后发送一个 content chunk，保持 Solver/事件接口不变。
- 模板模式 `GetModels` 零网络返回不支持；前端在 bridge 前禁用刷新。
- API 专用保存继续使用 `PatchValidated` 原子事务，只扩大连接字段白名单。

## 3. 阶段计划

| 阶段 | 操作 | 主要范围 | 完成标志 |
|---|---|---|---|
| 1 | TDD 扩展 protocol、Config、模板/Pointer/Auth 校验 | `pkg/llm/protocol/**`、`pkg/config/**` | 配置红灯转绿，旧字段兼容 |
| 2 | TDD 实现模板渲染和消息投影 | 新 `pkg/llm/template*.go` | 占位符、图片、历史和拒绝矩阵通过 |
| 3 | TDD 实现 `CustomJSONAdapter` | 新 `pkg/llm/custom_json*.go`、factory/service/probe | POST、鉴权、响应提取、零目录请求通过 |
| 4 | 修复 Keyless Solver 实际解题 | `pkg/solution/**` | 无 Key `httptest` 截图解题通过 |
| 5 | 扩展 Wails 草稿/安全 DTO/局部保存 | `app/llm.go`、测试 | 模板字段原子保存且不泄露 |
| 6 | 前端 Store 两模式、旧配置草稿迁移和竞态 | `settings.js`、回归测试 | payload、迁移、禁刷新和迟到隔离通过 |
| 7 | 重构设置 UI、独立场景页和响应式布局 | 三个组件、UI 回归 | 520px/窄屏无裁切，Footer/滚动正确 |
| 8 | 固定 Wails v2.11.0 bindings 与 README | bindings、README | 生成差异和说明审计通过 |
| 9 | 全量验证与 Revision B 候选 | Go/Node/Vite/Wails | 普通门禁和独立启动通过 |
| 10 | GUI/真实接口验收与提交边界 | Change/审计/INDEX | NEEDS_USER 项如实处理，待授权提交 |

逐文件、逐测试的操作步骤写入 [implementation-plan.md](implementation-plan.md)。

## 4. 允许业务文件

以 [design.md §12](design.md) 为权威。新增文件限于：

- `pkg/llm/template.go`
- `pkg/llm/template_test.go`
- `pkg/llm/custom_json.go`
- `pkg/llm/custom_json_test.go`

如实现证明两个文件可清晰合并，可减少文件；不得新增依赖或扩展到题库、Mobile、快捷键和截图采集实现。

## 5. 测试与验证

### 聚焦

- Config/template/auth/Pointer 校验。
- 消息、图片与 legacy 模板迁移。
- Custom JSON HTTP、错误、超限、取消和零模型目录请求。
- Keyless Solver 实际请求。
- Wails 局部保存和安全 DTO。
- 前端两模式、迁移、竞态、分页和响应式布局。

### 全量

```bash
go test ./... -count=1
go vet ./...
go build ./...
go mod verify
for test_file in frontend/*_test.mjs; do node "$test_file" || exit 1; done
npm --prefix frontend run build
git diff --check
```

相关 race：

```bash
go test -race ./pkg/config ./pkg/llm ./pkg/solution ./app -count=1
```

若继续在启动阶段返回 `0xc0000139`，记为 `BLOCKED`，不冒充 PASS。

Wails 候选必须按 `_dev/build-artifact-policy.md`：归档 Revision A 候选、冻结稳定版/Release/依赖/bindings/dist/index 哈希、固定 v2.11.0、独立启动、正常退出和残留进程检查。

## 6. 安全边界

- 自动测试仅使用 `httptest.Server`。
- 不读取、打印或提交真实 Key、Token、Header 值或认证 URL。
- API Key 不可作为模板 body 或 URL 占位符。
- 模板不执行代码、表达式、循环、文件或环境变量访问。
- HTTP 禁止重定向；响应 body 设上限；错误不回显响应正文。
- 真实 probe 和解题只由用户后续主动配置和触发。

## 7. 回滚

- Revision B 未通过时保留根稳定版和测试前稳定副本。
- Revision A 候选按原哈希归入 `archive/transient`，不冒充正式验证通过。
- 定向回滚 `custom_json` protocol、adapter、Config 字段和前端两模式；旧三个 adapter 仍在，可恢复 Revision A UI。
- 禁止 `git reset --hard`，已提交后只使用经授权的 `git revert`。

## 8. 授权边界

用户已批准 Revision B 设计与业务实施。当前没有真实 API 调用、用户配置修改、commit、push、稳定版晋升、Tag 或 Release 授权。

## 2026-11-08 实施补充：Keyless 入口一致性

实际审计发现 app/solve.go 的 TriggerScreenshot、triggerSend、solveInternal 仍以空 APIKey 拒绝；仅修 Solver 无法满足已批准的无鉴权实际解题目标。范围增加 `app/solve.go` 和 `app/solve_preflight_test.go`，仅把 Key 必填判断改为地址/模型连接就绪判断及无效 Provider 安全拒绝，并将旧 model 设置路由改为 api；不改截图采集、任务与题库行为。

## 2026-11-09 Revision C（历史，模型历史方向已被 Revision D 取代）

以下仅记录已经执行但尚未提交的历史计划。模型历史步骤不得继续实施或作为最终验收依据；快捷键 `comboID` 与提示同步步骤由 Revision D 保留。详细历史步骤见 [revision-c-implementation-plan.md](revision-c-implementation-plan.md)。当时按以下顺序实施：

1. 新增不含凭据的 `ModelHistoryEntry`，在 API 局部保存和全量保存事务成功前记录当前模型；旧配置读取不自动迁移。
2. 前端按规范化连接筛选历史，与目录候选分组、去重显示；模板模式只使用历史并继续禁止目录请求。
3. 修复 `key-recorded` 的 `comboID` 字段，并在录制成功后同步桌面运行态提示；保存全部设置后由 Config subscriber 同步手机端并持久化。
4. 将 F9 `toggle` 统一标为“隐身模式”，将 `toggle_window` 标为“隐藏/显示窗口”，同步双语 README。
5. 不修改题库产品代码；候选验证复用固定隔离 Profile，证明题库目录、模型历史和快捷键跨重启保留。

Revision C 新增允许文件：

- `pkg/config/model_history.go`
- `pkg/config/model_history_test.go`
- `frontend/model_history_regression_test.mjs`
- `frontend/shortcut_hint_sync_regression_test.mjs`
- 既有 Config/Manager、App LLM/Mobile、Settings Store、ModelSelect、SettingsModal、App/Welcome 组件及对应测试和 README。

明确禁止修改 `app/questionbank.go`、`frontend/src/stores/questionBank.js`、依赖、Wails runtime、正式用户配置或 `build/bin/releases/**`。当前未授权 commit、push、稳定版晋升、Tag 或 Release。

## 2026-11-09 Revision D：多条加密连接 Profiles

Revision D 详细设计见 [design.md §24～34](design.md)。本轮将：

1. 定向撤销未提交的 Revision C 模型历史代码、测试、前端分组和 Wails 类型，保留快捷键正确修复。
2. 在现有加密 Config 中增加最多 20 条完整 Connection Profiles 和活动 ID，根级连接字段镜像活动记录。
3. 兼容旧单连接：加载不写盘，设置页显示 `legacy-current`，首次成功保存后生成正式随机 ID。
4. 增加安全列表、显式加载、原子保存/激活和非活动删除接口。
5. 前端提供自动命名可修改的记录列表、新建、修改和应用内删除确认；选择记录无网络或运行态副作用。
6. 继续保留 `comboID`、桌面/手机提示同步和 GhostMode/WindowVisibility 文案修复。
7. 不修改题库产品代码；固定隔离 Profile 仅用于跨重启验证。

Revision C 原 [revision-c-implementation-plan.md](revision-c-implementation-plan.md) 仅保留为历史实施记录，不再作为后续执行计划。Revision D 逐文件 TDD 计划见 [revision-d-implementation-plan.md](revision-d-implementation-plan.md)；该计划通过用户复核后才能定向修改业务源码。
