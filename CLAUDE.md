# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **角色：Q-Solver 项目维护 Agent。** 默认使用简体中文交流和编写文档，代码标识符、命令及第三方产品名称保持原样。
>
> **项目规则层级**：[AGENTS.md](./AGENTS.md)（项目通用最高规则）> 本文件（Claude Code 执行规则）> 当前 Change 文档（单次功能治理记录）。开始任务按此顺序阅读；旧 AGENT.md 仅是兼容入口。此项目内层级不覆盖宿主平台的系统、安全或工具权限约束。
>
> **职责边界**：本文件把 AGENTS.md 的通用制度细化为 Claude Code 的工具、命令和操作步骤，不独立放宽通用规则。Change 可收窄本次范围、增加验收要求，但不能覆盖上位规则、授予 commit/push/release 权限，或成为其他 Change 的默认规则；全局语言偏好用于项目未约定的事项。
>
> **冲突处理**：暂停受影响操作并报告具体条款；需要变更通用制度时，先获得用户明确授权，再修改 AGENTS.md 并同步相关执行说明，不能仅修改 Change 记录绕过规则。
>
> **修订记录（2026-10-14）**：先完善长期维护规范并修正 Pinia 架构说明；本轮规则审计进一步建立 AGENTS.md → CLAUDE.md → Change 层级，统一提交和关闭条件。仅修改规则文档，保留已有未提交内容，不修改业务代码或历史 Change 状态，不执行 commit / push。

## 1. 项目身份

- **项目**：Q-Solver，Go + Wails 架构的 AI 桌面助手，支持截图解题、流式答案、历史上下文、桌面窗口控制及手机端输出。
- **仓库性质**：个人 fork 二次开发仓库，不是官方主仓库。
- **本地仓库**：`E:\DATA\求职_AI\求职笔试\Q-Solver`（2026-10-19 已完成完整副本复制、一致性验证和旧副本清理）。
- **origin**：https://github.com/lizhouya/Q-solver.git —— 用于个人开发与经授权的发布。
- **upstream**：https://github.com/jym66/Q-solver.git —— 用于同步官方项目；不得误向其推送个人开发内容。
- **开发环境**：Windows 11 Pro、PowerShell / Git Bash、WebView2；Wails Go 模块当前为 v2.11.0。
- **许可**：README 标明 CC BY-NC 4.0，仅限非商业个人学习与研究；维护文档时保留许可边界。

### 架构与数据流

- `main.go` 嵌入 `frontend/dist`，创建 Wails 窗口，将 `app.App` 绑定给前端并注册启动/关闭生命周期。
- `app/` 是编排和 Wails 接口层：启动时装配配置、状态、任务、截图、LLM、Solver、快捷键、Mobile 和托盘服务；关闭时释放快捷键、托盘、Mobile 服务并保存配置。
- 解题主链路：前端或快捷键 → `app/solve.go` → 截图缓冲与任务协调 → `pkg/solution` → `pkg/llm` → 回调 `App.EmitEvent`。KeepContext 的历史上下文由 Solver 管理，不能与前端展示历史混为一谈。
- 桌面前端是 **Vue 3 + Pinia + Vite**。`frontend/src/services/api.js` 封装 Wails 生成的 Go bindings，`services/events.js` 封装后端事件，stores 管理设置、答案和 UI；实际全局样式入口为 `styles/global.css`。
- `app/mobile.go` 只路由解题输出事件：`desktop`、`mobile`、`dual` 分别对应桌面、手机、双端；Mobile 不可用时手机独占模式降级为桌面。Toast 等非解题事件仍走桌面 Wails。
- `pkg/mobile` 维护答案快照、订阅和局域网 HTTP/SSE 服务，嵌入 `pkg/mobile/web` 作为远程 UI；这不是桌面 Vue 前端的同一构建入口。
- GhostMode、WindowVisibility、ClickThrough、OutputMode 是不同状态，不得合并语义；Windows/macOS 平台实现分别位于相应平台文件，Windows 验证不能冒充跨平台验证。

## 2. Git 仓库规则

每次 Git 操作前，必须先执行以下只读预检；预检命令自身不递归要求再做一轮预检：

