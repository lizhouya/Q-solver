# Change 026 现状快照

日期：2026-10-25
状态：业务代码修改前快照。

## 1. 工作树与基线

- 当前分支：`dev/qsolver-custom`，跟踪 `origin/dev/qsolver-custom`。
- 远程：`origin=https://github.com/lizhouya/Q-solver.git`；`upstream=https://github.com/jym66/Q-solver.git`。
- 工作树包含 Change 024/025 的未提交代码、测试、bindings 与治理文件；本 Change 不清理或覆盖。
- 依赖哈希：
  - `go.mod`: `ead5d49037577845346967f41fab9dc04138f75f500d09387b306ccce5738595`
  - `go.sum`: `22174ab9fc52ddc6ac87f6862961a0c5dddcb254bb05b0e30db300eaefbb7014`
  - `frontend/package.json`: `0595e7f0957dd72c62d42dc1a310498d4ae2a3ab6a5e1efb198ac6a2f06b846a`
  - `frontend/package-lock.json`: `827a54f1049d6d143ead365c00c6c0870d6f9eabde57f77ec0926c061f63d725`

## 2. typed-nil probe 返回缺陷

`app/llm.go` 当前把 `ProbeResult.Error` 传给接受 `error` interface 的 `providerErrorDTO`。成功结果是静态类型为 `*llm.ProviderError` 的 nil 指针，转换后 interface 非 nil；`errors.As` 返回 true，但目标指针仍为 nil，后续解引用触发 panic。

后端 probe 本身已有 15 秒 context timeout，但 mapper panic 发生在成功请求结束后；Wails callback 没有正常 resolve/reject，前端 Promise 因无 deadline 永久 pending，`finally` 不执行。失败路径已有 DTO 测试，成功经 App DTO 的路径没有覆盖。

## 3. 当前连接配置与 provider 路由

- `config.Config` 当前有 `apiKey`、`baseURL`、`apiProtocol`、`model`，无 `connectionMode`。
- `protocol.Normalize` 使空协议兼容为 `openai_chat`，并拒绝未知非空值。
- `NewProvider(config.Config)` 只按 `apiProtocol` 在 Chat、Responses、Anthropic adapter 间路由。
- Change 025 已统一生产入口走 `NewProvider`；模型名、品牌和 URL 当前不参与协议选择。
- `ConfigManager` 已采用候选快照事务：Update/Patch 先校验并 `saveSnapshot`，成功后再替换内存并在锁外通知 subscriber。
- 当前请求配置允许 http/https 且拒绝 userinfo，但没有 CCR 专用 root 规范化规则。

## 4. 当前模型目录和 probe

- `Service.ListModelsForConnection` 创建当前协议 provider，15 秒 timeout 后调用 `GetModels`，再去重保序。
- `Service.ProbeConnection` 先校验完整连接、创建 provider，再进行 text/vision probe。
- text probe 使用随机 `QSP-...` token，非流式单请求；vision probe 使用合成 PNG 和流式多模态路径。
- Wails DTO 为 `ConnectionDraft`、`ModelListResult`、`ModelProbeRequest/Result`；draft 尚无模式字段。
- 前端 API wrapper 直接等待 Wails Promise，没有界面级 deadline。

## 5. CCR 能力边界

已核实 CCR 暴露 `/v1/models` 与 Chat/Responses/Messages 入口。目录主要返回模型 ID 等目录字段，不能证明某模型应由 Q-Solver 选择哪种入站协议，也不能证明图片/PDF 能力。

因此本 Change 不做协议猜测。CCR 作为网关时，Q-Solver 固定使用 OpenAI Chat Completions 入站；上游 provider 和协议转换由 CCR 管理。

## 6. 前端连接生命周期现状

`frontend/src/stores/settings.js` 当前把 `apiKey + baseURL + model + apiProtocol` 放在同一 watcher。任一变化都会递增同一个 connection revision 并清除 `availableModels` 和 probe 状态。因此手填或选择 model 也会清空目录，不适合 CCR 模型切换。

当前只有 `sessionRevision`、`connectionRevision` 和三个 request version，没有独立的 catalog/probe identity，也没有“同 revision 在途复用/成功缓存”。

## 7. 模型页布局现状

- `.modal-content` 固定 `height: 580px; max-height: 85vh`。
- `.modal-body` 为剩余高度且 `overflow: hidden`。
- 普通 `.tab-pane` 可以纵向滚动，但 `.model-tab` 又覆盖为 flex + `overflow: hidden`。
- 上方模型和 probe 区不收缩；`.domain-group` 依赖 `flex: 1` 和剩余高度。
- 父组件写 `:deep(.domain-grid)`，子组件真实列表类名为 `.domain-list`，覆盖不生效。

这会使场景区被压缩或裁剪，且外层没有滚动出口。

## 8. 受保护程序基线

| 文件 | 大小 | SHA-256 |
|---|---:|---|
| `Q-Solver.exe` | 23126528 | `5dcabb17933bfa9ab4dc15f289b4b8e50af99879c316b58831b09bbb4e39aa11` |
| `Q-Solver-change017-validation.exe` | 23291392 | `3b85644beb8a726c7138d01ce843526538b4d028aea80fd75403a0844bd338c5` |
| `Q-Solver-change018-continuation-validation.exe` | 23363072 | `775196521b00528824e4620cc4df1ad80c8bbb78c7820d40c626b528f67fe8e0` |
| `Q-Solver-change018-layout-validation.exe` | 23363584 | `853e64c3ccd59101317d0fc4e92414a10f1d12695deed9694b0d689ee2e07343` |
| `Q-Solver-change019-answer-summary-preview-validation.exe` | 23366656 | `cd4b354ef37613029d2bb0e39b9d62fd924480aefa0eafbf762ca60135d19cf7` |
| `Q-Solver-change020-round-history-selection-validation.exe` | 23369216 | `3ea25ab19009d373c304f440c126bfeb18ce7c0d4c218d8e9be8204abdaa75bf` |
| `Q-Solver-change021-start-next-exam-validation.exe` | 23387648 | `95f4b3c17b96db0e25fa682ed337bb067458046784a09d3580d1eff4839e3c4c` |
| `Q-Solver-change022-answer-copy-numbering-validation.exe` | 23440896 | `5cde3df3a8bdd160f7740ffeebc6b68c7dcb3f87bbfc5ff7226ce6bf2950e751` |
| `Q-Solver-change023-numbered-answer-copy-validation.exe` | 23443456 | `44fbb9381179b0d739003c53df1abe9c06847255447f87ff786fdbe8a711d339` |
| `Q-Solver-change024-organize-failure-recovery-validation.exe` | 23464960 | `1953484b5fb42576f2db013691bdc6190f7a60ce7e1aabbdadad539b3f35d274` |
| `Q-Solver-change025-multi-protocol-validation.exe` | 44017664 | `589443f464f0e3f0c8adf5c6100beed81d5380d1c2ac98eb54b7f0a0bf6efa37` |

## 9. 已知但不在本 Change 解决的问题

- Change 024/025 的 Windows GUI 和真实 provider 项仍为 `NEEDS_USER`。
- Windows race 启动阶段历史上返回 `0xc0000139`，当前事实为环境阻塞而非业务竞态结论。
- Change 024/025 尚未提交；本 Change 不代替其提交边界审计或授权。
