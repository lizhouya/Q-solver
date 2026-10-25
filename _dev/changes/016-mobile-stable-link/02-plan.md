# 实施方案 — 02

## 1. 方案概述

Mobile Token 与普通 `config.Config` 完全分离。新增专用状态管理器，以现有 AES-GCM 加密并使用双槽文件事务保存固定模式、Token 与单调 `clearSeq`；Bridge 增加每进程随机 `sessionId` 和可重放清屏边界；Server 由外部注入 Token，并通过认证代次撤销旧请求和活动 SSE；App 用 `mobileOpMu` 串行所有 Mobile 生命周期操作；每个解题任务捕获 display epoch。桌面端以共享 Pinia store 驱动设置页、TopBar 与唯一确认层。

## 2. 分步实施计划

| 步骤 | 操作 | 预期产出 | 完成标志 |
|---|---|---|---|
| 1 | 专用 Mobile 状态管理 | 固定/临时 Token 与 clearSeq 的加密双槽存储 | 单槽损坏可恢复，失败不切 manager current |
| 2 | Bridge 协议扩展 | `sessionId + clearSeq`、原子代际发布、可靠清屏屏障 | 旧 epoch 不能广播，满队列仍收到 Clear |
| 3 | Server 鉴权生命周期 | Token 注入、全路由鉴权、SSE lease 撤销、无锁等待 Stop | 旧 URL 与旧 SSE 失效 |
| 4 | App 编排与任务代际 | Clear/Reset/固定模式 API、shutdown 栅栏、task-scoped emitter | 先落盘后切运行态，旧任务不复活 Mobile |
| 5 | Mobile 页面 | 跨进程 session、clear 水位、DOM/history/rAF 清理 | 刷新重连不恢复旧内容 |
| 6 | 桌面前端 | 共享 store、设置管理、TopBar 清屏、唯一确认层 | 双入口状态一致且无 URL 泄漏 |
| 7 | bindings、文档与验证 | Wails bindings、README、Change 记录、自动/人工矩阵 | 自动结果可复现，人工缺口如实标注 |

## 3. 涉及业务源码的改动清单

以下为允许修改的业务与测试文件；不得未经记录越过清单：

### 配置与专用状态

- `pkg/config/mobile_state.go`（新增）：双槽持久化和运行态 Token 管理。
- `pkg/config/mobile_state_test.go`（新增）：恢复、事务、模式和泄漏测试。
- `pkg/config/manager.go`：仅抽取复用应用配置目录解析，保持普通 Config 行为。

### Mobile 协议与服务

- `pkg/mobile/state.go`：Snapshot 新增 `sessionId`、`clearSeq`。
- `pkg/mobile/bridge.go`：session、clear 屏障、`PublishIfCurrent`、Close。
- `pkg/mobile/server.go`：Token 注入、首页/SSE 鉴权、认证代次、SSE lease、Stop。
- `pkg/mobile/bridge_server_test.go`：协议、鉴权、撤销、队列与生命周期测试。
- `pkg/mobile/web/index.html`：session/clear 协议入口、清 DOM/history/rAF、外链保护。
- `pkg/mobile/web_dom_test.mjs`：新 session、清屏、旧帧和滚动回归。

### App

- `app/app.go`：Mobile manager、operation lock 和 closing 状态。
- `app/mobile.go`：setup/Retry/Clear/Reset/固定模式 API 与结果 DTO。
- `app/solve.go`：任务捕获 display epoch 和 task-scoped Mobile 路由。
- `app/mobile_commands_test.go`（新增）：确认门、持久化失败、未监听 mutation、shutdown 栅栏和任务显示代际测试；同时提供 App Mobile 测试共用的 fake manager。
- `app/mobile_test.go`：适配新 Bridge/Server 构造器并注入 fake manager，验证状态查询与 Retry。
- `app/phase4_test.go`：适配新 Bridge/Server 构造器及专用状态 manager，保持既有窗口策略回归可编译、可执行。
- 直接调用 solve 的其他既有测试：显式传入 display epoch。
- `app/questionbank_capture.go` 只读核对；若无需改动不得制造差异。

### 桌面前端与 bindings

