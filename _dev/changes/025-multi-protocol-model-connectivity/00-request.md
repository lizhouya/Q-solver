# Change 025 变更请求：多协议模型连接兼容

日期：2026-10-25
状态：计划已批准，事前治理进行中；未经授权不提交、不推送、不发布。

## 背景

用户修正请求地址后，DeepSeek 可以使用；Claude 的文字连接测试成功，但截图解题回复称无法看到图片；GPT/Codex 无法连接。只读核查确认当前生产调用全部走 OpenAI Chat Completions，连接测试只发送文字 `hi`，因此既不能证明图片已传输，也不能适配 Responses-only endpoint。界面中的图片图标和上下文长度来自静态模型目录，不是当前 endpoint 的实测能力。

现有证据不能确认中转是否丢图、模型映射是否错误、模型是否具有视觉能力，也不能确认 GPT/Codex 失败是协议、参数、模型名、权限、额度、地址或网络中的哪一种。本 Change 不伪造具体 provider 根因。

## 目标

- 保留 OpenAI-compatible Chat Completions，并新增 OpenAI Responses API 和 Anthropic 原生 Messages API。
- 使用显式 `apiProtocol` 选择协议，不按模型名、品牌、域名或 BaseURL 自动推断或 fallback。
- 主解题、简历 PDF、题库整理与设置诊断统一使用同一 provider 工厂和不可变配置快照。
- 分离文字连通性测试与图片实际传输/识别测试；每次点击只发一个请求。
- 支持手工模型 ID；模型列表只作为建议，失败不阻止保存和实测。
- 使用结构化、脱敏错误区分配置、鉴权、权限、协议/模型、限流、额度、请求过大、上游、网络、超时和响应终态。
- 修复配置先发布后落盘的问题：候选配置必须先成功持久化，再替换内存并通知订阅者。
- 清理本调用链中可能输出完整 Prompt、回答或 SDK 原始错误的日志。

## 验收标准

- 三个稳定协议值分别命中 Chat、Responses、Anthropic Messages，旧配置缺字段时保持 Chat。
- 任何真实业务或 probe 失败均不自动切协议、换参数重发或跟随跨主机重定向。
- 图片 probe 使用本地合成图和流式多模态路径；文字成功不显示为图片成功。
- OpenAI Chat 的图片、PDF、终态和流中断处理正确；Responses 和 Anthropic 使用各自原生内容块。
- SDK 自动 retry 关闭；每个业务动作或 probe 至多一个 provider 请求。
- 保存失败时磁盘、内存、subscriber 和 active provider 均保持旧状态。
- UI 显示“已配置 / 未验证”，静态能力明确标为目录参考；A→B→A 与关闭/重开后迟到响应无效。
- 所有自动测试只访问 loopback `httptest`，不读取真实密钥、配置或用户资料，不调用真实模型。
- 生成独立 `Q-Solver-change025-multi-protocol-validation.exe`，不覆盖正式或旧验证程序。

## 非目标

- 不确认当前 Claude、GPT 或 Codex 的真实 provider 根因。
- 不做协议自动探测、fallback、双发、多模型投票或业务级重试。
- 不启用 tools、MCP、thinking、prompt caching、beta、background 或服务端会话。
- 不全面重构设置系统、LLM message schema、题库 schema、KeepContext 或日志框架。
- 不调用真实模型，不自动上传用户截图、简历、题库或历史对话。
- 不覆盖、删除或迁移旧题库、`_记录`、截图、导出、正式 EXE 或旧验证程序。

## 依赖与版本边界

- 固定 `GOTOOLCHAIN=go1.25.13`、Wails CLI v2.11.0。
- `github.com/openai/openai-go` 保持 v1.12.0。
- 唯一计划新增的直接依赖为 `github.com/anthropics/anthropic-sdk-go/v2 v2.3.0`。
- 主模块 `go` directive 仅允许从 `1.24.1` 调整为 `1.25.0`，不得新增 `toolchain` directive。
- 不升级 Wails、前端依赖或其他无关直接依赖；所有 indirect 变化必须能追溯到 Anthropic SDK。

## 授权边界

本计划已获实施批准；依赖安装仅限上述 Anthropic SDK 与必要 Go directive 调整。commit、push、Tag、Release 和真实 API 调用未授权。禁止 `git add .`、`git add -A`、`git reset --hard`、`git clean`。任何超出本 Change 白名单的范围都必须停止并重新确认。
