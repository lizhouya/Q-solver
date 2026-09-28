# 03 — 移动端状态同步数据结构设计（State Sync Data Design）

> 只读设计审计。定义**手机端应展示 / 同步**的数据结构与状态机。
> 本节只设计**数据结构**，不选传输协议（传输选型见 02 + 最终报告）。

---

## 一、状态机（来自 01 关键帧的自然映射）

```
                +-----------------------+
                |  IDLE（未解题/空闲）    |
                +----------+------------+
                           | TriggerSend → start-solving
                           v
                +-----------------------+
                |  WAITING（等待中）     |   ← 桌面截图已拿到，等待流开始
                +----------+------------+
                           | solution-stream-start
                           v
        +------------------------------------------------+
        |  STREAMING（分析中）                            |
        |    └─ THINKING（思考链） → CONTENT（正文流式）   |
        +-----------------------+-----------------------+
                           | solution（最终答案）
                           v
                +-----------------------+
                |  DONE（完成）          |  ← answer 全量 + error 可空
                +-----------------------+

状态枚举：`idle / waiting / streaming(thinking|content) / done / error`

---

## 二、核心同步数据结构（建议稿）

```jsonc
// 服务端 → 手机 的单条「状态包裹」——不含具体业务字段
{
  "seq": 42,                // 单调递增序号，手机用于丢包检测去重
  "ts": 1730000000000,      // 事件时间戳 ms
  "status": "streaming",    // 状态：idle|waiting|streaming|done|error
  "stage": "content",       // 子阶段：thinking|content（仅 streaming 时有效）
  "payload": {
    "question": "…",         // 题目（截图描述或在桌面识别；可选）
    "thinking": "…",         // 思考链（增量或全量，取决于展示模式）
    "answer": "…",           // 答案文本（全量在 done）
    "rounds": 1,             // 当前第几轮（保存 keepContext 语义）
    "model": "…",            // 当前模型名（展示用）
    "elapsedMs": 12345       // 耗时（可选）
  },
  "error": { "code": "", "msg": "" }   // status=error 时
}
```

**说明**：
- `seq + ts` 让手机端检测乱序/丢失，断线重连后拉全量快照对齐。
- 全量/增量策略由产品决定（见 06-D-Mobile 003：答案/答案+简单解析/完整流程），此处框架兼容。
- `rounds` 为将来 F-001（真多轮）预留——当前后端每轮独立，rounds 恒为 1。

## 三、手机端渲染要求（非实现）

- 需要满足 **Markdown + 数学公式（KaTeX/MathJax）+ 代码块**——与桌面 `renderMarkdownWithLatex`（frontend/utils/markdown-latex.js）语义一致，减少两端渲染差异。
- 思考链折叠/展开、答案可滚动、错误态呈现。

---

## 四、与现有结构（桌面）对照

| 字段 | 桌面端（solution.js） | 移动端（本设计） |
|---|---|---|
| 状态集合 | `isLoading/isAppending/isThinking` | `status/stage`（单一事实源） |
| 流式缓冲 | `streamBuffer / thinkingBuffer` | `payload.answer / payload.thinking`（增量） |
| 最终答案 | `rounds[].aiResponse` | `payload.answer` 终结态 |
| 历史 | `history[]`（多题列表） | `rounds`（若需切题查看） |

> 移动端更偏向「**单一当前题**」展示；跨题历史迁移需求待产品定义（D-Mobile-003/004）。

---

*（本节完成。仅设计数据结构，不修改任何源码。）*