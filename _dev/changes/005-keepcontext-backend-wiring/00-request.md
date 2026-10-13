# 变更请求 — 00 · 005-keepcontext-backend-wiring

> **阶段**：Decision + Design Initialization（仅创建文档，不改业务代码，不 git add/commit/push）
> **状态**：Not Started

## Change 名称

`005-keepcontext-backend-wiring`

## 问题

Q-Solver 的 KeepContext 功能，**UI 已存在，但后端 `Solve()` 实际不读取 `chatHistory`**。

具体：用户开启 KeepContext 后，前端会新增 round 卡片、显示多轮对话；但后端每次 `Solve()` 都只构造 `[system, currentUser]` 两条消息，从不携带历史，且在成功路径强制清空 `chatHistory`。结果是——**前端看似保留 rounds，模型实际每轮无上下文**。

## 用户可见后果

开启 KeepContext 后，用户在求职笔试/测评场景中连续追问时，界面显示多轮，但模型每次都"失忆"，上轮问答内容不被携带。

## 目标

让既有 KeepContext 行为真正生效——把**已实现的 `chatHistory` 能力正确接线**到 `Solve()` 请求构造中。不新增能力，只把已存在的死代码激活。

## Non-goals（本 Change 明确不做）

- ❌ 不改 frontend（`solution.js` 的 rounds 渲染已正确）
- ❌ 不改 mobile / SSE / Bridge
- ❌ 不改 LLM Provider / Prompt 设计
- ❌ 不持久化 history（不写磁盘 / localStorage / config）
- ❌ 不做 summary memory / 上下文摘要记忆
- ❌ 不做 token-aware compression / 图片去重 / OCR 替换
- ❌ 不新增 conversation abstraction（复用现有 `chatHistory`）

## 参考

- 原始审计发现：F-001（`_dev/audits/001-qsolver-architecture-audit/04-findings.md`）
- 调用链失效点：`_dev/audits/001-qsolver-architecture-audit/03-call-chain.md` §6