# Change 028 事前现状

日期：2026-10-26

## Git 基线

分支 dev/qsolver-custom；HEAD 43022e10cd8fb1b38b72b56d95bc39b8bb4d201d。
42 个已跟踪修改、35 个未跟踪条目（目录折叠），暂存区为空。024–027 已有未提交内容，028 未分配；已核对 INDEX 与实际目录。

## 已确认事实

- ProviderError 有安全分类、阶段、状态码、协议及重试标记，支持 Unwrap；Solver 失败只发 err.Error()。
- questionSolveCapture 收到 solution-error 后 terminal=true；之后独立详情事件会被拦截。现有 variadic 参数完整透传，可安全增加第二参数。
- DesktopSolveEvent 已有 epoch/sessionId/recordId 关联，前端终态后清 streamTarget，迟到事件被拒绝。
- Mobile 使用全量 Snapshot，经 clearSeq 拒绝旧任务；错误字符串为 Error；新增可选字段不会改变现有 SSE 事件类型。
- 题库 Record 的 OrganizeFailure 专用于二次整理，不可挪作主解题错误。本次不持久化诊断，不迁移 schema。
- Responses SDK v1.12.0 有 Response.Error.Code/Message；Code 是字符串类型，有服务、限流及图片错误常量，无 HTTP 状态字段。HTTP 错误另由 typed SDK error 提供。
- 当前 failed 分支一律 provider_unavailable；未知普通错误一律 network_error，需要停止推测。
- logger_prod.go 为空实现，必须以界面诊断为交付，不能依赖日志。

## 保护与工具链

工作树及既有 EXE 已记录 SHA-256，目标文件另保存在仓库外的临时基线目录，只用于逐文件和增量审计，不批量还原。
本机默认 Go 1.27.1；已有缓存 Go 1.25.13，可离线固定使用；Node 24.14.0/npm 11.9.0。历史测试并非本次结果。

## 待验证

真实模型失败原因仍未知；本次不读真实凭据、不作真实请求。GUI、真实上游验证留给用户。
