# Change 006 架构审计范围

> **状态**：Audit 完成；Decision 已批准进入 `006-mobile-sse-handshake`
> **基线分支**：`dev/qsolver-custom`
> **基线 HEAD**：`c3db919 chore: add change 005 governance records`
> **基线工作树**：clean

## 审计目标

在 Change 001～005 已关闭的基线上，重新核对 Window、Shortcut、Mobile Output、Solver 与 Frontend 的剩余问题，并从中选择最多 1～2 个高收益、低风险且同一方向的 Change 006 目标。

## 审计边界

- Audit 阶段仅读取业务源码，不修改业务行为。
- 不修改 Mobile 协议、LLM Provider、Prompt 系统、窗口架构或 OutputMode 架构。
- 不新增依赖、数据库或持久化。
- 所有 Finding 分类为 `ALREADY IMPLEMENTED`、`DOCUMENTATION ONLY`、`DEFER` 或 `SUITABLE FOR CHANGE006`。

## 重点范围

1. `pkg/window`、`pkg/platform`：窗口层级、可见性状态机、Tray/Shortcut/SecondInstance 链路。
2. `pkg/shortcut` 与前端 Settings：配置显示、空配置兼容、默认键同步。
3. `pkg/mobile`、`app/mobile.go`：SSE 状态、MobileInfo、hidden 模式反馈矩阵。
4. `pkg/solution`：KeepContext 并发、生命周期、multimodal history、ClearHistory。
5. Frontend ModelSelect、Settings、Solution UI：能力过滤、快捷键提示、OutputMode UI。
