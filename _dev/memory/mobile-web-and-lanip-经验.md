# Mobile Web 与 LAN IP 经验（Change 001 沉淀）

> 类型：已确认 Bug 经验（Confirmed）
> 来源：`_dev/changes/001-mobile-output/03-implementation.md`、`04-validation.md`
> 写入：2026-09-27 ｜ 归属：Q-Solver 二次开发

## 经验一：嵌入页 JS 运行时异常 + 语法异常，Go 单测都无法暴露

> 长期约束 — Mobile Web UI 必须包含真实浏览器/DOM 验证，且语法检查必须用 `vm.Script`（Script 目标），而非 `node --check`。

Q-Solver Change 001 曾出现「Go 侧全部单测（HTTP/SSE/状态机/鉴权）全绿，但手机浏览器打开页面即空白」的缺陷。**两次叠加，根因升级**：

**第一层（已修复）**：`pkg/mobile/web/index.html` 的 `setBadge()` 用 `statusBadge.childNodes[2].textContent = text` 写文案，而 DOM 只有 `childNodes[0]`/`childNodes[1]` → `TypeError`，首帧 snapshot 渲染中断 → 空白。改为 DOM 重建。

**第二层（真正元凶，2026-09-27 确认）**：classic `<script>`（非 module）**顶层**直接 `return;`（无 token 提前退出分支）→ 浏览器 **parse 期** `SyntaxError: Illegal return statement`，整段脚本在解析时失效，`EventSource`/`snapshot` 监听/`setBadge` 修复**统统没执行** → 页面永远停留静态初始态。修复：启动段包进 `init()` 函数，`return` 只作用于函数内部。

**为什么 Go 测试测不出**：`//go:embed` 打进 exe 的静态页面在端到端 HTTP 测试里只校验 `200`/`Content-Type`/`含固定文本`，JS 语法/运行时在 Go 侧断言触达不到。

**对策（两层互补，入库为回归脚手架）**：
- 语法层：新增 `_dev/mobile-check/check-raw-syntax.js`——从 `index.html` 原样提取 `<script>` 内联 JS，用 **`vm.Script`**（Node `vm` 模块）按浏览器 classic `<script>` 的 **Script 目标**编译，顶层 `return`/非法语句直接抛 SyntaxError，退出码非 0 即失败。
  - ⚠️ **不要用 `node --check`**：实测它对顶层 `return` 返回 0（`.js` 按 CommonJS 模块语义解析，模块本身包在函数里，顶层 return 合法）——与 `new Function` 同源假阴性，误导性极强。
- 运行时：`_dev/mobile-check/dom-mobile-page.js`——DOM shim 执行内联 JS，模拟 SSE 推 idle/waiting/streaming-thinking/streaming-content/done/error 六帧，断言零运行时异常（捕获「收到 snapshot 后渲染异常」）。注意它用 `new Function` 执行，会对顶层 return 假阴性，故**必须**与 check-raw-syntax 互补。
- `TestIndexHTML`：校验 `GET /` 200 + `text/html` + body 含「Q-Solver 手机同步端」；无 token 也应 200。
- 每当改动 `web/index.html`，三层都跑：`check-raw-syntax.js` → `dom-mobile-page.js` → `go test`，不能只依赖 `go test`。

## 经验二：LAN IP 自动发现必须排除虚拟网卡

> 长期约束 — Mobile LAN IP discovery 必须避免选择 WSL/VPN/TUN 等虚拟接口。

`resolveLANIP()` 原实现用 `net.InterfaceAddrs()`（**无接口名**）取「第一个 RFC1918 IPv4」。在装有 WSL/Hyper-V + 代理（Mihomo/Meta）的机器上，接口枚举顺序不可控，会误选 `vEthernet (WSL)` 的 `172.22.x.x` 或代理虚拟网卡地址作为手机接入 URL——手机在同一 Wi-Fi 也连不上。

**对策**：改为 `net.Interfaces()` + 接口名加权评分：
- 虚卡关键词（vEthernet/WSL/Hyper-V/Mihomo/Meta/Clash/TUN/TAP/VPN/Tunnel/Docker/Virtual/Loopback/Bridge/Tailscale/ZeroTier/WireGuard/utun）→ 强负分；
- 物理网络（WLAN/Wi-Fi/Wireless/以太网/Ethernet）→ 高分；
- 链路本地 APIPA（169.254/16）→ 直接排除。
- 选择逻辑抽为纯函数 `chooseBestIPv4(candidates)` + `scoreIfaceName(name)`，注入候选数据可单测；无理想候选时安全 fallback / 127.0.0.1，不 panic、不影响 desktop 模式。

**本机实测**：评分后选中 `WLAN` 的 `192.168.1.102`（屏蔽了 WSL 172.22.16.1、Mihomo 198.18.0.1、APIPA）

## 附录：本次检查「不做」项

- 未记录 PID、临时端口、临时日志、用户 token（含 URL 中的）——policy 禁写。
- 未新增任何第三方依赖：SSE（stdlib `net/http`）、页面（Go embed）、token（`crypto/rand` hex）。