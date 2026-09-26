# agentmemory 接入评估报告

> **评估对象**：Q-Solver 二次开发项目的 Agent 长期记忆方案 —— [agentmemory](https://github.com/rohitg00/agentmemory)（`@agentmemory/agentmemory`）
> **评估时间**：2026-09-26
> **评估性质**：静态配置检查 + 临时启动服务做链路验证后恢复原状（未持久改动任何配置）
> **结论**：**整套基础设施已安装可运行，但当前未对 Q-Solver 项目生效（无记忆、无会话、功能 flags 全关），属于"待启用"状态。**

---

## 1. 安装状态（已确认）

| 组件 | 路径/版本 | 状态 |
|---|---|---|
| npm CLI | `agentmemory` v0.9.29（`C:\Users\lizhou\AppData\Roaming\npm` 全局） | ✅ 已安装 |
| Claude Code 插件 | `agentmemory@agentmemory` v0.9.29（安装于 2026-08-20，git sha `2d38dafe…`） | ✅ 已安装 |
| 插件缓存 | `~/.claude/plugins/cache/agentmemory/agentmemory/0.9.29/`（hooks/ scripts/ skills/） | ✅ 存在 |
| 第三方引擎 | `iii`（zero-LLM：BM25 + on-device embeddings，无需外部 LLM key） | ✅ 随包 |
| 本地数据目录 | `E:\AI-Memory\agentmemory\`（iii-config.yaml、索引、stream log） | ✅ 存在（历史） |
| MCP server（可选） | `.mcp.json` 定义：`npx -y @agentmemory/mcp`（`AGENTMEMORY_URL=:3111`） | ✅ 已配置（未被 settings.json 引用） |

## 2. 配置状态（测试确认）

### 2.1 关键 feature flags（来源 `~/.claude/plugins/.../.env.example` / 项目 .env）

| 变量 | 项目 .env 当前值 | 含义 | 状态 |
|---|---|---|---|
| `AGENTMEMORY_AUTO_COMPRESS` | `no` | 对话自动压缩记忆 | 🔴 关 |
| `AGENTMEMORY_REFLECT` | `no` | 反思/固化 | 🔴 关 |
| `AGENTMEMORY_SLOTS` | `no` | 槽位记忆 | 🔴 关 |
| `AGENTMEMORY_GRAPH_EXTRACTION_ENABLED` | `no` | 知识图谱提取 | 🔴 关 |
| `AGENTMEMORY_CONSOLIDATION_ENABLED` | `no` | 记忆整合 | 🔴 关 |
| `CLAUDE_MEMORY_BRIDGE` | `no` | Claude Memory Bridge（Claude-SDK 桥） | 🔴 关 |
| `AGENTMEMORY_ALLOW_AGENT_SDK` | `no` | 允许 Agent SDK 接入 | 🔴 关 |
| `AGENT_ID` / `AGENTMEMORY_AGENT_SCOPE` | `no` | Agent 身份/作用域 | 🔴 关 |
| `TEAM_ID` / `USER_ID` | 注释态（未启用） | 团队/用户身份 | 🔴 关 |

**解读**：当前处于 **zero-LLM 记忆收集的守卫基线**——只保留 hooks 自动记录能力（SessionStart、UserPromptSubmit、PreToolUse/PostToolUse、Subagent…等 12 类事件），所有消费型/推理型功能全部关闭。这是**安全默认**，符合"不擅自开启 agentmemory 功能"的项目约定。

### 2.2 Claude 接入链路

| 接入方式 | 状态 |
|---|---|
| Claude 插件（12 hooks 系统级） | ✅ 已装 |
| `~/.claude/settings.json` | 无 hooks / 无 mcpServers（不含 agentmemory MCP） |
| 独立 MCP 注册 | 🔴 未在 settings 中注册（agentmemory MCP 属于**可选补充**，当前不走 MCP 通道） |
| 服务运行状态 | 🔴 默认未运行（仅验证时临时启动，已恢复停止） |

**结论**：当前 Claude ↔ agentmemory 的连接为"**插件已装、hooks 就位、但服务未常驻、推理 flags 全闭**"，memory 实际未被使用。

## 3. 服务链路验证（临时启动 → 验证 → 已恢复停止）

| 验证项 | 结果 |
|---|---|
| 启动 | ✅ `iii-engine` 进程启动成功，zero-LLM 模式（BM25 + on-device embeddings） |
| REST API | ✅ `:3111` liveness = ok，health = healthy，circuit 关闭 |
| 引擎 | ✅ 272 个 function，workers 连接，`memory_heap_tight (95% RSS 145MB)` |
| Streams / Viewer | ✅ `:3112`（ws）/ `:3113`（web viewer） |
| **Q-Solver 项目识别** | ❌ **未识别**——smart-search 查询 Q-Solver/求职/截图解题 均返回 `results:[]`，sessions 仅余 2026-08-20 的 `test-project` 试跑数据 |
| 记忆库内容 | ❌ 无 Q-Solver 相关实体/观测（记忆库基本为空，仅有历史测试数据） |
| 恢复 | ✅ 验证后已停止服务，端口释放，回到"未运行"原状 |

**关键发现**：
1. **Q-Solver 项目从未被 agentmemory 收录**（无 session、无记忆实体）——因为服务一直未运行 + 项目未 setup。
2. 记忆库（`E:\AI-Memory\agentmemory\`）存在历史 test-project 试跑数据，可作功能演示但不属本项目。
3. 服务若需纳入工作流，需"常驻运行"才能让 hooks 注入上下文；当前默认关闭,不会自动生效。

## 4. 优势（为何选择该方案）

1. **零 LLM、本地化**：BM25 + on-device embedding，无外部 API key，数据留本机，隐私性好。
2. **深度集成 Claude Code**：插件 12 hooks 覆盖完整会话生命周期（读文件、Subagent 事件、PreCompact 等），无需手动维护。
3. **多入口**：REST :3111 / Streams :3112 / Viewer :3113 / MCP / skills（17 个），便于调试与扩展。
4. **相对成熟**：版本 0.9.x，API 稳定，schema 齐全。

## 5. 当前缺口 / 需注意

1. **项目未被记忆**：Q-Solver 零记忆。若启用，需"确立项目身份"（AGENT_ID/项目 scope）+ 首次接入流程。
2. **服务常驻问题**：agentmemory 需要常驻才会自动（hooks 注入、上下文检索）。当前默认关，若上线需计划自动启动（开机/按需）。
3. **推理类功能未验证**：AUTO_COMPRESS/REFLECT/SLOTS 等均未开启，实际效果未经真实工作流检验。
4. **`memory_heap_tight` 告警**：引擎 RSS 145MB/hold 95%，长时间运行需监控内存。
5. **多项目共存**：当前靠 cwd 识别项目；`E:\DATA\求职_AI` 与 `agent memory\test-project` 等历史数据需区分。

## 6. 建议

| 优先级 | 动作 | 说明 |
|---|---|---|
| P1（当前） | **保持现状关闭** | 符合项目约定；不擅自开启推理 flags |
| P1（当前） | 本评估归档 | 本文入 `_dev/memory/`，后续启用时可作实测依据 |
| P2（启用时） | 明确 Q-Solver 项目身份 | 设置 `AGENT_ID` + 项目 scope，让记忆归属本项目 |
| P2（启用时） | 规划服务常驻策略 | 常驻才能让 hooks 自动读写；考虑按需启停脚本 |
| P3 | 分阶段开启 | 先开基础（项目识别+记录），再评估 AUTO_COMPRESS / REFLECT 等推理功能 |
| P3 | 监控 | 观察引擎内存（heap_tight 告警） |

## 7. 变更授权情况

本次评估**未修改任何配置**：不新增 hook、不开 flag、不写入 .env、不提交 secret；未改动业务源码；服务验证后已恢复原状（停止）。如需启用任何 agent入口功能，走 `_dev/changes/` 归档流程。

---
*附：本评估对应的环境证据（版本号、路径）均采集于 2026-09-26 会话；后续版本升级后如需复核，应重新 capture。*