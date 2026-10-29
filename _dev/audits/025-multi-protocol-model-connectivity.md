# Change 025 边界审计

日期：2026-10-25
状态：实施后审计与普通自动验证完成；GUI/真实 provider `NEEDS_USER`；race `BLOCKED`；未提交。

## 1. 基线与工作树保护

- 实施前分支为 `dev/qsolver-custom`，基线 HEAD 为 `43022e10cd8fb1b38b72b56d95bc39b8bb4d201d`。
- 工作树原有 Change 024 未提交改动，Change 025 未执行 reset、clean、checkout 覆盖或批量暂存。
- 正式程序、Change 017～024 验证程序和其余既有 EXE 均保留；构建前后关键受保护程序哈希不变。
- Change 025 新程序单独命名，未覆盖正式 `Q-Solver.exe` 或 Change 024 程序。
- 题库 `_记录`、截图、旧导出及其目录未被删除、覆盖或迁移。

## 2. 实施边界

Change 025 新增或修改的范围包括：

- `pkg/llm/protocol`：显式协议枚举和规范化。
- `pkg/llm/errors.go`、`http_client.go`、`factory.go`：安全错误、禁止重定向 HTTP 策略和统一工厂。
- `pkg/llm/openai.go`、`openai_responses.go`、`anthropic.go`：三个独立 wire protocol adapter。
- `pkg/llm/probe.go`：文字和合成图片实测。
- `pkg/config`：协议字段及候选配置原子保存。
- `pkg/resume`、`pkg/solution`、`pkg/questionbank`、`app`：必要的工厂接入、Wails DTO 和日志安全修复。
- 前端设置 store、UI、API service、provider/model 控件、回归测试和 Wails bindings。
- `go.mod/go.sum` 中受限的 Anthropic SDK 与 Go directive 变化。
- Change 025 治理文档、审计和 INDEX 状态。

以下重叠文件同时包含 Change 024 既有未提交工作，不得在未来提交前按整文件粗暴删除或回退：

- `app/questionbank.go`
- `pkg/questionbank/organize.go`
- 相关题库测试和前端题库文件
- Wails bindings
- `_dev/changes/INDEX.md`

## 3. 依赖审计

结果：`PASS`。

- 主模块 `go` directive：从 `1.24.1` 调整为 Anthropic SDK v2.3.0 要求的 `1.25.0`。
- 固定执行工具链：`go1.25.13`。
- 未新增 `toolchain` directive。
- 唯一新增直接模块：`github.com/anthropics/anthropic-sdk-go/v2 v2.3.0`。
- `github.com/openai/openai-go` 保持 `v1.12.0`。
- `github.com/wailsapp/wails/v2` 保持 `v2.11.0`。
- 其他既有直接 Go 依赖未做无关升级。
- npm 依赖版本未变化。
- 新增 indirect 可追溯到 Anthropic SDK 模块图；历史 `testify` indirect 保留，避免无关依赖漂移。
- `GOTOOLCHAIN=go1.25.13 go mod verify` 输出 `all modules verified`。

## 4. 协议与网络边界

结果：`PASS`。

- 协议只来自 `apiProtocol`，模型名、品牌、域名和 BaseURL 不参与路由。
- 空协议仅为旧配置兼容映射到 Chat；未知非空协议拒绝。
- OpenAI Chat、OpenAI Responses 与 Anthropic Messages 使用独立 adapter。
- 任意业务调用或 probe 失败后不 fallback、不双发、不换模型、不换 endpoint、不改参数重发。
- OpenAI 与 Anthropic SDK retry 均为 0。
- 共享 HTTP client 拒绝 301/302/307/308 自动重定向；测试断言跳转目标零请求。
- Anthropic adapter 使用 `WithoutEnvironmentDefaults` 后再设置显式草稿值，伪 `ANTHROPIC_*` 环境变量不能覆盖当前连接。
- typed SDK error 只用于安全分类，不向 UI 或日志传递 `Error()`、RawJSON、请求体或响应体。

## 5. Probe 与数据隔离

结果：`PASS`。

- 文字 probe 只携带随机 token 指令，非流式单请求，精确匹配才通过。
- 图片 probe 只携带合成七段数字 PNG 和不含答案的固定 prompt，通过流式多模态路径单请求。
- probe 不读取 screenshot buffer、简历、题库、角色 Prompt、历史消息或本地私人文件。
- 自动测试仅命中 loopback `httptest` 和 fake provider，没有真实 provider 网络调用。
- 图片错值与无法唯一解析分别映射为 `vision_mismatch` 和 `vision_inconclusive`；网络/认证等错误不伪装成“不支持图片”。

