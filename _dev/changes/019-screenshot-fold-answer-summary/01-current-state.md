# Change 019 现状快照

日期：2026-10-23
状态：业务修改前快照。

## 涉及文件与现状

| 文件 | 当前职责 | 现状 |
|---|---|---|
| `frontend/src/components/RoundScreenshots.vue` | 主解题界面按 Record 展示截图 | `image.data` 有值即直接显示图片；无值时显示读取按钮；没有独立展开状态，也不能再次收起。 |
| `frontend/src/components/SolveView.vue` | 主界面历史与答案渲染 | 按 `round.recordId || idx` 遍历轮次，并把 `load` 事件交给 `solution.loadRoundImage`。 |
| `frontend/src/stores/solution.js` | 会话历史、流事件与历史图片缓存 | 历史图包含 `data/loading/error/requestVersion`；`loadRoundImage` 已处理缓存、并发、迟到响应和第一图 `userScreenshot` 兼容。 |
| `pkg/domain/prompts.go` | 统一行为规范和场景角色 Prompt | 已要求综合全部截图、选择题明确选项；尚未要求识别全部题、跨图合并同题、重叠去重和多题答案汇总。 |
| `pkg/solution/solution.go` | 组装 system 与多模态请求 | 一次 Solve 把 1～3 张截图按顺序放入同一条 user multipart 消息；每次按当前配置重建 system。 |
| `pkg/solution/solution_test.go` | Solver 上下文和多模态回归 | 已有 fake provider、当前 system 替换和单图多模态历史测试；没有三图单次调用与完整汇总正文透传测试。 |

## 关键行为

- 实时请求的 `event.images` 会直接写入 `round.images[].data`，因此当前截图进入内容区后常驻显示。
- 历史记录先建立空 `data` 描述符；首次点击读取成功后，按钮被图片替换，不能再次隐藏。
- `loadRoundImage` 在已有 `image.data` 或 `image.loading` 时拒绝重复读取，并在 Record/Session 失效或请求代际变化时拒绝迟到回写。
- 第一张历史图读取成功后同步到 `round.userScreenshot`，用于既有兼容路径；显示折叠不得清空该字段。
- `GetSystemBehaviorPrompt()` 与当前 Scene Prompt、ResumeContent 组成唯一 system；历史 Answer 只作为 assistant。
- 当前多图不按图片分别调用模型，也没有额外汇总请求或结构化输出 schema。

## 构建与测试现状

- 前端回归使用 Node + Pinia/VM 与 Vue 源码静态断言，没有 Vitest/Vue Test Utils。
- 固定 Wails CLI：`_dev/changes/015-rc-release/tmp/tools/wails.exe` v2.11.0。
- 固定 Go 工具链：`GOTOOLCHAIN=go1.25.13`。
- Change 018 普通自动验证通过；Windows GUI 仍 `NEEDS_USER`；race 启动阶段曾因 `0xc0000139` 标记 `BLOCKED`。

## 工作区基线

当前分支为 `dev/qsolver-custom`。工作区叠加 Change 016～018 的大量未提交和未跟踪文件，其中包括本次将继续修改的 `SolveView.vue`、`solution_test.go` 等。Change 019 只增加自身局部差异，不通过 reset、checkout、clean 或覆盖方式清理既有内容。

## 已知但本次不处理

- `general-assistant` 默认 ID 通过未知 ID 回退获得 `gen-solver` Prompt，界面选中状态可能不一致。
- 一次提交最多三张截图。
- Prompt 只能提高模型遵循概率，自动测试不能证明真实模型一定正确识题或汇总。
