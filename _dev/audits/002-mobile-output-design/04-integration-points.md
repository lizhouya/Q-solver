# 04 — 集成点分析（Integration Points & Modes）

> 只读设计审计。包含三项专题：
> ① 后端承载位置 + 生命周期（第十一部分）；
> ② 前端承载形式比较（第十二部分）；
> ③ OutputMode 输出模式分析（第十三部分）。
> 全部为分析，不修改代码、不选最终方案。

---

## 一、后端承载位置与生命周期（第十一部分）

### 1.1 候选位置

| 位置 | 说明 | 依据 | 评述 |
|---|---|---|---|
| `app/`（App 结构体） | 作为 App 的成员，`Startup()` 初始化 | app/app.go:18-30，App 已持所有 service | ✅ **推荐候选**：具备 `ctx`、`EmitEvent`、`configManager`，天然可与事件总线 / 配置联动 |
| `pkg/platform/` | 现有纯 win32 包装（ghost/click-through/hotkey） | platform_windows.go:195-250 | ⚠️ **不推荐**：该包职责是桌面窗口/系统 API，与网络服务无关联，混入会破坏单一职责 |
| 新 `pkg/mobile/`（或 `pkg/bridge/`） | 独立包，被 App 引用 | 无既有参与 | ✅ **可与 app 并列**：网络服务与未来一切移动端能力单点收容，维护清晰 |

> **建议方向（非拍板）**：网络服务放 **`pkg/mobile` 独立包 + 在 `App` 上挂载**（App 持一个 `mobileBridge` 实例）。`pkg/platform` 保持纯窗口平台层职能不动。
> 理由：`platform` 目录当前在与 `os` win32 平台强耦合，放网络服务会让 `F-004`（SetWindowLevel）未来修复时更复杂。

### 1.2 生命周期（Startup → run → Shutdown）

```
Wails Options.OnStartup (main.go:40 → app.Startup)
   └─ app.Startup()（app/app.go:43-72）
        ├─ 读取配置
        ├─ … 现有服务 init（llm/solver/resume/shortcut）
        └─ [新] mobileService.Start(ctx)      ← 在 Startup 末尾：绑定端口、生成 token/二维码、启动网关

运行期
   └─ mobile 服务 监听局域网端口；接收 App.EmitEvent 镜像 / 提供查询接口；管理手机连接

Wails OnShutdown（main.go:60 → app.OnShutdown）
   └─ app.OnShutdown()（app/app.go:86-93）
        ├─ shortcutService.Stop()
        ├─ configManager.Save()
        └─ [新] mobileService.Stop()          ← 关闭端口、清理连接如果手机浏览还开着

```

**要点**：
- 端口 / 绑定 IP 由配置提供（默认 `0.0.0.0` 或 `127.0.0.1` 取决于产品决策，见 06 D-Mobile-002/005）。
- 服务生命周期与 Wails 进程一致，`Stop()` 必须资源可清理。
- 「仅在手机连接时起服务」为可选优化（产品决策 D-五）。

### 1.3 部署形态 & 网络

- Windows 防火墙需放行入站端口（局域网访问时）。此点标注为**运行时风险**（见 05）。
- 端口冲突处理：动态端口 + 失败回退。

---

## 二、前端承载形式比较（第十二部分）

桌面端 `main.go` 用 `//go:embed all:frontend/dist` 嵌入 `frontend/dist` 为 AssetServer。手机端页面承载有三个候选：

| 方案 | 描述 | 维护成本 | 侵入程度 | 适用度 |
|---|---|---|---|---|
| **方案 1：复用现有 Vue（同一个 SPA）** | 手机浏览器打开同一个构建好的 SPA，由路由/参数切到「移动视图」 | 中 | **中**（桌面 SPA 需加路由分叉，可能影响桌面渲染复杂度） | 中低：桌面 SPA 面向幽灵窗口（宽 1024），移动端适配需大量响应式改造；且 WebView2 与手机浏览器面对同一份代码可能因 UA 分支引入 bug |
| **方案 2：新增 mobile 子页面（独立 SPA）** | `frontend` 里建 `mobile/` 子目录，独立构建出 `dist/mobile/`，后端 AssetServer 或独立路由提供给 `/mobile` | 低~中 | **中**：前端构建多一个 entry，但桌面代码零改动 | ✅ 高：隔离清晰，移动端只面对精简页面（等待/答案/流式），可免包一堆桌面组件；风险是 double-build 流程（build:desktop + build:mobile） |
| **方案 3：Go embed 纯静态页（HTML/CSS/JS 单页 + 手写）** | Go 内 `//go:embed` 一个小型静态页（含 markdown 渲染库） | 高（需手写渲染逻辑，无 Vue 生态） | **低侵入**（纯新增 embed，不动现有前端） | 中：无框架时答案渲染（Markdown+LaTeX+代码）需自己处理，复杂度身高；但与 Vue 版本独立，可完全按需裁 |

