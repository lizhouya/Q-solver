# _dev/changes/ — 二次开发变更总索引

本目录归档 Q-Solver 二次开发的**所有变更**。每项变更一个子目录，编号从 `001` 起连续递增，**不得复用已分配编号**。

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

> 分配规则：新增变更时，找到表中最小的未使用编号（或 `最大编号+1`），创建目录 `NNN-名称/`（名称用短横线连接、无空格、建议 < 30 字符），并在本表登记。

## 变更状态说明

| 状态 | 含义 |
|---|---|
| `规划中` | 已创建 `00-request`，尚未进入实现 |
| `进行中` | 已创建 `01-current-state` + `02-plan`，正在实现 |
| `已完成` | 全部 6 份文档齐备，变更合并 |
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
