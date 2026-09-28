# 现状快照 — 01 · 002-remote-output-ui

> 动手修改前，对将涉及的相关文件做客观快照。改动期间的演进记录放 `03-implementation.md`。
> 快照日期：2026-09-27。基线：`dev/qsolver-custom`。

---

## 1. 涉及文件清单（本次变更范围内）

### 1.1 `pkg/mobile/web/index.html`（唯一业务源码改动点）— 现状

| 部分 | 现状 | 关键行 | 备注 |
|---|---|---|---|
| 文档结构 | `<!doctype html>` + `<head>`(内联 `<style>`) + `<body>`(内联 `<script>`) 单文件 | 整文件 321 行 | Go `//go:embed` 直接嵌入，无外部 js/css |
| CSS | 深色主题；`.card`/`.badge`/`.thinking-box`/`.md`/`.error-box`/`.waiting` 等 | style 块 line 9-113 | 当前是**单栏**贯穿所有宽度，无 `@media` 断点 |
| 卡片 | `waiting-card` / `thinking-card`(分析过程) / `answer-card` / `error-card` | line 128-151 | **thinking-card = 分析过程展示，002 将不再显示** |
| JS-状态入口 | `setBadge` / `setConn` / `setStatus` / `setState` / `applySnapshot` | line 213-275 | 业务状态走 badge；连接走 conn-line（002 沿用分离语义） |
| JS-启动 | `init()`（含 token 读取 + EventSource 建连 + visibilitychange 重连） | line 277-318 | 001 的 Mobile Web Browser Syntax Bug 修复成果；**保留** |
| JS-Markdown | `renderMarkdown()`（escape → inline → block 拆分） | line 156-200 | 支持代码块/引用/标题/列表/粗体/行内代码；**无表格**；需按 002 增强与校验 |
| JS 无历史 | 仅有「当前最新 snapshot」（`applySnapshot` 直接渲染） | — | **002 将新增 sessionHistory / selectedSeq / latestSnapshot** |
| JS 无去重 | 无历史，故无去重逻辑 | — | 002 需按 `seq` 去重 |

### 1.2 `_dev/mobile-check/`（测试脚手架）

| 文件 | 现状 | 作用 |
|---|---|---|
| `check-raw-syntax.js` | `vm.Script` 真实 Script 目标编译 `index.html` 提取内联 JS；顶层 return → 抛错 | 语法层护栏（保留，不改） |
| `dom-mobile-page.js` | DOM shim + 模拟 SSE 六帧（idle/waiting/streaming-thinking/streaming-content/done/error），断言零异常 | 运行时/行为护栏。**需扩展**：当前六帧里 thinking 帧断言的是「能处理」，002 后要求「streaming-thinking 不显示 analysis、能进 history、去重、点击切换」 |

### 1.3 后端（`pkg/mobile/**`）— **本次不改，仅记录协议**（供 002 实现对齐）

| 文件 | 关键内容 | 002 使用方式 |
|---|---|---|
| `pkg/mobile/state.go` | `Payload{Question, Analysis, Answer, Model}`；`Snapshot{Seq,Ts,Status,Stage,Payload,Error}`；状态 `idle/waiting/streaming/done/error`；阶段 `thinking/content` | Remote UI 只读：**Analysis 不渲染**；Answer 渲染；`seq` 用作去重键 |
| `pkg/mobile/server.go` | SSE `/sse?token=` + snapshot 首帧 + heartbeat | 不变 |
| `pkg/mobile/bridge.go` | `Bridge.Publish` 映射事件 → snapshot | 不变 |

### 1.4 前端 PC 参考（不入本变更，仅视觉参考）

- `frontend/src/components/SolveView.vue`：左 `left-panel`(200px) + 右 `right-panel`(flex:1)；`HistoryItem`(摘要/时间/active) + `ThinkingBlock`(分析)。002 tablet 布局视觉上参考「左历史 + 右答案」，但**不复用 Vue 代码**（远程为纯标准 HTML）。

---

## 2. 关键现状行为

1. **业务状态**：`applySnapshot(snap)` 按 `snap.status` 切换 `waiting/streaming/done/error/待命`。streaming 阶段 thinking 会展示 `payload.analysis`（002 将停止）。
2. **连接状态**：`setState('已连接 · 同步最新')` 等（保留）。
3. **历史**：当前**无**。刷新（SSE 重连）后仅收最新 snapshot，手机上看到最新一题（001 行为，000 保持）。
4. **Markdown**：`renderMarkdownToHTML` 存在但**不含表格**、部分列表样式朴素（002 按 §12 精修视觉层级但不改安全模型）。

---

## 3. 相关配置

| 项 | 值 | 备注 |
|---|---|---|
| `pkg/mobile/web/index.html` | Go embed 单页（无 Vue/npm） | 本次唯一业务源码改动 |
| `ssHeartbeatInterval` | 20s | 不变 |
| SSE URL | `/sse?token=<token>` | 不变 |
| 端口 | 38511（`DefaultPort`） | 不变 |

---

## 4. 构建与运行现状

- 构建：`wails build`（`//go:embed` 打包 index.html）；本环境 `wails` CLI **不可用**（`command not found`），自行组件 web 修改后可用 `go test ./pkg/mobile/...` + 两个 JS 护栏验证，`wails build` 由用户在终端执行。
- 桌面回归：后端协议不改 → `go build` 应无差异。