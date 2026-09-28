# Q-Solver AgentMemory Policy

> 本文件定义 Q-Solver 二次开发阶段 **agentmemory（长期记忆）的使用策略**。
> 一句话：**记忆是辅助，源码是权威。**
> 编写日期：2026-09-26 ｜ 依据：AGENT.md 红线 + `_dev/memory/agentmemory-assessment.md` 评估结论 + 架构审计 001 结论。

---

## 一、目的（Purpose）

为 Q-Solver 二次开发建立一套**可长期驻留、跨会话复用、永不污染业务源码**的 AgentMemory 使用规范。目标是：

1. 把已确认的架构事实（`Confirmed` 级）沉淀为长期记忆，避免每个会话重复审计。
2. 防止 AgentMemory 中的数据被当作「事实来源」而超越源码本身。
3. 明确「什么能存、什么不能存、冲突时信谁」，让多会话协作行为一致。

## 二、允许保存（Allowed）

以下**已由源码/文档/Git 验证（Confirmed / High Confidence）**的信息可存入 AgentMemory：

- 架构决策与设计取舍（如：配置订阅驱动、事件流式输出）。
- 产品方向决策与用户拍板（如：D- 系列决策）。
- 技术方案结论（如：候选方案比较结论，标注未实现）。
- 已解决 Bug 的根因与修复点（供后续参考）。
- 用户 / Agent 协作偏好（中文交流、先归档后改码、不碰 `main` 等）。
- 长期约束类（分支纪律、只读红线、出于安全考虑的结构约束）。

## 三、禁止保存（Forbidden）

以下**一律不得**写入 AgentMemory：

- ❌ API Key、Secret、Token、密码及任何机密 **实值**（只允许 SET / NOT SET）。
- ❌ 临时日志、调试输出、一次性的运行噪音。
- ❌ 未经源码/审计柱证的猜测、推断（一律标 `Needs`，禁止写下）。
- ❌ 大量源码原文、完整配置文件、完整提示词（对象不持久化，记索引要点即可）。
- ❌ 单次安装 / 环境错误（可记「已验证可用」结论，不记错误事故经过）。

> 违规方式：写入时即坏——会用错误记忆污染以后会话的判断。**宁可少存，不可错存**。

---

## 四、记忆先后优先级（决定「冲突时信谁」）

当 AgentMemory 内容与其它来源冲突时，按以下顺序裁决，**Agent 越低**：

1. **当前源码**（`app/** pkg/** frontend/src/** main.go go.mod …`）——最高的唯一真相
2. **AGENT.md**（项目硬红线，皇权最高规则；CLAUDE.md 为 Claude Code 补充）
3. **`_dev/` 正式文档**（baseline / audits / changes / memory 中的 policy 与评估文档）
4. **Git history**（`git log` / `git show` 可证实的记录）
5. **AgentMemory**（辅助记忆，仅会在上面 4 项无法回答时参考）

> 规则：**记忆与源码冲突 → 源码获胜**。记忆会话中说是 A，源码里写着 B，一律以 B 为准，并同时更新内存条目消除冲突。

## 五、写入纪律（总结来源）

- **默认**：由 Claude 插件 hooks 自动记录工具使用。**不要求主动写入**。
- **主动写入**：仅限关键结论（SET 级）、用户的选择决定、审计发现确定级别为 `Confirmed` 的结论。
- 所有写入的 `NEVER` 写秘密实值；涉及环境信息一律 `SET` / `NOT SET`。
- 被临时启动的 agentmemory 服务，用完后必须按 AGENTS.md 恢复原状（停止）。

## 六、Scope（项目归属）

| 项 | 值 |
|---|---|
| 项目名（期望） | Q-Solver |
| 项目路径 | `E:\DATA\求职_AI`（当前 workspace root） |
| **agentmemory 自动派生名** | `求职_AI`（按 git 顶层目录 basename） |
| 是否需要显式创建 scope 文件 | **不需要** —— 插件按会话 cwd/git 根自动归组 |
| 若要强制 `Q-Solver` 名 | 设 `AGENTMEMORY_PROJECT_NAME=Q-Solver`（需经变更流程，本次未改） |

> 说明：agent 插件在 `SessionStart` 时把 `{sessionId, project, cwd}` POST 到本地 REST（`http://localhost:3111`），`project` 即 git 顶层 basename。**无需手工创建 scope**。当前项目记忆自动落在 `求职_AI` 分组。

## 七、第一批已保存的记忆索引

> 写入方式：2026-09-26 **临时启动** worker（zero-LLM 模式，`~/.agentmemory/.env` 仅 `AGENTMEMORY_INJECT_CONTEXT=true`，其余推理 flags 全部默认关闭），REST `POST /agentmemory/remember` 写入 4 条，`smart-search` 验证全部命中（score 1.0 / 0.98 / 0.98 / 1.0），随后 **`agentmemory stop` 停止服务**（端口 3111-3113 已释放，记忆持久化到磁盘，符合 AGENT.md「临时进程用完恢复原状」红线）。

| # | 内容摘要 | obsId（管理端可查） | 验证 |
|---|---|---|---|
| 1 | 项目技术栈/架构/开发分支（Wails2+Go+Vue3） | `mem_muii484l_...` | ✅ 命中 |
| 2 | 开发纪律：归档先行 + 分支 + secret | `mem_muii485...` | ✅ 命中 |
| 3 | 审计 001 关键发现 F-001/002/004/011/012 | `mem_muii486y_...` | ✅ 命中 |
| 4 | Mobile Output 未来方向（audit 002 未实现） | `mem_muii487j_...` | ✅ 命中 |

> 服务已停止；需重新全文读取时用 `npx @agentmemory/agentmemory` 拉起（zero-LLM）。

---

## 八、参考文档

- `_dev/CLAUDE.md` —— Claude Code 规则（环境、工具偏好、编译验证）
- `_dev/audits/001-qsolver-architecture-audit/REPORT.md` —— 审计 001 结论（F-01~F-20）
- `_dev/memory/agentmemory-assessment.md` —— agentmemory 能力评估 / flags
- AGENTS.md —— Go 项目最终规则

*（policy 建立于 Q-Solver 二次开发开始前，0 业务源码变更）*