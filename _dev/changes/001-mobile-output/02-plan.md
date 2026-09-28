# 实施方案 — 02 · 001-mobile-output

> 基于 `00-request` 与 `01-current-state` 制定的具体实现步骤。**本文件在动手写代码前完成**，实现中若方案变化，在 `03-implementation` 中记录偏差。

---

## 1. 方案概述

在**不动 `Solver.Solve` 的 LLM / 截图 / Prompt 逻辑**的前提下，于 `App.EmitEvent` 这个唯一事件出口旁路插入一个 **Mobile Bridge**：它镜像 `solution-stream-*`/`solution` 事件，维护一份独立的**统一状态快照**（只读、内存态），并通过一个新增的**本地局域网 HTTP + SSE Server**（`pkg/mobile`）把状态实时推送给手机浏览器。手机端是 **Go embed 的独立静态移动页**（不含答案回落前不被侵入；纯手写 HTML/CSS/JS + 轻量 markdown），用 `EventSource` 订阅（自带自动重连）。桌面设置新增 **OutputMode（desktop/mobile/dual）** 切换：`mobile` 模式下桌面**跳过渲染**答案 DOM，`dual` 双端渲染，`desktop`（默认）保持现状零改动。连接以**二维码为主、IP 手动输入兜底**；二维码库若依赖增量过大则第一版只显示 URL。

**架构一句话**：`TriggerSend → Solver.Solve → App.EmitEvent` 依旧动；`pkg/mobile.Server` 持有 `pkg/mobile.Bridge`，`App.EmitEvent` 调用 `bridge.Publish(event, data)` 镜像事件；手机 `EventSource` 订阅 `/sse` 获得标准 JSON 流。

---

## 2. 分步实施计划

| 步骤 | 操作 | 涉及文件（新增为主） | 预期产出 | 完成标志 |
|---|---|---|---|---|
| **1. SSE Server** | 新建 `pkg/mobile`，实现本地 HTTP + SSE（`net/http` 标准库，无新依赖），端口默认 `38511`，`StartHost/Stop` 生命周期，`0.0.0.0` 绑定 | 新增 `pkg/mobile/server.go` | 局域网可访问 `/`、`/sse`；SSE 长连接 | 桌面启动后 `curl http://127.0.0.1:38511/sse` 收到握手 |
| **2. Event Bridge** | 在 `pkg/mobile` 内建 `Bridge`，实现统一状态机 `{seq, status, stage, payload{question, analysis, answer, model}, error}`；把 `solution-stream-start/thinking/chunk/solution/error` 映射进去 | 新增 `pkg/mobile/bridge.go`、`pkg/mobile/model.go` | 一个全局最新状态（内存锁）+ 逐个订阅者推送队列 | 单测模拟事件序列，断言状态收敛 |
| **2.2 状态扩展** | SSE 连接时先发**全量快照**（让手机刷新/重连后自动恢复），随后增量；`seq` 单调递增防丢；携带 `rounds` 字段留给 F-001（第一版固定 1） | 并入 `bridge.go` | 「刷新即恢复」语义 | 断线重连后，手机显示与新连接前一致 |
| **3. 接入现有事件流** | `app.go` 挂载 `mobile`：`Startup` 时启动、`OnShutdown` 时停止；`App.EmitEvent` 内旁路调用 `bridge.Publish` | `app/app.go`（Startup/OnShutdown/EmitEvent 各动一行） | 桌面照常 + Mobile 同步 | 桌面解题时手机实时看到状态变化 |
| **4. 移动端页面** | 新增 **Go embed 独立静态页** `pkg/mobile/web/`（index.html + app.js + style.css，手写轻量 markdown），订阅 `/sse` 渲染「等待 → 分析 → 答案」，带自动重连 + 刷新恢复 | 新增 `pkg/mobile/web/*` | 手机扫码打开即看实时结果 | 手机浏览器访问 `/` 显示状态变化 |
| **5. 二维码 + URL 入口** | 生成连接 URL `http://<IP>:38511/?token=...`；二维码图像**优先**用新增 `go-qrcode` 库；若依赖决策不过（增加过大）则第一版**仅显示 URL 文案**（用户已默认） | 新增 `pkg/mobile/qrcode.go`；改动 `app` 暴露接口给前端；`settings.js` 一个连接状态面板/弹窗 | 桌面 UI 显示可扫码二维码 + 备选 URL | 手机扫二维码（或手输 IP）连通 |
| **6. OutputMode** | 新增配置键 `outputMode: desktop/mobile/dual`（默认 `desktop`）；后端配置 `config.go` 增字段；前端 `settings.js` + `SettingsModal` 加三态开关；前端 `App.vue` 按 mode 决定是否渲染答案 DOM（mobile 时跳过答案 DOM，双端外仍发射事件 | 改 `pkg/config/config.go`、`frontend/src/stores/settings.js`、`frontend/src/components/SettingsModal.vue`、`frontend/src/App.vue` | desktop 默认零影响；mobile 桌面隐藏答案；dual 双端 | 切换 mode 后桌面渲染行为符合预期 |
| 7. 安全 / token / 收尾 | 随机 session token（`crypto/rand`），二维码/URL 携带 token；SSE 校验；「不对外网开、仅局域网」；日志中 token 只 `SET/NOT SET`；关闭后端口释放验证 | `pkg/mobile/server.go`（token 校验）、`pkg/mobile/bridge.go`（清理） | 无明文 token 落盘、无残留进程 | 关闭 Q-Solver → 端口释放；`netstat` 无残留 |

