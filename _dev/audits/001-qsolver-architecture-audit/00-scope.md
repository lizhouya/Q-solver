# 审计范围 — 00

> 审计编号：`001-qsolver-architecture-audit`
> 审计类型：**只读架构审计**（read-only architecture audit）
> 执行日期：2026-09-26
> 分支 / 基线：`dev/qsolver-custom`（baseline HEAD `834e9fe`，治理提交 `b34bb04` 之后）

## 1. 审计目的

建立 Q-Solver **二次开发前的真实架构认知**：当前系统由哪些模块构成、如何调用、数据如何流动、哪些能力真实存在、哪些只是"看起来有"。

## 2. 审计边界（本阶段必须遵守）

- 🔒 **只读**：禁止修改任何 Q-Solver 业务源码（`app/**`、`pkg/**`、`frontend/src/**`、`main.go`、`go.mod`、`go.sum`、`package.json`、`package-lock.json`、`wails.json`）。
- 🚫 禁止设计最终改造方案。
- 🚫 禁止开始编码。
- 📁 所有产出写入本目录：`_dev/audits/001-qsolver-architecture-audit/`

## 3. 产出文档

| 文件 | 内容 |
|---|---|
| `00-scope.md` | 本文件：范围、原则、事实来源 |
| `01-project-architecture.md` | 项目整体架构（分层 + 模块 + 文件位置 + 关键函数） |
| `02-module-analysis.md` | 前后端各模块职责分析 |
| `03-call-chain.md` | 核心调用链追踪（LLM、截图、快捷键、历史等） |
| `04-findings.md` | 问题分类（Bug Candidate / Design Limitation / …） |
| `05-user-decisions.md` | 需要用户做产品决策的点 |
| `06-risk-analysis.md` | 风险分析 + AgentMemory 使用边界 |
| `REPORT.md` | 最终审计报告（入口文档，汇总 01~06） |

## 4. 审计原则

一切结论必须且只能来自**以下三类来源**，不得使用猜测、记忆推断、既往讨论结论或未经验证的 AgentMemory 信息：

1. **当前源码**（本分支 `dev/qsolver-custom` 工作区 = 官方基线 + 治理文件）
2. **Git history**（`git log` / `git show` 可查证的提交记录）
3. **已存在开发文档**（`_dev/baseline/Q-Solver 开发环境准备报告.md` —— 前期环境验证的正确结论可引用为既有文档事实，但它对行为细节的描述需以源码见证为准）

### 4.1 确定性分级

| 级别 | 含义 | 使用场景 |
|---|---|---|
| `Confirmed` | 源码/git/文档三源可采信，且亲眼验证 | 直接采信 |
| `High Confidence` | 源码见证 + 文档佐证，行为基本确定的间接观察 | 采信但有推断成分 |
| `Needs Verification` | 当前证据不足以定论，或存在与文档矛盾的描述 | **必须标出，留待验证** |

### 4.2 发现标记

- 发现的问题统一登记在 `04-findings.md`，用连续 `F-001` 编号。
- 类别只能取：
  - `Bug Candidate` / `Design Limitation` / `Maintenance Risk` / `Security Concern` / `UX Limitation` / `Compatibility Limitation` / `Missing Coverage` / `Improvement Opportunity`
- 给出：模块、证据位置（`文件:行号`）、当前行为、影响、确定性级别。
- **不得写解决方案**（本阶段不做方案设计）。

## 5. 输出约定

- 所有结论附证据（`文件:行号` 或 commit 引用）。
- 无法从源码验证的描述一律标 `Needs Verification`，不臆断。
- 最终报告格式：`# Q-Solver 架构审计报告`，含 10 节。