# _dev/changes/ — 二次开发变更总索引

本目录归档 Q-Solver 二次开发的**所有变更**。每项变更一个子目录，编号从 `001` 起连续递增，**不得复用已分配编号**。

> **规则归属**：本索引只记录 Change 编号、状态并说明归档操作，服从 [AGENTS.md](../../AGENTS.md)；Claude Code 同时遵守 [CLAUDE.md](../../CLAUDE.md)。单次 Change 记录不覆盖上位规则，也不构成提交、推送或发布授权。

## 编号分配表

| 编号 | 名称 | 状态 | 创建日期 | 归档目录 |
|---|---|---|---|---|
| `001` | mobile-output（解题结果手机同步镜像） | 已完成 | 2026-09-26 | `001-mobile-output/` |
| `002` | remote-output-ui（远程端 Answer-first 双栏 UI + 会话内历史） | 已完成 | 2026-09-27 | `002-remote-output-ui/` |
| `005` | 005-keepcontext-backend-wiring | 已完成 | 2026-10-13 | `005-keepcontext-backend-wiring/` |
| `006` | mobile-sse-handshake（SSE 建连原子订阅） | 进行中 | 2026-10-13 | `006-mobile-sse-handshake/` |
| `007` | clickthrough-toggle-feedback（鼠标穿透切换反馈） | 进行中 | 2026-10-13 | `007-clickthrough-toggle-feedback/` |
| `008` | toast-icon-compatibility（Change 007.1 Toast 图标兼容） | 进行中 | 2026-10-13 | `008-toast-icon-compatibility/` |
| `009` | toast-semantic-feedback（Toast 类型语义传递） | 进行中 | 2026-10-13 | `009-toast-semantic-feedback/` |
| `010` | question-bank-export（笔试题库归档） | 待人工验收 / 未提交 | 2026-10-18 | `010-question-bank-export/` |
| `011` | job-workspace-layout（工作区整理） | 迁移与旧副本清理完成 / 待人工验收 | 2026-10-18 | `011-job-workspace-layout/` |
| `012` | question-bank-quick-export（自动分段与一键题库导出） | Mobile 回修自动验证完成 / 待人工复验 / 未提交 | 2026-10-19 | `012-question-bank-quick-export/` |
| `013` | question-bank-export-dialog（题库导出弹窗与试卷格式） | 进行中 | 2026-10-20 | `013-question-bank-export-dialog/` |
| `014` | api-settings-recovery（API 配置加载与接入修复） | 自动验证完成 / 待人工验收 / 未提交 | 2026-10-22 | `014-api-settings-recovery/` |

| `015` | rc-release（RC 整理与发布） | RC1 CI 失败 / RC2 本地回修完成 / 待远端验证 | 2026-10-22 | `015-rc-release/` |
| `016` | mobile-stable-link（固定同步链接与同步端清屏） | 自动验证与用户确认的 GUI 人工验收完成 / race BLOCKED / 已纳入整合提交 / 不发布 | 2026-10-22 | `016-mobile-stable-link/` |
| `017` | question-bank-history（题库历史浏览与归档元数据） | 实现与普通自动验证完成 / GUI 待验收 / race BLOCKED / 已纳入整合提交 / 不发布 | 2026-10-23 | `017-question-bank-history/` |
| `018` | question-bank-continuation（题库历史分段真正续接） | 实现与普通自动验证完成 / GUI 待验收 / race BLOCKED / 已纳入整合提交 / 不发布 | 2026-10-23 | `018-question-bank-continuation/` |
| `019` | screenshot-fold-answer-summary（主界面截图折叠与多图多题答案汇总） | 普通自动验证完成 / GUI 与真实模型待验收 / race 与归档 BLOCKED / 已纳入整合提交 / 不发布 | 2026-10-23 | `019-screenshot-fold-answer-summary/` |
| `020` | round-history-selection（历史记录按轮展示与单轮选择） | 普通自动验证与验证程序归档完成 / GUI 待验收 / race BLOCKED / 已纳入整合提交 / 不发布 | 2026-10-23 | `020-round-history-selection/` |
| `021` | start-next-exam（开始下一套笔试） | 普通自动验证与统一归档完成 / GUI NEEDS_USER / race BLOCKED / 已纳入整合提交 / 不发布 | 2026-10-24 | `021-start-next-exam/` |
| `022` | answer-copy-numbering-sync（答案复制与分段题号同步） | 普通自动验证与统一归档完成 / GUI 与真实模型 NEEDS_USER / race BLOCKED / 已纳入整合提交 / 不发布 | 2026-10-24 | `022-answer-copy-numbering-sync/` |
| `023` | numbered-answer-copy（复制题号＋答案） | 普通自动验证与归档完成 / GUI 用户确认通过 / 已纳入整合提交 / 不发布 | 2026-10-25 | `023-numbered-answer-copy/` |
| `024` | question-bank-organize-failure-recovery（题库整理失败恢复） | 本轮全量自动验证通过 / GUI 与真实模型 NEEDS_USER / 纳入 2026-10-29 整合提交 / 未发布 | 2026-10-25 | `024-question-bank-organize-failure-recovery/` |
| `025` | multi-protocol-model-connectivity（多协议模型连接兼容） | 本轮全量自动验证通过 / GUI 与真实 provider NEEDS_USER / race BLOCKED / 纳入 2026-10-29 整合提交 / 未发布 | 2026-10-25 | `025-multi-protocol-model-connectivity/` |
| `026` | ccr-gateway-validation（CCR 网关接入与连接诊断修复） | 本轮全量自动验证通过 / GUI 与真实 CCR NEEDS_USER / race BLOCKED / 纳入 2026-10-29 整合提交 / 未发布 | 2026-10-25 | `026-ccr-gateway-validation/` |
| `027` | vision-failure-detection（视觉失败可见化与历史图片预算收敛） | 本轮全量自动验证通过 / GUI 与真实 CCR NEEDS_USER / race BLOCKED / 纳入 2026-10-29 整合提交 / 未发布 | 2026-10-26 | `027-vision-failure-detection/` |