- `frontend/src/stores/mobile.js`（新增）：共享状态、请求版本、busy 与 pending action。
- `frontend/src/components/MobileActionConfirm.vue`（新增）：唯一确认层。
- `frontend/src/services/api.js`：新增正式 App API 包装。
- `frontend/src/components/SettingsModal.vue`：固定模式、URL、Retry/Clear/Reset 管理。
- `frontend/src/components/TopBar.vue`：运行状态中的高频清屏入口，不展示 URL。
- `frontend/src/App.vue`：挂载唯一确认层。
- `frontend/mobile_store_regression_test.mjs`（新增）与 `frontend/settings_regression_test.mjs`：状态竞争和设置隔离测试。
- `frontend/wailsjs/go/app/App.js`、`App.d.ts`、`models.ts`：正式生成 bindings。

### 文档

- `README.md`、`README_EN.md`、本 Change 的 `03`～`06`、`_dev/changes/INDEX.md`。

明确不改：ActionBar、solution store、题库 Store、Solver chat history、截图缓存清理与项目依赖版本。

### 2026-10-23 补充修复白名单

补充范围只允许修改：

- `pkg/config/config.go`、`pkg/config/manager.go` 及 `config_test.go`、`manager_test.go`：Windows 默认绑定和旧配置无冲突迁移。
- `pkg/shortcut/service.go`、`service_test.go`：`clear_mobile_display` 只发 `request-clear-mobile-display` 事件。
- `frontend/src/stores/mobile.js`、`frontend/src/App.vue`、`frontend/mobile_store_regression_test.mjs`：refresh 后建立 pending，busy/pending 防重复。
- `frontend/src/components/TopBar.vue`：pending watcher 与 dropdown 层级。
- `frontend/src/stores/settings.js`、`frontend/src/components/SettingsModal.vue`、`frontend/settings_regression_test.mjs`：Windows-only 清单、录制和空绑定显示。
- `README.md`、`README_EN.md`、Change 016 文档、INDEX、验证 EXE 与哈希清单。

固定接口为：

```text
action = clear_mobile_display
event = request-clear-mobile-display
ComboID = 67+160
KeyName = Shift+C
```

初版计划要求 action 缺失且 `67+160` 被占用时保留旧 action、清屏补空绑定；该规则已被第 8 节用户实测回修替代。最终规则见第 8 节：默认组合归清屏，旧冲突 action 清空。快捷键事件不得直接调用 `ClearMobileDisplay(true)`，必须经过 `mobile.refresh() → mobile.requestClear() → MobileActionConfirm`。macOS 不增加默认项、不显示该 action。

补充验证覆盖默认 ComboID 唯一性、三类旧配置迁移、service 单事件无 payload、读取/长按/pending 防重、TopBar 层级与同步关闭、Windows/macOS 设置清单及空绑定文案。

## 4. 关键接口与事务边界

```go
type MobileRuntimeState struct {
    StableLink bool
    Token string
    ClearSeq uint64
    Generation uint64
}

func NewMobileStateManager() *MobileStateManager
func (m *MobileStateManager) LoadOrCreate() (MobileRuntimeState, error)
func (m *MobileStateManager) AdvanceClearSeq() (MobileRuntimeState, error)
func (m *MobileStateManager) ResetLink() (MobileRuntimeState, error)
func (m *MobileStateManager) SetStableLink(enabled bool) (MobileRuntimeState, error)
```

双槽为 `mobile-state.0.enc` / `mobile-state.1.enc`。候选写入非当前有效槽，执行 Sync、关闭、回读解密与字段校验后才切换 manager current。固定模式默认开启；临时模式磁盘 Token 为空。未知 schema fail closed，单槽截断时使用另一有效槽。

```go
func NewBridge(initialClearSeq uint64) (*Bridge, error)
func (b *Bridge) CurrentClearSeq() uint64
func (b *Bridge) PublishIfCurrent(expected uint64, event string, data ...interface{}) bool
func (b *Bridge) ClearDisplay(next uint64)
func (b *Bridge) Close()

func NewServer(b *Bridge, token string) (*Server, error)
func (s *Server) RotateToken(newToken string)
```

持久化后的 App 字段赋值、Bridge Clear 和 Server Rotate 是无 I/O、无失败返回的确定性步骤。Clear/Reset/模式切换顺序统一为“专用状态提交 → App current → Bridge clear → 可选 Server rotate”。

App 对前端公开 `ClearMobileDisplay(confirmed)`、`ResetMobileOutputLink(confirmed)`、`SetMobileStableLink(enabled, confirmed)`，返回稳定结果码与不含 Token/session/clear 的 `MobileInfo`。

