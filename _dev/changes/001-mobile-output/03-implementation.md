# 实现过程记录 03 — Mobile Output

> 边做边记录：时间线、每步改动、关键决策与偏差。

## 实现日志

| 时间 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-09-26 | Step 1 | 新增 `pkg/mobile/` 包：状态机、事件桥、SSE Server、嵌入手机页 | `pkg/mobile/state.go` `bridge.go` `server.go` `web/index.html` | 只新增，不动既有业务；Go stdlib `net/http` 实现 SSE，不引第三方依赖（决策 D-数字见 05） |
| 2026-09-26 | Step 2 | `App.EmitEvent` 接入 OutputMode Router：仅 `solution-*` 事件进路由，其余永远走桌面 | `app/app.go` `app/mobile.go` | 约束一：事件输出路由不扩大影响面 |
| 2026-09-26 | Step 2.2 | 桥事件 → 快照映射，含 `payload.model` 注入 | `pkg/mobile/bridge.go` | Payload 结构对齐 D-Mobile-003 |
| 2026-09-26 | Step 5 | 配置 `OutputMode` 字段 + 前端三态 + 手机连接面板 | `pkg/config/config.go` `frontend/src/*` `frontend/wailsjs/go/app/*` | 手动同步 wailsjs 绑定（wails CLI 未装），不改 `wails.json` |
| 2026-09-27 | Bug 1 | 手机页空白：修复 `setBadge` childNodes[2] 越界（**表象**；真正的元凶是顶层 return，见「Mobile Web Browser Syntax Bug」） | `pkg/mobile/web/index.html` | `childNodes` 只有 0/1，越界会抛 TypeError；但此前整段 JS 已在 parse 期被顶层 return 杀死，越界从未真正执行到 |
| 2026-09-27 | Bug 2 | 修正 `snap.model` → `payload.model` 读取 | `pkg/mobile/web/index.html` | Go 端 model 在 `payload` 内层 |
| 2026-09-27 | 网络 | 重构 LAN IP 发现：虚拟网卡评分过滤，选中 WLAN 192.168.1.102 | `pkg/mobile/server.go` `lanip_test.go` | WSL/Mihomo 虚卡不被选为首选接入地址 |
| 2026-09-27 | 测试 | 补充 `TestIndexHTML` + DOM 五状态回归脚本 | `pkg/mobile/bridge_server_test.go` `_dev/mobile-check/dom-mobile-page.js` | Go 单测无法发现嵌入页面 JS 运行时异常 |
| 2026-09-27 | Bug 1.2 | 确认「Mobile Web Browser Syntax Bug」= classic `<script>` 顶层 `return`，整段 JS parse 期失败 | `pkg/mobile/web/index.html` | 确定性根因：`return;` 位于脚本顶层，浏览器报 `Illegal return statement`；此前所有 HTTP/SSE/单测/`setBadge` 修复都在脚本执行前被一并废弃 |

## Mobile Web Browser Syntax Bug（确认根因，2026-09-27）

### 现象

手机打开 APP 给出的 URL：页面停留静态初始态（等待桌面连接… / 未连接 / 正在连接…），既不是等待/已连接，也不会通过 snapshot 更新。

### 确认根因

`pkg/mobile/web/index.html` 的 classic `<script>`（**非 module**）顶层直接写了：

```js
if (!token) {
  setStatus('无连接', 'error');
  setState('缺少连接凭据。请从桌面端设置中获取二维码/URL。', true);
  return;   // ← 顶层 return
}
```

classic script 顶层 `return` 是 **parse 期语法错误**（`SyntaxError: Illegal return statement`）。浏览器解析整段脚本即失败，`EventSource` 从未创建、`snapshot` 监听从未注册、之前修好的 `setBadge`/`payload.model` 也从未执行——页面永远停留在初始 HTML。

后端 curl 实测一切正常：

- `GET /` → `200` + `text/html` + 固定文本
- `GET /sse?token=<valid>` → `200` + `text/event-stream`，snapshot 首帧 + heartbeat `ping` 正常