| `028` | llm-error-diagnostics（主解题安全诊断与 Responses 失败分类） | 本轮全量自动验证通过 / GUI 与真实上游 NEEDS_USER / race BLOCKED / 纳入 2026-10-29 整合提交 / 未发布 | 2026-10-26 | `028-llm-error-diagnostics/` |
| `029` | build-artifact-lifecycle（构建产物生命周期与整合提交） | 产物整理、全量自动验证、最终 EXE 晋升与边界审计完成 / 纳入本轮治理提交 / 未发布 | 2026-10-29 | `029-build-artifact-lifecycle/` |
| `030` | latest-validation-build（当前工作树独立验证版） | 自动验证完成 / 验证产物已归档 / 被 Change 029 最终版取代 / 纳入本轮治理提交 / 未发布 | 2026-10-28 | `030-latest-validation-build/` |
| `031` | inline-question-explanations（整理稿固定逐题解析） | 本轮全量自动验证通过 / GUI 真实导出 NEEDS_USER / 纳入 2026-10-29 整合提交 / 未发布 | 2026-10-28 | `031-inline-question-explanations/` |
| `032` | model-dropdown-scroll-lock（模型下拉滚动锁定） | 自动验证与 GUI 人工验收通过 / 最终版已晋升 / 已纳入双提交 / 未发布 | 2026-10-29 | `032-model-dropdown-scroll-lock/` |
| `033` | mobile-answer-follow（同步端答案空间与滚动优化） | 已完成 / 自动与用户 GUI 验证通过 / 已提交推送并纳入最新稳定版 / 未发布 | 2026-10-29 | `033-mobile-answer-follow/` |
| `034` | question-bank-hard-delete（题库当前分段物理清空） | 已完成 / 自动与用户 GUI 验证通过 / 已提交推送并晋升稳定版 / 未发布 | 2026-10-29 | `034-question-bank-hard-delete/` |

> `030` 由用户在阶段二任务中显式指定；`029` 后续按最小未用编号分配给构建产物生命周期治理。

> 分配规则：新增变更时，交叉核对本表、实际目录及历史记录，取最小未用三位编号；已使用编号不得因索引遗漏而复用。创建目录 `NNN-名称/`（名称用短横线连接、无空格、建议 < 30 字符），并在本表登记。

## 变更状态说明

| 状态 | 含义 |
|---|---|
| `规划中` | 已创建 `00-request`，尚未进入实现 |
| `进行中` | 已创建 `01-current-state` + `02-plan`，正在实现 |
| `已完成` | 00~06 共七份文档职责齐备，必要验证通过、提交边界审计及授权提交完成，按 AGENTS.md 关闭；不要求合并 main |
| `已回滚` | 按 `05-rollback` 回滚完成 |
| `已废弃` | 变更被放弃，档案保留仅供参考 |

## 目录内文件约定

每个变更目录统一按编号前缀命名，顺序即生命周期：

```
00-request.md           变更请求（为什么做、目标、验收标准）
01-current-state.md     现状快照（动手前对相关代码/配置的客观描述）
02-plan.md              实施方案（步骤、涉及文件、风险与回滚预案）
03-implementation.md    实现过程记录（时间线、每步改动、关键决策）
04-validation.md        验证记录（测试、构建、运行、验收对照）
05-rollback.md          回滚预案与执行记录（若发生回滚）
06-notes.md             总结与注意事项（踩坑、后续影响、遗留项）
```

## 使用提示

- 前三份文档（00/01/02）必须在**业务代码被修改之前**完成——它们是变更的"事前记录"。
- `03` 之后的记录允许边做边补，但不得缺失；数字一旦分配不得改号。
- 模板见 [`_template/`](_template/)，示例见 [`000-examples/`](000-examples/)。

## Change 005

- **名称**：`005-keepcontext-backend-wiring`
- **类型**：Feature / Backend Fix
- **状态**：Completed
- **摘要**：Wire existing KeepContext implementation into Solver request history flow.
- **包含**：
  - `KeepContext=true` 携带历史上下文
  - `KeepContext=false` 独立请求
  - 成功后保存 User + Assistant history
  - 失败轮不写入
  - 复用 `trimChatHistory`
- **提交状态**：Pending Commit
- **Commit hash**：

## Change 006