```bash
git status
git branch
git remote -v
git diff
```

- 确认工作区改动、当前分支、远程目标和未暂存差异；涉及提交时还必须检查 `git diff --cached`，不能以 `git diff` 为空推断暂存区为空。
- 禁止 `git add -A`、`git add .`。经授权暂存时，只能使用明确文件路径，例如 `git add -- <已确认的文件路径>`。
- 禁止未确认直接 commit 或 push；“修改完成”“测试通过”“请收尾”均不构成提交或推送授权。
- 所有提交必须先列明：**提交目的、提交文件范围、完整 commit message**；推送另须明确 remote、分支或 Tag。
- 不得夹带用户已有改动、个人配置、构建产物或无关生成文件；发现非本任务改动时保留并报告。
- 遵守 AGENTS.md：禁止未经授权的 `git reset --hard` / `git push --force`，不得为获得干净工作区而丢弃已有改动。

## 3. 分支管理规则

- 二次开发分支固定为 **`dev/qsolver-custom`**。
- **禁止直接修改 `main`**，其作为官方基线保留；不擅自切换、合并、重置或重写该分支。
- 若当前分支不是预期分支，先报告并等待用户决定，不自动切换。
- 所有功能开发必须创建 **`Change XXX`**，其中 `XXX` 为三位编号，例如 `Change 010`。
- 分配编号先查 `_dev/changes/INDEX.md`，再核对实际目录与历史记录；已使用的编号不得复用。003/004 已有目录，不能因索引漏登就重新分配。
- 对应目录为 `_dev/changes/XXX-名称/`；未完成事前归档，不得修改 AGENTS.md 列出的业务源码、依赖或 Wails 配置。

## 4. Change 生命周期规范

按 AGENTS.md 第三、六节执行生命周期与关闭条件。以下是 Claude Code 的执行映射，不是 Change 文档自行定义的全局规则；审计与变更档案相互引用：

```text
_dev/
├── audits/                     架构审计、设计审计、提交边界审计
└── changes/
    ├── INDEX.md                Change 编号、状态与摘要
    ├── _template/              新 Change 模板
    └── XXX-名称/               单项变更的全过程记录
```

| 阶段 | 名称 | 必须完成的工作 |
|---|---|---|
| Phase 1 | 需求确认 | 明确目标、非目标、允许文件范围和验收标准；编写 `00-request.md`。 |
| Phase 2 | 架构审计 | 核查当前调用链、风险和平台约束；编写 `01-current-state.md`，关联 `_dev/audits/` 审计记录。 |
| Phase 3 | 设计方案 | 编写 `02-plan.md`，明确实现、测试及回滚方案；方案确认后才能动业务代码。 |
| Phase 4 | 实现 | 限定在确认范围内小步修改，同步记录 `03-implementation.md`。 |
| Phase 5 | 测试验证 | 执行相关测试、构建和必要人工验收，在 `04-validation.md` 如实记录结果。 |
| Phase 6 | Commit Boundary Audit | 审计文件白名单、代码/治理分离、测试证据、secret 和生成噪音；列明每个提交的目的、范围与 message。 |
| Phase 7 | Commit | 仅在用户明确确认后按第 5 节提交；未授权就保持“待提交”，不能自动越过。 |
| Phase 8 | Close | 验证完成、归档完整、提交信息核实后，更新总结与 INDEX；报告遗留项及最终工作区状态。 |

- 新 Change 复制 `_template/` 的七份文档：`00-request.md`、`01-current-state.md`、`02-plan.md`、`03-implementation.md`、`04-validation.md`、`05-rollback.md`、`06-notes.md`。
- 现有部分档案采用 `05-decisions.md` + `06-rollback.md`；维护时尊重已有命名，不能仅因缺少 `06-notes.md` 就认定整个档案不存在，也不擅自重命名历史文件。
- **没有完成验证，不允许关闭 Change。** `FAIL`、`BLOCKED`、`NEEDS_USER` 或未执行的必要验收不能标为通过；提交完成不等于验收完成。
- 按 AGENTS.md 第六节提前规划总结、INDEX 与治理提交；提交后有新文档改动时如实报告并另行请求授权，不自动 amend 或补交，不要求文档写入自身提交哈希。必要验证和授权提交完成前，不报告正式关闭。

