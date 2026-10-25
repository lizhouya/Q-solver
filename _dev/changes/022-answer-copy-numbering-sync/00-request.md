# Change 022 变更请求：答案复制与分段题号同步

日期：2026-10-24
事前状态（2026-10-24）：计划已批准，事前治理完成；最新实施与验证状态见 03/04。

## 1. 背景与问题

当前完整模型回答仅以 Markdown 字符串保存在 `Record.Answer` 和前端 `round.aiResponse` 中。“最终答案汇总”只在多题时建议输出，格式可为列表或表格，单题不保证存在。因此现有复制入口只能复制题干、解析和汇总在内的全文，无法可靠得到用户真正需要填写的答案。

同一题库 Session 可能跨多次提交、`KeepContext=false`、应用重启和历史分段续接。Solver 最近 10 轮对话不是稳定题号账本，无法保证跨轮题号连续，也无法在截图采用新编号规则时安全切换。

Mobile 快捷键提示虽然使用桌面当前保存的 `KeyName`，但 `app/mobile.go` 的显示白名单尚未包含 Change 016 的 `clear_mobile_display` 和 Change 021 的 `start_next_exam`；页面标题“快捷键”也未说明这些按键实际应在答题端触发。

## 2. 目标

- 单题和多题的完整回答末尾都只出现一次固定格式的 `## 最终答案汇总`。
- Go 后端统一严格解析、规范化和校验答案汇总，并将完整回答与结构化答案原子持久化。
- Desktop 与 Mobile 使用同一结构化结果提供“复制答案”，复制内容只包含每题可填写答案值；现有“复制全文”保持不变。
- 界面仍只显示完整 Markdown 中的一份答案汇总，不重复绘制第二份答案表。
- 题库 Session 内维护受限、持久化的编号摘要；跨轮编号不依赖 Solver 对话，也不新增第二次模型调用。
- Mobile 只展示桌面当前实际绑定且非空的快捷键，并将标题改为“答题端快捷键”。
- 修复 `KeepContext` 从开启切换为关闭时 Solver history 与持久化 `conversationRefs` 可能不一致的问题。

## 3. 固定答案汇总契约

完整回答最后一节必须为：

```markdown
## 最终答案汇总
| 题号 | 题型 | 可填写答案 |
|---|---|---|
| 12 | 单选 | B |
| 13 | 多选 | AC |
| 14 | 填空 | 第一空；第二空 |
```

题型仅接受：`单选`、`多选`、`填空`、`判断`、`简答`、`计算`、`代码`、`其他`，Go 端映射为稳定枚举。

- 单选：一个大写选项字母。
- 多选：移除空格与常见分隔符后，保留大写、不重复的选项字母组合。
- 其他题型：保留可直接填写答案，不含题号、题干、解析、Markdown 标题或“答案：”前缀。
- 一题多空：使用中文分号 `；` 留在同一行。
- 多题复制：按表格显示顺序每题一行；单题复制只有一行。
- 任一行答案或题号为“待核对”、表格结构无效、题号重复或值超限时，不回退复制全文；完整回答仍保留并显示，但整轮“复制答案”禁用或不出现。

## 4. 编号规则

- 当前截图明确显示的完整题号或章节号永远优先。
- 仅当截图确实无题号时，才参考同一 Session 已用题号和安全的下一编号建议。
- 编号规则不同或无法确定时，按截图实际内容显示，或标记“待核对”，不得强行续号。
- 新一套笔试创建新 Session 后编号摘要为空，不继承上一套编号。
- `KeepContext=false`、应用重启和真正续接 ended/exported/paused 分段后，编号参考仍来自当前 Session 的持久化受限摘要。

## 5. 验收标准

### 答案契约与复制

- [ ] 单题和多题回答末尾均有唯一固定三列表格。
- [ ] `pkg/answersummary` 是解析、校验、复制文本和编号提示的唯一事实源。
- [ ] Desktop 与 Mobile 的“复制答案”只使用 Go 端生成的 `answerCopyText`，不自行解析或规范化 Markdown。
- [ ] 单选只复制如 `B`，多选只复制如 `AC`，填空/判断/简答只复制可填写值，多题按显示顺序逐行复制。
- [ ] “复制全文”继续复制完整 Markdown；空结构化结果不得回退为复制全文。
- [ ] 含待核对项或汇总契约无效时，完整回答仍显示和持久化，但“复制答案”不可用。
- [ ] Desktop/Mobile 浏览历史轮时复制所选轮；后台新轮完成不得改变当前浏览轮的复制目标。

### 编号与上下文

