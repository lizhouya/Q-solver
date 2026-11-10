# 现状快照 — 01

> **时点说明：** 本文件是 Change 037 实施前的只读现状快照（Revision A 提出时点），按规范保留“变更前事实”，不随 Revision B 实现回写。Revision B 的实现后架构事实见 [03-implementation.md](03-implementation.md) 与 [审计](../../audits/037-unified-api-connection.md)。

## 1. 涉及文件清单

| 文件 | 当前职责摘要 | 关键符号 / 行为 |
|---|---|---|
| `pkg/config/config.go` | 单份持久化主配置 | `APIKey`、`BaseURL`、`ConnectionMode`、`APIProtocol`、`Model`；无 Endpoint/Headers/response format |
| `pkg/config/connection.go` | 连接模式和协议规范化 | `direct` 支持三协议；`ccr` 强制 `openai_chat` 并规范化 URL |
| `pkg/config/manager.go` | 加密读取、全量更新和订阅通知 | 配置整体加密落盘；读取失败不覆盖；`UpdateFromJSON` 不是局部 merge |
| `pkg/llm/factory.go` | Provider 唯一工厂 | 按有效协议创建 Chat、Responses 或 Anthropic adapter；无 CCR adapter |
| `pkg/llm/service.go` | 模型目录、草稿 probe、旧兼容测试 | 目录/probe 用临时 Config；CCR probe 在 Service 层被拒绝 |
| `pkg/llm/probe.go` | 文字/图片真实能力验证 | 文字随机 token；图片随机数字 PNG；不读取用户业务数据 |
| `pkg/llm/openai.go` | OpenAI Chat adapter | SDK 固定 Chat endpoint；模型目录读取第一页 |
| `pkg/llm/openai_responses.go` | Responses adapter | 强类型 Responses 请求与流式解析 |
| `pkg/llm/anthropic.go` | Anthropic Messages adapter | 强类型 Messages 请求与流式解析 |
| `app/llm.go` | Wails 草稿 DTO 和模型/测试接口 | `ConnectionDraft` 仅五字段；新旧接口并存 |
| `app/settings.go` | 设置读取与全量保存 | `GetSettings`、`UpdateSettings` |
| `frontend/src/components/SettingsModal.vue` | 设置弹窗与标签编排 | API/模型分栏；模型页刷新和两类 probe；全局保存 |
| `frontend/src/components/ProviderSelect.vue` | Provider/协议/CCR 配置页 | `direct/ccr` 选择；CCR 专属文案、默认地址和跳转按钮 |
| `frontend/src/components/ModelSelect.vue` | 手填模型与候选下拉 | CCR 隐藏 provider/capability 推断；不覆盖手填模型 |
| `frontend/src/stores/settings.js` | 配置草稿、目录/probe 竞态控制、保存 | CCR 强制 Chat、自动目录同步、禁 probe；Key/BaseURL 必填 |
| `frontend/src/services/api.js` | Wails API 封装 | 结构化目录/probe 与旧兼容入口 |
| `frontend/ccr_*_test.mjs` | CCR UI/生命周期回归 | 固化 CCR 专属模式、单飞目录和零 probe |
| `frontend/model_connectivity_regression_test.mjs` | 多协议前端回归 | 草稿请求、迟到结果、跨 origin 清凭据 |

## 2. 当前数据与调用链

### 持久化

当前只有一份主连接配置：

```text
APIKey + BaseURL + ConnectionMode + APIProtocol + Model
```

主配置整体加密存储。没有连接 profile、连接 ID、自定义 endpoint、额外 headers、请求模板或模型目录 endpoint 配置。`AssistantModel` 只有模型名，不构成第二套连接。

### 生产调用

```text
ConfigManager
  → llm.Service 订阅配置
  → NormalizeConnection
  → NewProvider
  → OpenAI Chat / OpenAI Responses / Anthropic Messages adapter
  → Solver
```

CCR 没有独立 adapter。它通过 `connectionMode=ccr` 强制有效协议为 `openai_chat`，最终仍进入 OpenAI Chat adapter。

### 未保存草稿

```text
Settings Store 草稿
  → ListModelsForDraft / ProbeModelConnection
  → 临时 Config
  → NewProvider
  → 模型目录或 probe
```

目录和 probe 不修改 ConfigManager。前端以 session/revision/request version 防止迟到结果覆盖新草稿。

## 3. 当前界面行为

- “API”标签负责连接模式、Provider、协议、Base URL 和 API Key。
- “模型”标签负责模型输入、模型目录刷新、文字/图片实测和场景选择。
- CCR 有独立模式、专属提示和进入模型页按钮。
- 进入 CCR 模型页会自动同步一次目录；Direct 不自动同步。
- CCR 目录只读 `/models`，不做真实推理；Store 和 Service 都拒绝 CCR probe。
- 全局“保存”提交整个设置草稿并关闭弹窗。
- 没有 API 专用保存按钮。

## 4. 当前校验和安全边界

- 草稿刷新要求 Key、Base URL；probe 另要求 Model。
- CCR URL 只接受根地址或 `/v1`，拒绝 userinfo、query、fragment 和额外路径。
- Direct 协议必须是三个显式值之一。
- Key 和配置整体加密，不应进入日志和安全诊断。
- Provider/协议切换到不同 origin 时，前端清空草稿 Key 和 Model。
- 真实 probe 可能产生费用，前端有明确提示。

## 5. 当前测试能力

已有测试覆盖：

- 旧配置默认 Direct。
- 多协议工厂、目录和文字/图片 probe。
- CCR URL 规范化、固定 Chat、自动目录单飞和禁 probe。
- 手填模型不被目录结果覆盖。
- 连接字段变化使 probe/目录结果失效。
- 设置关闭/重开和 A→B→A 场景拒绝迟到结果。
- 跨 origin 清除草稿凭据。

当前没有覆盖 Endpoint、Headers、Key 可空、API 局部保存或旧 CCR 规范化保存的测试。

## 6. 已知风险与限制

1. 新旧 Wails 测试入口语义不完全一致，旧入口可能绕过 CCR probe 禁令。
2. Config、ConnectionDraft、前端草稿和 bindings 新增字段时容易遗漏并造成降级丢失。
3. SDK 固定 endpoint；自定义 Endpoint 必须通过受控连接层实现，不能改成任意请求模板。
4. 模型目录成功不等于所选模型能推理，更不等于支持图片。
5. 目录只读取第一页；本 Change 不扩展分页。
6. Windows `go test -race` 历史上可能在启动阶段返回 `0xc0000139`，如复现必须记为 BLOCKED。
7. 旧 Change 025/026 的 GUI/真实 provider 验证记录不能直接当作本 Change 当前证据。

## 7. 本轮只读审计状态

- Git 分支：`dev/qsolver-custom`。
- 工作区：开始审计时 clean。
- 配置实值：未读取、未打印、未修改。
- 真实 API：未调用。
- 业务源码、依赖、bindings：未修改。