> **说明**：步骤 4 中移动页面采用「Go embed 纯静态页」，避免双前端构建（`build:desktop` + `build:mobile`）带来的复杂度，符合 00-request 非目标「不升级第三方依赖、不建双构建」。桌面 SPA 完全不动移动端。若后续需要，可升级为独立 mobile SPA。
>
> **二维码决策（2026-09-26 用户确认：方案 A）**：第一版**不引入** `github.com/skip2/go-qrcode`。仅实现局域网 URL + 文案显示 + 一键复制。二维码作为后续独立小变更处理。

---

## 2.1 OutputMode 路由约束（约束一，用户确认 2026-09-26）

**目标**：`desktop`/`mobile`/`dual` 必须真正控制「答案输出端」，不能在 `runtime.EventsEmit` 后简单镜像。

**路由位置**：`App.EmitEvent`（app/app.go:95-97）—— 因为 `Solver.Solve` 的一切事件正是通过 `cb.EmitEvent = a.EmitEvent` 发出，此处是唯一可做分流的输出 Router。

**受 OutputMode 控制的「解题输出事件」白名单（仅以下 5 个）**：

| 事件 | 语义 | desktop | mobile | dual |
|---|---|---|---|---|
| `solution-stream-start` | 开始 | Wails | Bridge | 两者 |
| `solution-stream-thinking` | 分析过程 | Wails | Bridge | 两者 |
| `solution-stream-chunk` | 正文流式 | Wails | Bridge | 两者 |
| `solution` | 最终答案 | Wails | Bridge | 两者 |
| `solution-error` | 错误 | Wails | Bridge | 两者 |

**永不屏蔽、永远走桌面 Wails 的事件**（这些不是解题输出，mobile 模式不得影响桌面）：
`toast` `require-api-key` `open-settings` `start-solving` `user-message` `screenshot-*` `screenshots-cleared` `init-status` `key-recorded` `shortcut-*` `toggle-visibility` `click-through-state` `scroll-content` `copy-code` 等一切 `solution-` 前缀之外的事件。

**桌面 UI 状态收敛**：mobile 模式下桌面不渲染答案，但 `start-solving` 仍会触发 loading 状态；为避免桌面永远卡 loading，后端在 mobile 模式**终态**（`solution`/`solution-error`）额外向 Wails 发**空状态事件 `solution-hidden`**（只复位 loading/status 文本，不携带任何答案内容），桌面显示「已转发至手机端」，不泄露内容。

## 2.2 客户端写真降级约束（约束二，用户确认 2026-09-26）

- 默认端口 `38511`；**Mobile Server 是附加能力**。
- 端口被占用 / HTTP 启动失败 / 取局域网 IP 失败 → **仅记录日志 + 标记「Mobile 不可用」**，绝不影响 Q-Solver 主进程启动、截图、解题。
- mobile 模式且 Mobile Server 不可用 → `solution` 事件**自动降级走 desktop（Wails）**，桌面仍能看答案。
- **main.go 仅在 App lifecycle 不足以管理 Mobile 时才改**；预期不改——接入点在 `App.Startup` / `App.OnShutdown` 内。

---

## 3. 涉及业务源码的改动清单

> **二次开发禁止越过此清单直接改代码。** 本清单只覆盖 001 变更必须触碰的内容。

### 新增文件