## 6. 配置与运行状态

结果：`PASS`。

- ConfigManager 候选配置先校验和落盘，成功后才替换内存并锁外通知；保存失败不改变当前配置、oldConfig 或活动 provider。
- 草稿列表和 probe 不替换保存配置、Solver provider 或用户手填模型。
- 题库 provider 工厂错误发生在封段、busy/cancel 和 worker 启动之前。
- `RawOnly` 保持零 provider 调用。
- 前端通过 session、connection 和 request revisions 拒绝字段变化、A→B→A、关闭/重开后的迟到结果。
- 打开、加载、保存、切品牌和切协议均无隐式模型调用。

## 7. 日志和错误 DTO

结果：`PASS`。

- `ProviderError` 的内部 cause 不序列化，`Error()` 只返回固定安全文案。
- Wails DTO 只复制 code、stage、message、retryable、HTTP status 和 protocol 白名单字段。
- 404 只提示端点、协议或模型待检查，不宣称唯一根因。
- 解题日志不再记录完整 system prompt、简历、模型正文、思考内容或 SDK 原始错误。
- sentinel 日志测试确认私有简历、答案和 SDK error 不泄漏，同时保留安全长度和固定事件元数据。

## 8. Bindings 与构建审计

结果：`PASS`。

- PATH 中的 Wails v2.16.0 未使用。
- 实际使用已核验的 Wails CLI v2.11.0 生成 bindings 和构建独立程序。
- 新 bindings 包含 `ConnectionDraft`、`ProviderErrorDTO`、`ModelListResult`、`ModelProbeRequest`、`ModelProbeResult`、`ListModelsForDraft` 和 `ProbeModelConnection`。
- 旧 `GetModels`、`TestConnection`、`GetModelsForConnection`、`TestConnectionForConnection` 仍存在。
- Wails runtime 文件无实际内容差异。
- 生成阶段只构造 App 并解析绑定；`Startup` 未执行，`ConfigManager.Load` 未调用，没有读取真实配置内容。
- 生成文件未包含真实 Key、Token 或私人配置。

独立程序：

```text
build/bin/Q-Solver-change025-multi-protocol-validation.exe
44017664 bytes
SHA-256 589443f464f0e3f0c8adf5c6100beed81d5380d1c2ac98eb54b7f0a0bf6efa37
```

新 EXE 是本地验证产物，不纳入未来 Git 提交。

## 9. 验证审计

- `GOTOOLCHAIN=go1.25.13 go test ./... -count=1`：`PASS`。
- `GOTOOLCHAIN=go1.25.13 go vet ./...`：`PASS`。
- `GOTOOLCHAIN=go1.25.13 go build ./...`：`PASS`。
- `GOTOOLCHAIN=go1.25.13 go mod verify`：`PASS`。
- 全部前端 Node 回归：`PASS`。
- Vite production build：`PASS`，仅有既有 chunk size 提示。
- `git diff --check`：`PASS`。
- Windows race：`BLOCKED`，目标测试程序启动时统一返回 `0xc0000139`，未获得 Go race 报告。
- Windows GUI：`NEEDS_USER`。
- 真实 OpenAI Chat、Responses、Anthropic Messages 与自定义网关：`NEEDS_USER`。

## 10. 隐私、输出和 Git 审计

- 未读取或输出用户真实 API Key、Token、Mobile Token、完整私人 BaseURL 或配置内容。
- 未调用真实模型，未上传用户截图、简历、题库、Prompt 或历史会话，未消费真实额度。
- 未修改 agent memory 或私人快捷键配置。
- 未停止占用端口的外部进程。
- 未暂存文件。
- 未 commit、push、创建 Tag、创建 Release 或发布。

## 11. 结论

Change 025 的实现、普通自动验证、依赖/隐私/边界审计、固定 Wails bindings 和独立验证构建均已完成。由于 Windows race 环境返回 `0xc0000139`，race 状态保持 `BLOCKED`；Windows GUI 和所有真实 provider 状态保持 `NEEDS_USER`。在用户完成人工验收并另行授权前，不得将本 Change 标记为最终完成、提交或发布。
