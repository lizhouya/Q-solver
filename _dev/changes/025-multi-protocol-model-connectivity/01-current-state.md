# Change 025 现状快照：多协议模型连接兼容

日期：2026-10-25

## 仓库与工作树基线

- 分支：`dev/qsolver-custom`，跟踪 `origin/dev/qsolver-custom`。
- HEAD：`43022e10cd8fb1b38b72b56d95bc39b8bb4d201d`。
- 事前工作树不是 clean：仅包含已审计、尚未提交的 Change 024 修改与新增文件；Change 025 不清理或覆盖这些改动。
- 下一可用编号：025。
- remote：`origin` 为用户 fork，`upstream` 为上游仓库；本 Change 未执行远程写操作。

## 依赖文件基线

| 文件 | 大小 | SHA-256 |
|---|---:|---|
| `go.mod` | 2016 | `6583908a30917f6864adc5cb7d36d36e56cb5ed6447889fa18e55e1a0784a298` |
| `go.sum` | 12724 | `d6b12513771dd06b45257192185c07519cedf89a0eab1af7967cc44112d121a4` |
| `frontend/package.json` | 708 | `0595e7f0957dd72c62d42dc1a310498d4ae2a3ab6a5e1efb198ac6a2f06b846a` |
| `frontend/package-lock.json` | 96050 | `827a54f1049d6d143ead365c00c6c0870d6f9eabde57f77ec0926c061f63d725` |

直接依赖中 `openai-go` 为 v1.12.0、Wails 为 v2.11.0；主模块 `go` directive 为 1.24.1。完整 `go list -m all` 已在实施前获取，后续用于差异审计。

## 已确认的模型调用链

1. `App.TriggerScreenshot` / `TriggerSend` 保存 data URL 截图并进入 `App.solveInternal`。
2. `pkg/solution.Solver.Solve` 通过 `llm.ImagePart` 组装 multipart user message。
3. 当前 `pkg/llm.OpenAIAdapter` 把图片转为 Chat Completions `image_url`。
4. 当前所有生产模型调用最终使用 Chat Completions；配置没有协议字段。
5. 当前文字连接测试只发送 `hi`，固定 `max_tokens=17`，不携带图片、不验证正文或流式终态。
6. `openai-go v1.12.0` 已包含 Responses API 类型，不需要升级 SDK。

因此可以确认应用进入 adapter 前仍保留截图，也可以确认当前连接测试不能证明截图通道可用；不能据此确认中转或上游最终是否收到图片。

## 现有架构缺口

- `OpenAIAdapter` 保存 `*config.Config`，client 与随后读取的 model 可能来自不同配置时刻。
- `pkg/llm/service.go` 的草稿模型列表直接访问 concrete `provider.client`。
- 生产代码共有 8 个直接 `NewOpenAIAdapter` 构造点，分布于 LLM Service、Resume 和 Question Bank。
- Resume 的 PDF 当前被 Chat adapter 当成图片 URL。
- Chat 非流式没有完整验证 empty/refusal/finish reason；流式只检查传输错误，没有强制合法终态。
- SDK 错误当前通过 `err.Error()` 截 JSON，且模型列表可能记录完整原始错误。
- App 与前端把多种失败压缩成同一个“检查地址、密钥和模型”文案。
- 设置页只能选模型列表；列表失败时可能无法使用实际可生成的网关模型。
- 静态模型目录中的图片图标和 context length 被展示为确定能力。
- `ConfigManager.UpdateFromJSON` / `Patch` 先替换内存并通知 subscriber，再保存磁盘；保存失败会产生运行态分裂。
- 前端只比较值，A→B→A 时旧 A 响应可能污染新 A。

## 可复用能力

- `pkg/llm.Message`、`ContentPart` 和 `Role` 已是 provider-neutral 抽象。
- `ParseBase64DataURL` 已能拆分 data URL，可扩展严格 MIME/Base64 校验。
- `Provider` 已被 Solver、Resume、Question Bank 和大量 fake provider 使用，应保持主接口稳定。
- `openai-go v1.12.0` 已提供 Chat、Responses、图片、file、流事件和 typed API error。
- 设置 store 已有 `backendSnapshot`、模型请求 version 和连接请求 version，可在此基础上加强 revision。
- Change 024 已有安全的 `OrganizeFailure` schema，可消费 typed provider error 的 Retryable 而不迁移题库历史。

## 不能确认的事项

- Claude 模型是否收到图片，或中转是否丢弃/转换了图片。
- 用户网关是否将模型名映射到不同上游模型。
- GPT/Codex endpoint 是 Responses-only、参数不兼容、模型不存在、权限不足、额度不足、地址错误还是网络问题。
- 静态模型目录与当前 endpoint 的能力是否一致。

## 受保护程序基线

实施前共记录 29 个 EXE 的路径、大小与 SHA-256。关键文件：

- `build/bin/Q-Solver.exe`：`5dcabb17933bfa9ab4dc15f289b4b8e50af99879c316b58831b09bbb4e39aa11`
- `build/bin/Q-Solver-change024-organize-failure-recovery-validation.exe`：`1953484b5fb42576f2db013691bdc6190f7a60ce7e1aabbdadad539b3f35d274`
- `build/bin/releases/v1.1.0-rc.3/q-solver-windows-amd64.exe`：`44fbb9381179b0d739003c53df1abe9c06847255447f87ff786fdbe8a711d339`

全部 29 项完整清单保留在本 Change 的实施审计过程中；Change 025 只能新增独立文件名，不得改写任一基线 EXE。
