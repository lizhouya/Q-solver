# 现状记录 — 01 · 006-mobile-sse-handshake

> **阶段**：业务代码修改前只读快照  
> **基线**：`c3db919`

## 当前 Bridge

- `Bridge` 使用 `sync.RWMutex` 保护 `state`、`subs`、`seq` 和 `model`。
- `GetState()` 独立取得 RLock 并返回当前快照。
- `Subscribe()` 独立取得 Lock，创建容量 64 的 channel 并写入 `subs`。
- `Publish()` 在 Lock 内更新全量快照并调用 `broadcastLocked()`；发送非阻塞，满 channel 丢帧。

## 当前 Server 建连顺序

`handleSSE()` 在鉴权与 header 设置后执行：

1. `initial := s.bridge.GetState()`
2. `writeEvent(..., "snapshot", initial)`，内部立即 `Flush()`
3. `sub := s.bridge.Subscribe()`
4. 循环消费 sub、心跳和 request context

## 缺陷窗口

步骤 1 后、步骤 3 前发生的 Publish 不会出现在 initial 中，也不会发送到尚未注册的 sub。客户端虽然成功建连，但可能永久持有旧状态。

现有端到端测试在连接后显式 `Sleep(100ms)`，注释说明其目的是等待 handler 完成订阅，因此测试本身也记录了该时序弱点。

## 不变约束

- 一次 Publish 仍更新一个全量 Snapshot。
- 多客户端独立 channel。
- 慢客户端允许丢中间帧。
- 断线重连首先收到 `event: snapshot`。
- 后续广播使用 `event: message`。
- 不改变 token、安全边界或 HTTP 路由。