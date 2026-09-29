# 审计范围 00 — Change 003 Desktop Window Toggle（前置架构审计）

> 本审计为 **Change 003 开发前置**，只读探查，**不提出最终方案，不编写代码**。
> 目的是确认：当前 Q-Solver 是否具备实现「桌面窗口显示/隐藏切换（Window Toggle）」的架构条件，以及若要实现，涉及哪些层、存在哪些风险、需要用户拍板的决策点。

## 1. 审计目的

- 摸清当前 **快捷键体系**（shortcut / hotkey / keybinding / 触发链）。
- 摸清当前 **窗口控制能力**（Show / Hide / Minimize / Focus / AlwaysOnTop / GhostMode）。
- 分析 **GhostMode 现状**与「隐藏窗口」的关系。
- 分析 **Mobile Output（001/002）** 与「隐藏窗口」是否耦合——需求方最关心的：**隐藏窗口后 SSE / App 生命周期 / Shortcut 是否继续**。
- 摸清 **Windows 实现层**（HWND / syscall / Win32 API）现状。
- 产出 **风险清单（F- 编号）** 与 **用户决策清单（D-003- 编号）**。

## 2. 审计范围（只读）

| 范围 | 内容 |
|---|---|
| 源码 | `main.go`、`app/`、`pkg/state`、`pkg/shortcut`、`pkg/platform`、`pkg/mobile`、`pkg/config`、`frontend/src`（window/settings 相关） |
| 不触碰 | 不修改 app/pkg/frontend 任何文件；不改快捷键；不改窗口逻辑 |
| 产物 | 本目录 `00-scope.md` ~ `REPORT.md`（共 8 份） |

## 3. 审计方法

1. 读取 AGENT.md / CLAUDE.md / `_dev/README.md` / `_dev/memory/qsolver-memory-policy.md`（治理规则）。
2. 通读 `main.go`（Wails options 窗口初值）。
3. 读 `app/*.go`（窗口绑定、shortcut 绑定、mobile 路由）。
4. 读 `pkg/state/manager.go`（窗口可见性/焦点/句柄）。
5. 读 `pkg/shortcut/*` （服务、Manager、Windows 低层 hook）。
6. 读 `pkg/platform/platform_windows.go`（Win32 API 封装、GhostMode）。
7. 读 `pkg/mobile/*`（Server/Bridge，确认是否依赖窗口）。
8. 生成 Findings（F-）与用户决策（D-003-）清单。

## 4. 审计纪律（复查用）

- 本目录仅新增 8 份文档；`git status` 应只出现 `_dev/audits/003-desktop-window-toggle/`。
- 未建 `_dev/changes/003/`——尚未决定是否开发。
- 所有结论基于当前工作树（`dev/qsolver-custom`，三层提交后 clean）。