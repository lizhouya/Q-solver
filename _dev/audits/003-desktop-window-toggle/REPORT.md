# REPORT — Change 003 Desktop Window Toggle 架构审计

> 前置架构审计（只读，未写代码，未改动 app/pkg/frontend）。日期：2026-09-28。分支 `dev/qsolver-custom`（bcf5ff8+b8acfa3+1e27512 之后，clean）。

## 0. 摘要

Q-Solver 当前没有「桌面窗口实体显示/隐藏」能力（只有防录屏隐身）；若开发 Change 003，需要跨 4 层新增功能，但**架构上完全可行**：全局快捷键钩子独立于窗口生命周期、Mobile SSE 服务独立于窗口可见性。**未发现阻断性障碍**，风险集中在「恢复路径缺失」「语义混淆」「键位冲突」三处。是否进入设计，取决于 D-003 系列决策。

## 1. 当前快捷键架构（结论）

- **存在统一 Manager**：`pkg/shortcut/Service + Manager`（Windows 低层 `WH_KEYBOARD_LL` + `WH_MOUSE_LL` 全局钩子；action 驱动；支持录制）。
- 触发链：按键 → hook → `onKeysChanged` → `OnTrigger(action)` → `Service.handleTrigger` → App delegate。
- 默认键（Windows）：F8 截图 / Ctrl+J 发送 / Ctrl+D 删除 / F9 toggle（隐身）/ F10 穿透 / Alt+方向 移动 / Alt+Pg 滚动。
- 扩展方式成熟：新增 action 即可接线新快捷键（含前端录制 UI 复用）。

详见 `02-hotkey-system-analysis.md`。

## 2. 当前窗口架构（结论）

- 主窗口：Wails 1024×768 固定、AlwaysOnTop、透明、单实例（第二实例 → `app.Show()`）。
- 窗口状态：state 持有 HWND；`ToggleVisibility` = **防录屏隐身**（SetDisplayAffinity），非实体隐藏。
- 已有：Show（runtime.WindowShow）、置顶、穿透、焦点、移动、尺寸持久化。
- **无**：WindowHide、Minimize、任务栏隐藏/托盘、实体可见性状态机。

详见 `01` / `03`。

## 3. GhostMode 关系

`applyGhostMode(hwnd)`（一次应用，启动时）为**样式 + 防录屏**：
- 无边框、工具窗、置顶、NOACTIVATE、防录屏（WDA_EXCLUDEFROMCAPTURE，失败降级黑屏）、Win10 去 DWM 模糊。
- **不含快捷键逻辑**，**不含实体隐藏**。
- 归类 → **B（含防录屏/显示隐藏的一类，即 SetDisplayAffinity 隐身）**，但非「窗口整体消失」。

> 与 003 的关系：003 的实体 Hide/Show 与 GhostMode 正交，隐藏时可保留 GhostMode 样式与防录屏状态，属设计决策（D-003-004）。

## 4. Mobile Output 关系（关键：隐藏后 SSE 等是否继续）

**结论：完全解耦，隐藏窗口不会中断 Mobile。**

- `mobile.Server`（HTTP/SSE）是独立 goroutine 服务，生命周期仅绑定 `App.Startup/OnShutdown`；**不引用 HWND、不依赖窗口可见性**。
- `decideRoute` 纯函数（OutputMode + server 可用 → desktop/mobile/dual），与窗口状态无关。
- 因此：
  - **隐藏窗口后 SSE 继续** ✅
  - **隐藏窗口后 App 生命周期继续** ✅（进程存活，仅 Win32 样式属性变化）
  - **隐藏窗口后 Shortcut（全局 hook）继续** ✅（低层钩子不依赖窗口消息循环）
- 唯一缺口：恢复方式（隐藏后如何召回窗口），见 F-105 / D-003-002。

## 5. Windows 实现层盘点

- `pkg/platform/platform_windows.go` 已封装 Win32：HWND 获取、`SetWindowPos`、`SetWindowLong/GetWindowLong`、SetDisplayAffinity、SetWindowToolWindow（可以隐藏任务栏）等。
- **未封装**：`ShowWindow(SW_HIDE / SW_SHOW)` 实体隐藏、最小化 API。
- 若实现 `Hide/Show`，需改动的层次：

| 层 | 改动 |
|---|---|
| `pkg/platform` | 新增 `ShowWindow`（SW_HIDE/SW_SHOW）或复用 `runtime.WindowHide/Show` |
| `pkg/state` | 实体可见状态 + Hide/Show 方法 |
| `pkg/shortcut` | 新增 action `toggle_window` 接线 |
| `app/` | 新绑定方法（HideWindow/ShowWindow/IsVisible），更新 delegate |
| `frontend` | 设置面板增加入口/快捷键录制（如适用） |

（详见 `04-call-chain.md`）

## 6. Findings（F-101 ~ F-110）

见 `05-findings.md`。核心三条需要重点应对：
- **F-102**：区分「隐身(F9) / 实体隐藏 / 最小化」三个概念。
- **F-105**：必须定义隐藏后的恢复途径（快捷键/托盘/二次启动）。
- **F-107**：macOS 是否有需求（若跨平台增补工作量）。
- 均为**设计可解，非技术阻断**。

## 7. 用户决策清单

见 `06-user-decisions.md`（D-003-001 ~ D-003-009）。**已确认（2026-09-28）：**

| 编号 | 用户选择 |
|---|---|
| D-003-001 | C＝两者兼有（快捷键 + mobile 自动隐藏） |
| D-003-002 | C＝隐藏任务栏 + 系统托盘 |
| D-003-003 | B＝不独立支持最小化到托盘（托盘仅作恢复入口） |
| D-003-004 | A＝实体隐藏与«隐身（防录屏）»独立可并存 |

剩余待决：D-003-005（mobile 远程唤醒与否）、D-003-007（macOS）、D-003-008/009（快捷键录制入口与默认键位）——为开放实现，不阻塞立项。

## 8. 是否具备进入设计阶段条件

**具备（条件性通过）**：

✅ 满足的：
- 架构支持可行性高：无技术阻断。
- 改动面清晰（4 层，均现有扩展点）。
- 与已提交 001/002 无冲突（mobile 与窗口状态解耦）。

⚠️ 需先回答的（D-003-001/002/004/005 为核心）：
- 触发方式（快捷键 / mobile 自动 / 两者）
- 隐藏后任务栏/托盘策略与恢复途径
- 与“隐身（防录屏）”的语义区分
- mobile 模式下是否要远程唤醒窗口

> 一旦用户回复上述待决问题，即可进入 `_dev/changes/003-desktop-window-toggle` 的前 3 份文档（00/01/02）设计阶段。**未回复前，不建 changes/003，不改任何代码。**

## 9. 审计产物

- 本目录 8 份文档即权威审计产物。
- `git status` 应仅有当前目录新增（已验证，见下节）。