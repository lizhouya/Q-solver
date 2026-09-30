# 实现计划 — 02 · 004-mobile-status-visibility

> **状态**：设计期计划（只读规划，未编码）

## 修改范围

仅：

- `app/mobile.go`

## 逻辑调整

### `GetMobileOutputInfo()`

当前（`app/mobile.go:119-122`）：

```go
return MobileInfo{
	Available: true,
	URL:       a.mobileServer.URL(),
}
```

调整为：

```go
return MobileInfo{
	Available: true,
	URL:       a.mobileServer.URL(),
	Port:      a.mobileServer.Port(),
}
```

> `Server.Port()`（`pkg/mobile/server.go:69-73`）已存在，返回 `s.port`，默认 `DefaultPort = 38511`。
> 仅给 available 分支补齐真实端口，unavailable 分支（`Port: 0`）保持不变。

## 保持不变（严格不修改）

- `pkg/mobile` 协议与实现
- SSE / Bridge / State / Snapshot / Payload
- Frontend（`SettingsModal.vue` 只读展示，消费 `port` 已有输入）
- OutputMode / Window / Tray / Ghost

## Phase 划分

- **Phase 1（代码修改）**：`app/mobile.go` 补齐 `Port`。
- **Phase 2（测试）**：`app/mobile_test.go` 补充 `GetMobileOutputInfo` 端口断言。