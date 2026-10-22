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

| `015` | rc-release（RC 整理与发布） | 已授权 / 发布准备中 | 2026-10-22 | `015-rc-release/` |

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
