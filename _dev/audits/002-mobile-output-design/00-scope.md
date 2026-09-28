# 审计范围 — 002 Mobile Output Design

> 审计编号：`002-mobile-output-design`
> 审计类型：**只读架构设计审计**（read-only design audit）
> 执行日期：2026-09-26
> 分支 / 基线：`dev/qsolver-custom`（HEAD `b34bb04`，治理提交基线之后）
> 前置依赖：审计 001（`001-qsolver-architecture-audit`）的结论为事实基础

---

## 1. 审计目的

为 Q-Solver 规划 **「手机端浏览 AI 答案/解析」能力（Mobile Output）**，输出：

1. 当前 AI 结果产生 → 桌面展示的**完整调用链**（作为插入点基础）；
2. 候选传输架构（WebSocket / HTTP Polling / SSE / 其它）的利弊**比较**（**不选型**）；
3. 后端承载位置（app / pkg/platform）、前端承载形式（复用 Vue / 新 mobile 页 / embed）分析；
4. 手机端实时状态同步的数据结构设计；
5. 连接方式（二维码 / Token / 局域网 / 断线重连）；
6. 与审计 001 关键 Finding（F-001/002/004）的关联分析；
7. 产品决策清单（D-Mobile-001~006）——**不代用户选择**；
8. 风险分级。

## 2. 审计边界（本阶段必须遵守）

- 🔒 **只读**：禁止修改任何业务源码（`app/**`、`pkg/**`、`frontend/src/**`、`main.go`、`go.mod`、`go.sum`、`package.json`、`package-lock.json`、`wails.json`）。
- 🚫 禁止提供最终实现方案（候选可比较，不拍板）。
- 🚫 禁止开始编码。
- 📁 所有产出写入本目录：`_dev/audits/002-mobile-output-design/`

## 3. 产出文档

| 文件 | 内容 |
|---|---|
| `00-scope.md` | 本文件：范围、原则、产出 |
| `01-current-output-analysis.md` | 当前输出链路追踪（6 问：产生/存储/传输/事件/绑定/插入点） |
| `02-mobile-architecture-options.md` | 方案 A WebSocket / B HTTP Polling / C SSE / D 其他 — 利弊比较（不选型） |
| `03-data-flow-design.md` | 手机端状态同步数据结构（等待/截图完成/分析中/流式/最终） |
| `04-integration-points.md` | 后端承载位置（Wails/生命周期）+ 前端页面方案比较 + OutputMode 分析 |
| `05-technical-risks.md` | 风险分级（Confirmed / Potential / Needs Verification） |
| `06-user-decisions.md` | 产品决策问题（D-Mobile-001~006），不替用户做选择 |
| `REPORT.md` | 最终审计报告（入口文档，汇总 01~06） |

## 4. 审计原则

一切结论必须且只能来自**以下三类来源**，不得使用猜测、记忆推断或未经验证的 AgentMemory 信息：

1. **当前源码**（本分支 `dev/qsolver-custom` 工作区 = 官方基线 + 治理文件）
2. **Git history**（`git log` / `git show` 可查证的提交记录）
3. **既有开发文档**（`_dev/baseline/`、`_dev/audits/001/`、`_dev/memory/qsolver-memory-policy.md` —— 其中行为细节以源码见证为准）

### 4.1 确定性分级

| 级别 | 含义 | 使用场景 |
|---|---|---|
| `Confirmed` | 源码/git 三源可采信，且亲眼验证 | 直接采信 |
| `High Confidence` | 源码见证 + 文档佐证，行为基本确定 | 采信但有推断成分 |
| `Needs Verification` | 当前证据不足，或与文档矛盾 | **必须标出，留待验证** |

### 4.2 产品决策问题

- 所有「需要产品/负责人拍板」的非技术问题，统一登记在 `06-user-decisions.md`，编号 `D-Mobile-001` 起。
- **本审计不做选择**，仅列问题与依据。
- 技术性结论在 01~05 中给出，但**最终方案选择一样留给用户**。

## 5. 输出约定

- 所有结论附证据（`文件:行号` 或 commit 引用），无法验证的标 `Needs Verification`。
- 最终报告格式：`# Q-Solver AgentMemory + Mobile Output Design Report`，含 9 节。