- **名称**：`006-mobile-sse-handshake`
- **类型**：Correctness Fix / Mobile Output
- **状态**：Implementation Completed / Pending Commit
- **摘要**：原子化 SSE subscriber 注册与初始 Snapshot 读取，消除建连窗口内唯一终态丢失风险。
- **包含**：
  - Bridge 原子返回 subscriber + initial Snapshot
  - Server 在首次 Flush 前完成订阅注册
  - 确定性首次 Flush 顺序测试
  - 移除端到端测试的 100ms 人为等待
- **协议变化**：无
- **提交状态**：Pending Commit
- **Commit 1**：功能代码 + 测试（待提交）
- **Commit 2**：Audit + Change 文档 + INDEX（待提交）

## Change 007

- **名称**：`007-clickthrough-toggle-feedback`
- **类型**：UX Feedback / Correctness
- **状态**：Implementation Completed / Pending Commit
- **摘要**：鼠标穿透成功切换后，根据平台确认的真实新状态显示开启/关闭 Toast；失败时不显示成功文案。
- **包含**：
  - State/App/Shortcut 透传真实状态与错误
  - 复用既有 `EmitEvent("toast", ...)` 通知链路
  - 开启、关闭、失败、单次调用和自定义快捷键测试
  - HWND 未初始化失败语义测试
- **Frontend 变化**：无
- **依赖/协议/配置变化**：无
- **提交状态**：Pending Commit
- **Windows 真机验收**：T-WIN-01～04 待人工执行

## Change 008（用户命名：Change 007.1）

- **名称**：`008-toast-icon-compatibility`
- **类型**：Frontend Compatibility / UX Presentation
- **状态**：Implementation Completed / Pending Commit Boundary
- **摘要**：将 Toast 前置 SVG 状态图标替换为 CSS 圆框与稳定字符，避免 Windows WebView2 中呈现为 `×`。
- **允许范围**：`frontend/src/App.vue`、`frontend/src/styles/global.css`
- **业务调用链变化**：无
- **依赖/协议/配置变化**：无
- **自动验证**：Frontend build、Go test/build、Wails debug build、diff-check 全部通过
- **视觉验收**：Windows WebView2 肉眼确认 `NEEDS_USER`
- **提交状态**：Pending Commit Boundary；未暂存、未提交、未推送

## Change 009

- **名称**：`009-toast-semantic-feedback`
- **类型**：Event Semantics / Backward Compatibility
- **状态**：Implementation Completed / Pending Commit Boundary
- **摘要**：为 ClickThrough Toast 传递 success/info/error 类型，并保留旧字符串 payload 兼容。
- **允许范围**：`pkg/shortcut/service.go`、`pkg/shortcut/service_test.go`、`frontend/src/App.vue`
- **视觉组件变化**：无，保留 Change 008 template/CSS
- **依赖/配置变化**：无
- **自动验证**：聚焦测试×50、全量 Go test/build、Frontend build、Wails debug build、diff-check 通过
- **提交状态**：Pending Commit Boundary；未暂存、未提交、未推送

## Change 010

- **名称**：question-bank-export（笔试题库归档）
- **状态**：实现与普通自动验证完成；人工验收 NEEDS_USER，race 运行环境 BLOCKED；未提交、未关闭。
- **档案**：[010-question-bank-export](010-question-bank-export/00-request.md)

## Change 011

- **名称**：job-workspace-layout（求职工作区整理）
- **状态**：2026-10-19 已完成完整副本复制、全量哈希/Git/测试/构建验证及旧副本清理；2026-10-20 已扁平化旧题库输出并移除旧空“自动网申”；待人工验收，未提交、未关闭。
- **档案**：[011-job-workspace-layout](011-job-workspace-layout/00-request.md)
- **目标层级**：`求职笔试/Q-Solver/` 已包含当前工具的独立完整副本；“求职笔试”继续预留同级的其他工具，不把输出或网申目录混入仓库。

## 2026-10-19 迁移完成后独立复验

- Change 010/011 的普通测试与构建再次通过，未提交改动仍完整保留。
- `go vet ./...` 存在既有 Windows hook unsafe.Pointer 告警；race 仍在启动阶段失败，不能标为通过。
- 新输出根路径已通过模块只读探针；GUI、手机、退出保护和真实模型整理仍待人工验收，不 Close。
- 证据与未提交资产归属见 [独立复核报告](../audits/010-011-post-migration-check.md)。

## Change 012

- **名称**：question-bank-quick-export（自动分段与一键题库导出）
- **状态**：Mobile 人工反馈回修实现与自动验证完成；人工复验 NEEDS_USER，race 运行环境 BLOCKED；未提交、未关闭。
- **档案**：[012-question-bank-quick-export](012-question-bank-quick-export/00-request.md)
- **范围**：题库自动分段与精简路径、主设置集成、ActionBar 一键整理导出、Mobile 连接状态与安全重试、按钮/键盘/局部 wheel 滚动控制、Windows hook vet 修复。
- **自动验证**：Mobile DOM 回归、聚焦与全量 Go 测试、`go vet`、`go build`、Frontend build、正式 bindings 再生成、备用输出名 Wails 生产构建和 `git diff --check` 通过；race 启动阶段 `0xc0000139`，标记 BLOCKED。
- **人工验收**：Windows GUI 的连接状态与局部 wheel、手机真机历史保持与三种路由、真实模型整理和退出保护标记 NEEDS_USER。
- **安全边界**：模型整理默认关闭；不删除旧记录或旧输出；不提交、不推送、不发布。

