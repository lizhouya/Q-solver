# 决策记录 — 05 · 004-mobile-status-visibility

> **状态**：设计期已完成（以下为设计阶段确定的设计决策，待用户实现阶段验收时确认）

## D-004 决策

| 编号 | 项 | 选择 | 含义 |
|---|---|---|---|
| D-004-001 | `MobileInfo.Port` 语义 | **状态信息修复，不改协议** | Port 属 `MobileInfo` 已定义的字段，仅补全 available 分支赋值；`pkg/mobile` 协议与 SSE/Bridge/State 均不改动 |
| D-004-002 | 扩展范围 | **不引入实时状态 / SSE client count / last snapshot / 多设备管理** | 这些属未来 Change，不进 004 |

## 设计原则

- 修改面最小化：仅 `app/mobile.go` 一处。
- 后端 `Server.Port()` 已有，为唯一数据来源，不重复取逻辑。
- 前端 `SettingsModal.vue` 已能消费 `port`，被动修正展示。