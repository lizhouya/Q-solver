# 实现记录 — 03 · 003-desktop-window-toggle

> **状态：Phase 1/2/3 + 3.1 一致性收口 + Phase 4（mobile 自动隐藏）+ 4.1（availability 统一守卫）已完成**。
> 依据 `02-plan.md` 的 Phase 计划。

## Phase 1 — WindowVisibilityManager + 三态分离 ✅

> **实现取舍**：按 003 指令「允许新增 `pkg/window/`」，`WindowVisibilityManager` 落在新包 `pkg/window`，而非设计稿初版的 `pkg/state`；状态三态完全独立：
> - GhostMode（防录屏隐身/F9）→ 仍在 `pkg/state`，**未改动**；
> - WindowVisibleState（实体可见性）→ 新增于 `pkg/window`；
> - OutputMode（desktop/mobile/dual）→ config，未读未写。

- [x] 新增 `pkg/window/window.go`：`WindowVisibilityManager`（`Hide/Show/Toggle/State/IsVisible` + 可见性回调），并发安全
- [x] 新增 `pkg/window/window_windows.go`：`windowsController` 经 `platform.ShowWindow` 封装 Win32
- [x] 新增 `pkg/window/window_darwin.go`：macOS 占位（003 非目标，返回错误）
- [x] `pkg/platform/platform_windows.go` 新增 `ShowWindow(hwnd, SW_HIDE/SW_SHOW)` 封装（procShowWindow）
- [x] `pkg/platform/platform_darwin.go` 追加 `ShowWindow` 占位（保证跨平台编译一致）
- [x] `app` 新增绑定：`HideWindow / ShowWindow / ToggleWindowVisible / WindowVisible`
- [x] 事件推送：`WindowVisibilityManager.SetVisibilityChange` 回调（锁外同步触发），Phase 4 接前端
- [x] 窗口句柄复用：`app` 注入 `stateManager.GetHwnd` 作为 provider，**不重复保存 HWND**

## Phase 2：快捷键触发 ✅

- [x] `pkg/shortcut` 新增 action `toggle_window`（`handleTrigger` 分支）
- [x] ServiceDelegate 新增 `ToggleWindowVisible() (string, error)`，由 `App.window.go` 既有绑定满足（Phase 1 已有 `ToggleWindowVisible`）
- [x] 默认键 `Ctrl+Shift+H`（ComboID `72+160+162` = VK_LCONTROL(162)+VK_LSHIFT(160)+H(72)，升序拼接，与既有 `send=74+162` 模式一致）
- [x] `pkg/config` `getDefaultShortcuts` 增加 `toggle_window` 默认项（Windows `72+160+162` / macOS 占位 `Cmd+4`，003 非目标）
- [x] 设置面板录制入口：前端 `settings.js` `shortcutActions` 增加 `toggle_window` 一行（**最小必要修改**，仅数据驱动列表行，无新 Wails binding、无新事件）
- [x] 冲突检查走既有 `handleRecordingComplete`（ComboID 撞车 → `shortcut-error` + 重启录制），默认键经查与 F8-10 / Ctrl+J / Ctrl+D / Alt+方向 / Alt+PgUp/PgDn 无冲突
- [x] 错误处理：`toggle_window` 触发失败仅 `logger.Printf` 记录，HWND 为空安全降级（不 panic / 不退出）

> **与 F9 的关系（D-003-004）**：`toggle_window`（实体隐藏，`ToggleWindowVisible`）与 `toggle`（F9 防录屏隐身 GhostMode，`ToggleVisibility`）为两个独立 action，互不调用、互不清除，可自由组合。

## Phase 3：系统托盘 ✅

> **实现取舍**：架构沿用 `pkg/window` 验证过的「Manager + TrayController 抽象 + 平台 init 绑定 + mock 单测」模式；平台层只封装「必要 Win32 原语」，回调/消息循环/生命周期编排全部留在 `pkg/tray`。

- [x] 新增 `pkg/platform/tray_windows.go`：纯 Win32 底层能力（无业务依赖）
  - 宿主隐藏窗口创建/销毁（`HWND_MESSAGE`，仅消息窗口不入任务栏）
  - `Shell_NotifyIconW` NIM_ADD / MODIFY / DELETE（`NOTIFYICONDATAW` 原生结构）
  - 消息泵原语：TranslateMessage / DispatchMessage / PostMessage / PostQuitMessage
  - 图标：`LoadAppIcon()`（优先复用主窗口类图标 `GetClassLongPtr(GCLP_HICON)`，零额外资源；失败降级系统默认图标）
  - 右键菜单：`ShowTrayMenu`（CreatePopupMenu/AppendMenuW/TrackPopupMenu 全生命周期封装）
