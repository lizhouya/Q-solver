# 决策记录 — 05 · 003-desktop-window-toggle

> 本文件记录 003 的需求决策（D-003 系列）与设计期新增实现决策（LD-003 系列）。
> 用户拍板的 D-003 见 `_dev/audits/003/06-user-decisions.md`，此处为设计阶段归档合并视图。

## 1. 用户已确认决策（D-003）

| 编号 | 项 | 选择 | 含义 |
|---|---|---|---|
| D-003-001 | 隐藏触发方式 | **C 两者兼有** | 快捷键 + OutputMode=mobile 自动隐藏 |
| D-003-002 | 隐藏后任务栏策略 | **C 隐藏任务栏 + 系统托盘** | 实体隐藏连任务栏一并消失；托盘作恢复入口 |
| D-003-003 | 最小化到托盘 | **B 不独立支持** | 托盘仅恢复入口 |
| D-003-004 | 与 GhostMode 关系 | **A 独立可并存** | F9 隐身与新隐藏互不清除 |

## 2. 补充默认决策（设计阶段确定，待用户验收时确认）

| ID | 项 | 默认 | 依据 |
|---|---|---|---|
| D-003-005 | mobile 远程唤醒 | 不支持，仅本地恢复（托盘+快捷键） | 不改 SSE 协议/001 已确认无远程指令 |
| D-003-007 | 平台 | 仅 Windows | 00-request §4 非目标明确 mac 不做 |
| D-003-008 | 快捷键录制 | 复用 `pkg/shortcut` 现有录制体系 | 001/002 已配置驱动化；**Phase 2 已落地**（`StartRecordingKey` + `shortcutActions` 一行） |
| D-003-009 | 默认键位 | `Ctrl+Shift+H` | 现有键已满（F8-10/Ctrl+J/D/Alt+…）；Phase 2 已落地 ComboID `72+160+162` |

## 3. 实现设计决策（LD-）

