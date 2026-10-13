# Change 006 Decision

> **决策日期**：2026-10-13  
> **结论**：仅选择 F-006-01，Change 名称为 `006-mobile-sse-handshake`。

## 优先级排序

1. **P1 — F-006-01：SSE 建连原子订阅**（进入 Change 006）
2. **P2 — Solver 配置并发竞态**（延期，需要独立锁粒度设计）
3. **P2 — Model capability 与截图模型策略**（延期，需要产品策略）
4. **P2/P3 — multipart 内存、Shortcut 删除语义、Windows SetWindowLevel**（延期或仅文档）

## 选择理由

- 用户影响真实：建连瞬间可能错过唯一终态，手机长期显示旧状态。
- 修改范围集中在 `pkg/mobile`，与 Mobile Output 同一方向。
- 不修改 SSE/JSON 协议，不新增事件，不改变 OutputMode 路由。
- 不新增依赖、数据库或持久化。
- 可通过 Bridge 单元测试和 handler 首次 Flush 顺序测试验证。
- 回滚只需恢复 Bridge 订阅 API与 Server 调用顺序。

## 明确不纳入

- 不扩展手机协议，不传截图或题目。
- 不改变慢客户端丢帧策略。
- 不处理 Solver 并发锁。
- 不修改 Window、Shortcut、Frontend、Config、LLM Provider 或 Prompt。
- 不实现二维码、持久化历史或 Mobile UI 重构。

## Decision Gate

本目标不需要扩大范围、修改协议/架构、新增依赖或持久化，允许进入 Design 与 Implementation。