## Change 013

- **名称**：question-bank-export-dialog（题库导出弹窗与试卷格式）
- **状态**：实现与普通自动验证完成；人工验收 NEEDS_USER，race 运行环境 BLOCKED；未提交、未关闭。
- **档案**：[013-question-bank-export-dialog](013-question-bank-export-dialog/00-request.md)
- **范围**：单次导出弹窗与命名/目录、无图试卷格式、明确的分段入口和顶部状态；不改变解题或 Mobile 协议。
- **自动验证**：聚焦与全量 Go 测试、`go vet`、`go build`、Frontend build、备用输出名 Wails 生产构建和 `git diff --check` 通过；race 启动阶段 `0xc0000139`，标记 BLOCKED。
- **人工验收**：Windows GUI、目录选择器和用户自愿开启后的真实模型整理标记 NEEDS_USER。
- **安全边界**：打开或取消弹窗不调用模型；不覆盖旧输出；不提交、不推送、不发布。

## Change 014

- **名称**：api-settings-recovery（API 配置加载与接入修复）。
- **档案**：[请求与授权](014-api-settings-recovery/00-request.md)、[验证记录](014-api-settings-recovery/04-validation.md)。
- **范围**：主配置安全加载、完整单次 API 草稿、题库与普通设置独立保存；保留旧绑定。
- **验证**：Go 全量/vet/build、前端与 Mobile 回归、Vite 和 Wails 通过；race 为 BLOCKED，GUI/真实端点为 NEEDS_USER。
- **交付**：本地 build/bin/Q-Solver-api-fix.exe，不替换旧程序，不改写用户配置。未提交、未关闭。

### Change 012 实时同步复验补充（2026-10-22）

- 用户反馈须刷新才显示答案；已修正真实快照序号不递增及慢消费者丢终态两项缺陷。
- 新增真实 Bridge → HTTP/SSE → 页面脚本连续两题测试，全量 Go、前端、Wails 构建通过。
- 最新本地交付：build/bin/Q-Solver-sync-fix.exe（含 Change 014 API 修复），旧 EXE 保留。状态仍为待人工复验、未提交、未关闭。

- **2026-10-22 用户确认**：Change 012 的跨电脑实时显示（无需逐题刷新）、答题电脑控制同步端滚动人工验收通过；其他未明确验证项目不变，未提交、未关闭。

## Change 016

- **名称**：mobile-stable-link（固定同步链接与同步端清屏）。
- **状态**：原实现及 2026-10-23 补充修复的普通自动验证完成；用户已确认 Windows GUI 人工验收通过；race 运行环境 BLOCKED；未提交、未推送、未发布。
- **档案**：[016-mobile-stable-link](016-mobile-stable-link/00-request.md)。
- **范围**：可选固定 Mobile Token、专用加密双槽状态、链接重置与活动 SSE 撤销、`sessionId + clearSeq` 清屏协议、旧任务代际过滤、设置页与 TopBar 双入口；补充修复状态面板层级，并新增 Windows 可录制左 `Shift+C` 清屏请求。
- **安全边界**：Token 不进入普通 Config、Snapshot、日志、文档或浏览器存储；固定端口不保证 IP 永久不变；Clear 不删除桌面、题库、截图、导出或模型上下文。
- **自动验证**：Mobile 专用状态、Bridge/Server、App 命令、浏览器 DOM、共享 store、Windows 快捷键默认值/旧空绑定纠偏/录制运行态/事件、Go 全量/vet/build、Vite、Wails 生产构建、归档哈希和 diff-check 通过；race 启动阶段 `0xc0000139`，标记 BLOCKED。
- **人工验收**：用户已确认最终验证版的确认层无遮挡、左 `Shift+C` 单例与防重、快捷键重新录制、清屏后刷新/重连为空、下一题同步及桌面/题库/截图/导出/模型上下文保留通过；race 仍 BLOCKED，Change 未关闭。


## Change 017

- **名称**：question-bank-history（题库历史浏览与归档元数据）。
- **状态**：业务实现、普通自动验证、正式 bindings 和独立验证构建完成；Windows GUI 人工验收待执行；race 因 `0xc0000139` BLOCKED；未提交、未推送、未发布。
- **档案**：[017-question-bank-history](017-question-bank-history/00-request.md)。
- **范围**：软清空成功后重置桌面实时历史；最近 20 段只读详情与安全图片读取；招聘单位、出题方和题型权威元数据；复用模型整理生成待核对候选；两阶段确认后纯本地导出。
- **数据边界**：第 21 段及更早段仅从 UI 隐藏，磁盘完整保留，不自动导出、不调用模型、不删除；历史预览不切换后台 current。
- **验证入口**：`build/bin/archive/validation/change-017/Q-Solver-change017-history-view-fix-validation.exe`；SHA-256 `3b85644beb8a726c7138d01ce843526538b4d028aea80fd75403a0844bd338c5`。
- **授权边界**：本 Change 不含 commit、push、Tag 或 Release 授权。

## Change 018