锁顺序固定：`mobileOpMu` → manager 内部锁（方法返回前释放）→ `mobileMu` → 单独 Bridge 或 Server 锁。不得同时持有 Bridge 与 Server 锁，不在 `mobileMu` 下写盘、关闭 socket、等待 handler 或调用普通 Config subscriber。

## 5. 测试与验证计划

### 自动测试

- `GOTOOLCHAIN=go1.25.13 go test ./pkg/config -run Mobile -count=1 -v`
- `GOTOOLCHAIN=go1.25.13 go test ./pkg/mobile -count=1 -v`
- `GOTOOLCHAIN=go1.25.13 go test ./app -run 'Mobile|DisplayEpoch|Settings' -count=1 -v`
- `node pkg/mobile/web_dom_test.mjs`
- `node frontend/settings_regression_test.mjs`
- `node frontend/mobile_store_regression_test.mjs`
- `GOTOOLCHAIN=go1.25.13 go test ./... -count=1`
- `GOTOOLCHAIN=go1.25.13 go vet ./...`
- `GOTOOLCHAIN=go1.25.13 go build ./...`
- `npm --prefix frontend run build`
- Wails 2.11.0 + Go 1.25.13 唯一命名验证构建，不覆盖现有 EXE。
- `git diff --check` 与 secrets/URL/DTO 边界检查。

### 人工验收

覆盖固定模式重启、临时模式重启、Retry、Reset 旧链接失效、多历史 Clear、断网清屏后重连、流式中清屏、端口占用恢复和数据不删除。未执行项标 `NEEDS_USER`；race 环境启动失败标 `BLOCKED`。

## 6. 回滚预案

- 仅还原 Change 016 代码与文档，不删除题库、输出、截图、普通配置或 Mobile 专用状态。
- 专用状态默认保留；旧版本会忽略，再升级可继续使用。手工删除会轮换链接，不作为常规回滚。
- 若鉴权撤销或状态事务不满足验收，停止交付并修复，不通过关闭鉴权、记录 Token 或删除用户状态绕过。
- 回滚、提交、推送、Tag 与 Release 均需独立授权。

## 7. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 写盘失败后运行态已变化 | Token/链接分裂 | 候选回读通过后切 manager；后续步骤无 I/O、无失败 |
| Token 轮换竞态 | 旧 SSE 继续收内容 | auth generation + lease cancel；鉴权与登记同锁 |
| 新进程 seq 归零 | 页面忽略新答案 | 每进程随机 sessionId，变化时重置去重命名空间 |
| 清屏后旧内容复活 | 考试前页面不干净 | 单调 clearSeq、可靠队列屏障、任务 epoch、客户端整帧拒绝 |
| 关闭阶段死锁 | 应用无法退出 | operation lock 串行；Server 解锁后关闭和等待 |
| 长期 Token 泄漏 | 未授权访问 | 独立加密存储、全路由鉴权、no-referrer、禁止日志与浏览器存储 |
| 用户误解成功语义 | 误以为所有设备确认 | UI 明示“已提交并广播”，不声称有客户端 ACK |

## 8. 2026-10-23 `Shift+C` 隐藏问题回修计划

允许继续修改 `pkg/shortcut/service.go`、`service_test.go`、`pkg/config/config.go`、`config_test.go`、`manager_test.go`、相关前端回归测试、README 与 Change 016 文档。

1. 将 `handleRecord()` 限定为预览事件，不写 `manager.Shortcuts`。
2. `handleRecordingComplete()` 先检查当前运行态冲突；失败只发错误且保持双方原绑定，成功后才写目标 binding 并发送 `shortcut-saved`。
3. 调整 Windows 清屏迁移：旧 JSON 缺少 action 时，清空其他 action 的默认 `67+160` 后给清屏 action 设置默认绑定；action 显式使用默认绑定时也清理重复占用。若旧版已持久化空清屏绑定且其他 action 仍占用默认组合，同样纠偏；显式自定义绑定，以及没有默认冲突的显式空绑定不被改写。
4. 用服务单元测试覆盖预览不提交、冲突不污染、成功一次提交和成功事件；用配置测试覆盖旧占用迁移及显式绑定边界。
5. 重跑聚焦、全量 Go、前端回归、Vite、Wails 验证构建与安全/差异检查；GUI 行为继续标记 `NEEDS_USER`。

本回修不修改 Windows Hook 的 VK 算法、不创建第二个事件或确认层、不读取或覆盖用户私人配置，也不执行 commit、push、Tag 或 Release。
