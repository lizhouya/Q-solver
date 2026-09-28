# 技术决策记录 05 — Remote Output UI

> 只记录**真正需要拍板**的产品/技术决策。普通渲染细节（CSS 数值等）属实现，见 `03-implementation.md`。

## 产品决策（用户已确认 — D-UI 系列）

| 编号 | 决策 | 备选 | 结论与理由 | 对应文件 |
|---|---|---|---|---|
| D-UI-001 | 远程端是否展示 AI Thinking / Analysis | 展示分析过程 vs **不展示** | **不展示**：答题场景要快速看结论；Thinking 属于桌面体验 | `pkg/mobile/web/index.html` |
| D-UI-002 | 远程端主要内容 | 分析+答案 vs **仅最终 Answer** | **仅 Answer**：结论优先，内容更聚焦 | 同上 |
| D-UI-003 | 平板横屏布局 | 单栏拉伸 vs **History 侧栏 + Answer 主区** | 参考 PC：左历史(约 200–240px) + 右答案 flex:1 | 同上 |
| D-UI-004 | 第一版远程历史 | 持久化(localStorage/IndexedDB/磁盘) vs **仅浏览器会话内存** | 本阶段**不持久化**，避免手机端长期保留答题内容；刷新即丢（已接受） | 同上 |
| D-UI-005 | SSE 数据协议 | 修改协议 vs **不改** | `analysis` 继续传但不渲染；协议零改动，后端零风险 | `pkg/mobile/**`（未动） |

## 实现决策

| 编号 | 决策 | 备选 | 结论与理由 | 对应文件 |
|---|---|---|---|---|
| D-UI-006 | History 去重键 | 按 `answer` 对比 vs **按 `seq`** | 用 `seq`：SSE 可能对同一完成态重复推送 snapshot/message；同 `seq` 则 update 该条而非新增 | `pkg/mobile/web/index.html` |
| D-UI-007 | History 交互状态 | 复杂状态机 vs **`latestSnapshot` + `selectedSeq` + `sessionHistory`** | 最小三态：新 done 自动选中 latest；点旧历史 `selectedSeq`=该条；streaming 到达时仅当 `selectedSeq`=latest 才覆盖右侧 | 同上 |
| D-UI-008 | 响应式断点 | 单一断点 vs **三档**（<768 单栏 / 768–899 单列+strip / ≥900 双栏）| 覆盖手机/平板竖屏/平板横屏三形态；<900 用顶部横向 history strip 作为紧凑入口 | 同上 |
| D-UI-009 | Tablet History 宽度 | 固定 240px vs **`clamp(200px, 22vw, 240px)`** | 响应式钳制，避免过大占用主区 | 同上 |
| D-UI-010 | Markdown 渲染策略 | 直接 innerHTML vs **escape→transform→render** | 维持安全模型；补表格；答案/解析等标签做视觉强调 class（不语义推断 A/B/C/D） | 同上 |

## 明确不做（防 Scope creep）

- **不持久化远程历史**；**不修改 SSE 协议 / Bridge / Payload**；**不动 LLM/Solve/Prompt**；**不新增 npm 依赖 / 不升级 Wails**。
- **不解析自然语言推断题号/选项**——只对模型输出已有的 Markdown 做视觉格式化。
- **不把 PC Vue 组件翻译成远程代码**：远程保持单个 standalone HTML + 无 Vue。
- **第一版不恢复刷新前更早历史**：刷新后仅 SSE 最新 snapshot（接受限制，记 04/05）。