- **名称**：question-bank-continuation（题库历史分段真正续接）。
- **状态**：业务实现、普通自动验证、正式 bindings 和独立验证构建完成；Windows GUI 人工验收待执行；race 因 `0xc0000139` BLOCKED；未提交、未推送、未发布、未关闭。
- **档案**：[018-question-bank-continuation](018-question-bank-continuation/00-request.md)；[实现后审计](../audits/018-question-bank-continuation.md)。
- **范围**：从最近 20 个 ended/exported/paused 分段真正切换题库 current，在主截图解题界面恢复历史截图与完整答案；以持久 turn 引用或明确标记的旧 Record 重建恢复非 system 模型上下文；切换、完成落盘失败和 Retry 均保持 Store、Solver 与 App refs 一致。
- **数据边界**：discarded 永久只读；第 21 段及更早数据仅隐藏并保留；旧导出目录不覆盖，再次导出创建 `_02` 等后缀版本。
- **安全边界**：运行任务、未发送截图、未落盘、整理或导出时拒绝；旧数据明确标记按题库记录重建；后续使用当前模型/场景/Prompt；不保存密钥、完整配置、Prompt、简历或 thinking。
- **验证入口**：最新布局验证程序 `build/bin/archive/validation/change-018/Q-Solver-change018-layout-validation.exe`；SHA-256 `853e64c3ccd59101317d0fc4e92414a10f1d12695deed9694b0d689ee2e07343`。原续接验证程序继续保留。
- **授权边界**：本 Change 不含 commit、push、Tag 或 Release 授权。

## Change 019

- **名称**：screenshot-fold-answer-summary（主界面截图折叠与多图多题答案汇总）。
- **状态**：业务实现与普通自动验证完成；Windows GUI 与真实模型效果 NEEDS_USER；race 因 `0xc0000139` BLOCKED；统一验证归档已由 Change 029 完成；已纳入 2026-10-25 整合提交，未发布、未关闭。
- **档案**：[019-screenshot-fold-answer-summary](019-screenshot-fold-answer-summary/00-request.md)；[实现后审计](../audits/019-screenshot-fold-answer-summary.md)。
- **范围**：主解题界面每张截图默认收起并可独立展开/隐藏；统一系统行为增加本轮多图多题识别、跨图合并、重叠去重、题号映射、待核对和最终答案汇总规则。
- **边界**：复用既有图片缓存与一次 multipart 模型请求；不新增模型调用、题库结构、Wails API、Mobile 协议或依赖；未经授权未调用真实模型。
- **验证入口**：`build/bin/archive/validation/change-019/Q-Solver-change019-answer-summary-preview-validation.exe`；SHA-256 `cd4b354ef37613029d2bb0e39b9d62fd924480aefa0eafbf762ca60135d19cf7`；归档目录和 `SHA256SUMS` 尚未更新。
- **授权边界**：本 Change 不含 commit、push、Tag 或 Release 授权。

## Change 020

- **名称**：round-history-selection（历史记录按轮展示与单轮选择）。
- **状态**：业务实现、普通自动验证、独立验证构建和统一归档完成；Windows GUI 人工验收待执行；race 因 `0xc0000139` BLOCKED；未提交、未推送、未发布、未关闭。
- **档案**：[020-round-history-selection](020-round-history-selection/00-request.md)；[实现后审计](../audits/020-round-history-selection.md)。
- **范围**：保持 Session/Record 权威结构不变，在桌面前端派生逐轮历史条目并独立管理浏览目标；点击一轮只显示该轮，新请求仍按 `epoch + sessionId + recordId` 写入真实 Record；legacy 流也使用独立写入目标。
- **边界**：删除和图片导出仍是父 Session 级；不修改 Go 后端、Wails API/bindings、Mobile、题库 schema、模型请求或依赖。
- **验证入口**：`build/bin/archive/validation/change-020/Q-Solver-change020-round-history-selection-validation.exe`；SHA-256 `3ea25ab19009d373c304f440c126bfeb18ce7c0d4c218d8e9be8204abdaa75bf`。
- **授权边界**：本 Change 不含 commit、push、Tag 或 Release 授权。

## Change 021

- **名称**：start-next-exam（开始下一套笔试）。
- **状态**：业务实现、普通自动验证、正式 bindings、独立验证构建和统一归档完成；Windows GUI 人工验收待执行；race 因 `0xc0000139` BLOCKED；未提交、未推送、未发布、未关闭。
- **档案**：[021-start-next-exam](021-start-next-exam/00-request.md)；[实现后审计](../audits/021-start-next-exam.md)。
- **范围**：主界面按钮与 Windows 默认 `Ctrl+Shift+N` 打开同一单例确认层；后端原子封存 active/paused 段，ended/exported 幂等接受，持久化成功后清理桌面、待发送截图与 Solver 上下文；下一题懒创建新段。
- **边界**：不根据异步导出状态自动切换；解题、capture、整理、导出、未落盘或写入阻塞时拒绝且零副作用；保留题库原始记录、归档截图、旧导出和 Mobile 固定链接/当前显示。
- **bindings 审计**：固定 Wails v2.11.0 根据当前累计 Go API 完整再生成并通过编译；事前只有哈希无文件副本，不能声称除 `StartNextExam` 外逐字节不变。
- **验证入口**：`build/bin/archive/validation/change-021/Q-Solver-change021-start-next-exam-validation.exe`；SHA-256 `95f4b3c17b96db0e25fa682ed337bb067458046784a09d3580d1eff4839e3c4c`。
- **授权边界**：本 Change 不含 commit、push、Tag 或 Release 授权。

