# Change 028 实现记录

日期：2026-10-26
状态：实现与普通自动验证完成；GUI、真实模型待验证；未暂存、未提交、未关闭。

## 1. 实施顺序

1. 重新读取规则、README、INDEX、Change 024–027 与源码，核实 HEAD 和已有工作树。
2. 记录 1286 个已有文件及程序的 SHA-256，备份本次可能编辑文件的任务入口版本；创建 00/01/02 后开始测试与代码。
3. 改动前相关 Go 包测试通过。首次使用工具链自动选择器并禁用 checksum 数据库时被工具链验证阻止，随后直接运行已缓存的 Go 1.25.13，未下载安装。
4. 新增错误摘要、Responses 分类、Solver 传播、题库透传、桌面封套、Mobile 快照和前端回归测试。诊断类型未实现时 Go 测试 RED；前端缺详情 RED；Responses 实现前矩阵复现错误归类；Mobile DOM 缺显示区域 RED。
5. 实现安全诊断与兼容传递，再补充真实 loopback SSE、HTTP 状态、prod 无日志及取消/空响应测试。
6. 全量测试、vet、build、前端回归与 Vite 构建通过；固定 Wails v2.11.0 独立构建通过。
7. 对比入口基线与最终增量，确认依赖、bindings、旧 EXE 和白名单外文件未变。

## 2. 安全摘要

新增 `pkg/llm/diagnostic.go`：

- `DiagnoseError` 通过 `errors.As` 提取多层包装中的 `ProviderError`，防御 typed nil。
- `ErrorDiagnostic` 只含 code、stage、httpStatus、protocol、retryable、message。
- 类别、阶段、协议均为白名单；message 由已知类别重新生成，不信任可修改的 ProviderError.Message。
- 实际状态在 100–599 时保留，缺失时省略；未知类别与 response_failed 的 retryable 为 null，表示暂无可靠建议。
- `CleanDiagnostic` 在封套、快照和复制边界再次筛选，并复制状态/布尔指针，订阅者不能修改桥接内部状态。
- 不附加模型 ID、URL、Provider 标识、请求 ID、响应正文或 cause；现有类型没有可靠的这些标识。
- `ClassifyProviderError` 保留已有包装 ProviderError；无依据的普通错误归 unknown，不再一律 network_error。

## 3. 事件兼容与展示

保留 `solution-error` 第一参数字符串，第二参数增加安全摘要；不新增独立详情事件。原因：现有题库捕获在错误后立即进入 terminal，独立后续事件可能被丢弃；同一事件也没有新旧事件乱序配对风险。

- Solver 仅改错误报告路径，未改截图、上下文、Prompt、模型请求或结果提交逻辑。
- `newDesktopSolveEvent` 把摘要放入现有 epoch/sessionId/recordId 封套的可选 errorDetail。
- 前端沿用 streamTarget 关联；终态后、下一题和不同代际的迟到事件拒绝；下一题清除当前详情。
- 原始 Wails 字符串消费者仍收到相同位置的字符串，新增第二参数由升级后的 App.vue 消费。
- 新 `ErrorDiagnostics.vue` 同时用于整页 ErrorView 和分轮 SolveView，原友好错误文案保留。
- 取消仍保留 `context canceled` 字符串；无可证明协议的内部取消/空响应摘要显示协议未知，不从模型名或配置猜测。

## 4. Mobile 与题库

- Mobile SSE 仍发送原 snapshot/message 类型；仅增加可选 errorDetail，原 error 字符串不变。
- 错误及详情一次广播；下一题、成功和清屏清除摘要，旧 clearSeq 不得覆盖新状态。
- 手机页增加 details 展开区，使用 textContent 和字段白名单，不解析任意 HTML/SDK 文本。
- prod logger 关闭不影响桌面事件、Mobile 快照或诊断显示。
- 两个题库捕获器原样保留：第二参数由 variadic passthrough 透传，不改变失败状态保存。
- 不复用二次整理的 OrganizeFailure，不修改 Record schema、不迁移数据、不保存主解题诊断。退出/重新加载题库历史后无法找回旧详情，Mobile 仅保留当前快照。

## 5. Responses 分类

读取本地 SDK v1.12.0 的真实 `Response.Error.Code` 字符串类型，流式 failed、非流式 failed 与 error event 共用白名单分类器：

| 服务端业务码 | 内部类别 | 手动重试建议 |
|---|---|---|
| rate_limit_exceeded | rate_limited | 是 |
| server_error | provider_unavailable | 是 |
| vector_store_timeout | timeout | 是 |
| image_too_large、image_file_too_large | request_too_large | 否 |
| image_content_policy_violation | refused | 否 |
| invalid_prompt、无效图片/格式/Base64/URL/模式、图片过小、解析失败、空文件、下载失败、文件不存在 | invalid_request | 否 |
| invalid_request_error（兼容扩展码） | invalid_request | 否 |
| invalid_api_key、authentication_error（兼容扩展码） | authentication_failed | 否 |
| permission_denied、permission_error（兼容扩展码） | permission_denied | 否 |
| insufficient_quota（兼容扩展码） | quota_exhausted | 否 |
| 未知或缺失 | response_failed | 未知 |

认证/权限兼容码不是 SDK 的枚举常量；真实 Code 是字符串类型，因此只在端点明确返回这些值时识别，不声称原生服务必然这样返回。真正的 HTTP 401/403/429/503 仍通过 typed SDK error 分类并保留实际状态；业务事件不合成 HTTP 状态。incomplete、refusal、empty、EOF 原语义保留。

## 6. 产物

独立 Windows 验证程序：`build/bin/Q-Solver-change028-llm-error-diagnostics.exe`。

使用本地固定 Wails v2.11.0、Go 1.25.13，prod 标签；跳过 bindings、前端安装/二次构建、mod tidy 和模块同步。前端已单独 Vite build。未覆盖正式或旧验证 EXE，未启动产物。

## 7. 实施偏差与限制

- 使用同一错误事件的可选第二参数，而非独立详情事件：为降低终态丢失及乱序风险，已在事前方案说明。
- 不做主解题诊断持久化；详情由用户在当前失败后查看。
- 全树 diff-check 的 85 处空白错误均来自任务入口即存在的 models.ts，哈希未变，未清理用户工作。
- prod 全套相关包测试中的既有日志断言失败；诊断专项在 prod 下通过。未修改无关旧测试。
- race 仍在 Windows 测试进程启动时 0xc0000139，未获得有效竞态结果。
