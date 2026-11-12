# 实施方案 — 02

## 1. 方案概述

在全局 `QuestionAnsweringRules` 中增加选择题映射优先和四方一致性约束：先识别截图原始字母与选项文本，再判断正确文本，最后回填原字母；正文结论必须同时显示原始字母与对应文本；汇总前复核字母、文本、解析和表格。无法可靠映射时使用“待核对”。同步收紧 `gen-speed` 的内部流程，保持外部简洁但禁止跳过映射。解析器和调用链不变。

## 2. 分步实施计划

| 步骤 | 操作 | 涉及文件 | 预期产出 | 完成标志 |
|---|---|---|---|---|
| 1 | 增加全局和快速模式提示契约红灯 | `pkg/domain/prompts_test.go` | 旧提示缺少新规则并按预期失败 | 失败只指向缺失片段 |
| 2 | 增加映射优先、正文结论和最终复核规则 | `pkg/domain/prompts.go` | 所有解题场景继承一致性约束 | 聚焦测试通过 |
| 3 | 调整 `gen-speed` 内部 Process/Output 文案 | `pkg/domain/prompts.go` | 快速模式不跳过映射，输出仍简洁 | 快速提示契约通过 |
| 4 | 执行聚焦与全量门禁 | 测试/构建 | 无解析、前端或依赖回归 | 必要命令如实记录 |
| 5 | 固定 Wails v2.11.0 构建候选并执行保护核对 | ignored 产物 | 唯一 Change 041 候选 | 不覆盖稳定版/Release |
| 6 | 更新 03–06、审计和 INDEX | `_dev/**` | 完整验证和边界记录 | 保持未授权提交状态 |

## 3. 涉及业务源码的改动清单

允许修改：

- `pkg/domain/prompts.go`：全局选择题映射与一致性规则、`gen-speed` 内部映射步骤。
- `pkg/domain/prompts_test.go`：提示词契约测试。

禁止越界修改：

- `pkg/answersummary/**`、`pkg/solution/**`、`app/**`、`frontend/**`。
- Provider 请求次数、配置 schema、依赖、正式配置和 Change 040 文件。

## 4. 测试与验证计划

1. `go test ./pkg/domain -run 'TestSystemBehaviorPromptChoiceConsistencyContract|TestSpeedPromptPreservesChoiceMappingCheck' -count=1` 红灯。
2. 同一命令转绿。
3. `go test ./pkg/domain -count=1`。
4. `go test ./... -count=1`。
5. `go vet ./...`。
6. `go build ./...`。
7. `go mod verify`。
8. 全部 `frontend/*_test.mjs` 和 `npm --prefix frontend run build`。
9. `git diff --check`、VS Code diagnostics 和提交边界审计。
10. 固定 Wails v2.11.0 保护构建；候选启动若受当前单实例影响，记为 `BLOCKED`。
11. 用户使用候选复验选择题字母、选项文本、解析和汇总一致性。

## 5. 回滚预案

定向恢复新增提示规则和测试，不迁移数据、不修改历史答案。若已有提交，仅在用户授权后使用 `git revert`，不使用 `git reset --hard`。

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 规则过长削弱快速模式 | 输出变慢或冗长 | 映射与复核要求内部执行，外部仍允许简短 |
| 模型仍违反提示 | 个别题仍可能错误 | 不承诺绝对正确；不安全自动改字母；必要时另立二次视觉校验 Change |
| 场景提示覆盖全局规则 | 快速模式跳过映射 | 同时修改 `gen-speed` 并增加独立契约测试 |
| 关键词自动修复误伤 | 错改其他题 | 明确不实现关键词或固定字母映射 |