## 5. Commit 规范

按 AGENTS.md 第四节的**双提交模式**执行：同一个 Change 的两类资产独立提交，不混入其他 Change：

| 提交 | 文件范围 | commit message 示例 |
|---|---|---|
| Commit 1 | 功能代码 + 对应测试 | `feat: <功能说明>` 或 `fix: <修复说明>` |
| Commit 2 | 审计、Change 文档、INDEX 等治理记录 | `chore: add change XXX governance records` |

- 两个提交分别列出文件清单和 message，得到明确授权后再操作；一次批准不能隐含授权 push 或 release。
- 纯治理任务只规划治理提交，不为满足双提交而制造空的功能提交；本次规则编写任务不提交。
- 通用提交制度以 AGENTS.md 为准，本节提供 Claude Code 的文件划分与 message 示例；不从旧 Change 的提交记录推导新的授权。
- Commit Boundary Audit 后若代码或测试发生变化，须重新验证相关内容并复核提交边界。

## 6. 测试规则

任何功能修改必须运行相关测试；至少执行：

```bash
go test ./...
go build ./...
```

涉及 frontend 时执行：

```bash
npm --prefix frontend run build
```

涉及 Wails 桌面绑定、窗口集成或打包时执行：

```bash
wails build
```

### 已核实的开发与聚焦验证命令

以下命令在仓库根目录运行；依赖安装、启动应用和构建前说明影响范围，不擅自升级依赖。

| 用途 | 命令 / 说明 |
|---|---|
| 安装锁定的前端依赖 | `npm --prefix frontend ci` |
| 桌面联调 | `wails dev` |
| 仅启动 Vite | `npm --prefix frontend run dev`；不等同于完整 Wails 运行环境。 |
| 运行解题模块测试 | `go test ./pkg/solution -count=1` |
| 运行单个 KeepContext 测试 | `go test ./pkg/solution -run '^TestTKC02KeepTrueCarriesHistory$' -count=1 -v` |
| 运行 SSE 首次 Flush 测试 | `go test ./pkg/mobile -run '^TestSSESubscribesBeforeInitialFlush$' -count=1 -v` |
| Go 静态检查 | `go vet ./...`；与测试、构建结果分别记录。 |
| 发布构建（README / CI 已使用） | `wails build -ldflags "-s -w" -tags prod` |
| 差异空白检查 | Git 预检后执行 `git diff --check`。 |

- README / CI 使用 Go 1.25+、Node.js 22+；`go.mod` 的 Go 指令为 1.24.1，Wails 模块为 v2.11.0，不擅自将其升级到本机 CLI 版本。
- `main.go` 需要嵌入 `frontend/dist`。新环境若因缺少前端产物而导致 Go 构建/测试失败，先构建前端，再重新执行并记录结果。
- 当前前端 scripts 只有 `dev` / `build` / `preview`，未定义 `test` 或 `lint`；不得编造 `npm test` / `npm run lint` 已通过。
- 每项验证记录命令、环境、退出结果、通过/失败及已知问题；无法执行的注明原因，必要人工验收标为 `NEEDS_USER`。禁止伪造测试结果，也不得把历史测试结果写成当前验证。
- 历史记录存在 Windows hook 的 `unsafe.Pointer` vet 告警；若复现，保留原始失败结果。禁用检查后的结果必须单列，不能替代标准检查通过。
- Wails 构建可能改动 `frontend/wailsjs`、`go.mod`、`go.sum` 等生成或依赖文件；构建前后核对差异，不能混入提交或未检查就批量恢复。
- 纯规则文档任务可只做文档完整性、链接和 diff 检查，明确说明未运行业务测试/构建，不宣称业务验证通过。

## 7. GitHub CLI 管理规则

项目使用 **GitHub CLI `gh`**，GitHub 操作优先使用 gh。执行前核实目标仓库，优先显式指定个人 fork，避免 fork 环境默认指向 upstream：

```bash
gh repo view --repo lizhouya/Q-solver
gh pr list --repo lizhouya/Q-solver
gh issue list --repo lizhouya/Q-solver
```

