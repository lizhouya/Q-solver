# 变更请求 — 00 · 006-mobile-sse-handshake

> **状态**：已批准实施  
> **日期**：2026-10-13  
> **来源**：`_dev/audits/006-architecture-audit/` F-006-01

## 背景

Mobile SSE handler 当前先读取并 Flush 初始 Snapshot，之后才注册 Bridge 订阅。若解题事件在两步之间发布，新客户端会收到旧快照且错过新事件；当错过的是 done/error 且没有后续事件时，手机会长期停留在错误状态。

## 目标

让 SSE 客户端的“订阅注册 + 初始 Snapshot 读取”具备原子顺序保证：初始 Snapshot 之后发生的每个事件，要么已包含在初始 Snapshot 中，要么进入该客户端订阅 channel。

## 验收标准

1. Bridge 在同一临界区内注册 subscriber 并复制当前 Snapshot。
2. Server 在第一次向客户端 Flush 前已经完成订阅注册。
3. Snapshot JSON、SSE event 名、token 鉴权、channel 容量和慢客户端丢帧策略不变。
4. 现有 Mobile 测试通过，并新增首次 Flush 时已订阅的确定性测试。
5. `go test ./...`、`go build ./...`、`go vet ./pkg/mobile/...`、前端 build、`git diff --check` 通过。

## 允许范围

- `pkg/mobile/bridge.go`
- `pkg/mobile/server.go`
- `pkg/mobile/*_test.go`
- `_dev/audits/006-architecture-audit/`
- `_dev/changes/006-mobile-sse-handshake/`
- `_dev/changes/INDEX.md`（收尾登记）

## 禁止范围

不修改 Mobile 协议、Frontend、App、Config、Window、Shortcut、Solver、LLM Provider、Prompt、`go.mod`、`go.sum`、`frontend/wailsjs/`；不新增依赖、数据库或持久化。