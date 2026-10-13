# 决策记录 — 05 · 005-keepcontext-backend-wiring

> **状态**：设计决策已实现并验证（2026-10-13）

## 1. 用户已确认决策（D-005）

| 编号 | 决策项 | 值 |
|---|---|---|
| D-005-001 | KeepContext = **false** | 当前 Solve 不读旧 history；每题独立；清空已有 history；本轮完成后不保存上下文 |
| D-005-002 | KeepContext = **true** | 请求携带已有有效 history → 追加当前 user；成功后把 current user + final assistant 写入 history |
| D-005-003 | 历史存储 | **仅内存**；禁止写磁盘 / localStorage / config 持久化 / App 重启恢复 |
| D-005-004 | 历史裁剪 | 必须用现有 `MaxConversationRounds` + `trimChatHistory()`，不建第二套裁剪 |
| D-005-005 | 失败轮次 | Solve 失败不写入不完整 assistant / 失败轮不作为完整成功 round 保存 |
| D-005-006 | ClearHistory | 现有 `ClearHistory()` 继续作为唯一显式清空入口；调用后 `chatHistory = empty` |
| D-005-007 | System Prompt | 每次 Solve 以当前有效 System Prompt 为准；用 `ensureSystemPrompt()` 避免历史中旧 system 无限累积 |

## 2. 局部设计原则（LD-005）

| 编号 | 原则 |
|---|---|
| LD-005-001 | **不创建新的 conversation abstraction**；复用现有 `chatHistory []llm.Message` |
| LD-005-002 | 历史存放**保留实际发送给模型的 user message**（multipart 含截图），不丢图片、不压缩 |
| LD-005-003 | round 数量继续由现有 `MaxConversationRounds` 控制（1 round = user + assistant） |
| LD-005-004 | 历史写入**只发生在完整成功响应后**；streaming 阶段不写 |
| LD-005-005 | OFF 分支不调用 `ensureSystemPrompt()` 或读取 history；直接构造 `[current system, current user]`，并在 `Solve()` 内清除残留 history。这是对设计期伪代码的实现细化，直接落实 D-005-001；`ClearHistory()` 仍是唯一 public 显式清空 API |

## 3. 记录在案的并发考虑

- 当前产品路径由 `task.Coordinator` 串行化（新 Solve 取消旧任务），同一时刻单来一个 Solve → 竞争风险低。
- `Solver` 本身无 mutex。本 Change **不引入锁**（避免范围蔓延）；仅把「未来并发隐患」记录在 01，作为后续优化候选。

## 4. 不纳入（Non-goal 汇总）

- 图片摘要 / OCR 替代 / token 精确计算 / 历史压缩 / 图片去重
- history 持久化、service restart 恢复
- summary memory / 上下文摘要
- 并发重构 / 新增锁（除非实现期最小可靠性需要）