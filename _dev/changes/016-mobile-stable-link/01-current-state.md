# 现状快照 — 01

日期：2026-10-22。以下内容记录 Change 016 修改前状态。

## 1. 仓库基线

- 分支：`dev/qsolver-custom`
- HEAD：`da273bdea8c665b38bd8407b3e0257074f48ba63`
- 与 `origin/dev/qsolver-custom`：ahead 1
- 工作区：clean
- `v1.1.0-rc.2`：不存在
- 当前系统 Go 为 1.27.1；已验证发布兼容工具链为 `GOTOOLCHAIN=go1.25.13`，Wails CLI/runtime 为 2.11.0。

## 2. 涉及文件与当前职责

| 文件 | 当前职责 | Change 016 前关键事实 |
|---|---|---|
| `pkg/config/manager.go` | 普通配置加载、保存、订阅 | `Config` 返回前端并全量回写；保存不是 Mobile Token 的安全边界 |
| `pkg/config/crypto.go` | AES-256-GCM 加解密 | 应用内固定常量经 SHA-256 派生 key，可复用但不等同系统密钥库 |
| `pkg/mobile/state.go` | SSE Snapshot 数据结构 | 只有业务 `seq` 与 `controlSeq`，没有进程 session 或清屏边界 |
| `pkg/mobile/bridge.go` | 桌面事件转 Snapshot 并广播 | `NewBridge()` 无 session；`Reset()` 只建立 idle 快照；队列满时有流式/终态/控制策略 |
| `pkg/mobile/server.go` | HTTP 页面与 SSE 服务 | `NewServer()` 每次随机生成 Token；只有 `/sse` 鉴权；Stop 持锁调用 Shutdown |
| `pkg/mobile/web/index.html` | 同步页面及会话内历史 | 以 `seq` 去重；`showWaiting()` 隐藏但不擦除全部旧 DOM；无 clear 水位 |
| `app/mobile.go` | Mobile setup、状态、路由与 Retry | Retry 复用同进程 Server；无 Clear、Reset、固定模式 API |
| `app/solve.go` | 解题任务启动与回调 | 任务不捕获 Mobile 显示代际，迟到输出可覆盖后续清屏 |
| `frontend/src/components/SettingsModal.vue` | 设置页输出配置与 Mobile 状态 | 使用组件局部 Mobile 状态，支持读取、复制与 Retry |
| `frontend/src/components/TopBar.vue` | 顶部运行状态与控制 | 无同步端清屏入口 |

## 3. 关键代码现状

### Token 生命周期

`pkg/mobile/server.go::NewServer` 调用 `randomToken(16)`，Token 仅存在 Server 内存；完整应用重启会创建新 Server，因此 URL 改变。同一进程内 Retry 调用现有 Server 的 Start，通常复用 Token。

### 鉴权和撤销

`authToken` 只在 SSE 建连时比较 query Token；首页 `/` 不鉴权。修改内存 Token 不能撤销已经通过鉴权的 SSE。`http.Server.Shutdown()` 在 Server 锁内调用，长 SSE 可能造成等待与锁耦合。

### Bridge 与客户端状态

Bridge 的 `seq` 每个进程从零开始。固定 Token 后，已打开浏览器跨进程重连仍保留旧 `lastBusinessSeq`，可能忽略新进程的低 seq。`Bridge.Reset()` 不清浏览器 `sessionHistory`、旧答案/错误 DOM、历史选择或已经排队的滚动 rAF。

### 配置写入边界

`config.Config` 是公开 DTO，会通过 `GetSettings()` 返回前端，并由 `UpdateFromJSON()` 全量回写。`Patch()` 与 `UpdateFromJSON()` 都先改内存并通知 subscriber，再 Save；因此 Token 不适合加入普通 Config，也不能在持有 `mobileMu` 时通过普通 Config subscriber 管理 Mobile 状态。

## 4. 运行与构建现状

