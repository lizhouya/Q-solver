# AGENTS.md — Q-Solver 项目通用最高规则

> 本文件面向**所有 AI 编程 Agent**，是项目通用规则的唯一权威入口。旧文件 [AGENT.md](./AGENT.md) 仅保留兼容跳转，不维护第二套规则。
>
> **修订说明（2026-10-14）**：从 AGENT.md 迁移通用规则，统一规则层级、双提交模式、源码修改边界和验收条件；本次仅修改规则文档，不修改业务代码、不提交、不推送。

## 一、规则层级与职责

项目内规则按以下顺序解释；此层级不覆盖宿主平台的系统、安全或工具权限约束：

1. **AGENTS.md：项目通用最高规则。** 定义所有 Agent 共同遵守的源码边界、Git 授权、安全、Change 生命周期和验收底线。
2. **CLAUDE.md：Claude Code 执行规则。** 在通用规则之下说明项目架构、工具使用、开发命令和操作步骤；可以细化，不得放宽通用底线。
3. **当前 Change 文档：单次功能治理记录。** 记录需求、确认范围、设计、实现、验证、回滚和关闭状态，仅对该 Change 生效，不是全局行为规则或 Git 操作授权。

- 阅读顺序：AGENTS.md → 适用 Agent 执行规则（Claude Code 读取 CLAUDE.md）→ 当前 Change 的 00/01/02 及后续记录。
- Change 可以在上位规则内收窄文件范围、增加验收条件；这些约束应同时遵守。“更具体”不意味着能覆盖上位规则或自动扩大权限。
- `_dev/README.md`、INDEX 和模板是归档操作说明，必须与本文件一致；审计、旧 Change、历史日志和外部记忆仅作为事实证据，不具有全局规则效力。
- 冲突时暂停受影响操作，报告冲突条款与处理方案；如需调整通用规则，先取得用户针对规则修改的明确授权，再更新本文件及相关执行说明。不以修改 Change 文档代替规则变更。
- 通用制度在本文件维护，CLAUDE.md 的相关说明是执行细化；修改制度时同步检查执行规则，避免维护互相矛盾的副本。

---

## 仓库性质与本文件目的

本仓库是 **Q-Solver**（Wails v2 + Go 后端 + Vue 3 前端的桌面 AI 截图解题工具）的**二次开发工作副本**。仓库内存在**两套严加区分的资产**：

1. **官方源码（A 类）**：`app/`、`pkg/`、`frontend/src/`、`main.go`、`go.mod`、`go.sum`、`frontend/package.json`、`frontend/package-lock.json`、`wails.json` —— **main 保留官方基线；开发分支仅允许在完成事前归档、确认方案与文件范围后修改，禁止未归档或越界修改**。
2. **开发过程资产（B/C/D 类）** 应在 `_dev/` 等明确位置沉淀，见下文。

## 二、硬性红线（绝对不可违反）

| 规则 | 说明 |
|---|---|
| **禁止修改业务源码** | 除非该项变更已走 `_dev/changes/` 归档流程，否则不得改动 `app/**`、`pkg/**`、`frontend/src/**`、`main.go`、`go.mod`、`go.sum`、`frontend/package.json`、`frontend/package-lock.json`、`wails.json` 中的任何文件。 |
| **变更前必须归档** | 任何业务相关变更，先创建 `_dev/changes/<编号>-<名称>/`，在**修改代码之前**完成 `00-request.md`、`01-current-state.md`、`02-plan.md` 三份文档，之后按 `03` → `04` → `05` → `06` 补齐整个生命周期。 |
| **不提交 secret** | 密钥、Token、API Key 一律不得出现在代码、文档、日志、提交信息中。环境相关只能以 `SET` / `NOT SET` 占位说明。 |
| **分支纪律** | 二次开发仅在 `dev/qsolver-custom` 分支上工作，**不得直接修改或破坏 `main`（官方基线）**。禁止非授权 `git reset --hard` / `git push --force`。 |
| **不修改 agentmemory 相关配置** | 不得擅自打开/关闭 agentmemory 的推理类 feature flag（如 `AGENTMEMORY_AUTO_COMPRESS`、`AGENTMEMORY_ALLOW_AGENT_SDK`、`CLAUDE_MEMORY_BRIDGE` 等）。如需调整，走变更归档流程并说明理由。 |

## 三、开发记录归档（_dev/）

所有开发记录统一存放 `_dev/`：

```
_dev/
├── README.md                   归档区总览与使用规范
├── audits/                    架构、设计与提交边界审计
├── baseline/                  环境与开发基线（启动报告等）
├── changes/
│   ├── INDEX.md               变更总索引（编号分配表）
│   ├── _template/             变更模板（复制使用）
│   └── NNN-名称/                每项变更的完整档案
└── memory/
    ├── README.md              长期记忆说明
    └── agentmemory-assessment.md
```

**新增一项变更的流程：**