- [x] 新增 `pkg/tray/`：`TrayManager`（Start/Stop 幂等 + IsRunning）+ `TrayController` 接口 + `TrayEventHandlers` 回调 + `currentController` 平台工厂（对齐 pkg/window 模式）
- [x] `pkg/tray/tray_windows.go`：真机控制器
  - `runtime.LockOSThread` 保证「创建窗口的线程」=「消息循环线程」（Win32 硬约束）
  - 独立 goroutine 消息循环，不阻塞 Wails Startup；`ready/stopped` 双通道做就绪与退出握手
  - wndProc 分派：左键 → `OnShow`（显示主窗口）；右键 → 菜单（显示主窗口 / 退出）
- [x] `pkg/tray/tray_darwin.go`：macOS 占位返回错误（003 非目标，D-003-007）
- [x] app 集成：`app/tray.go` 回调 + `app.go` Startup `setupTray()` / OnShutdown `trayManager.Stop()`
  - 托盘左键：窗口隐藏 → `windowVisibility.Show()`（状态 + 实体一致）；可见 → `runtime.WindowShow`（前台聚焦）
  - 托盘退出 → `runtime.Quit`（触发 OnShutdown 走既有清理链）
  - 失败仅降级（logger），绝不影响主流程
- [x] 单元测试：`pkg/tray` 9 case（初始态 / Start 委托与 handlers 传递 / Start 幂等 / 控制器失败传播 / Start 失败不标记运行 / Stop 委托 / Stop 幂等 / 未启动 Stop 无操作 / Stop 失败状态复位）全 PASS

## Phase 3.1：托盘菜单一致性收口 ✅

> **背景**：Phase 3 初版右键菜单为「显示主窗口 / 退出」，未含「隐藏 Q-Solver」；且 `app.Show()`（SecondInstance / 托盘恢复的共同入口）直接 `runtime.WindowShow`，绕过了 `WindowVisibilityManager` 状态机，存在「实体已显示但状态仍 Hidden」的漂移隐患。3.1 一并收口。

- [x] **托盘右键菜单三向补齐**：`pkg/tray/tray.go` 新增 `MenuHide` + `TrayEventHandlers.OnHide`；`tray_windows.go` 菜单改为「显示 Q-Solver(&S) / 隐藏 Q-Solver(&H) / ─ / 退出(&X)」，分派新增 `onHide()` 
- [x] **托盘菜单分派可测化**：`onRightClick` 的 switch 抽出为独立 `dispatchMenu(id)`，补 `TestDispatchMenu`（show/hide/exit 各恰好 1 次 + 未知 ID 静默），tray 单测 9 → **10 case**
- [x] **隐藏唯一入口收敛**：托盘「隐藏 Q-Solver」→ `app.trayOnHide` → `WindowVisibilityManager.Hide()`（状态机 + SW_HIDE）；禁止任何业务层直接 `ShowWindow(SW_HIDE)`
- [x] **`app.Show()` 状态同步**：`app.go` `Show()` 改为「第一步 `windowVisibility.Show()`（Hidden→Visible 状态机同步 + 幂等），第二步 `runtime.WindowShow`（Wails 前台聚焦）」；`trayOnShow` 复用 `app.Show()`，消除可见/隐藏分支不一致
- [x] **唯一 Source of Truth 梳理**：业务层 Show/Hide 全部收敛到 `WindowVisibilityManager`（Shortcut `ToggleWindowVisible` / Tray Show / Tray Hide / SecondInstance `app.Show`）；`platform.ShowWindow` 仅 `pkg/window/window_windows.go` 的 controller 内部调用
- [x] **app 层单测**：新增 `app/tray_window_test.go` 4 case（`trayOnHide` 恰调用一次 / 重复隐藏幂等 / `app.Show` Hidden→Visible / 已可见重复 Show 幂等），通过 `NewWindowVisibilityManagerWithController` 注入 mock 控制器（`pkg/window` 新增测试专用构造器）
- [x] 回归：`go build ./...` / `go vet ./pkg/tray ./pkg/window ./app` / `go test ./pkg/tray ./pkg/window ./app` 全绿；`git diff --check` 干净

## Phase 4：OutputMode=mobile 自动隐藏桌面主窗口 ✅

> **实现取舍**：按 Phase 4 指令，Auto Hide 是 transition-triggered action——只在进入 mobile 时触发一次 `Hide()`，mobile→desktop/dual 不自动 Show，用户在 mobile 模式手动 Show 后不再次强制 Hide。
> 启动顺序保证：shortcut → mobile server → tray → auto hide（隐藏前所有恢复路径已装配）。
> 依赖方向：`app` 层负责 OutputMode × WindowVisibility 的协调，`pkg/config` 不 import `pkg/window`。

