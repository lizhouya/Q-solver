# 实施方案 — 02 · 002-remote-output-ui

> 基于 `00-request` 与 `01-current-state`。**动码前完成**；实现偏差记入 `03-implementation.md`。

---

## 1. 方案概述

在 **不改后端协议与数据模型** 的前提下，重做单个远程页面 `pkg/mobile/web/index.html`：

1. **隐藏 Analysis**：删除 `thinking-card`/`thinking-text` 的渲染分支；`stage=thinking` 仅更新业务状态文案（`处理中…`），不写 analysis 文本；当 `payload.answer` 出现（streaming content）直接渲染 Answer。
2. **响应式三档**：
   - `<768px`（Phone）：单栏，Answer-first（Header / 连接状态 / Answer 靠上）。
   - `768–899px`（Tablet Portrait）：单列 Answer + 紧凑 History 入口（顶部横向 history strip）。
   - `≥900px`：双栏（左 History 约 200–240px + 右 Answer flex:1），贴近 PC。
3. **Remote History**：`sessionHistory[]`（内存）+ `latestSnapshot` + `selectedSeq`；仅 `status===done` 写入；按 `seq` 去重（update 而非新增）。
4. **History 交互**：done 自动加入/选中/渲染最新；点击旧历史 → 右侧切该历史 Answer；收到新 streaming 不破坏正在查看的历史（除非选中为 latest）。
5. **Markdown 增强**：沿用 escape→transform→render；补表格（简单）；视觉强化「答案/解析」等标签以提升扫描性（不做语义推断）。
6. **Header**：Header 保留 Q-Solver / 模型 / 连接 / 业务状态，紧凑（平板横排 / 手机可折行）。

---

## 2. 分步实施计划

| 步骤 | 操作 | 涉及文件 | 预期产出 | 完成标志 |
|---|---|---|---|---|
| **1 布局骨架** | 重写 `index.html` 的 `body`：`header`(品牌+模型+badge+conn) + `main`(Answer 区) + `aside`(History 栏，仅 ≥900 显示) + 顶部 history strip(仅 <900 显示) | `pkg/mobile/web/index.html` | 页面三段结构；断点样式 | Phone/Tablet 视觉满足 00 §4 |
| **2 隐藏 Analysis** | 移除 `thinking-card`/`thinking-text` 渲染路径；`applySnapshot` 中 `streaming` 且 `stage===thinking` 只统一业务状态；renderAnswer 仅消费 `payload.answer` | 同上 | 不再显示分析过程 | DOM 脚本断言 streaming-thinking 中无 analysis 文本 |
| **3 History** | 加 `sessionHistory=[]`、`latestSnapshot`、`selectedSeq`；`done` → `upsertHistory()`（按 seq 去重）+ 自动选中 + 渲染；`renderHistoryList()` 生成卡片；`selectHistory(seq)` 切换 | 同上 | 会话内历史 + 去重 + 交互 | 脚本验证：done 加入、重复 seq 不新增、点击切换 |
| **4 History UI** | 卡片：question 首行或 answer 首行摘要 + 时间 + 选中高亮；Tablet（≥900）侧栏；<900 顶部横向 strip | 同上 | 历史卡片 / 侧栏 / strip | 视觉验收 1024 横屏 |
| **5 Markdown 增强** | `renderMarkdown`：保留安全（escape→transform→render）并补 `table`；强化标题层级；对「答案：/解析：/正确答案：」等行加视觉强调类（仅视觉、不做语义推断）| 同上 | 答案更易扫描 | DOM 渲染含表格用例 |
| **6 刷新恢复** | 保持 SSE 重连 + 首帧 snapshot → `latestSnapshot` 直接渲染（历史窗口丢失为可接受，见 00 §5） | 同上 | 刷新显示最新题 | 手动/E2E |
| **7 测试更新** | 更新 `dom-mobile-page.js`：不再要求 thinkingCard 显示；新增 8 项行为断言（待命/streaming-thinking 无正文 / streaming-content 更新 Answer / done 显示 / done 入史 / 同 seq 不重复 / 点击切换 / 新 done 加卡） | `_dev/mobile-check/dom-mobile-page.js` | 行为护栏升级 | `node` 冒烟通过 |
| **8 验证** | `check-raw-syntax.js`（保留 vm.Script）+ `dom-mobile-page.js` + `go test ./pkg/mobile/... ./app/...` + `go build ./...` | — | 自动验证全绿 | 见 `04-validation.md` |

---

## 3. 涉及文件（改动清单，仅限范围内）

| 文件 | 动作 | 说明 |
|---|---|---|
| `pkg/mobile/web/index.html` | 重构 | 唯一业务源码改动点 |
| `_dev/mobile-check/dom-mobile-page.js` | 更新 | 扩展行为断言 |
| `_dev/mobile-check/check-raw-syntax.js` | 不改 | 保留，重跑验证 |
| `_dev/changes/INDEX.md` | 更新 | 登记 002 |

**后端 `pkg/mobile/**`、前端桌面 `frontend/**`、`go.mod`、Wails 版本** —— 一律不动。

---

## 4. 测试与验证计划

| 用例 | 方法 | 预期 |
|---|---|---|
| 语法护栏 | `node _dev/mobile-check/check-raw-syntax.js` | ✅（vm.Script，顶层无非法 return） |
| DOM/行为护栏 | `node _dev/mobile-check/dom-mobile-page.js` | ✅ 8 项行为断言全过 |
| 后端回归（协议未动） | `go test ./app/... ./pkg/mobile/...` | ✅ 原测试全过（应为 green） |
| 构建 | `go build ./...` | ✅ |
| `wails build` + 真机 | 用户在终端执行（本环境无 wails CLI） | ⏳ iPhone / iPad 验收 |

---

## 5. 回滚预案

| 场景 | 操作 |
|---|---|
| `index.html` 回归 | `git checkout -- pkg/mobile/web/index.html`（单文件，回滚瞬时恢复 001 页面） |
| 历史交互状态混乱 | 回退历史逻辑为「仅展示 latest」（001 语义），separate 本变更内 `sessionHistory` 分支 |
| 测试脚本失效 | `git checkout -- _dev/mobile-check/dom-mobile-page.js` 回滚脚本到 001 |
| 后端被误动 | 全都不动（00 §5 红线），若发现误动则 `git checkout` 对应 Go 文件 |

> 详细勾选见 `05-rollback.md`（本变更以「单文件重构」为特征，回滚成本极低）。