1. 规划编号：查 `_dev/changes/INDEX.md`，并核对实际目录和历史记录，取最小未用三位编号 `NNN`；已分配编号不得因索引遗漏而复用。
2. 建目录 `_dev/changes/NNN-名称/`，复制 `_template/` 的 7 份文档进去。
3. 在**动代码前**写 `00`（请求）、`01`（现状快照）、`02`（方案）。
4. 边做边写 `03`、`04`；如需回滚用 `05`；收尾 `06`。
5. 按验证、提交和关闭的真实进度回写 `INDEX.md` 状态与摘要，不提前标记完成。

统一生命周期：**需求确认 → 架构审计 → 设计方案 → 实现 → 测试验证 → Commit Boundary Audit → 经授权 Commit → Close**。前 3 阶段对应动码前的 00/01/02；03/04 记录实现与验证，05/06 保存回滚和总结。历史档案的 05-decisions / 06-rollback 命名继续保留，不为统一名称改写历史。

纯规则文档维护不涉及业务实现时，无需虚构功能 Change；必须限定规则文件范围、记录修改原因并完成文档校验。

---

## 四、开发流程纪律

1. **先确认，再动手**：任何涉及业务源码的改动，先在归档栏出方案/计划，再实施。
2. **提交边界**：一个 Change 不混入其他 Change。功能变更采用双提交：Commit 1 为功能代码与测试（`feat:` / `fix:`），Commit 2 为审计、Change 档案与 INDEX 等治理记录（`chore: add change XXX governance records`）。纯治理任务只需治理提交，不制造空功能提交。
   - 每次 Git 操作前检查 `git status`、`git branch`、`git remote -v`、`git diff`；预检本身不递归。提交前另查 `git diff --cached`。
   - 禁止 `git add -A`、`git add .`；仅按经确认的具体文件路径暂存。提交前明确目的、文件范围和完整 message。
   - commit、push、release 均须用户明确授权；方案、Change 文档或审计通过不等于操作授权。发布还须确认目标仓库、Tag 和自动化副作用，禁止误操作 upstream。
   - 未获提交授权时停在待提交；用户要求不提交时不得为关闭 Change 自动提交。
3. **不擅自开启功能**：不主动开启 agentmemory 推理类功能；不主动新增依赖；不做范围蔓延。
4. **构建与验证**：功能修改必须通过相关测试及 `go test ./...`、`go build ./...`；涉及前端运行 `npm --prefix frontend run build`，涉及 Wails 集成或打包运行 `wails build`。必要的人工验收不能以构建代替。逐项记录通过、失败、阻塞、未执行及已知问题，禁止伪造结果。纯规则文档修改可只做内容、引用与 diff 校验，明确业务测试/构建未执行。
5. **运行状态**：临时启动的进程（如 agentmemory 服务）用完必须恢复原状态，不停止原本已运行的用户服务。秘密以 SET/NOT SET 呈现。

---

## 五、长期记忆（agentmemory）接入规范

- **记忆本体在仓库外**（本机 `E:\AI-Memory\agentmemory\`，`iii` 引擎，zero-LLM 模式）。
- **写入**：默认由 Claude 插件 hooks 自动记录；Agent 主动记录的只有关键结论/SET。
- **读取**：`agentmemory` 的 `smart-search` / hooks 注入经 MCP/REST 获取历史记忆。
- **评估依据**：见 `_dev/memory/agentmemory-assessment.md`。

---

## 六、验收与完成标准

- 功能 Change 的 00~06 共七份档案完整且真实；历史命名可保留，但需求、设计、验证和回滚等职责不可缺失。
- **验证未完成不得 Close**；必要验收存在 FAIL、BLOCKED、NEEDS_USER 或未执行项时保持待处理状态，不能把已提交等同于已验收。
- 关闭前完成提交边界审计及经授权的功能/治理提交，核实相关提交并同步 INDEX 与总结。关闭不要求向 main 合并，也不隐含 push 或 release。
- 治理提交前准备好总结和 INDEX；若提交后又修改文档，应如实报告未提交差异，另行请求授权，不自动 amend 或补交，也不要求文档记录其自身提交哈希。
- 工作区只能存在已说明的预期改动；未提交时如实报告“仅预期未提交修改”，不称为 clean，不为清理状态丢弃用户改动。
- 本规则文件的任何调整也应记录原因（可在 `_dev` 相应变更或本文件内注明）。

## 2026-10-22 v1.1.0-rc.1 一次性授权例外

用户在确认提交边界、公开目标仓库、RC Tag、自动发布影响和已知验证缺口后，明确批准“RC 整理与发布”。仅本次允许：

- 将叠加的 Change 010～014 功能与测试整合为一个提交；维护规则单独提交，发布工具链及治理记录单独提交。后续仍遵守原有逐 Change 提交规则。
- 在如实披露 race 环境阻塞及未完成的人工验收时，发布到个人 fork 的 v1.1.0-rc.1 **预发布**，不标为稳定版或 Latest；此例外不等于 Close 旧 Change，不伪造验证通过。
- 推送目标仅 origin/dev/qsolver-custom 和 v1.1.0-rc.1，不涉及 main/upstream，不授予其他 Tag、强制推送或删除 Release 权限。

此条记录用户本次明确决定，不构成后续默认授权。其他安全、配置、数据保护规则保持有效。实施记录见 `_dev/changes/015-rc-release/`。
