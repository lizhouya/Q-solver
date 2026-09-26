# CLAUDE.md — Q-Solver 项目 Claude Code 专用规则

> 本文件是 **Claude Code**（VS Code 扩展 / CLI）在本仓库工作的**专属补充规则**。适用于本文档的运行环境：Claude Code 在 VSCode 扩展内运行，API 经代理（`ANTHROPIC_BASE_URL`）接入，并装有 agentmemory 插件。
>
> **规则层级**：`AGENT.md`（通用，最高优先） > 本文件（Claude Code 专用）> 全局 `~/.claude/CLAUDE.md`（通用语言规范）。
> 本文件**仅补充** Claude Code 特有内容，**不复制** AGENT.md 内容——完整通用规则见 [AGENT.md](./AGENT.md)。

---

## 1. 环境概要

| 项目 | 值 |
|---|---|
| 技术栈 | Wails v2（`wails/v2@v2.11.0`）+ Go 后端 + Vue 3（Vite）+ WebView2 |
| 后端入口 | `main.go` + `app/`，业务模块在 `pkg/`（screen、llm、solution、shortcut、state、platform …） |
| 前端入口 | `frontend/src/`（Vue 3 + Vuex stores + services） |
| 构建 | `wails build`；前端 `npm run build`（vite build） |
| 开发分支 | `dev/qsolver-custom`（**不得触碰 `main` 官方基线**） |
| 运行环境 | Windows 11 Pro，PowerShell / Git Bash 均可 |

## 2. Claude Code 特有规则

### 2.1 工具使用偏好

- **文件读取编辑**：优先使用内置的 Read / Edit / Write / Glob / Grep 工具，而非 shell 的 cat / grep / find（shell 中文路径可能乱码）。
- **中文路径注意**：工作目录 `E:\DATA\求职_AI` 含中文，Bash 下 `cd` 可能乱码——目录类操作优先用 PowerShell；文件内容操作优先用 Claude 工具。
- **MCP 工具**：basic-memory（个人知识库）已连接，需要跨会话记忆时可用；不要将其误当成项目记忆本体。
- **secret 展示**：任何配置/环境变量值一律 `SET` / `NOT SET` 输出，绝不打印实值。

### 2.2 默认遵循 AGENT.md 红线

- 本文件默认**遵守** [AGENT.md](./AGENT.md) 的全部硬性红线（不碰业务源码、变更先归档 `_dev/changes/`、不提交 secret、分支纪律、不擅自开启 agentmemory 推理 flags）。
- 若出现与 AGENT.md 冲突的指示，**以 AGENT.md 为准**。

### 2.3 日常操作约定

| 场景 | 约定 |
|---|---|
| 新增变更 | 查 `_dev/changes/INDEX.md` 拿编号 → 建 `NNN-名称/` → 动码前写 `00/01/02` → 边做写 `03/04` → 收尾 `06` |
| 修改业务源码前 | 先确认归档文档已建；未归档不直接改代码 |
| 构建验证 | `go build ./...`、前端 `npm run build` 或 `wails build` 通过后再收尾 |
| 记忆查询 | 会话内用 agentmemory（如服务运行中）或读 `_dev/memory/agentmemory-assessment.md` |
| 环境信息 | 见 `_dev/baseline/Q-Solver 开发环境准备报告.md` |

## 完成条件

- 所有改动符合 AGENT.md 与归档流程。
- `git status` 干净（仅预期新增/修改文件）。
- 建即可在 VSCode 里查看；**commit / push 仅在用户明确要求时执行**（本仓库默认不自动提交）。