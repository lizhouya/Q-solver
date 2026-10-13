# 实现记录 — 03 · 006-mobile-sse-handshake

> **状态**：Implementation 完成，待提交  
> **日期**：2026-10-13

## 实际实施

### Bridge

`Subscribe()` 现在一次返回 subscriber channel 与 initial Snapshot，并在同一写锁临界区内完成：

1. 创建容量 64 的 channel；
2. 注册到 `subs`；
3. 复制当前 `state`；
4. 返回二者。

因此 Publish 与订阅建立具有确定全序：更早事件进入 initial，更晚事件进入 channel。

### Server

`handleSSE()` 在写出初始快照前先调用新的原子 Subscribe，并立即登记 `defer Unsubscribe`。初始 `event: snapshot` 仍先于 channel 中的后续 `event: message` 写给客户端。

### Tests

- 新增 `TestSSESubscribesBeforeInitialFlush`：测试专用 Flusher 在第一次 Flush 时读取同包 Bridge subscriber 数量，断言为 1，并通过 request context cancel 验证 handler 退出。
- `TestSSEHandshakeAndPublish` 删除 `Sleep(100ms)`；HTTP 响应建立后立即 Publish，仍收到初始 snapshot 与 waiting message。
- SSE 两项测试重复 50 次通过。

## 边界偏差

无。未修改协议、事件名、Snapshot schema、鉴权、channel 容量、慢客户端策略或范围外模块；未新增依赖。

## 变更文件

- `pkg/mobile/bridge.go`
- `pkg/mobile/server.go`
- `pkg/mobile/bridge_server_test.go`

治理文件另见 Audit 目录、本 Change 目录及 INDEX。