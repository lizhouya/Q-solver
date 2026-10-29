# Change 025 备注

日期：2026-10-25
状态：实现与普通自动验证完成；GUI 与真实 provider `NEEDS_USER`；race `BLOCKED`；未提交。

## 关键边界

- 用户选择“三协议同时支持”是架构和实施批准，不是 Git、发布或真实 API 调用授权。
- DeepSeek 在用户修正地址后可用，不作为 Change 025 对其他 provider 根因的证明。
- Claude 图片与 GPT/Codex 的真实 provider 失败原因仍未知；不能把中转丢图、模型映射、权限、额度、模型名、地址或 Responses-only 等猜测写成已确认事实。
- 文字实测、图片实测和本地静态模型目录是三个不同事实源，必须分开显示和表述。
- 一个 probe 只向当前显式协议发送一个请求；禁止自动 fallback、双发、改 endpoint、改模型或改参数重试。
- 自动测试只使用 loopback `httptest`、fake provider、合成七段数字 PNG 和测试 PDF。

## 配置与兼容

- 旧配置缺少 `apiProtocol` 时仅在运行时解释为 `openai_chat`，不会因加载而强制写回。
- 未知非空协议拒绝，不静默降级。
- OpenAI-compatible 网关中的 Claude 可显式选择 Chat 或 Responses；品牌和模型名称不决定协议。
- Anthropic 原生 SDK 的 BaseURL 使用 API root，例如 `https://api.anthropic.com`，不填写完整 `/v1/messages` endpoint。
- 模型列表只是建议；刷新失败、为空或不包含手填模型都不应清空手填值。
- 有 API Key 只表示已配置，未经主动 probe 时显示“已配置 / 未验证”。

## 安全与隐私

- 新 adapter 均显式禁用 SDK retry，HTTP client 禁止自动重定向。
- 用户可见错误只使用安全白名单字段和固定中文文案，不包含 SDK 原始错误、响应体、完整 URL、Authorization、API Key、Prompt、Base64 或私人路径。
- 解题日志不再记录完整 system prompt、完整模型答案或原始 SDK 错误，只记录安全事件和长度。
- Wails bindings 生成只解析 App 方法；该阶段未执行 `Startup`，未调用 `ConfigManager.Load`，因此没有读取真实配置内容。
- 自动验证未调用真实模型、未上传用户截图/简历/题库、未消费真实 API 额度。

## 依赖与工具链

- 固定工具链：`GOTOOLCHAIN=go1.25.13`。
- 主模块 `go` directive：`1.25.0`；无 `toolchain` directive。
- 唯一新增直接模块：`github.com/anthropics/anthropic-sdk-go/v2 v2.3.0`。
- `github.com/openai/openai-go` 保持 `v1.12.0`。
- Wails module 和生成 CLI 均保持 `v2.11.0`；PATH 中检测到的 v2.16.0 未用于生成或构建。
- npm 依赖未升级。

## 验证程序

```text
build/bin/Q-Solver-change025-multi-protocol-validation.exe
```

- 大小：`44017664 bytes`
- SHA-256：`589443f464f0e3f0c8adf5c6100beed81d5380d1c2ac98eb54b7f0a0bf6efa37`
- 构建未启用 clean bin，未覆盖正式 `Q-Solver.exe` 或任何旧验证程序。
- 新验证程序是本地构建产物，不纳入未来 Git 提交。

## 验证状态

- 普通 Go 测试、vet、build、module verify：`PASS`。
- 全部 Node 回归和 Vite build：`PASS`。
- `git diff --check`：`PASS`。
- Windows race：`BLOCKED`，启动测试二进制时统一返回 `0xc0000139`；不得记为 `PASS`。
- Windows GUI：`NEEDS_USER`。
- OpenAI Chat、OpenAI Responses、Anthropic Messages 和用户自定义网关真实验收：`NEEDS_USER`。

## 尚未执行

- 未暂存文件。
- 未 commit。
- 未 push。
- 未创建 Tag。
- 未创建 Release。
- 未发布。

## 2026-10-29 整合提交状态

- 用户批准 Change 024～031 一次性跨 Change 整合。
- 功能/测试提交：`2a8a506636496a1eb6c38662c6ef6eaccc1fe958`（Change 030 无业务源码，仅归档治理记录）。
- 本 Change 的治理档案纳入 `chore: archive changes 024-031 and define artifact lifecycle`。
- 本轮全量自动验证通过；原有 `NEEDS_USER` 与 race `BLOCKED` 项保持，不因提交自动关闭。
- 不创建 Tag、GitHub Release 或新 RC。
