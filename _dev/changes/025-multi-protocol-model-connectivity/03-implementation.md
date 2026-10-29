# Change 025 实施记录

日期：2026-10-25
状态：实现与普通自动验证完成；Windows GUI、真实 provider 待用户验收；race 被环境阻塞；未提交。

## 1. 治理与基线

- 在业务代码修改前建立 `00-request.md`、`01-current-state.md`、`02-plan.md`、实施/验证/回滚/备注模板和独立审计文件。
- 只读保留 Change 024 未提交工作树，没有清理、覆盖或迁移其文件。
- 记录正式程序、Change 017～024 验证程序、`go.mod`、`go.sum` 和前端依赖文件基线。
- 固定 `GOTOOLCHAIN=go1.25.13`、Wails CLI v2.11.0、OpenAI SDK v1.12.0 和 Anthropic SDK v2.3.0。

## 2. 协议与配置

新增 `pkg/llm/protocol`：

- `openai_chat`
- `openai_responses`
- `anthropic_messages`

协议为空时仅为兼容旧配置解释为 `openai_chat`；未知非空值拒绝。模型名、品牌、域名和 BaseURL 不参与协议推断。

`config.Config` 新增可选 `apiProtocol`。配置管理器改为候选快照事务：先校验并持久化，再替换内存，最后在锁外通知 subscriber。保存失败不会污染当前配置、旧配置快照或活动 provider。

## 3. Provider 安全基础设施

新增统一 provider 工厂、安全错误与 HTTP 策略：

- `NewProvider(config.Config)` 按显式协议创建 adapter。
- adapter 持有配置值快照，不引用共享可变配置。
- SDK 自动重试设为 0。
- HTTP client 禁止自动重定向，避免凭据跟随跳转到其他主机。
- `ProviderError` 只向 UI 暴露白名单字段和固定中文文案；内部 cause 不序列化。
- 错误分类覆盖配置、网络、TLS、认证、权限、404/协议不匹配、限流、额度、请求过大、空响应、无完整终态、拒答和图片 probe 结果等类别。
- 不从 SDK 原始错误字符串或响应 JSON 猜测能力或根因。

## 4. 三种协议 adapter

### 4.1 OpenAI-compatible Chat Completions

修正并验证：

- system/user/assistant 和多图顺序。
- 图片使用 `image_url`。
- PDF 使用 SDK file content part，不再伪装成图片。
- 非法 data URL/Base64 在发请求前失败。
- 正常和异常流终态、空响应、refusal 与不完整流的安全分类。
- 低成本文字 probe 使用 `max_completion_tokens=32`。
- 任意失败不更换字段、模型或协议重发。

### 4.2 OpenAI Responses API

新增独立 adapter：

- 只访问 Responses endpoint，不回落 Chat。
- 本地历史转为 Responses input，图片为 `input_image`，PDF 为 `input_file`。
- `store=false`，不使用 `previous_response_id`、background 或服务端会话。
- 处理 output text、公开 reasoning summary、completed、failed、incomplete 和 SSE error。
- completed 无 delta 时从 `Response.OutputText()` 恢复结果。
- 首个终态后立即结束，异常 EOF 分类为 `stream_interrupted`。
- 文字 probe 使用 `max_output_tokens=32`。

### 4.3 Anthropic 原生 Messages API

新增基于官方 `github.com/anthropics/anthropic-sdk-go/v2 v2.3.0` 的 adapter：

- 使用 `/v1/messages` 和原生 Models API。
- system 使用顶层 blocks；user/assistant、图片和 PDF 转为原生 content blocks。
- 使用 API root，SDK 自行追加 `/v1` endpoint。
- 调用 `WithoutEnvironmentDefaults`，再显式传入 Key、BaseURL、HTTP client 和零重试，避免环境变量污染连接草稿。
- 处理文字、思考增量、error event、message stop 和异常 EOF。
- 业务最大输出使用固定 8192；文字 probe 使用 `max_tokens=32`。
- 不启用 tools、MCP、prompt caching、beta、自动 fallback 或服务端会话。

依赖门禁结果：

- 唯一新增直接模块：`github.com/anthropics/anthropic-sdk-go/v2 v2.3.0`。
- 主模块 `go` directive 从 `1.24.1` 调整为 SDK 要求的 `1.25.0`。
- 无 `toolchain` directive。
- OpenAI、Wails 和前端既有直接依赖版本未升级。

## 5. 模型列表和文字/图片实测

新增内部 `Prober` 编排，不扩张现有 `Provider` 主接口。

### 5.1 文字实测

