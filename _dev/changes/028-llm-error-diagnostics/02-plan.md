# Change 028 实施方案

日期：2026-10-26
状态：按用户明确授权任务实施；以下方案在业务代码修改前建立。

## 兼容设计

保留 solution-error(message) 的首参数契约，增加第二个可选 ErrorDiagnostic 参数；桌面封套和 Mobile Snapshot 增加可选 errorDetail。相比独立事件，该方案使详情与错误原子关联，避开 capture 终态丢事件和乱序问题，不新增网络调用或 App 方法。

安全摘要使用 errors.As，白名单化 code/stage/protocol，重新生成固定 message，不复制 SDK 文本、cause、配置、URL、模型或请求 ID。HTTP 未知省略；retryable 不可判定时为 null。裸错误、typed nil 与未知类别安全降级 unknown；既有 ProviderError 保留已知字段。

Responses 流式和非流式 failed 使用同一分类器；已知服务/限流/图片及明确认证权限兼容码映射既有分类，未知/空码使用 response_failed，重试建议未知。不会把事件业务码变成 HTTP 状态。incomplete/refusal/empty/EOF 语义不变。

桌面复用新增 ErrorDiagnostics.vue，在整页和分轮错误区展示。当前请求与诊断复用现有 epoch/sessionId/recordId 隔离。旧事件缺详情显示暂无信息，新请求清旧详情；不新增无关联的独立详情监听。

Mobile 仅给现有快照添加可选摘要，复制快照时深拷贝；错误终态一次广播；新题、成功、清屏清除旧摘要。手机错误卡以 textContent 展示固定字段；旧 SSE 客户端忽略新字段。

题库捕获继续透传可选参数，保持原失败落盘行为。无主解题诊断持久化；桌面退出/重载历史、Mobile 进程重启或清屏后详情不可追溯。

## 文件白名单

- pkg/llm/errors.go、errors_test.go、openai_responses.go、openai_responses_test.go；新增 diagnostic.go、diagnostic_test.go、responses_failure_test.go。
- pkg/solution/solution.go（仅失败诊断），新增 error_diagnostics_test.go。
- app/mobile.go（仅错误封套），新增 error_diagnostics_test.go；题库捕获用测试证明兼容，不改持久化。
- pkg/mobile/state.go、bridge.go、web/index.html（仅诊断），新增 error_diagnostics_test.go；DOM 测试按需增补。
- frontend/src/App.vue、stores/solution.js、components/ErrorView.vue、SolveView.vue；新增 components/ErrorDiagnostics.vue、llm_error_diagnostics_regression_test.mjs。
- 本 Change 七份文档、evidence、安全审计和 INDEX 本条记录。

不改 go.mod/go.sum、依赖清单、生成 bindings、Provider 工厂、配置、Prompt、图片或历史语义。

## 验证顺序

1. 记录保护基线；离线执行现有相关测试获得基线。
2. 测试先行：包装错误/未知/隐私，Responses 模拟响应，Solver→capture→desktop/Mobile，前端状态和 UI。
3. 实现至 GREEN；执行全量 Go test/vet/build、前端回归、Mobile DOM 和 Vite build、diff-check。
4. 无 App API 改动，不重新生成 bindings。检查固定 Wails CLI 与参数，若可跳过生成及安装则独立构建；否则记录阻塞，使用明确标注的 Go 集成编译补充验证，不伪称 Wails 通过。
5. 对比基线哈希及增量差异；只保留白名单修改；不暂存、不提交、不推送。

## 风险

所有诊断字段都可能来自外部错误，必须白名单化。Mobile 指针复制须独立。不能将 CCR 返回状态当成最终模型厂商状态，也不能把 retryable 当成自动重试授权。
