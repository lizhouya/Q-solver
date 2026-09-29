# 变更请求 — 00 · 003-desktop-window-toggle

| 字段 | 值 |
|---|---|
| 变更编号 | `003-desktop-window-toggle` |
| 提出人 | 产品/负责人（Q-Solver 二次开发） |
| 提出日期 | 2026-09-28 |
| 前置审计 | `_dev/audits/003-desktop-window-toggle/`（已完成：架构现状/风险 F-101~F-110/决策 D-003） |
| 前置变更 | `001-mobile-output`、`002-remote-output-ui`（均已提交） |
| 状态 | `设计阶段`（本文档 + 01/02 应随后齐备后进入实现） |

## 1. 背景与目标

Q-Solver 当前是「常显透明悬浮工具」：无边框、总置顶、GhostMode 防录屏（F9 隐身）。但**没有「桌面窗口整体显示/隐藏」入口**：不能把整个程序藏起来，也不能在 mobile 模式不占屏的同时保持进程与 SSE 服务存活。

本变更新增 **WindowVisibilityManager（窗口可见性管理器）**：提供 `Hide() / Show() / Toggle() / State()`，并把「窗口隐藏」接入两类触发——**手动快捷键**（`Ctrl+Shift+H` 可配置）+ **切到 OutputMode=mobile 时自动隐藏**；隐藏后经**系统托盘图标**恢复。隐藏期间手机端 **SSE / 解题 / Shortcut 全部继续**（进程在，SSE goroutine 与全局钩子与窗口可见性解耦）。

## 2. 需求要点（已确认决策）

| 项 | 决策 |
|---|---|
| 触发方式 | **C＝两者兼有**：快捷键 + mobile 自动隐藏（D-003-001） |
| 隐藏后任务栏 | **C＝隐藏任务栏 + 托盘恢复**（D-003-002） |
| 最小化到托盘语义 | **B＝不独立做**，托盘仅恢复入口（D-003-003） |
| 与 GhostMode 关系 | **A＝独立可并存**（F9 隐身与新隐藏互不影响）（D-003-004） |
| mobile 远程唤醒 | 不支持，仅本地恢复（D-003-005 补充默认） |
| 平台 | 仅 Windows（D-003-007） |
| 快捷键录制 | 复用 `pkg/shortcut` 现有录制体系（D-003-008） |
| 默认键位 | `Ctrl+Shift+H`（可在设置面板修改）（D-003-009） |

## 3. 验收标准

- [ ] 桌面启动 → 快捷键 `Ctrl+Shift+H`（含可改）→ 主窗口整体隐藏（任务栏入口同步消失）
- [ ] 隐藏后进程存活、SSE 继续推送、解题继续、F9 隐身状态不受影响
- [ ] 再次按 `Ctrl+Shift+H`（或托盘左键）→ 窗口恢复到原位置/尺寸/置顶/隐身状态
- [ ] OutputMode=mobile 且有手机连接时 → 自动隐藏主窗口；切回 desktop/dual → 自动恢复
- [ ] 托盘图标在隐藏/显示过程始终存在，托管恢复动作（左键恢复，右键菜单含退出）
- [ ] 设置面板可自定义隐藏快捷键（复用现有录制 UI），默认 `Ctrl+Shift+H`
- [ ] 三个状态**独立存储**：GhostModeState / WindowVisibleState / OutputModeState 互不合并
- [ ] `go build ./...`、`go vet`、`go test ./app/... ./pkg/...`、两层 JS 护栏全绿
- [ ] 真机（用户执行）：`wails build` 后手工验收上述交互

## 4. 非目标 / 明确不做

- ❌ 不远程唤醒窗口（无 SSE 扩展指令）
- ❌ 不做 macOS 实现
- ❌ 不新增独立的「最小化到托盘」状态（托盘仅恢复入口）
- ❌ 不修改 `pkg/mobile` SSE 协议 / `mobile.Bridge` / LLM / Solve / Prompt
- ❌ **不修改 GhostMode 现有实现**（防录屏逻辑原样保留）
- ❌ 不新增 npm/go 依赖（托盘用 Windows platform 层原生，不引入第三方）

## 5. 影响面预估

- 新增：`pkg/window/`（或并入 `pkg/state`）VisibleManager；`pkg/platform` 增 Windows `ShowWindow`/`Shell_NotifyIcon`；`app` 增 bindings；frontend settings 增键位。
- 修改建议最小化、默认低风险；`pkg/mobile` 不外改（见 02）。