- [x] `applyOutputModeWindowPolicy(previousMode, newMode string)` 核心方法（`app/app.go`）：
  - 仅 `previousMode != mobile && newMode == mobile` 时调用 1 次 `windowVisibility.Hide()`（one-shot 语义）
  - mobile→desktop/dual 不自动 Show（OutputMode 与 WindowVisible 为独立状态）
  - 无轮询/循环检测
  - **Phase 4.1**：`mobileOutputEnabled()` 统一守卫——无论 startup 还是 runtime，只有 Server 可用时 Hide
- [x] **Scenario A：启动时检测**——`App.Startup()` 末尾检测 `OutputMode=mobile`，走同一 `applyOutputModeWindowPolicy` 方法
- [x] **Scenario B：运行时切换**——`onConfigChanged()` 中调用 `applyOutputModeWindowPolicy(oldConfig.OutputMode, newConfig.OutputMode)` 检测 transition
- [x] **启动顺序**：Config Load → shortcutService.Start() → setupMobile() → setupTray() → auto hide
- [x] **恢复入口预装配检查**：`shortcutService == nil` 时跳过自动隐藏
- [x] **Server 可用性守卫（Phase 4.1）**：`mobileOutputEnabled()` 在 `applyOutputModeWindowPolicy` 内部统一检查——覆盖 startup 和 runtime 两条路径；不可用时记录 warning 并跳过
- [x] **Hide 失败处理**：仅记录错误，不 panic/exit，保持窗口显示
- [x] **启动闪烁**：第一版允许 Wails 创建后短暂存在，不引入 ShowWindow hack
- [x] **三态保持独立**：自动隐藏不修改 GhostMode/DisplayAffinity/F9 行为
- [x] 自动测试（17 case PASS）：`app/phase4_test.go`

### Phase 4 局部设计决策

| ID | 决策 | 依据 |
|---|---|---|
| LD-004-01 | mobile 只在进入时 Hide 一次（one-shot） | 避免「强制隐藏循环」打破用户手动恢复 |
| LD-004-02 | mobile→desktop/dual 不自动 Show | OutputMode 与 WindowVisible 独立状态 |
| LD-004-03 | 用户手动 Show 后不再次强制 Hide | mobile 模式用户 Show→保持 Visible |
| LD-004-04 | Mobile Server 不可用时 Startup/Runtime 均不 Auto Hide | 避免「窗口消失但手机服务不可用」场景（Phase 4.1 统一覆盖） |
| LD-004-05 | 启动顺序：shortcut → mobile → tray → auto hide | 保证恢复路径装配后才隐藏 |
| LD-004-06（4.1） | Server 稍后恢复时不补偿 Hide；仅在新的 non-mobile→mobile transition 再次判断 | 不引入 server-ready watcher / retry / timer |

## 完成标准（渐进）

- [x] `go build ./...` 通过
- [x] `go vet` 改动包干净（`pkg/window`、`pkg/platform`、`pkg/tray`、`pkg/config`、`app` exit=0；`pkg/shortcut` 的 unsafe.Pointer 警告为**基线既有**，HEAD worktree 同样存在）
- [x] `go test ./app/... ./pkg/...` 全绿（tray 10 case、window 12 case、app 新增 4 case、shortcut/config/mobile 均 ok）
- [x] 前端 npm build（Phase 2 改 settings.js、本次 Phase 4.1 回归已跑通 `npm --prefix frontend run build`）
- [ ] 真机手测（见 04-validation）

## 变更日志

| 日期 | Phase | 内容 |
|---|---|---|
| 2026-09-28 | 1 | 实现 `pkg/window` + platform.ShowWindow + app 绑定 + 12 case 单测 |
| 2026-09-28 | 2 | shortcut action `toggle_window` + delegate 方法 + config 默认键 + 前端一行 + 9 case 单测 |
| 2026-09-28 | 3 | `pkg/platform/tray_windows.go` 原语 + `pkg/tray`（manager/控制器/darwin 占位）+ app 集成 + 9 case 单测 |
| 2026-09-28 | 3.1 | 托盘菜单补「隐藏 Q-Solver」+ 分派可测化（tray 9→10 case）+ `app.Show()` 走状态机 + app 层 4 case 单测（见 Phase 3.1 节） |
| 2026-09-29 | 4 | OutputMode=mobile 自动隐藏主窗口（transition-triggered one-shot），`applyOutputModeWindowPolicy`（见 Phase 4 节） |
| 2026-09-29 | 4.1 | `mobileOutputEnabled()` 统一守卫收敛到 `applyOutputModeWindowPolicy` 内部；Startup/Runtime 共享相同 server availability 判断（见 Phase 4 节） |