| 文件 | 内容 | 状态 |
|---|---|---|
| `pkg/mobile/server.go` | HTTP + SSE Server，`StartHost/Stop`，token 校验 | 新增 |
| `pkg/mobile/bridge.go` | `Bridge` 事件镜像 + 统一状态机 + 快照/广播 | 新增 |
| `pkg/mobile/state.go` | 统一消息结构 / 状态枚举 | 新增 |
| `pkg/mobile/qrcode.go` | 二维码生成（可选，依赖许可） | 新增（可选） |
| `pkg/mobile/web/index.html` | 移动端入口页面 | 新增 |
| `pkg/mobile/web/app.js` | EventSource 订阅 + markdown 渲染 | 新增 |
| `pkg/mobile/web/style.css` | 移动端样式 | 新增 |

### 修改文件（仅限以下紧密触碰点）

| 文件 | 改动 | 最小侵入依据 |
|---|---|---|
| `app/app.go` | `App` 增加 `mobileServer`/`mobileBridge` 字段；`Startup` 末尾启动 mobile；`OnShutdown` 停止；`EmitEvent` 旁路 `bridge.Publish` | 「事件出口唯一」原则，旁路不改现有流程 |
| `main.go` | 仅在需要 embed 移动页到 dist 之外时最小改动 | 若移动页不进 `frontend/dist` 则不需要改 |
| `pkg/config/config.go` | `Config` 加 `OutputMode string` + 默认 `desktop` + `Validate` 检查 | 配置后端 |

### 前端修改（非侵入，另归档）

| 文件 | 改动 |
|---|---|
| `frontend/src/stores/settings.js` | `settings.outputMode`、`tempSettings.outputMode`、`applyConfig`/`buildConfigToSave` 双向带出 |
| `frontend/src/components/SettingsModal.vue` | 新增输出模式三选一 |
| `frontend/src/App.vue` | 订阅到 `solution` 时按 mode 决定是否渲染（desktop/mobile/dual） |

---

## 4. 测试与验证计划

| 用例 | 方法 | 预期 |
|---|---|---|
| 构建 | `go build ./...`；前端 `npm run build` | 两构建通过 |
| 桌面回归 | `wails build` 启动；截图→发送→解法 | 桌面流程完全不变 |
| SSE 握手 | 启动后 `curl http://127.0.0.1:38511/sse` | HTTP 200 + `text/event-stream` 握手 |
| 手机实时 | 手机浏览器连 `http://<局域网IP>:38511/?token=…` | 实时收到 waiting→streaming→done |
| 刷新恢复 | 手机刷新页面 | 显示当前最新快照（自动重连 + 全量快照） |
| OutputMode | 设置切 `mobile` → 桌面不渲染答案、手机正常；`dual` 双端 | 行为符合定义 |
| 安全 | 日志 grep token 实值 | 无实值；仅 `SET/NOT SET` |
| 关闭清理 | 退出 Q-Solver → `netstat -ano | findstr 38511` | 端口释放无残留 |
| 防火墙 | 手机首次连需要放行专用网络入站 | 提示引导 |

---

## 5. 回滚预案

| 场景 | 操作 |
|---|---|
| 桌面流程被破坏 | 若 `EmitEvent` 旁路导致异常，`app.go` 回退 `EmitEvent` 为纯 `runtime.EventsEmit`（桥挂载挪到可选） |
| 配置损坏 | `OutputMode` 无效值 → `Validate` 拒绝返回默认 `desktop` |
| 网络服务冲突 | 端口冲突 → 动态端口 + 失败回退（见 06 风险） |
| 前端改动伤人 | `settings.js`/`App.vue` 变更可 `git checkout` 回滚（前端独立出此变更） |

> 详细勾选见 `05-rollback.md`。

---

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 二维码库新增依赖（`go-qrcode`） | go.mod 增加依赖（AGENT.md 依赖纪律） | **优先降级为 URL 纯文本**接入；若必须二维码且依赖可控，单独立项归档其决策 |
| 局域网端口触发 Windows 防火墙 | 首次访问被阻 | 默认 `0.0.0.0` + 记录引导；文档提示「专用网络」放行 |
| SSE 连接与 State lock 竞争 | 数据竞态 | `Bridge` 全局 mutex + 每个订阅者独立 goroutine 带缓冲队列（或合并） |
| 移动端有残留进程 | 端口未释放 | `OnShutdown` 强制 `Stop`，`netstat` 验证 |
| `OutputEvent` 旁路影响桌面 | panic / 渲染异常 | `App.EmitEvent` 三行内接 bridge，bridge 无会话时零开销