因此**这不是** SSE 故障 / 网络故障 / Token 故障——后端自始自终是好的，断在纯前端 parse。

### 为什么原 DOM 回归脚本没捕获

`_dev/mobile-check/dom-mobile-page.js` 用 `new Function('console', inlineJs)` 执行内联 JS。`new Function` 会把脚本体**当成函数体**包裹，于是「顶层 return」变成「函数内 return」→ **合法**，同一类 false negative。连 `node --check` 也不行：`.js` 按 CommonJS 模块语义解析，模块本就包在函数里，顶层 return 同样合法。

### 修复（结构性最小，不改协议）

将启动段整体包进 `init()` 函数（`return` 只作用于函数内部），末尾 `init()` 调用。SSE 数据协议（`/sse?token=`、snapshot/EventSource 事件名、payload 形状）**零改动**。同时借机把两层概念显式分开：

- **连接层**（`conn-line`）：连接中… / 已连接 / 已连接 · 同步最新 / 连接断开，正在自动重连…
- **业务层**（badge，由 `snapshot.status` 驱动）：待命 / 等待 / 流式中 / 已完成 / 出错

其余不动：server.go SSE、Token 鉴权、LAN IP、Wails 生命周期、LLM、Solve、Prompt、OutputMode 均未触碰。

### 修复后的 JS 回归护栏（两层互补）

| 脚本 | 作用 | 捕获缺陷类别 |
|---|---|---|
| `check-raw-syntax.js`（新增） | `vm.Script` 真实 Script 目标编译 `index.html` 提取的内联 JS | **parse 期**：顶层 return / 非法语句 |
| `dom-mobile-page.js`（保留） | DOM shim + 模拟 SSE 六帧 | **运行时**：收到 snapshot 后渲染异常 |

`vm.Script` 选用理由（已实证）：它按浏览器 classic `<script>` 语义编译；`node --check` 对顶层 return 实测返回 0（CommonJS 语义包函数），同属假阴性，**不可靠**。验证记录见 `04-validation.md`。

## 与计划的偏差

| 02-plan 原计划 | 实际偏差 | 原因 |
|---|---|---|
| 手机端展示「二维码为主、IP 手动输入兜底」 | **第一版只显示 URL + 一键复制，不做二维码** | 用户确认方案 A：`go-qrcode` 依赖增量过大，后续单独立项 |
| `bindings` 由 `wails generate` 自动生成 | 手动编辑 `frontend/wailsjs/go/app/App.js` + `.d.ts` | 本机无 `wails` CLI；改动 shape 极小（只加一个 binding），避免引入 Wails 版本升级 |
| `pkg/mobile/server.go` 展示真实 IP 选择 | 加入接口名评分过滤策略后再选 IP | 实测发现 `net.InterfaceAddrs()` 无接口名，会误选 WSL/vEthernet 虚卡 IP |

## 关键决策纪要

- **SSE 方案保留**：本机 `gorilla/websocket` 虽已被使用，但手机端 `EventSource` 自带自动重连，无心跳协议开销，实现最简。
- **Go embed 保留**：手机页作为静态资源打进 exe，避免运行时依赖外部文件。
- **不升级 Wails 版本**：`wails/v2` 的版本升级（2.11→2.16）是环境里 `wails` CLI 自动行为，**不属于本变更**，见第十一步清点。
- **LAN IP 采用候选接口评分/过滤**：物理网卡（WLAN/Ethernet）高分，虚卡（vEthernet/WSL/Mihomo/TUN/…)强负分，找不到理想接口时安全 fallback 或 127.0.0.1，不影响 desktop。

## 遗留未完成项

- [ ] 二维码能力：后续独立变更（D-802），当前版本手机端以 URL 文本接入。
- [ ] `frontend/wailsjs/runtime/*`、`go.mod`/`go.sum` 中 Wails 版本噪声待用户决定是否清理（见治理清点）。