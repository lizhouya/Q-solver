# 实现过程记录 03 — 002-remote-output-ui

> 边做边记录：时间线、每步改动、关键决策与偏差。

## 实现日志

| 时间 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-09-27 | Step 1 | 先归档：更新 INDEX 登记 002；写 00-request / 01-current-state / 02-plan / 05-decisions | `_dev/changes/INDEX.md` `002-remote-output-ui/00/01/02/05` | 动码前文档齐备（AGENT.md 红线） |
| 2026-09-27 | Step 2 | 重构 `pkg/mobile/web/index.html`：响应式三档布局 + 移除 Analysis 展示 + 会话内 History + 去重 + 点击切换 | `pkg/mobile/web/index.html` | 见下方小节分解 |
| 2026-09-27 | Step 2.1 | 重写 `dom-mobile-page.js`：DOM shim 支持 children/attr/style/click；断言 1–8（14 项） | `_dev/mobile-check/dom-mobile-page.js` | 用户 §16 新行为验证 |
| 2026-09-27 | Step 2.2 | shim 修正：`style.display` 闭包 + `_attr` 命名 + `textContent` 聚合子节点 + `answerHtml()` 函数名 | 同上 | 三轮迭代至全绿 |
| 2026-09-27 | Step 3 | 验证链全绿（见 `04-validation.md`） | — | syntax + dom + go vet/test/build |

## 实现细节

### 「不显示 Analysis」的处理选择

按 00-request §6 与用户 15 节：**直接删除** `thinking-card`/`thinking-text`（DOM/CSS 全部移除），而非保留隐藏。理由：

- DOM 回归测试已迁移到新断言（不再依赖 thinking-card），删除不会破坏测试（实测全绿）。
- 保留无用 DOM 会增大后续维护面；`analysis` 字段仍在后端传输（D-UI-005），前端只是不渲染，回滚成本低。

`applySnapshot` 中 streaming 分支：
- `stage=thinking` → 仅 `setBadge('处理中', 'thinking')` + `showWaiting('正在分析题目…')`，payload.analysis 完全不被消费。
- `stage=content` 且有 `payload.answer` → 直接 `showAnswer(renderMarkdown(answer))`。

### 响应式断点

- `< 899px`：`#history-panel` 隐藏；`.history-strip`（横向滚动小卡）显示；header `flex-wrap: wrap`（Phone 允许折行）。
- `≥ 900px`（横屏板）：`.history-panel` 显示（`width: clamp(200px, 22vw, 240px)`）；strip 隐藏。
- 无强制的 768px 断点，因为用户要求 768–899 采用「单列 + 紧凑 strip」，与 <768 共用同一套布局（这正好满足 17-responsivity 的 portrait 验收）。在 `03` 中明确此选择。

### Remote History（内存态）

- `sessionHistory[]` 仅内存，**不持久化**（D-UI-004）；`latestSnapshot` 用于「跟随最新」判断；`selectedSeq`（-1=跟随最新）。
- 写入时机：仅 `status === 'done'` 时 `upsertHistory(snap)`，保存 `{seq, ts, question, answer, model}`，**不保存 analysis**。
- 去重：`findHistory(seq)` → 存在则 update（同 `seq` 覆盖），否则 push + `sort` by seq 升序（最新在列表末，渲染顺序 → 大 seq 靠后；用于 测试 断言顺序）。
- 交互：`selectHistory(seq)` 把 `selectedSeq` 设为该条；若点击的即最新则回落到 `-1` 跟随最新；`showAnswer` 渲染 `renderMarkdown(e.answer)`。
- 刷新：SSE 重连 → 首帧 snapshot → `applySnapshot`（恢复最新一题）；更早历史丢失为该版本可接受限制（用户 11-节明确）。

### Markdown 安全模型

- `toInline(escapeHtml(x))`：先 escape 再套标记；返回 safe HTML。
- 代码块 `<pre><code>escapeHtml`；表格 `th/td` 内 `toInline(escapeHtml)`。
- 视觉强调：`# {1,4}` 标题（H1/H2 高亮色+下边框）、`strong` 提亮、答案/解析前缀行 → `<span class="tag-answer/tag-analysis">`（仅视觉格式化，不推断选项值——用户 13-节红线）。

## 与 02-plan 的偏差

| 02-plan 原计划 | 实际 | 原因 |
|---|---|---|
| 步骤 1 含「768–899 断点」 | 未单独断点，直接共享 < 899 的逻辑 | 用户 6-节明确 portrait 可用「横向 strip/按钮」共享手机布局；合并更简洁且稳定 |
| 步骤 4 History UI 用 question 优先 | question 为空时用 answer 首行摘要（符合用户 9-节「question 不可用→答案首行」） | 后端 `payload.question` 目前为空（001 占位） |
| 步骤 5「补 table」 | 已加入 `renderMarkdown` 表格分支（表头/行解析） | 用户 12 节「如现有实现容易支持，可保留」 |

遗留未完成（见 06）：真机 `wails build` 验收、`_dev/mobile-check` 断言的 tablet 视口真实尺寸渲染（shim 无法做 layout 断言，仅 DOM 行为）。