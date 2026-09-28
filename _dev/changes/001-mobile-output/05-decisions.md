# 技术决策记录 05 — Mobile Output

> 只记录**真正需要拍板的技术决策**。普通 Bug 修复（如 `setBadge` 索引越界）属实现细节，见 `03-implementation.md`，不记为架构决策。

| 编号 | 决策 | 备选 | 结论与理由 | 对应文件 |
|---|---|---|---|---|
| D-Mobile-001 | 实时推送通道 | WebSocket vs SSE | **SSE**：`EventSource` 浏览器端自动重连 + 自动重发握手，无需自定义心跳协议；Go stdlib 即可，零新增依赖 | `pkg/mobile/server.go` |
| D-Mobile-002 | 手机页托管 | 外部静态目录 vs 打包 | **//go:embed 打进 exe**：单文件分发、无路径/资源泄漏；手机页只读不影响桌面 | `pkg/mobile/server.go` `web/index.html` |
| D-Mobile-003 | 手机载荷范围 | 全量思考链 vs 精简载荷 | **只传「答案 + 简短分析」**（`Analysis` 累积自 thinking 事件、`Answer` 累积自 chunk/solution），不传截图/题目大字段 | `pkg/mobile/state.go` `bridge.go` |
| D-Mobile-004 | 连接鉴权 | 无鉴权重听 vs 随机 token | **随机 session token**（`crypto/rand` hex），仅内存态、只随 URL 下发；日志/文档一律 `SET/NOT SET` 不打印实值 | `pkg/mobile/server.go` |
| D-Mobile-005 | 局域网 IP 选择 | 取首个 RFC1918 vs **候选接口评分/过滤** | **按接口名加权评分**：物理 WLAN/Ethernet 高分，vEthernet/WSL/Hyper-V/Mihomo/TUN/TAP/VPN/Docker 强负分；纯函数可单测；无理想候选走 fallback，不 panic、不影响 desktop | `pkg/mobile/server.go` `lanip_test.go` |
| D-Mobile-006 | 手机接入交互 | 二维码 + URL 兜底 vs 仅 URL | **第一版仅 URL 文案 + 一键复制**：`go-qrcode` 依赖增量超限，二维码留独立变更 D-802（后续） | `frontend/src/components/SettingsModal.vue` |
| D-Mobile-007 | OutputMode 路由边界 | 全局 EmitEvent 拦截 vs 仅白名单路由 | **仅 5 个 `solution-*` 事件走 Router**，其余（toast/screenshot/hotkey…）永远直送桌面 Wails | `app/app.go` `app/mobile.go` |

## 明确不做（防 Scope creep）

- **不升级 Wails / OpenAI 依赖**：`go.mod` 中 2.11→2.16 版本变化由本机 `wails` CLI 自动同步产生，**不属于 001**，须与业务改动分开治理。
- **不改 `Solver.Solve` / LLM / 截图 / Prompt 逻辑**：变更边界保持「旁路镜像」。
- **不落地、不持久化手机端状态**：`Bridge` 全内存态，重启即失忆（符合只读镜像语义）。