# 回滚方案 — 06 · 004-mobile-status-visibility

> **状态**：设计期记录

## 回退策略

如果实现阶段出现问题：

- **回退文件**：`app/mobile.go`
  - 恢复 `GetMobileOutputInfo()` available 分支（去掉 `Port: a.mobileServer.Port()`）。

使用单文件还原即可，例如：

```bash
git checkout -- app/mobile.go
```

## 不涉及的文件

- `pkg/mobile`（协议/实现）——未改动，无回退
- `frontend` ——未改动，无回退
- `pkg/config` ——未改动，无回退
- Window / Tray / Shortcut / OutputMode ——未改动，无回退

## 影响面评估

本 Change 影响面极小：仅 `app/mobile.go` 一处字段赋值，属纯增量修复；回退零风险、零副作用。