# 变更请求 — 00 · 002-remote-output-ui

| 字段 | 值 |
|---|---|
| 变更编号 | `002-remote-output-ui` |
| 提出人 | 产品/负责人（Q-Solver 二次开发） |
| 提出日期 | 2026-09-27（修订产品决策后正式成文） |
| 前置变更 | `001-mobile-output`（已完成：手机页可连接、SSE 实时、LAN IP、token） |
| 关联审计 | 无（本变更为 001 交付页面的产品化修订） |
| 状态 | `进行中`（规划中 → 已在本文档 + 01/02/05 完成后进入实现） |

## 1. 背景与问题

001 交付了手机页 `pkg/mobile/web/index.html`：手机可连接本机 SSE → 实时显示 **状态 + 分析过程（thinking）+ 答案**。但真机试用后产品决策调整：

- 手机上滚动看「分析过程」才能看到最终答案，**答案不靠上**，答题场景要快速看结论；
- 平板横屏仍是窄手机容器 + 左右大面积空白，未利用大屏；
- 无历史记录：解题多题后无法回看前几题答案。

本次变更基于**已确认的新产品决策**（见 §3）修订远程输出 UI，不动后端协议与数据模型。

## 2. 目标

1. 远程端（手机/平板）**不再展示 AI Thinking / Analysis Process**，只突出**最终 Answer**。
2. **Phone（<768px）**：单栏 Answer-first（Header / 连接状态 / Answer 在最上方），无需滚过 Analysis。
3. **Tablet / 大屏（≥900px）**：双栏（紧凑 History 侧栏 + 宽 Answer 主区），布局贴近 PC（历史 | 答案）。
4. **Tablet Portrait（768–899px）**：不强制窄双栏；采用单列 Answer + 紧凑 History 入口（按钮 / 顶部横向 strip）。
5. **Remote History（第一版）**：仅维护当前浏览器页面会话内历史（内存态），**不持久化**。
6. Answer 排版参考 PC（标题/段落/列表/粗体/答案/解析），视觉上答案结论更易扫描。
7. **不修改 SSE 数据协议、不修改 Bridge 核心、不修改 LLM/Solve/Prompt、不增加 npm 依赖、不升级 Wails。**

## 3. 产品决策确认（用户已确认 — D-UI 系列）

| 决策编号 | 决策项 | 确认值 |
|---|---|---|
| D-UI-001 | 远程端是否展示 AI Thinking / Analysis | **不展示** |
| D-UI-002 | 远程端主要内容 | **仅最终 Answer** |
| D-UI-003 | 平板横屏布局 | 参考 PC：**History 侧栏 + Answer 主区域**（双栏） |
| D-UI-004 | 第一版远程历史存储 | **仅当前浏览器会话内，内存态；不持久化**（不用 localStorage / IndexedDB / 磁盘） |
| D-UI-005 | SSE 数据协议 | **不改**；`analysis` 字段可继续传输，但远程 UI **不渲染** |

## 4. 验收标准

- [ ] Phone（~390px 宽）：单栏，打开 URL 后 Answer 优先（无需先滚过 Analysis 即可见 Answer/待命）
- [ ] Phone：Header（Q-Solver/模型/连接/业务状态）紧凑不占过多首屏高度
- [ ] Tablet 横屏（≥900px，如 1024）：左侧紧凑 History（约 200–240px）+ 右侧宽 Answer，无明显大面积左右空白
- [ ] Tablet Portrait（768–899px）：单列 Answer + 紧凑 History 入口
- [ ] 收到 `status=done` 的 snapshot → 自动加入 History（去重）、选中新答案、右侧显示最新 Answer
- [ ] 同一 `seq` 的重复 `done` 推送 → History 不重复
- [ ] 点击历史卡片 → 右侧切换显示该历史 Answer；选中项高亮
- [ ] streaming（stage=thinking）仅更新业务状态（处理中），**不显示 analysis 正文**；streaming（content）直接渲染 Answer
- [ ] 刷新浏览器 → SSE 重连 → 展示当前 session 内最新一题快照（历史丢失为已接受限制）
- [ ] Markdown 渲染（标题/粗体/有序/无序/行内代码/代码块/引用）安全（escape → transform → render）
- [ ] `node _dev/mobile-check/check-raw-syntax.js`（vm.Script）与 `node _dev/mobile-check/dom-mobile-page.js`（六状态+历史交互扩展）均通过
- [ ] `go test ./app/... ./pkg/mobile/...`、`go build ./...` 通过（后端协议不动，应为干净回归）
- [ ] 真实设备（iPhone / iPad）由用户在 `wails build` 后手工验收

## 5. 非目标 / 明确不做

- ❌ 不修改 SSE 协议 / Bridge / `payload` 结构（`analysis` 继续传但不渲染）
- ❌ 不修改 LLM / Solve / Prompt / 截图
- ❌ 不新增 npm 依赖、不升级 Wails
- ❌ 不持久化远程历史（不做 localStorage / IndexedDB / 磁盘存储）
- ❌ 第一版历史**只在当前浏览器页面会话内**，刷新后仅恢复最新 snapshot，不恢复更早历史（已接受限制）
- ❌ 不解析自然语言去推断题号/选项（只对模型输出已有的 Markdown 做视觉格式化）
- ❌ 不在远程端做任何控制（只读）

## 6. 影响面预估

- 修改（仅`pkg/mobile/web/**`、`_dev/mobile-check/**`、`_dev/changes/002-remote-output-ui/**`）：
  - `pkg/mobile/web/index.html`（单页 HTML/CSS/JS：布局响应式 + 隐藏 analysis + History + 去重 + 历史交互 + Markdown 增强）
  - `_dev/mobile-check/dom-mobile-page.js`（更新：不再要求 thinkingCard 显示；新增 history/去重/点击切换验证）
  - `_dev/mobile-check/check-raw-syntax.js`（保留，不改；确证仍通过）
- 未改而后端或接口变化导致 → 立即回滚（见 `05-rollback.md`）
- 风险：单一 `index.html` 若改动过大易引入回归 → 依赖两层 JS 护栏（语法 + DOM 行为）