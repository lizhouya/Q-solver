# 现状快照 — 01 · 001-mobile-output

> 动手修改前，对将**涉及的相关代码/配置做客观快照**。目的：后续可以对比"改动前 vs 改动后"，也能作为回滚依据。
>
> **本文件在 02-plan 之前完成**，且基本不再变动（改动期间的演进记录放 `03-implementation`）。
> 快照日期：2026-09-26。基线：`dev/qsolver-custom`（HEAD `b34bb04`）。

---

## 1. 涉及文件清单

### 1.1 后端（Go）— 现状职责

| 文件 | 当前职责摘要 | 关键行/符号 | 备注 |
|---|---|---|---|
| `main.go` | Wails 入口；`Options.OnStartup` 指向 `app.Startup`、`OnShutdown` 指向 `app.OnShutdown`；`//go:embed all:frontend/dist` 承载桌面 AssetServer | main.go:40（Startup）/ main.go:60（Shutdown） | 移动端接入点之一（仅在需要时改） |
| `app/app.go` | `App` 结构体 + 生命周期；`EmitEvent` 是**唯一事件出口** | app.go:18-30（struct）、app.go:43-72（Startup）、app.go:86-93（OnShutdown）、app.go:95-97（EmitEvent） | `EmitEvent` = `runtime.EventsEmit`，天然可作 mobile 镜像点 |
| `app/solve.go` | 截图缓冲 + 触发解题入口 | solve.go:77-125（`TriggerSend`）、solve.go:135-153（`solveInternal` 装配 `solution.Callbacks`） | `solveInternal` 内 `cb := solution.Callbacks{EmitEvent: a.EmitEvent}` |
| `pkg/solution/solution.go` | 核心 `Solver.Solve`；`GenerateContentStream` 流式产出 | solution.go:45-139（`Solve`）、solution.go:92（流式回调）、solution.go:133-135（`solution` 事件） | **后端不存储答案**；`chatHistory` 虽存在但未真正接入（F-001） |
| `pkg/config/config.go` | 配置结构体 + 默认值 + 校验 | config.go:9-36（`Config`）、config.go:40-70（`NewDefaultConfig`）、config.go:111-122（`Validate`） | **现无 `outputMode` 字段，需新增** |

### 1.2 前端（本次）— 现状职责

| 文件 | 当前职责摘要 | 关键行/符号 | 备注 |
|---|---|---|---|
| `frontend/src/stores/solution.js` | 解题结果 Pinia store；流式缓冲 + 状态机 | `handleStreamStart`(112)、`handleStreamChunk`(151)、`handleThinkingChunk`(178)、`handleSolution`(211)、`errorState`(35) | 桌面渲染的核心状态保存地（内存态） |
| `frontend/src/App.vue` | 应用壳；订阅 solution 事件并驱动渲染；settings 弹窗 | App.vue:87-258 事件订阅区；`SolutionView`/`SettingsModal` | **OutputMode 渲染开关落点**（桌面何时渲染 DOM） |
| `frontend/src/services/events.js` | `runtime.EventsOn` 封装 | `on(event, handler)` | 桌面事件订阅抽象 |
| `frontend/src/stores/settings.js` | 设置 store；`buildConfigToSave` 组装保存对象 | settings.js:257-277（`buildConfigToSave`）、settings.js:145-165（`loadSettings`） | **现无 `outputMode`，需新增配置键** |
| `frontend/src/components/SettingsModal.vue` | 设置弹窗 UI（截图/模型/快捷键等） | — | **OutputMode 切换 UI 落点** |
| `frontend/src/components/SolutionView.vue` | 展示答案/流式内容（历史卡片） | — | mode 为 `mobile` 时可跳过渲染的地方 |

### 1.3 前端移动端（现状）— 无

当前**不存在**任何移动端页面/构建。`frontend/dist` 仅打包桌面 SPA，由 `//go:embed` 嵌入。

---

## 2. 关键代码现状（原文摘录，未修改）

### 2.1 事件出口（唯一）— `app/app.go:95-97`

```go
func (a *App) EmitEvent(eventName string, data ...interface{}) {
	runtime.EventsEmit(a.ctx, eventName, data...)
}
```

