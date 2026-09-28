# 验证记录 04 — Mobile Output

> 实现的验收过程记录。对照 `00-request` 的验收标准逐项验证。**三类问题必须区分，不得混为一谈**：代码 Bug（setBadge）、环境问题（SingleInstanceLock）、网络接口选择 Bug（WSL 虚卡）。

## 构建验证

| 命令 | 结果 | 备注 |
|---|---|---|
| `go build ./...` | ✅ 通过 | |
| `go vet ./app/... ./pkg/mobile/... ./pkg/config/...` | ✅ 通过 | |
| `go test ./app/... ./pkg/mobile/...` | ✅ 通过 | app 0.50s / mobile 0.84s，含 LAN IP + TestIndexHTML |
| `npm --prefix frontend run build` | ✅ 通过 | 3.4s |
| `wails build` | ⏳ 请在用户终端执行 | `-debug -windowsconsole`；重新嵌入已修 HTML 后验证 |

## 运行时 BT 排查（已确认）

| 现象 | 根因分类 | 结论 |
|---|---|---|
| 双击 exe 无窗口、无日志、立即退出 | **环境问题**：旧 `q-solver-windows-amd64.exe`(PID 71108 from Downloads) 持有同 `SingleInstanceLock` | ✅ 非代码回归；清理后可正常启动 |
| 手机浏览器打开 URL 空白 / 停留在「等待桌面连接…未连接」 | **代码 Bug（JS parse 期）**：classic `<script>` 顶层 `return` → `Illegal return statement`，整段脚本失效（`setBadge` 修复与 `EventSource` 建连从未执行） | ✅ 已修复：启动段包进 `init()`；详见 03-implementation「Mobile Web Browser Syntax Bug」 |
| APP 显示 `172.22.16.1` 而非 WLAN IP | **网络接口选择 Bug**：`resolveLANIP` 用 `net.InterfaceAddrs()` 无接口名，误选 TUN(TUN) 私有 IP | ✅ 已重构评分策略，选中 `192.168.1.102` |

## 运行验证

> 手机与电脑同一 Wi-Fi；APP 显示 Mobile URL。

| 检查项 | 预期 | 实际 | 通过 |
|---|---|---|---|
| 新实例启动；Mobile 服务监听 | `0.0.0.0:38511` / `[::]:38511` | ✅ 实测监听 | ☑ |
| 启动日志 Mobile URL | 显示 `http://192.168.1.102:38511/?token=SET`（不含实值） | ✅ 本机 `resolveLANIP()` 实测 = `192.168.1.102` | ☑ |
| 手机页直接打开（不改 IP） | `GET /` 200 + `text/html` + body 含「Q-Solver 手机同步端」 | ✅ `TestIndexHTML` 通过；手机 Safari 实际可访问 | ☑ |
| Test A 手机连接 | 已连接 / 待命 状态 | ⏳ 待新 EXE 后核对 | ☐ |
| Test B desktop | 仅 PC 显示 thinking/analysis/answer；手机无答案 | ⏳ 待验证 | ☐ |
| Test C mobile | 手机 waiting→streaming→done；PC 不显示答案 | ⏳ 待验证 | ☐ |
| Test D dual | 双端同轮结果；仅一次 LLM 请求 | ⏳ 待验证 | ☐ |
| Test E 刷新恢复 | EventSource 重连后立即快照恢复 | ⏳ 待验证 | ☐ |
| Test F 断开 | 手机断开后 Solve 不阻塞、无 panic | ⏳ 待验证 | ☐ |

## 验收标准对照

对照 `00-request` §3 与 02-plan §4：

- [x] 构建与 Go 单测通过（含 LAN IP 纯函数、SSE 状态机、401、handshake、IndexHTML）
- [x] 手机页 JS **语法**：真实 Script 目标编译通过（`node _dev/mobile-check/check-raw-syntax.js`，`vm.Script`，顶层无非法 return）
- [x] 手机页 JS **运行时**：五状态 DOM 模拟无异常（`node _dev/mobile-check/dom-mobile-page.js`，六帧）
- [x] LAN IP 自动发现选中物理 WLAN（实机 = 192.168.1.102）
- [ ] desktop / mobile / dual 三模式真实截图解题（待新 EXE）

### 「Mobile Web Browser Syntax Bug」针对性验证

> 后端 curl 实测（2026-09-27）：`GET /` 200 + text/html；`GET /sse?token=<valid>` 200 + text/event-stream，snapshot 首帧 + heartbeat 正常。**后端始终健康**，断点在纯前端 parse。

| 检查项 | 预期 | 实际 | 通过 |
|---|---|---|---|
| `check-raw-syntax.js`（修复后） | 脚本语法通过 | ✅ 通过 | ☑ |
| 同一 `vm.Script` 兜底能力（修复前代码片段） | 抛 `SyntaxError: Illegal return statement` | ✅ 实测捕获 | ☑ |
| `dom-mobile-page.js` 六帧 | 无运行时异常 | ✅ 通过 | ☑ |
| `go test ./pkg/mobile/...` | 全绿（含 TestIndexHTML） | ✅ 0.768s | ☑ |
| `go build ./...` | 通过 | ✅ 通过 | ☑ |
| 连接层 vs 业务层 | conn-line 走连接状态，badge 走 snapshot.status | ✅ 代码已分离（见 03） | ☑ |
| `wails build -debug -windowsconsole` + 手机真机 | — | ⏳ 需用户在终端执行（本机无 `wails` CLI） | ☐ |

## 回归与副作用检查

- `pkg/config` `Validate()` 对旧配置 `OutputMode:""` 宽容 → 老配置兼容，无需迁移（已核实 `c.OutputMode != ""` 判断）。
- `app.EmitEvent` 旁路仅作用于 5 个 `solution-*` 事件；`toast`/`screenshot` 等始终走桌面。
- LAN IP 重构：`scoreIfaceName` 纯函数 8 用例通过；空候选 / 全虚卡不 panic，fallback 安全。

## 结论

代码层验证通过，含「Mobile Web Browser Syntax Bug」确认后的 JS 语法护栏闭环。

**未决项**：真实 `wails build -debug -windowsconsole` + 手机端 Test A–F 端到端验收，需用户在终端执行（本环境无 `wails` CLI）。前一轮怀疑的「手机未连接」已由本次根因确认排除——浏览器因顶层 return 拒绝执行整段脚本，`setBadge` 修复只有在 `init()` 修复后才会真正生效（重新构建进 `//go:embed`）。