- 生产 Mobile 端口固定 `38511`；测试已支持 Server port 设为 `0` 后读取内核分配端口。
- 用户已人工确认：API 解题、跨电脑同一 SSE 实时显示无需刷新、桌面端控制同步端滚动可用。
- 最近全量普通测试、vet、build、Vite 与 Wails 1.25.13 构建通过。
- `go test -race` 在当前 Windows 环境启动失败 `0xc0000139`，状态为 `BLOCKED`。
- `build/bin/` 顶层仅保留 `Q-Solver.exe`，历史与验证版本位于 `releases/`、`archive/`。

## 5. 已知现状问题

1. Token 未持久化，完整重启链接变化。
2. Token 若直接加入普通 Config 会暴露给前端并受全量回写影响。
3. 已连接 SSE 无 Token 轮换撤销机制。
4. 只有 SSE 鉴权，旧 Token 首页请求仍可加载页面。
5. 跨进程业务 seq 归零可能被旧页面错误去重。
6. Reset 不是可靠、可重放的页面清屏。
7. 清屏前任务的迟到输出可让旧内容复活。
8. 设置与 TopBar 没有共享 Mobile 操作状态和统一确认层。

本文件是修改前快照；实施演进记录进入 `03-implementation.md`。

## 6. 2026-10-23 补充修复前快照

- `frontend/src/components/TopBar.vue` 的 `.status-panel` 使用 `z-index: var(--z-tooltip)`；当前 token 值为 `99999`。根级 `MobileActionConfirm` 使用 `calc(var(--z-modal) + 20)`，当前为 `3020`，因此状态面板会覆盖确认层。
- 状态面板带 `@click.stop`，其内部点击不会触发 document 外部点击关闭；`panel-fade` 的 leave 动画还会保留约 150ms DOM。
- `mobile.requestClear()` 已只建立 pending，不直接调用后端；因此鼠标与键盘入口应继续复用该 store 和唯一确认层。
- `ConfigManager.Load()` 先以默认配置初始化 struct，再对旧 JSON 执行 `json.Unmarshal`；对预初始化的非 nil `shortcuts` map，Go 会把旧 JSON 键合并进默认 map，导致反序列化后无法区分新 action 是默认注入还是用户显式保存，因此迁移必须额外检查原始 JSON 中是否存在该 key。
- Windows 快捷键 Hook 按排序后的 VK 生成 `ComboID`，左 Shift 为 `160`、右 Shift 为 `161`；默认 `67+160` 只匹配左 Shift+C。匹配成功的全局按键会被 Hook 吞掉。
- 两个 action 若共享同一非空 `ComboID`，触发结果受 Go map 遍历顺序影响；迁移必须避免生成重复组合。
- 当前 `mobile.setPending()` 只检查 mutation/pending，未把 loading 纳入门禁；快捷键前置 refresh 需要把读取状态也纳入 busy 防重。
- 当前常规设置在绑定为空时回退显示默认文本，会把冲突迁移后的空绑定错误显示为已生效的 `Shift+C`。

## 7. 2026-10-23 `Shift+C` 隐藏问题根因

- `shortcut.Service.handleRecord()` 在每次录制按键变化时就写入 `manager.Shortcuts[action]`，而冲突检查直到 `handleRecordingComplete()` 才执行。
- 冲突路径只发 `shortcut-error` 并重新开始录制，没有恢复被提前覆盖的运行态绑定；因此运行态可同时存在两个相同的非空 `ComboID`。
- Windows Hook 通过遍历 Go map 查找第一个匹配 action；重复 `ComboID` 下命中顺序不确定，可能执行 `toggle_window` 并隐藏主界面，而不是请求清屏。
- 前端收到 `shortcut-error` 后只恢复 `tempShortcuts`，无法修复 Go Hook 中已经被污染的运行态 map。
- 旧迁移策略在默认 `67+160` 已被旧 action 占用时把清屏 action 设为空，因此用户按 README 中的 `Shift+C` 仍会执行旧 action。该策略与最新实测验收冲突，需要由本次反馈收窄为“默认组合保留给清屏”。