经用户确认内容和目标后，可以使用：

```bash
gh issue create --repo lizhouya/Q-solver
gh release create <tag> --repo lizhouya/Q-solver --verify-tag
```

- 创建 Issue 属于对外写入，先确认标题、正文和仓库。
- **未经用户明确确认，禁止创建 Release、删除 Tag、合并 PR、修改默认分支。**
- 不因 gh 已安装就声称认证或远程权限已验证；不输出 Token、账号凭据或认证配置实值。

## 8. Release 管理规则

Release 前必须确认：

- Git 预检完成，`git status` 工作区及暂存区均 clean。
- 发布所对应提交的必要测试通过、构建成功，人工验收已完成。
- README 已同步功能和使用方式，CHANGELOG 已更新；当前根目录未发现 CHANGELOG，首次发布准备时应先确认并补齐，不能假定已存在。
- Tag、目标 commit、发布仓库、产物和 Release 文案均已确认，例如 `v0.1.0`；示例不是预定版本号。

经明确授权后按顺序执行：

1. 在已确认的 commit 上创建 Tag。
2. 将该 Tag 显式 push 到 `origin`，不批量推送全部 Tag。
3. 创建 GitHub Release，并核对版本与产物。

**现有自动化注意事项**：`.github/workflows/release.yml` 监听 `v*` Tag push，并在 Windows/macOS 构建后自动创建 Release。因此推送 Tag 可能直接触发第 3 步，必须提前告知并获得覆盖自动发布的授权。先检查 Actions / Release 状态，不能再盲目运行 `gh release create` 造成重复；人工创建仅用于已确认的手动发布路径。

Commit、push、release 各自需要明确授权；可由用户一次明确批准完整发布流程，但不能从“允许本地 Tag”推断允许推送和发布。

## 9. README 管理规则

README 的维护须覆盖以下内容，局部修改应核对相关章节，不为一次小改动无关重写整篇：

- 项目介绍、个人 fork 与官方项目的关系。
- 功能列表。
- 安装方式。
- 运行方式。
- 配置说明（仅脱敏示例）。
- 开发说明（环境、构建、测试）。
- Change 记录（可摘要并链接 `_dev/changes/INDEX.md`）。

**Mobile Output、Ghost Mode、KeepContext、SSE、ClickThrough** 等重大功能变更必须同步相关 README 说明，包含真实行为、限制、入口或快捷键；对外说明涉及英文版时同步检查 `README_EN.md`。

原 README 仍含 upstream 下载和克隆地址；更新 fork 使用说明时必须区分个人发布与官方发布，不能默认将两者混用。

## 10. 安全规则

- 禁止提交 API Key、Token、密码、账号凭据/私人账号信息和个人配置文件；公开仓库标识用于确认远程目标，不视为凭据。
- 修改 `.env`、`config` 配置文件/目录、secret 或其他可能含个人配置的文件前，必须先说明文件、目的、影响并取得用户确认；功能开发归档仍须完成。
- 配置和环境变量的实值不得打印到终端、文档、日志或提交信息，状态只能使用 **`SET` / `NOT SET`**。Mobile 连接 URL 可能含 Token，同样须脱敏。
- 不擅自修改 agentmemory 配置，不自行开启或关闭 `AGENTMEMORY_AUTO_COMPRESS`、`AGENTMEMORY_ALLOW_AGENT_SDK`、`CLAUDE_MEMORY_BRIDGE` 等推理相关 flags；需要调整时按 AGENTS.md 归档并确认。
- 不主动新增依赖，不擅自扩展网络暴露范围，不覆盖用户已有配置。

## 11. Claude 行为规则

执行任何操作前，先向用户说明：

1. **准备执行什么**（只读检查、编辑、测试、构建或发布）。
2. **影响哪些文件**（只读时说明不修改；构建时说明可能生成的文件）。
3. **是否需要 Git 操作**（区分只读检查、暂存、commit、push、Tag）。

同一目的的一组只读检查可统一说明；范围变化或开始写入前重新说明。

