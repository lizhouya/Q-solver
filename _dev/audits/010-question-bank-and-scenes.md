# 题库归档与场景选择审计

日期：2026-10-18。静态代码审计，不代表真机行为已通过验收。


- 当前根目录就是 Q-Solver 仓库，分支为 `dev/qsolver-custom`；“自动网申”已核实为空。
- 已有修改：`AGENT.md`、`CLAUDE.md`、`_dev/README.md`、`_dev/changes/INDEX.md`；另有未跟踪的 `AGENTS.md`。保留这些内容，不覆盖、不清理、不擅自提交。
- `app/solve.go:113-152` 为一次请求复制完整截图，但桌面 `user-message` 只发送第 1 张。不能用桌面历史作为完整归档数据源。
- `frontend/src/stores/solution.js` 的历史仅在内存，包含覆盖、删除和多轮合并行为；已有导出是答案图片，并非整场 Markdown。
- `pkg/solution/solution.go` 的 `chatHistory` 是模型上下文，最多保留最近 10 轮；归档必须与 `KeepContext` 解耦。
- `app/mobile.go:35-49` 在手机独占模式下不把答案送到桌面；归档必须发生在输出分流之前。
- `pkg/config/manager.go:31-47` 正常使用系统用户配置目录存放加密配置。迁移仓库不要求迁移或重写密钥。
- 新 Change 编号已交叉核对 INDEX、实际目录及近期提交，下一编号为 010；003/004 已使用，不复用。


## 场景选择可行性（本次不实施）


### 手动更换：已有基础能力

- `DomainSelector.vue` → `SettingsModal.vue` → `settings.saveSettings()` → `UpdateSettings` 保存 `domainId`。
- 下一次 `solveInternal` 获取配置后，`Solver.Solve` 通过 `domain.GetPrompt` 注入新场景；已经发出的请求不会被新选择追溯修改。
- 当前没有做题页快捷切换入口，也没有完善的“流式处理中禁止/延后保存场景”的策略；静态调用链可行不等于并发行为已经验收。
- `KeepContext=true` 时 `ensureSystemPrompt` 会替换 system 提示词，但旧问答仍保留。跨题型切换可能受旧上下文影响，未来宜提供“保留/清空上下文”的显式策略。

### 自动选择：可以实现，但现状不等于已实现

- 通用解题 `gen-solver` 提示模型理解题目类型，不会把识别结果回写 `domainId`，也没有分类路由器。
- 可选方式：通用场景内适配（无额外调用，但不是真正切换）；关键词规则（便宜，但截图还需要文字识别）；视觉模型先分类再选择场景（可观察、可审计，但增加延迟/费用且有误判）。
- 后续推荐“自动推荐 + 手动锁定”：分类只允许现有场景白名单，低置信度回退通用；用户选择优先；当前请求锁定快照；不在流式输出中途替换。
- 人格风格与题目领域应分开，不根据题型推断或自动切换 MBTI 等人格设定。

本次只把上述结论和代码依据写入审计，不修改 DomainSelector、domain prompts 或场景配置行为。题库的题型标签不反向控制解题角色。


## 关联档案

- [Change 010](../changes/010-question-bank-export/02-plan.md)
- [Change 011](../changes/011-job-workspace-layout/02-plan.md)
