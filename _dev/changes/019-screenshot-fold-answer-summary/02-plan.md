# Change 019 实施方案

日期：2026-10-23
状态：用户已批准。

## 方案概述

前端把“是否显示”从图片缓存状态中分离：`RoundScreenshots.vue` 为每张图片维护本地展开集合，默认收起，展开未缓存图片时才复用既有 `loadRoundImage`，收起不取消请求也不清缓存。Prompt 在统一 system behavior 中增加仅对解题任务生效的多题规则，保持现有一次多图请求和各场景详略要求。

## 实施步骤

1. 先增加截图折叠与 Store 缓存保护回归，确认旧组件不满足新交互。
2. 实现本地展开状态、会话/Record 作用域重置、可访问按钮和图片点击收起。
3. 先增加 Prompt 语义与三图单次请求保护测试，确认规则缺口且请求链无需修改。
4. 仅修改 `GetSystemBehaviorPrompt()`，增加多图多题、去重、题号、待核对和答案汇总规则。
5. 运行专项、全量、race 和 diff-check；构建独立 Change 019 GUI 验证程序并更新治理记录。

## 业务源码白名单

- `frontend/src/components/RoundScreenshots.vue`
- `frontend/src/components/SolveView.vue`
- `frontend/round_screenshots_regression_test.mjs`
- `frontend/questionbank_continuation_regression_test.mjs`
- `pkg/domain/prompts.go`
- `pkg/domain/prompts_test.go`
- `pkg/solution/solution_test.go`

除非测试证明事前事实错误，否则不修改 `frontend/src/stores/solution.js`、`pkg/solution/solution.go`、图片后端/API、Wails bindings、`go.mod` 或 `go.sum`。如需越界，先暂停并更新现状与方案。

## 前端状态规则

- 每张图独立、默认收起；`scopeKey` 由当前 Session/历史项与 Record 组成，作用域改变时只清空本地展开集合。
- 展开未缓存图片时才 emit `load(round, image)`；已缓存或读取中不重复调用。
- 收起不写 `image.data/loading/error/requestVersion`，不写 `round.userScreenshot`，不取消请求。
- 加载中可以收起；迟到成功只更新 Store 缓存，不自动展开。
- 主按钮显示“显示截图（N）/隐藏截图（N）”，提供 `aria-expanded`；已显示图片本身也是可聚焦的收起按钮。

## Prompt 规则

- 仅在当前场景要求直接解题或最终答案时应用，不覆盖 OCR、纯翻译等非解题场景。
- 只处理当前 user 消息中的原题；历史只作为上下文，变式题答案单列。
- 截图数不等于题目数；跨图连续材料合并，重叠同题只答一次。
- 逐题沿用原题号；无题号按首次出现顺序编号；无法可靠辨认时标记“待核对”并禁止猜测。
- 两道及以上原题时，正文末尾输出 `## 最终答案汇总`；选择题汇总选项，其他题型汇总结果/结论，且必须与正文一致。
- 单题不机械重复汇总；`gen-speed` 继续简洁，`gen-similar` 不把新生成练习混入原题汇总。

## 验证计划

- 前端：新截图组件回归、continuation/history/settings/mobile 回归、Vite build。
- 后端：`pkg/domain` Prompt 契约、`pkg/solution` 三图单次请求与完整答案透传、`go test ./...`、`go vet ./...`、`go build ./...`。
- 并发：`go test -race ./pkg/domain ./pkg/solution`；若环境仍返回 `0xc0000139`，如实记 `BLOCKED`。
- 集成：固定 Wails v2.11.0 与 Go 1.25.13 构建 `Q-Solver-change019-answer-summary-preview-validation.exe`，归档并更新 LF-only SHA256SUMS。
- 人工：Windows GUI 交互记 `NEEDS_USER`；真实模型效果未经额外授权不执行，记 `NEEDS_USER`。

## 风险与对策

| 风险 | 对策 |
|---|---|
| 组件复用导致展开状态串到其他轮次 | 传入 Session + Record 的稳定 `scopeKey`，变化时清空本地集合。 |
| 收起触发重复读图或影响导出 | 显示状态与 `image.data`、`userScreenshot` 完全分离，并增加缓存保护回归。 |
| 公共 Prompt 破坏特殊场景 | 规则条件化，明确 OCR/翻译/秒杀/变式题边界，并加 Prompt 契约测试。 |
| Prompt 变更被误认为确定性模型保证 | 自动验证只证明规则注入和链路完整；真实模型单列人工验收。 |
| 叠加工作区误覆盖旧 Change | 只编辑白名单文件，不使用破坏性 Git 命令，记录实施前后边界。 |

## 回滚

只定向撤销 Change 019 在白名单文件中的局部改动与新测试，删除且仅删除 Change 019 新验证程序，保留治理档案记录原因；不回退 Change 016～018，不删除任何题库、截图或旧导出。
