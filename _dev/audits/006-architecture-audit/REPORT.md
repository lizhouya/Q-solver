# Change 006 Architecture Audit Report

## 审计结论

Window、Shortcut、Mobile Output、Solver 与 Frontend 已完成只读核查。多数用户可见链路已在 Change 001～005 中实现；本轮唯一同时满足“高收益、低风险、边界清晰”的目标是 **Mobile SSE 建连原子性**。

## 选定目标

`006-mobile-sse-handshake`

当前 handler 先读取并 Flush 初始 Snapshot，再注册订阅。建连窗口内发布的事件可能丢失，尤其当丢失的是 done/error 时，手机会保持旧状态直到下一题。

修复将让 Bridge 在同一临界区内返回：

- 已注册的订阅 channel；
- 与该订阅注册时刻一致的当前 Snapshot。

Server 随后先使用该结果写初始 Snapshot，再消费 channel。协议、JSON、事件名、鉴权和广播策略均不变。

## 其他重要结论

- Window 实体可见性入口已统一经过状态机。
- Windows `SetWindowLevel` 空实现当前仅对应 macOS 专用前端路径，不构成本轮 Windows 用户缺陷。
- Shortcut 默认键、Settings 显示与真实 config 基本同步。
- Mobile waiting/streaming/done/error 可见，截图阶段不可见属于既有协议决策；mobile-only 桌面终态反馈完整。
- Solver 无锁存在未来/配置并发风险，但正确修复需要独立设计，延期。
- ModelSelect 只展示能力、不强制过滤；未知和自定义模型使简单过滤存在误伤，延期。

## 范围许可

Change 006 仅允许修改：

- `pkg/mobile/bridge.go`
- `pkg/mobile/server.go`
- `pkg/mobile/*_test.go`
- `_dev/changes/006-mobile-sse-handshake/`
- `_dev/audits/006-architecture-audit/`
- `_dev/changes/INDEX.md`（收尾登记）

禁止修改 Frontend、App、Config、Window、Shortcut、Solver、LLM Provider、Prompt、`go.mod`、`go.sum` 与 `frontend/wailsjs/`。