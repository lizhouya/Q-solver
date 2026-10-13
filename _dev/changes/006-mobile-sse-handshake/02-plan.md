# 实施计划 — 02 · 006-mobile-sse-handshake

> **阶段**：设计完成，业务代码尚未修改

## 方案

### 1. Bridge 原子 API

将订阅 API 调整为一次返回 `(chan Snapshot, Snapshot)`：

1. 创建容量 64 的 channel。
2. 获取 Bridge 写锁。
3. 把 channel 注册到 `subs`。
4. 在同一锁内复制 `state`。
5. 解锁并同时返回 channel 与快照。

这样，任一 Publish 与该临界区必然有确定先后：

- Publish 先发生：返回的 initial 已包含它；
- Subscribe 临界区先发生：之后 Publish 会进入已注册 channel。

### 2. Server 顺序

`handleSSE()` 改为：

1. 取得 `sub, initial`，立即 defer Unsubscribe。
2. 写出并 Flush initial。
3. 进入既有 select 循环。

### 3. 测试

- 保留 Bridge 状态机、鉴权、HTML、health 测试。
- 更新 SSE 端到端测试，移除人为 `Sleep(100ms)`。
- 新增可控 ResponseWriter/Flusher 测试：在第一次 Flush 回调中读取 Bridge 订阅数量，断言已经为 1；随后取消 request context 结束 handler。
- 测试辅助类型只放在 `_test.go`，不新增生产 getter/hook。

## 文件范围

| 文件 | 计划 |
|---|---|
| `pkg/mobile/bridge.go` | 原子订阅 + 初始快照 API |
| `pkg/mobile/server.go` | 调整建连顺序 |
| `pkg/mobile/bridge_server_test.go` | 新增确定性顺序测试、移除 sleep |

## 风险与控制

| 风险 | 控制 |
|---|---|
| 初始事件与后续事件乱序 | 先注册并复制 state，再写 initial；channel 中只可能是更晚事件 |
| handler 提前退出泄漏订阅 | 取得 sub 后立即 `defer Unsubscribe` |
| 改变多客户端/慢客户端语义 | channel 容量与非阻塞广播保持不变 |
| 测试并发挂起 | 测试使用可取消 request context 与有界 timeout |

## 回滚

恢复 Bridge 原 `Subscribe() chan Snapshot`、Server 原调用形态并移除新增测试即可；无数据迁移、配置变化或协议兼容问题。