## Change 022

- **名称**：answer-copy-numbering-sync（答案复制与分段题号同步）。
- **状态**：业务实现、普通自动验证、正式 bindings、独立构建和统一归档完成；Windows GUI 与真实模型 NEEDS_USER；race BLOCKED；未提交、未推送、未发布、未关闭。
- **档案**：[022-answer-copy-numbering-sync](022-answer-copy-numbering-sync/00-request.md)；[实施后审计](../audits/022-answer-copy-numbering-sync.md)。
- **范围**：固定回答末尾三列表格、Go 统一解析/纯答案复制、Answer/AnswerItems/ConversationSnapshot 原子持久化、Session 受限编号摘要、Desktop/Mobile 复制与历史目标保持、实际快捷键提示，以及 KeepContext 引用一致性修复。
- **安全边界**：保持一次模型请求；解析失败保留全文，待核对禁用复制；保持 schema version 2 和旧 JSON 兼容；不新增依赖、不升级 runtime，不读写用户密钥配置，不删除题库/截图/旧导出。
- **验证**：Go test/vet/build、七份前端回归、Mobile DOM/SSE、Vite、Wails 及 12 条归档校验通过；28 项保护哈希不变；race 启动错误 `0xc0000139`。
- **验证入口**：`build/bin/archive/validation/change-022/Q-Solver-change022-answer-copy-numbering-validation.exe`；SHA-256 `5cde3df3a8bdd160f7740ffeebc6b68c7dcb3f87bbfc5ff7226ce6bf2950e751`。
- **授权边界**：本 Change 不含 commit、push、Tag 或 Release 授权。

## Change 023

- **名称**：numbered-answer-copy（复制题号＋答案）。
- **状态**：实现、普通自动验证、独立构建与归档完成；GUI 用户确认通过；未提交、未推送、未发布、未关闭。
- **档案**：[023-numbered-answer-copy](023-numbered-answer-copy/00-request.md)；[实施后审计](../audits/023-numbered-answer-copy.md)。
- **范围**：Desktop/Mobile 新增带实际题号复制；保留纯答案和桌面全文复制。不改 Go API、模型请求、题库结构或依赖。
- **验证**：Go test/vet/build、八份前端回归、Mobile DOM/SSE、Vite、Wails 通过；35 项旧产物与依赖哈希不变。GUI 用户确认通过，既有 race BLOCKED 未解决。
- **验证入口**：`build/bin/archive/validation/change-023/Q-Solver-change023-numbered-answer-copy-validation.exe`；SHA-256 `44fbb9381179b0d739003c53df1abe9c06847255447f87ff786fdbe8a711d339`。
- **授权边界**：不含 commit、push、Tag、Release 授权。

## Change 027

- **名称**：vision-failure-detection（视觉失败可见化与历史图片预算收敛）。
- **状态**：业务实现及本轮全量自动验证完成；Windows GUI 与真实 CCR NEEDS_USER；race 因 `0xc0000139` BLOCKED；已纳入 2026-10-29 整合提交，未发布、未关闭。
- **档案**：[027-vision-failure-detection](027-vision-failure-detection/00-request.md)；[实施后审计](../audits/027-vision-failure-detection.md)。
- **起因**：用户测试 `APMIX.AI-openai/claude-sonnet-4-6-free` 返回两道完全不相关的题目（实际 5 道），DeepSeek 正确。
- **已核实事实**：Q-Solver 多图构造与 OpenAI Chat 线格式正确，无丢图路径；`StartNextExam` 已清空历史并递增 epoch，无跨分段污染；`KeepContext` 默认关闭，开启时旧截图随历史重发，修复前请求携带 **11 轮**图片（`MaxScreenshots=3` 时上限 33 张，超 Anthropic 20 张阈值）；prompt 已禁编造但**无**视觉失败指令。
- **范围**：`pkg/solution/solution.go` 新增发送侧 `trimHistoryForSend`，为当前轮预留位置使请求总轮数收敛为 `MaxConversationRounds`；`pkg/domain/prompts.go` 新增一条视觉失败 `<Rule>`。仅 2 处业务改动。
- **安全边界**：不改 `trimChatHistoryLocked`、`MaxConversationRounds`、快照契约与 KeepContext 开关语义；不按模型名推断视觉能力；不改 CCR 入站协议；不新增依赖；不触碰题库/截图/导出/用户配置；零真实模型调用。
- **验证**：`go test ./...`（22 包）、`go vet`、`go build`、`go mod verify`、13 份前端回归、Vite build 全部通过；依赖与受保护 EXE 哈希不变；race `0xc0000139` BLOCKED。
- **关键证据**：修复前第 13 轮请求 22 张图 → 修复后 20 张（历史 18 + 当前 2），当前轮截图始终完整。
- **验证入口**：`build/bin/archive/validation/change-027/Q-Solver-change027-vision-failure-detection.exe`；60,950,016 字节；SHA-256 `f75e27398dfe3562ae7ac140bf33ec15d4ce6736ae50210d7981a1bb90c9c90a`。
- **根因边界**：同一请求同时发往 Claude 与 DeepSeek 而仅 Claude 异常，说明仍有 Q-Solver 之外的因子（CCR 路由/上游视觉透传），本 Change **不宣称**已定位唯一根因，该部分留 NEEDS_USER。
- **授权边界**：本 Change 不含 commit、push、Tag 或 Release 授权。

