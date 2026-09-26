# AGENT.md — Q-Solver 二次开发 Agent 行为规则（通用最高优先级）

> 本文件面向**所有 AI 编程 Agent**（Claude Code、Cursor、Codex、Gemini CLI 等），定义在本仓库工作的**统一行为边界**。优先级高于任何模型默认行为；具体 Agent 平台的特有规则分别在 `CLAUDE.md`（Claude Code）等文件中补充，若冲突以本文件为准。

---

## 0. 仓库性质与本文件目的

本仓库是 **Q-Solver**（Wails v2 + Go 后端 + Vue 3 前端的桌面 AI 截图解题工具）的**二次开发工作副本**。仓库内存在**两套严加区分的资产**：

1. **官方源码（A 类）**：`app/`、`pkg/`、`frontend/src/`、`main.go`、`go.mod`、`go.sum`、`package.json`、`package-lock.json`、`wails.json` —— **保持原样，与官方一致，不得被二次开发污染**。
2. **开发过程资产（B/C/D 类）** 应在 `_dev/` 等明确位置沉淀，见下文。

## 二、硬性红线（绝对不可违反）

| 规则 | 说明 |
|---|---|
| **禁止修改业务源码** | 除非该项变更已走 `_dev/changes/` 归档流程，否则不得改动 `app/**`、`pkg/**`、`frontend/src/**`、`main.go`、`go.mod`、`go.sum`、`package.json`、`package-lock.json`、`wails.json` 中的任何文件。 |
| **变更前必须归档** | 任何业务相关变更，先创建 `_dev/changes/<编号>-<名称>/`，在**修改代码之前**完成 `00-request.md`、`01-current-state.md`、`02-plan.md` 三份文档，之后按 `03` → `04` → `05` → `06` 补齐整个生命周期。 |
| **不提交 secret** | 密钥、Token、API Key 一律不得出现在代码、文档、日志、提交信息中。环境相关只能以 `SET` / `NOT SET` 占位说明。 |
| **分支纪律** | 二次开发仅在 `dev/qsolver-custom` 分支上工作，**不得直接修改或破坏 `main`（官方基线）**。禁止非授权 `git reset --hard` / `git push --force`。 |
| **不修改 agentmemory 相关配置** | 不得擅自打开/关闭 agentmemory 的推理类 feature flag（如 `AGENTMEMORY_AUTO_COMPRESS`、`AGENTMEMORY_ALLOW_AGENT_SDK`、`CLAUDE_MEMORY_BRIDGE` 等）。如需调整，走变更归档流程并说明理由。 |

## 三、开发记录归档（_dev/）

所有开发记录统一存放 `_dev/`：

```
_dev/
├── README.md                   归档区总览与使用规范
├── baseline/                  环境与开发基线（启动报告等）
├── changes/
│   ├── INDEX.md               变更总索引（编号分配表）
│   ├── _template/             变更模板（复制使用）
│   └── NN-名称/                每项变更的完整档案
└── memory/
    ├── README.md              长期记忆说明
    └── agentmemory-assessment.md
```

**新增一项变更的流程：**

1. 规划编号：查 `_dev/changes/INDEX.md`，取最小未用编号 `NNN`。
2. 建目录 `_dev/changes/NNN-名称/`，复制 `_template/` 的 7 份文档进去。
3. 在**动代码前**写 `00`（请求）、`01`（现状快照）、`02`（方案）。
4. 边做边写 `03`、`04`；如需回滚用 `05`；收尾 `06`。
5. 完成回写 `INDEX.md` 状态与摘要。

---

## 四、开发流程纪律

1. **先确认，再动手**：任何涉及业务源码的改动，先在归档栏出方案/计划，再实施。
2. **小步提交**：一次变更一个提交，commit message 说明动机；不要把一个变更拆进无关提交。
3. **不擅自开启功能**：不主动开启 agentmemory 推理类功能；不主动新增依赖；不做范围蔓延。
4. **构建与验证**：改完必须能 `go build ./...`、`npm run build`（前端）通过；能用 Wails 启动验证更好。
5. **运行状态**：临时启动的进程（如 agentmemory 服务）用完必须恢复原状态（停止）。秘密以 SET/NOT SET 呈现。

---

## 五、长期记忆（agentmemory）接入规范

- **记忆本体在仓库外**（本机 `E:\AI-Memory\agentmemory\`，`iii` 引擎，zero-LLM 模式）。
- **写入**：默认由 Claude 插件 hooks 自动记录；Agent 主动记录的只有关键结论/SET。
- **读取**：`agentmemory` 的 `smart-search` / hooks 注入经 MCP/REST 获取历史记忆。
- **评估依据**：见 `_dev/memory/agentmemory-assessment.md`。

---

## 六、验收与完成标准

- 变更归档完整（00~06 齐备）且信息真实。
- `INDEX.md` 状态已更新。
- `git status` 干净（除预期新文件外无意外变更）。
- 本规则文件的任何调整也应记录原因（可在 `_dev` 相应变更或本文件内注明）。