> **观察（非选型）**：方案 2（独立 mobile 构建）在「不动桌面」「可充分为移动优化」「构建清晰」三者平衡最好；方案 3 若只做「纯答案文本+M-建渲染」也可简；方案 1 对幽灵窗口形态侵入最大，通常不适合。

---

## 3. OutputMode 输出模式分析（第十三部分）

### 3.1 场景

手机显示时，电脑端可能**不显示**（Ghost 模式桌面隐身），需要在「谁看到答案」上做模态区分。

### 3.2 候选 OutputMode

| 模式 | 描述 | 桌面行为 | 手机行为 | 影响层级 |
|---|---|---|---|---|
| `desktop` | 仅桌面 → 当前行为 | 正常渲染 | 无连接/关闭 | 前端 `solution.js` 现有当日行为，零改动 |
| `mobile` | 仅手机 | 桌面**不渲染**（隐藏答案 cards）或置后台 | 手机完整流式+答案 | 前端：桌面端在 `handleStreamChunk/handleSolution` 时跳过渲染；后端：事件照发到 mobile 网关 |
| `both`（同步显示） | 双端同时展示 | 正常渲染 | 同步渲染 | 两端都消费事件；语义`=`当前行为 + mobile |

- 层涉及：**触达层**：前端渲染开关 + 事件分发。
- **实际上 OutputMode 最影响「前端渲染」而非后端**——后端一直 emit，前端根据 mode 决定是否落到 DOM；mobile 端永远消费事件。

### 3.3 与已有状态整合

- `settings.js` 目前无 output 相关字段；`KeepContext`、`screenshotMode` 已存在（settings.js:18-19）。OutputMode 可作为**新配置键**（如 `outputMode: 'desktop' | 'mobile' | 'dual'`）。
- 状态影响面：`App.vue` 的渲染逻辑（桌面 SVG/卡片的显隐）、`mobile` 网关是否启用、连接状态展示。

---

## 四、设备连接流程（第十五部分：二维码 / Token / 局域网 / 断线重连）

> 流程设计，不实现。

### 4.1 期望流程（桌面端）

```
手机与电脑同一局域网
  1. 电脑（Q-Solver 桌面端）启动后，若启用了 Mobile 服务：
       • 绑定端口（默认如 38511），生成一次性访问 Token
       • 主界面显示二维码（内容 = ws://IP:PORT?token=… 或 扫描即打开 URL）
  2. 手机浏览器扫二维码 / 手动输入 IP:PORT
       • 携带 token 建立连接（WebSocket/SSE/HTTP — 传输选型见 02）
  3. 连接成功 → 桌面端显示「手机已连接」；手机显示实时解题流程
  4. 断开/重连：手机切后台再回来、网络抖动时自动重连（EventSource 内置 / WS 应用层重连）
```

### 4.2 连接要素

| 要素 | 候选 | 权衡 |
|---|---|---|
| **发现方式** | ① 二维码（桌面展示，手机扫）② 手动输入 `IP:端口` ③ 两者兼有 | 二维码最顺（考试/键盘场景不低头输 IP）；手动输 IP 适合无扫码枪/画面被占 |
| **身份认证** | ① 无 Token（局域网信任）② 短随机 Token ③ 每次生成换新 | 本地决胜：建议**至少短 Token** 防局域网嗅探；Token 写入 URL 由二维码携带（有效一 session） |
| **IP 获取** | Go 枚举本机网卡选非 loopback 地址；若有多个网卡需用户选 | 输入框内让用户确认网关 IP 或自动检测 |
| **断线重连** | WS：应用层心跳 + 指数退避重连；SSE：EventSource 自动重连 + last-event-id 续传 | EventSource 重连免费；WS 需自己写 |
| **多设备** | 同一 Token 支持 1~N 手机；或每设备独立 Token | 视产品定位（单手机 vs 多人围观） |

### 4.3 关键约束

- **Token 属于敏感信息**：任何日志/状态报告不打印实值（遵循 AGENT.md `SET/NOT SET` 规则）；Token 存内存或加密配置，不作为明文写入 `_dev` 文档。
- 跨子网（不在同一局域网）场景需要公网中继（D3 in 02）→ 与本地优先冲突，明确不推进。
- Firewall：Windows「专用网络」入站端口，首次启动可能需要提示/放行。

---

*（本节完成。三个部分均为分析，无代码改动、无最终选型。）*