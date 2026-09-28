# 05 — 技术风险与分级（Technical Risks）

> 只读设计审计。针对 Mobile Output 的风险分级。
> 分级方式：`Confirmed`（源码/现行固有）/ `Potential`（预期性，需实现后验证）/ `Needs Verification`（见 06 决策与后续验证）。

---

## 一、风险清单

### 1. Wails / 进程生命周期

| 风险 | 级别 | 说明与依据 |
|---|---|---|
| 网络服务与 Wails 进程绑死 | `Confirmed` | 服务放 App 内则必须随 `OnShutdown`（main.go:60）关闭；不清理会残留连接。但它在 App 内也最简单。 |
| WebView2 崩/退出时 WebSocket 客户端失效 | `Potential` | 若选「前端二次转发」（01 方案 c），依赖 WebView2 存活，与幽灵窗口 + 隐身冲突。 |

### 2. WebSocket / SSE 传输

| 风险 | 级别 | 说明 |
|---|---|---|
| WS 心跳/超时处理缺失 → 陈旧连接 | `Potential` | 若选 WS 需自写心跳与清理，否则连接泄漏。SSE 无此问题（EventSource 自带重连）。 |
| SSE 单 faculty 不支持双向控制 | `Potential` | 若产品要手机「停止/重新解题」（D-Mobile-006），SSE 需补 POST — 方案变 Double-channel。 |

### 3. 状态同步 / 一致性

| 风险 | 级别 | 说明 |
|---|---|---|
| 断线重连后状态失步 | `Potential` | 需 `seq` 序号 + 断线后拉全量快照对齐；否则手机显示停滞/丢 chunk。 |
| 流式 chunk 乱序/丢帧 | `Potential` | 依赖协议保证顺序（SSE 顺序天然保；WS 需维护顺序）。 |
| 与 F-001 KeepContext 未来接入 +rounds 的语义 | `Potential` | 若真多轮修复（F-001），`rounds` 溢出到多个问题，移动端切题/进度需新增状态。 |

### 4. 网络权限 / 跨平台

| 风险 | 级别 | 说明 |
|---|---|---|
| Windows 防火墙阻断入站 | `Confirmed` | vs localhost 无防火墙问题，但局域网访问需放行专用网络入站；跨 `public/private` 网络差异。 |
| 网络多网卡 / IP 选择 | `Needs` | 笔记本同时有 Wi-Fi/虚拟网卡/热点时 IP 解析需枚举；二维码生成的必选 IP。 |
| macOS 未支持（当前客户端主 Win） | `Needs` | 若未来 mac 版，需复用相同网络层；现在先按 Win 设计。 |

### 5. 安全 / 隐私

| 风险 | 级别 | 说明 |
|---|---|---|
| Token 泄在 URL/日志 | `Confirmed`（规则风险） | URL 中的 token 可能进浏览器历史/代理日志；建议**session 短时效 token** + 传输层仅局域网 |
| 答案含题面/简历被局域网嗅探 | `Confirmed` | 手机输出把**答题内容 + 简历上下文（CandidateProfile）**暴露在同一 LAN；**明文传输需谨慎**（D-Mobile 004 是否含简历） |
| 手机连接无身份 ⇒ 恶意接入 | `Potential` | 无 token 时局域网内任意设备可订阅解题内容；**至少 Token 是底线** |
| 移动端渲染 v-html/XSS | `Needs` | 若手机页面复用 `marked.parse`（见 F-015），不可信内容渲染风险需重评估 |

### 6. 编译/构建影响（合规）

| 风险 | 级别 | 说明 |
|---|---|---|
| 增加构建复杂度 | `Potential` | 若选「方案2 独立 mobile 前端」→ 需 build:desktop + build:mobile 双构建，`wails.json`/CI 增加步骤 |
| go mod 新增依赖 | `Potential` | 若选第三方 ws 库 → `go.mod` 增依赖（AGENT.md 有依赖纪律，需归档决策） |

---

## 二、风险汇总（按优先级）

| 级别 | 关键项 |
|---|---|
| 🔴 高 | Token 安全 / 局域网数据暴露（截图+简历）；服务生命周期与 Wails 绑死；Firewall 入站阻断 |
| 🟡 中 | 断线重连状态失步；WS 双向缺省；多网卡 IP 选择；双构建复杂度 |
| 🟢 低 | macOS；移动端 XSS（本地低触发）；SSE 单向补充 |

> **实施前必须验证清单（Needs Verification）**：
> 1. 手机浏览器 EventSource/WebSocket 在同一 Wi-Fi 下访问 Win 桌面端口是否畅通（防火墙实测）。
> 2. 断线 10s 重连后手机端状态是否与桌面一致。
> 3. 多网卡（Wi-Fi + 虚拟网卡）自动选 IP 正确性。

---

*（本节完成。风险为分析，不修改源码。）*