## 2026-10-25 Change 016～023 整合提交

- 用户已明确批准一次性跨 Change 整合为功能/测试与治理两个提交，并连同既有 RC2 `da273bd` 推送至 origin/dev/qsolver-custom。
- 功能/测试提交：`e44bb6163f0cb1b88e2ec4b4da80a1446ddb1990`；治理记录随本次第二提交归档。
- [完整提交边界审计](../audits/016-023-commit-boundary.md)列明精确文件、完整消息、验证与排除项。
- 上述分段历史摘要中的未提交/未推送是当时状态，最新结果以 Git 历史与远端分支头为准；本次不创建 Tag 或 Release。
- Change 023 用户人工验收通过；其他未明确验收和 race 环境阻塞保持原状态，未自动 Close。

## Change 028

- **目录**：[028-llm-error-diagnostics](028-llm-error-diagnostics/00-request.md)。
- **范围**：主解题安全诊断、事件兼容、Responses 失败分类；不改变 CCR 或模型路由。
- **状态**：实现及本轮全量自动验证完成；GUI/真实上游 NEEDS_USER，race BLOCKED；旧日志基线限制保留；已纳入 2026-10-29 整合提交，未发布、未关闭。
- **审计**：[028 差异边界审计](../audits/028-llm-error-diagnostics.md)。
- **验证入口**：`build/bin/archive/validation/change-028/Q-Solver-change028-llm-error-diagnostics.exe`。
- **结论边界**：未调用真实模型，不宣称 GPT/Claude 已恢复；诊断不写入题库。

## Change 029

- **名称**：build-artifact-lifecycle（构建产物生命周期与整合提交）。
- **目录**：[029-build-artifact-lifecycle](029-build-artifact-lifecycle/00-request.md)。
- **状态**：产物整理、规则落地、全量自动验证、最终 EXE 晋升与提交边界审计完成；纳入本轮治理提交，未发布。
- **范围**：根目录唯一稳定版、测试中双产物、按 Change 归档、SHA-256 清单、回滚、Change 024～031 双实施提交与指定远端 push。
- **设计**：[构建产物生命周期与整合提交设计](029-build-artifact-lifecycle/design.md)。
- **最终产物**：`build/bin/Q-Solver.exe`；42,331,136 字节；SHA-256 `82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46`。
- **验证**：Go test/vet/build/mod verify、14 份前端回归、Vite、Wails 独立构建、候选与晋升后启动、44 个迁移前文件、25 份目录级哈希和 7 个 Release 保护全部通过。
- **安全边界**：二进制继续由 Git 忽略；不触碰 `releases/`，不纳入用户/CCR 配置、数据库、日志、仓库外探针或 secret；不推 upstream、不修改 main、不 force push、不创建 Tag/Release/RC。

## Change 030

- **名称**：latest-validation-build（当前工作树独立验证版）。
- **目录**：[030-latest-validation-build](030-latest-validation-build/00-request.md)。
- **状态**：自动构建、产物哈希、独立启动与正常退出验证完成；产物已归档并被 Change 029 最终版取代；纳入本轮治理提交，未发布、未关闭。
- **范围**：不改业务源码；使用当前工作树与现有依赖生成唯一命名的 `build/bin/archive/validation/change-030/Q-Solver-change030-latest-validation.exe`，不覆盖任何旧 EXE。
- **验证入口**：`build/bin/archive/validation/change-030/Q-Solver-change030-latest-validation.exe`；42,333,184 字节；SHA-256 `5924B7D00CE0B00A4BE5A0169EDDD8D900F9442E96B621ACE7F374DD28AB07EF`。
- **自动验证**：启动 8 秒后存活，WebView2 初始化，未启动 Node/Vite，未连接 CCR，窗口关闭路径退出码 0，无强制终止或残留进程；11 个关键保护文件哈希不变，Git 语义索引与保护副本一致且 staged 为 0。
- **索引副作用**：一次未禁用 Git 可选锁的只读差异检查刷新了 `.git/index` stat cache，二进制哈希变化但 1,194 条 stage listing 与保护副本逐项语义一致；未执行暂存、提交或越权回写。
- **基线限制**：全树 `git diff --check` 仍因构建前已存在的 `frontend/wailsjs/go/models.ts` 尾随空白返回 2，该文件构建前后哈希一致，本 Change 未修改。
- **安全边界**：不安装依赖，不同步 `go.mod`，不生成 bindings，不读取或修改用户 secret，不调用模型，不修改本机或远程 CCR。
- **授权边界**：用户批准本地构建和验证；不含 commit、push、Tag 或 Release 授权。

## Change 031