| ID | 决策 | 理由 |
|---|---|---|
| LD-003-01 | 新增独立 `WindowVisibilityManager`（**`pkg/window` 新包**，见 003 Phase1 指令「允许新增 pkg/window/」），不并入 `pkg/mobile` / `pkg/state` | 职责单一，避免窗口状态耦合移动包与既有状态包；三态（Ghost/Visible/OutputMode）物理隔离 |
| LD-003-02 | 三态独立存储：`StealthMode`（隐身）与新增 `Visible`（实体可见）分字段 | 防 F-102/F-109 语义混用 |
| LD-003-03 | `pkg/platform` 用原生 `Shell_NotifyIcon`，不引入第三方托盘库 | 00-request §4 禁新增依赖 |
| LD-003-04 | 隐藏/恢复统一走 `state.WindowVisibilityManager` 单入口 | 快捷键/Tray/mobile 三条触发路径收敛 |
| LD-003-05 | 前端事件 `window-visible-changed` 与新绑定方法 | 保持与 state 事件机制一致（F-009） |
| LD-003-06 | 托盘架构沿用 `pkg/window` 模式：`pkg/tray`（Manager + TrayController 接口 + mock 单测），`pkg/platform` 只封装必要 Win32 原语（宿主窗口/NotifyIcon/消息泵/图标/菜单），回调编排全在 `pkg/tray` | 可测性（mock 立测 9 case）+ 平台层零业务依赖（LD-003-03 延续） |
| LD-003-07 | 托盘宿主窗口为 `HWND_MESSAGE` 仅消息窗口（不入任务栏）；`runtime.LockOSThread` 保证建窗线程=消息循环线程 | 满足 Win32「消息只能被拥有线程分派」硬约束，且不干扰 Wails WebView 主窗口 |
| LD-003-08 | 托盘图标复用主窗口类图标 `GetClassLongPtr(GCLP_HICON)`，失败降级系统 `IDI_APPLICATION` | 「优先复用现有图标」，零引入新资源（00-request §4 禁新增依赖） |
| LD-003-09 | 托盘 Start 用 `ready/stopped` 双通道与消息循环 goroutine 握手：Start 阻塞到图标真正建立（毫秒级、错误如实上报），Stop 投递停止消息并限时等待退出 | 「消息循环不阻塞 Startup」要求 + Stop 可结束循环 |
| LD-003-10 | 托盘左键：窗口隐藏 → `windowVisibility.Show()`（状态/实体一致）；可见 → `runtime.WindowShow`（仅前台聚焦） | 避免与 windowVisibility 状态机打架、又满足第二实例唤起语义 |
| LD-003-11（3.1） | 托盘右键菜单三向：「显示 Q-Solver / 隐藏 Q-Solver / 退出」；`TrayEventHandlers` 增 `OnHide`，`onRightClick` 的 switch 抽出 `dispatchMenu(id)` 独立方法 | 满足 3.1 设计要求「显示/隐藏/退出」；分派逻辑可单测（10 case） |
| LD-003-12（3.1） | `app.Show()` 统一为两步：先 `windowVisibility.Show()`（Hidden→Visible 状态机同步 + 幂等），再 `runtime.WindowShow`（Wails 前台聚焦）；`trayOnShow` 直接复用 `app.Show()` | 「Hidden→Visible」全部业务入口（Shortcut/Tray/SecondInstance）都同步状态机；聚焦职责归属 Wails，状态机不背聚焦（修复 Phase 3 初版 SecondInstance 绕过状态机的漂移隐患） |
| LD-003-13（3.1） | `pkg/window` 新增 `NewWindowVisibilityManagerWithController`（显式注入控制器） | 仅测试注入 mock 用，生产路径仍走 `NewWindowVisibilityManager`，零生产行为改变 |
| LD-003-14（3.1） | `app.Show()` 两步语义：状态机同步 + Wails 前台聚焦；TrayHide/ShortcutHide 全部收敛到 `WindowVisibilityManager.Hide()` | 唯一 Source of Truth，消除 Phase 3 初版 SecondInstance 绕过状态机的漂移隐患 |
| **LD-004-01**（Phase 4） | mobile 只在进入时 Hide 一次（transition-triggered，one-shot 语义） | 避免「强制隐藏循环」导致用户无法稳定恢复窗口 |
| **LD-004-02**（Phase 4） | mobile→desktop/dual 不自动 Show | OutputMode 与 WindowVisible 为两个独立状态；恢复由 Ctrl+Shift+H/Tray 控制 |
| **LD-004-03**（Phase 4） | 用户手动 Show 后不再次强制 Hide（即使模式仍为 mobile） | 尊重用户意图，不循环强制隐藏（覆盖 T-P4-09） |
| **LD-004-04**（Phase 4） | Mobile Server 未启动成功时不自动隐藏 | 避免「窗口消失但手机服务不可用」的 UX 断层 |
| **LD-004-05**（Phase 4） | 启动顺序保证：shortcut → mobile server → tray → auto hide | 隐藏前恢复路径已装配；shortcut 为主恢复入口，Tray 允许降级 |
| **LD-004-06**（4.1） | Mobile Server 不可用时，Startup/Runtime 均不 Auto Hide（`mobileOutputEnabled()` 统一守卫收敛到 `applyOutputModeWindowPolicy` 内部） | 修复 Phase 4 初版「startup 有守卫、runtime 无守卫」的不一致；避免「窗口消失但手机服务不可用」的 UX 断层；覆盖 T-P4-13~15 |
| **LD-004-07**（4.1） | Server 稍后恢复时**不补偿 Hide**；仅在新的 non-mobile→mobile transition 再次判断 | 不引入 server-ready watcher / retry / timer / polling；保持 transition-triggered one-shot（覆盖 T-P4-16/17） |

## 4. 变更的影响面（已核对）

- 新增动点：`pkg/state`、`pkg/platform`（Windows）、`app`、`pkg/shortcut`（action）、frontend settings。
- Phase 4 新增动点：`app/app.go`（`applyOutputModeWindowPolicy` + `onConfigChanged` transition 检测 + `Startup` auto hide 调用）
- 不改动：`pkg/mobile`、SSE、LLM、Solve、Prompt、GhostMode 实现、index.html。