- 使用 `crypto/rand` 生成 `QSP-` 加 12 位大写十六进制 token。
- 非流式发送一次低 token 请求。
- trim 后必须与 token 精确相等；额外文字视为无效响应。
- 每次点击只触发当前显式协议的一次请求。

### 5.2 图片实测

- 用 Go 标准库生成 140×84 高对比七段数字 PNG。
- 随机答案为 `00`～`99`，prompt 不包含答案。
- 通过与主截图解题相同的流式多模态 adapter 路径发送一次请求。
- 只允许 trim、固定 `DIGITS:` 前缀和全角数字转半角。
- 正确数字为 `passed`；唯一两位错误数字为 `vision_mismatch`；不能唯一解析为 `vision_inconclusive`。
- 不读取用户截图、简历、题库、角色 Prompt 或历史。

### 5.3 Wails API

新增：

- `ListModelsForDraft(ConnectionDraft) ModelListResult`
- `ProbeModelConnection(ModelProbeRequest) ModelProbeResult`

DTO 只复制白名单错误字段。旧连接 API 保留原签名并固定为 Chat 兼容语义。

## 6. 生产入口统一

以下入口改为通过 `NewProvider` 使用同一个显式协议事实源：

- 活动 LLM Service。
- 主截图流式解题。
- Resume PDF 解析。
- 题库首次模型整理。
- 题库只重试失败项。
- 自动整理导出。
- 旧兼容整理入口。

`RawOnly` 继续使用 nil provider，保持零模型调用。题库工厂错误发生在封段、busy/cancel 状态变化和 worker 启动之前。Change 024 的全失败、部分失败、只重试失败和零题不落盘语义保持不变。

## 7. 前端设置生命周期

完成品牌、协议和模型能力分离：

- 设置草稿和保存配置均携带 `apiProtocol`。
- `ProviderSelect` 显示实际协议，支持 Chat Completions、Responses API、Anthropic Messages。
- 品牌只提供推荐 BaseURL 和协议；用户可随后独立修改协议。
- Anthropic 官方默认地址使用 `https://api.anthropic.com` API root。
- 同 origin 品牌预设保留草稿 Key；不同 origin 清空草稿 Key、model、建议列表和 probe 状态。
- `ModelSelect` 改为可输入 combobox；模型列表仅为建议，不覆盖手填值。
- 静态图片、文件和 context 标签标注“模型目录参考，非当前通道实测”。
- 文字与图片实测有独立按钮、loading、结果和“不确定”状态。
- 页面固定提示“每次点击会调用一次模型，可能产生少量费用”。
- 有 Key 时只显示“已配置 / 未验证”，不把静态目录或文字测试当作图片能力证明。
- 打开、加载、保存、切品牌和切协议均不自动调用模型。

竞态保护使用 session revision、connection revision 和三个 request version；四个连接字段变化、A→B→A、关闭和重新打开均会阻止旧结果覆盖当前草稿。

## 8. 敏感日志与 bindings

新增敏感日志回归测试，并完成定向清理：

- system prompt 只记录长度。
- 模型正文和思考内容只记录长度。
- LLM 请求失败只记录固定事件名，不记录 `%v` SDK 原文。
- 不在日志中保留简历、Prompt、完整答案、API Key、Base64、响应体或私人路径。

使用固定 Wails v2.11.0 重新生成：

- `frontend/wailsjs/go/app/App.js`
- `frontend/wailsjs/go/app/App.d.ts`
- `frontend/wailsjs/go/models.ts`

新旧 API 与新 DTO 均存在；Wails runtime 文件没有实际内容差异。

## 9. 自动验证与独立程序

自动结果：

- `go test ./... -count=1`：`PASS`
- `go vet ./...`：`PASS`
- `go build ./...`：`PASS`
- `go mod verify`：`PASS`
- 全部前端 Node 回归：`PASS`
- Vite build：`PASS`
- `git diff --check`：`PASS`
- Windows race：`BLOCKED`，所有目标启动时返回 `0xc0000139`

独立验证程序：

- 路径：`build/bin/Q-Solver-change025-multi-protocol-validation.exe`
- 大小：`44017664 bytes`
- SHA-256：`589443f464f0e3f0c8adf5c6100beed81d5380d1c2ac98eb54b7f0a0bf6efa37`

构建未清理 bin，未覆盖正式程序、Change 024 验证程序或其他旧程序。

## 10. 未执行事项

- Windows GUI 人工验收：`NEEDS_USER`。
- OpenAI-compatible Chat、Responses、Anthropic Messages 和自定义网关真实调用：`NEEDS_USER`。
- 自动阶段未读取真实密钥或配置内容，未调用真实模型，未上传用户资料，未消费真实额度。
- 未暂存、commit、push、创建 Tag、创建 Release 或发布。