- [ ] 当前截图显式题号优先；截图无题号时才使用当前 Session 的受限编号摘要。
- [ ] 三张图片仍只发起一次模型请求，图片顺序保持不变。
- [ ] 编号摘要只作为当前 user multipart 的第一个 text part，图片紧随其后；动态摘要不进入 system。
- [ ] 编号摘要不保存 API Key、Token、完整 Config、Prompt、题干、旧完整 Answer 或解析。
- [ ] `KeepContext=false` 仍可携带 Session 业务编号摘要，但 Solver 历史按既有规则清空。
- [ ] 历史 Answer 在真正续接时仍只作为 assistant，Solver 恢复继续使用 replace，不使用 append。
- [ ] 新 Session 不继承上一套编号；复杂章节号不猜测下一编号。

### 原子性与兼容

- [ ] 完整 `Answer`、`AnswerItems` 和 `ConversationSnapshot` 在同一候选 Session 中一次保存；保存失败不产生半完成内存状态。
- [ ] 解析失败不让解题失败，不发起补偿模型请求；完整回答照常落盘，结构化复制不可用。
- [ ] 保持 Session schema version 2；旧 v1/v2 JSON 无 `answerItems` 时仍可读取，读取历史不改写磁盘。
- [ ] Desktop 终态事件、历史 DTO 和 Mobile `Payload` 同步携带 `answerItems` 与 `answerCopyText`。
- [ ] failed/cancelled/running 状态不发布可复制终态。
- [ ] `true -> false -> true` 的 `KeepContext` 切换不会让旧 `conversationRefs` 复活；in-flight 旧 epoch 结果不能重新追加旧引用。

### Mobile 快捷键

- [ ] Mobile 快捷键卡包含实际绑定的 `clear_mobile_display` 和 `start_next_exam`，空绑定不显示。
- [ ] 标题为“答题端快捷键”，不宣称浏览器按键能触发 Windows 全局 Hook。
- [ ] 改绑后的 `KeyName` 可同步；不发送 action ID、ComboID、VK、Token 或配置路径。
- [ ] 开始下一套不清 Mobile 当前答案；既有清屏屏障仍阻止旧 epoch 回填。

### 验证与交付

- [ ] 按 TDD 先记录 RED，再完成最小实现和 GREEN。
- [ ] 聚焦测试、全量 Go test/vet/build、前端回归、Vite、race、diff-check 按计划如实记录。
- [ ] 真实模型严格遵循固定表格和 Windows GUI 行为保留为人工验收，未经授权不调用真实 API。
- [ ] 固定 Wails CLI v2.11.0、`GOTOOLCHAIN=go1.25.13`；不新增依赖、不升级 Wails runtime。
- [ ] 独立验证程序不得覆盖正式 `Q-Solver.exe` 或旧验证程序。

## 6. 非目标 / 明确不做

- 不新增第二次模型调用，不上传真实截图，不消费真实 API 额度。
- 不从自由 Markdown、题干或解析中猜测可复制答案，不扫描正文选项字母。
- 不为答案表额外绘制第二套可见 UI。
- 不新增 HTTP API、第二个 Mobile store 或第二套确认协议。
- 不迁移、删除、覆盖或改写 `_记录`、截图、旧导出、正式 `Q-Solver.exe`、旧验证程序或用户私人配置。
- 不改 Session schema version，不新增依赖，不升级 Wails runtime。
- 本次建档不修改任何业务源码、配置、依赖、题库、输出或验证程序。

## 7. 影响面预估

预计后续实施涉及：

- 新增 `pkg/answersummary` 纯 Go 包及测试。
- Prompt、Solver multipart、题库 Record/Store/续接、App capture 与历史 DTO。
- `KeepContext` 切换时运行态与持久化会话引用的一致性。
- Desktop Pinia store、ActionBar、SolveView、clipboard 工具与前端回归。
- Mobile Bridge/State/内嵌 Web UI、快捷键提示与 DOM 回归。
- Wails bindings、中英文 README、独立验证构建与治理记录。

主要风险为：模型未遵循契约、错误解析导致误复制、持久化半完成、跨轮编号误续、历史浏览目标被新流覆盖、`KeepContext` 旧引用复活、clipboard 成功状态误报、Mobile 清屏与答案序号语义被破坏、bindings 累积差异或构建产物覆盖。

## 8. 授权边界

用户已批准本计划并授权建立 Change 022 事前治理档案。业务实施尚未开始；本次不修改业务源码、配置、依赖、题库、输出或验证程序，不运行真实模型，不执行 commit、push、Tag 或 Release。后续实施仍须严格限于 `02-plan.md` 白名单，并如实记录未完成、`BLOCKED` 与 `NEEDS_USER` 项。
