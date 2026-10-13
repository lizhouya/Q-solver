# 决策记录 — 05 · 006-mobile-sse-handshake

## D-006-001：原子边界

选择在 Bridge 内以现有 mutex 原子完成注册和快照复制，而不是在 Server 添加额外锁或重试。

## D-006-002：协议保持

不增加 event、ack、replay buffer 或序列查询接口。现有全量 Snapshot + 单调 seq 足以表达状态，缺陷只在订阅时序。

## D-006-003：慢客户端

保留容量 64 和非阻塞广播；本 Change 只保证建连边界不丢失唯一状态，不承诺慢客户端接收每一帧。

## D-006-004：测试策略

使用测试专用 ResponseWriter/Flusher 观察首次 Flush 时的 subscriber 数量，不在生产代码增加测试 hook 或 getter。

## 延期项

Solver 锁、模型能力策略、Shortcut 删除语义、二维码与持久化均不纳入本 Change。