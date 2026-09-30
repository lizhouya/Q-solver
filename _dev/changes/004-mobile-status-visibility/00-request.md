# 变更请求 — 00 · 004-mobile-status-visibility

> **阶段**：设计初始化（只创建文档，不修改业务代码，不 git add/commit/push）
> **状态**：Not Started

## Change 名称

`004-mobile-status-visibility`

## 背景

Mobile Output 服务已经具备：

- SSE 服务（局域网监听，默认端口 38511）
- URL 获取（`mobileServer.URL()` 含 token）
- 手机连接（单页 + /sse + /health）

但后端 `GetMobileOutputInfo` 返回 `MobileInfo` 时：

- `Available = true`
- `URL` 正确
- `Port = 0`

导致 Settings 面板「当前端口 **0**」展示错误的设置状态——尽管服务真实监听于 38511。

## 目标

修复 `MobileInfo.Port` 字段，让 available 分支返回真实端口。

## 明确边界

- 本 Change **不扩展 Mobile 协议**。
- 不改 `pkg/mobile`、SSE、Bridge、State、Frontend、OutputMode、Window、Tray。
- 不引入实时状态 / SSE client count / last snapshot / 多设备管理（归未来 Change）。