- **名称**：inline-question-explanations（整理稿固定逐题解析）。
- **目录**：[031-inline-question-explanations](031-inline-question-explanations/00-request.md)。
- **状态**：实现、TDD 及本轮 Go/前端/Vite/Wails 全量自动验证完成；Windows GUI 真实导出 `NEEDS_USER`；纳入 2026-10-29 整合提交，未发布、未关闭。
- **范围**：保持 `原始记录.md` 和单一整理稿文件契约，将整理稿从“全部题目后集中答案”调整为每道题下紧跟参考答案、知识点、解析要点、来源与核对说明。
- **验证入口**：`build/bin/archive/validation/change-031/Q-Solver-change031-inline-question-explanations.exe`；42,331,136 字节；SHA-256 `82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46`。
- **自动验证**：聚焦与全量 Go 测试、`go build`、`go vet`、14 份前端回归、Vite 构建、Wails 独立构建和范围内 diff-check 通过；11 个保护文件及 Git index 哈希不变，0 staged。
- **历史再生成**：仅处理明确归属的 TCL 与中国移动；TCL 27 题位由完整离线恢复稿确定性迁移为 `_03`，中国移动 106 道结构化题重导为 `_02`；原始记录和 assets 哈希一致，未调用模型。经用户明确选择，已删除被 `_03` 取代的 TCL `_02` 目录；最初原始版、最新 `_03` 与仓库外保护副本保留。
- **安全边界**：不新增模型调用，不改 Prompt、JSON schema、题库持久化、Wails API、bindings 或依赖；不覆盖历史导出。
- **授权边界**：用户批准本地实现与验证；不含 commit、push、Tag 或 Release 授权。

## Change 032

- **名称**：model-dropdown-scroll-lock（模型下拉滚动锁定）。
- **目录**：[032-model-dropdown-scroll-lock](032-model-dropdown-scroll-lock/00-request.md)。
- **状态**：TDD、14 份前端回归、Vite、Go test/vet/build、Wails 构建、用户 GUI 人工验收及晋升后启动验证通过；最终版已晋升；已纳入经授权双提交，未发布。
- **范围**：模型下拉打开时锁定外层模型页，隔离 wheel/touchmove 和边界滚动链；不改 Go、CCR、模型请求或配置。
- **最终产物**：`build/bin/Q-Solver.exe`；42,331,648 字节；SHA-256 `C1F1873F054520F0F45FADFCCBCC087056D1556381DD1EFC66D4CD3748085F7B`。
- **授权边界**：当前仅批准本地实施验证，不含 commit、push、Tag、Release 或 RC。

## Change 033

- **名称**：mobile-answer-follow（同步端答案空间与滚动优化）。
- **目录**：[033-mobile-answer-follow](033-mobile-answer-follow/00-request.md)。
- **状态**：TDD、Mobile Go、14 份桌面前端回归、Vite、Go test/vet/build/mod verify、候选构建、用户 GUI 人工验收和稳定版晋升验证通过；已提交推送，未发布。
- **范围**：快捷键默认折叠并显示有效项数；最新流式/最终答案自动滚底，历史浏览不强拉回；远程键盘单次滚动由视口 70% 降至 40%。不改 Go 生产代码、SSE 协议、配置、依赖或模型链路。
- **验证入口**：原 Change 033 候选已归档到 `build/bin/archive/transient/`，原始 SHA-256 `19495E67DE3E84781E2E8598EE016EAB4AB4FD5599786817DD3F9EBE757016E9`；包含 Change 033 与 Change 034 的组合候选已归档并晋升为根目录稳定版。
- **稳定版边界**：`build/bin/Q-Solver.exe` 为最新稳定版，42,345,472 字节，SHA-256 `C033EAB4844721EB233841915C82583D213095965357E77091738D9FBC21B0B8`；旧稳定版位于 `archive/superseded/change-034/`。
- **授权边界**：用户已明确授权并完成 Change 033/034 提交、推送和稳定版晋升；不含 Tag、Release 或 RC 授权。

## Change 034

- **名称**：question-bank-hard-delete（题库当前分段物理清空）。
- **目录**：[034-question-bank-hard-delete](034-question-bank-hard-delete/00-request.md)。
- **状态**：TDD、Store/App 聚焦测试、Go test/vet/build/mod verify、14 份前端回归、Vite build、组合候选构建、用户 GUI 人工验收和稳定版晋升验证通过；已提交推送，未发布。
- **稳定版入口**：`build/bin/Q-Solver.exe`；42,345,472 字节；SHA-256 `C033EAB4844721EB233841915C82583D213095965357E77091738D9FBC21B0B8`；候选归档于 `archive/validation/change-034/`，旧稳定版归档于 `archive/superseded/change-034/`。
- **范围**：“清空本段”二次确认后物理删除当前分段记录、答案和截图，成功后清理实时历史及该分段模型上下文；“结束本段”保留本地记录并清理解题界面；导出流程不变。
- **删除安全**：仅允许受管理记录根和合法 Session ID；同目录原子暂存后删除；失败恢复并逐文件校验，无法完整恢复时阻止继续写入；旧软清空历史不追溯删除。
- **数据边界**：自动测试只使用 `t.TempDir()`，未触碰用户题库、其他分段或导出目录；用户已确认组合候选 GUI 验证通过。
- **授权边界**：用户已明确授权并完成 Change 033/034 提交、推送和稳定版晋升；不含 Tag 或 Release 授权。