> 结论：桌面一切事件的**唯一发布点**。Mobile 镜像即在此时同步一份到手机端。

### 2.2 解题 Callbacks 装配 — `app/app.go:148-152`

```go
cb := solution.Callbacks{
	EmitEvent: a.EmitEvent,
}
```

### 2.3 事件名序列 — `pkg/solution/solution.go`

| 事件 | 触发位置 | 载荷 | 语义 |
|---|---|---|---|
| `solution-stream-start` | solution.go:88-90 | — | 流开始（`waiting`） |
| `solution-stream-thinking` | solution.go:99 | chunk | 思考链流式 |
| `solution-stream-chunk` | solution.go:100-101 | chunk | 正文流式 |
| `solution` | solution.go:133-135 | `response.Content` | 完整答案（`done`） |
| `solution-error` | solution.go:109 / 116 / 128 | error 字符串 | 出错（`error`） |

### 2.4 前端 Store 消费（`stores/solution.js`）

- `App.vue` 通过 `services/events.js` 的 `on()` 订阅上述事件 → 调用 `store.handleStreamStart / handleStreamChunk / handleThinkingChunk / handleSolution`。
- 前端不直接接触后端 LLM，仅依赖事件 + 载荷。

### 2.5 配置持久化 — `pkg/config/config.go`

- `Config` 当前无与输出/移动端相关字段；`Validate` 校验 `screenshotMode` 只能是 `fullscreen` 或 `window`。

---

## 3. 相关配置

| 配置项 | 当前值 | 备注 |
|---|---|---|
| `.env` / API Key | `SET`（不展示实值） | — |
| `wails.json` | outputfilename `Q-Solver`，embed `frontend/dist` | 移动端页面 embed 需单独挂载 |
| screenshotMode 默认 | `fullscreen`（`NewDefaultConfig` 中） | 前端 settings.js 默认 `'window'`——两处默认不一致（已有，不在本次处理） |
| OutputMode | **不存在** | 本次新增键 |

---

## 4. 依赖与构建现状

- `go.mod`：`github.com/wailsapp/wails/v2 v2.11.0`（直接）；**`github.com/gorilla/websocket v1.5.3` 已在间接依赖树**（Wails）——SSE 用标准库 `net/http` 即可，**不发生 WebSocket**。
- 移动端如需二维码：当前无 `go-qrcode` 依赖，需新增（`github.com/skip2/go-qrcode`）→ **若依赖增加过大，则第一版仅显示 URL（用户已授权此可能，见 00-request §6）**。
- 构建命令：`go build ./...`（后端）、`npm run build`（前端，Vite → `frontend/dist`）、`wails build`（产物）。

---

## 5. 运行行为现状

- 桌面运行：`wails develop`/`wails build` 产出 WebView2 桌面态；`main.go:38-47` 幽灵透明窗口 + `click-through`。
- 解题流程：截图 → `TriggerSend` → `Solver.Solve` → 事件流 → `solution.js` → 桌面 `App.vue` 渲染渲染。
- 桌面渲染与事件**强解耦**（唯一耦合点是 `App.EmitEvent`），Mobile 只需在 `AppEmitEvent` 处旁路。
- 当前**无任何局域网端口/网络服务**（纯桌面应用）。

---

## 6. 已知现状问题（本次不处理）

| 编号 | 现象 | 说明 |
|---|---|---|
| F-001 | `KeepContext` 后端无效 | `solution.go` 中 `chatHistory` 未接入 `Solve` 的消息组装；Mobile 只针对当前单轮，不依赖此 |
| F-002 | `solve` 快捷键未绑定 | 手机端第一版只读，不依赖 |
| F-004 | Win `SetWindowLevel` 空实现 | 移动端并存桌面；desktop 模式不受影响；幽灵模式下配手机主屏，可接受 |
| F-012 | `logger.Println` 打印完整 system prompt + 简历 | 与 mobile 无关，但局域网页面**不**外传原始简历；注意不在 mobile 内容中含简历全文 |

> 以上问题**均非本次变更目标**，仅记录交接。