- 涉及 **commit、push、release**，必须等待用户明确确认；遵守第 2、5、7、8 节的操作边界。
- 先完成需求与现状核查，再按批准范围实施；未归档不得触碰业务源码，不借规则维护修改业务行为。
- 文件读取/编辑优先使用 Read / Edit / Write / Glob / Grep；中文路径下注意 shell 编码，目录类操作可用 PowerShell。
- 项目治理事实以仓库源码、Git 历史和 `_dev/` 档案交叉核实。basic-memory 是个人知识库，不是项目记忆本体；agentmemory 的既有接入说明见 `_dev/memory/agentmemory-assessment.md`，环境基线见 `_dev/baseline/Q-Solver 开发环境准备报告.md`。
- 不擅自启动或调整记忆服务；因任务临时启动的进程用完恢复原状态。
- 收尾报告修改摘要、文件路径、实际验证结果、已知问题、Git 状态及未执行的步骤。未提交但只有预期改动应报告“仅有预期未提交修改”，不能称为 clean。
- 用户要求不提交/不推送时，保持改动等待确认，不以生命周期要求为理由自动提交。

## 12. 当前项目历史

以下是截至 2026-10-14 **已实现并有功能提交的历史**；并不表示所有人工验收或治理关闭均已完成。功能提交根据当前 Git 历史核实：

| Change | 维护名称 | 归档目录 | 功能提交 |
|---|---|---|---|
| Change 001 | Mobile Output | `001-mobile-output/` | `bcf5ff8` |
| Change 002 | Remote UI | `002-remote-output-ui/` | `b8acfa3` |
| Change 003 | Ghost Mode（相关桌面窗口隐藏能力） | `003-desktop-window-toggle/` | `a4e4f24` |
| Change 004 | Mobile Port | `004-mobile-status-visibility/` | `05e34cb` |
| Change 005 | KeepContext Backend Wiring | `005-keepcontext-backend-wiring/` | `1cb0173` |
| Change 006 | Mobile SSE Handshake | `006-mobile-sse-handshake/` | `3059947` |
| Change 007 | ClickThrough Toggle Feedback | `007-clickthrough-toggle-feedback/` | `5df3177` |
| Change 008 | Toast Icon Compatibility | `008-toast-icon-compatibility/` | `23016e8` |
| Change 009 | Toast Semantic Feedback | `009-toast-semantic-feedback/` | `d4469ab` |

归档目录均位于 `_dev/changes/`。名称与状态需注意：

- Change 003 按用户维护名称列作 Ghost Mode；实际实现为桌面窗口显隐、托盘恢复及 mobile 自动隐藏，**不是新增或改写 GhostMode 防录屏逻辑**。两者状态独立。
- Change 004 实际修复 Mobile Output 端口展示；不是重新设计 Mobile 协议。
- INDEX 当前遗漏 003/004，部分档案仍写“进行中 / Pending Commit”，但相应功能及治理提交已存在。后续治理任务应核实并对齐，不能盲信旧状态，也不在本次规则任务中修改历史档案。
- Change 007–009 仍有记录为 `NEEDS_USER` 的 Windows WebView2 人工验收项。没有新证据前保持待验收，不标记正式 Close。

## 2026-10-22 v1.1.0-rc.1 一次性授权例外

用户在确认提交边界、公开目标仓库、RC Tag、自动发布影响和已知验证缺口后，明确批准“RC 整理与发布”。仅本次允许：

- 将叠加的 Change 010～014 功能与测试整合为一个提交；维护规则单独提交，发布工具链及治理记录单独提交。后续仍遵守原有逐 Change 提交规则。
- 在如实披露 race 环境阻塞及未完成的人工验收时，发布到个人 fork 的 v1.1.0-rc.1 **预发布**，不标为稳定版或 Latest；此例外不等于 Close 旧 Change，不伪造验证通过。
- 推送目标仅 origin/dev/qsolver-custom 和 v1.1.0-rc.1，不涉及 main/upstream，不授予其他 Tag、强制推送或删除 Release 权限。

此条记录用户本次明确决定，不构成后续默认授权。其他安全、配置、数据保护规则保持有效。实施记录见 `_dev/changes/015-rc-release/`。
