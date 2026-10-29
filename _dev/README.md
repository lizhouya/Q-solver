# _dev/ — 二次开发归档目录

本目录是 Q-Solver 二次开发的**正式治理归档区**，存放开发过程中产生的所有**非正式源码**信息：开发记录、变更档案、环境准备报告与 Agent 记忆评估。

> **目录定位**：本目录**不属于** Q-Solver 官方源码，不参与应用构建；它是项目开发治理的"安全带"，让开发过程可追踪、可回溯、可交接。

## 规则关系

项目规则优先级：[AGENTS.md](../AGENTS.md)（通用最高规则）> [CLAUDE.md](../CLAUDE.md)（Claude Code 执行规则）> 当前 Change 文档（单次功能治理记录）。本总览、INDEX 和模板仅解释归档操作，不覆盖上位规则；旧 AGENT.md 仅作兼容跳转。历史审计和 Change 是证据，不构成当前 Git 操作授权。

## 目录结构

```
_dev/
├── README.md                    # 本文件：归档区总览与使用规范
├── audits/                      # 架构、设计与提交边界审计
├── build-artifact-policy.md      # 被忽略构建产物的目录、测试、晋升、哈希与回滚政策
├── baseline/                    # 开发环境与基线情报（开发起点快照）
│   └── Q-Solver 开发环境准备报告.md
├── changes/                     # 二次开发变更档案（每项变更一个子目录）
│   ├── INDEX.md                 # 变更总索引
│   ├── _template/               # 变更归档标准模板（复制使用，不直接修改）
│   └── 000-examples/            # 模板使用示例
└── memory/                      # Agent 长期记忆相关说明与评估
    ├── README.md
    └── agentmemory-assessment.md
```

## 使用规范

### 1. 归档区与源码的边界

| 区域 | 位置 | 说明 |
|---|---|---|
| **正式源码** | 仓库根（app/、pkg/、frontend/src/、main.go 等） | main 保留官方基线；开发分支仅按事前归档、确认方案与文件范围实施 |
| **开发记录** | `_dev/` | 所有过程性信息统一放这里 |
| **行为规则** | `AGENTS.md` / `CLAUDE.md` | 通用规则 / Claude Code 执行规则；AGENT.md 仅为兼容入口 |
| **长期记忆** | agentmemory（外部） | 见 `_dev/memory/README.md` |

### 2. 新增一项变更（严格流程）

1. 在 `_dev/changes/` 下创建归档目录：`<编号>-<短名称>/`（编号从 `001` 递增，见 `INDEX.md` 分配）。
2. 将 `_template/` 下的文档模板**复制**进该目录（`00-request` → `01-current-state` → `02-plan` 为前三份，必须**在任何业务代码修改前**创建）。
3. 边开发边按 `03` → `04` → `05` → `06` 补齐文档，并按 AGENTS.md 执行验证、提交边界审计、经授权的双提交和 Close；INDEX 必须反映真实进度，文档齐备不等于验收完成。

### 3. 禁区（绝对遵守）

- **不**将开发报告、计划和治理记录混入业务源码；功能代码在经确认的源码范围内修改，对应过程记录归档于 `_dev/changes/<编号>-<名称>/`，不得把归档要求误解为功能代码也应放入 `_dev/`。
- **不**允许任何开发过程文件（报告、计划、方案）常驻仓库根目录或 `docs/`，一律收进 `_dev/`。
- **不**提交任何密钥/Token：环境变量一律用占位符（如 `SET` / `NOT SET`）展示，绝不输出实值。

### 4. Git 纳管与忽略

- `_dev/` 全目录默认**纳入版本控制**（保留 git 历史），因此**不要**在 `.gitignore` 中忽略它。
- 模型文件、个人隐私数据、大型二进制若需放进 `_dev/`，放入前先确认是否应改为 `.gitignore` 条目或放置到仓库目录之外。

## 配套文档

| 文件 | 作用 |
|---|---|
| `AGENTS.md` | 面向所有 AI Agent 的项目通用最高规则 |
| `CLAUDE.md` | Claude Code 执行规则，服从 AGENTS.md |
| `AGENT.md` | 旧名称兼容入口，仅指向 AGENTS.md |
| `_dev/build-artifact-policy.md` | 被 Git 忽略的构建产物生命周期、SHA-256、晋升、归档和回滚政策 |
| `_dev/changes/XXX-名称/` | 单次功能治理记录，不覆盖上位规则 |
| `_dev/memory/README.md` | 长期记忆接入方案说明 |
| `_dev/memory/agentmemory-assessment.md` | agentmemory 接入评估结果 |
