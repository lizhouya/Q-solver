# _dev/memory/ — 长期记忆说明

本目录存放 Agent 长期记忆接入方案说明与评估，**不存放记忆本体**。

## 记忆本体在哪里？

Agent 的长期记忆由 **agentmemory**（[rohitg00/agentmemory](https://github.com/rohitg00/agentmemory)）管理，与本仓库解耦：

- **配置**：`~/.agentmemory/`（全局）或项目内 `.agentmemory/`
- **本地数据**：位于 `E:\AI-Memory\agentmemory\`（本机部署，非仓库内）
- **引擎**：`iii`（zero-LLM 模式：BM25 + on-device embeddings，无外部 LLM key 依赖）
- **接入 Claude 方式**：Claude Code 插件（12 个 hooks，见 `~/.claude/plugins/`）+ 可选 MCP server + REST API

## 为什么记忆不放在仓库内？

1. **隐私/权责**：记忆含会话内容、偏好、评估结论，放在仓库会随 git 扩散。
2. **性能**：索引存储（`state_queue.db` 等）体积随使用增长，不适合版本控制。
3. **关注点分离**：源码 = 代码资产；`_dev/` = 开发过程资产；记忆 = Agent 运行时资产。

## 本目录文件

| 文件 | 内容 |
|---|---|
| `agentmemory-assessment.md` | agentmemory 接入本项目的评估：现状、能力、缺口、建议 |
| （备选）`bridge-notes.md` | 若启用 Claude Memory Bridge 后的配置与说明 |

## 关于 AGENTS.md（多 Agent 协作时机）

仓库根级 `AGENTS.md` **当前未创建**。判断标准：

- [AGENT.md](../../AGENT.md) 已承载"通用行为规则"（所有 Agent 最高优先级）。
- `AGENTS.md` 的追加价值仅在出现**多 Agent 协作拓扑**时体现，例如：定义了 `.claude/agents/` 子代理、启用 teammates 多会话、或需要为 reviewer / implementer / 记忆维护者等分工时。

**启用触发条件**（任意满足即应补充 `AGENTS.md`）：
1. 引入 Claude Code 子代理（`.claude/agents/*.md`）从事独立子任务；
2. 需要两个以上角色并行协作（如"实现 Agent + 审查 Agent"）；
3. 明确了跨 Agent 的记忆/归档交接契约。

**在启用前**：所有协作细节收敛于 AGENT.md + CLAUDE.md，无需 AGENTS.md。

## 使用约定

- **读**：Agent 在会话中可查询 agentmemory（经 hooks 注入或 REST/MCP）获取历史记忆。
- **写**：由 hooks / 插件按事件（SessionStart、UserPromptSubmit、PostToolUse 等）自动记录；关键结论也可主动 `agentmemory` 写入。
- **备份额外**：若某记忆需要随仓库保留（团队交接、里程碑结论），将它**转写为 markdown 存到本目录**（如 `agentmemory-assessment.md`），并注明来源。
- **不**把运行态记忆库（db/索引）放进仓库。