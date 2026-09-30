# 现状记录 — 01 · 004-mobile-status-visibility

> **状态**：设计期现状核实（只读源码，未改代码）

## 当前链路

```
SettingsModal.vue
        ↓  mobileInfo.port = info?.port || 0（SettingsModal.vue:206）
api.js  getMobileOutputInfo: () => GetMobileOutputInfo()
        ↓  前端→Wails binding
app/mobile.go  GetMobileOutputInfo()
        ↓
mobileServer  Port() / Start() / Started()
```

## MobileInfo 字段定义

`app/mobile.go`：

```go
type MobileInfo struct {
	Available bool   `json:"available"`
	URL       string `json:"url"`
	Port      int    `json:"port"`
}
```

## 现状

| 字段 | 来源 | 当前生产值 |
|---|---|---|
| `Available` | `mobileOutputEnabled()` | available 分支 true |
| `URL` | `mobileServer.URL()` | 正确（含 token，脱敏由调用方处理） |
| `Port` | 字段定义存在，见消费端 | **available 分支未赋值 → 恒 0** |

## 结论

- **字段定义存在**（`MobileInfo.Port`）✅
- **消费端存在**（`SettingsModal.vue` `mobileInfo.port` / DOM「当前端口」）✅
- **生产未赋值**（`GetMobileOutputInfo` available 分支 return 只给 `Available` + `URL`）❌
  — 服务真实监听 38511，UI 却恒显示 0，属确定性状态信息缺陷。

## 相关源码

- `app/mobile.go:108-123` — `MobileInfo` 结构 + `GetMobileOutputInfo`
- `pkg/mobile/server.go:69-73` — `Server.Port()` 返回 `s.port`（默认 `DefaultPort = 38511`）
- `frontend/src/components/SettingsModal.vue:150,202-206` — 端口消费展示