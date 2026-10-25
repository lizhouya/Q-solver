# Change 022 现状快照：答案复制与分段题号同步

日期：2026-10-24
状态：业务源码修改前快照；尚未实施。

## 1. 当前数据形态

- 完整模型回答以 Markdown 字符串保存在 `pkg/questionbank.Record.Answer`，桌面实时轮次对应 `round.aiResponse`。
- 当前没有独立、稳定的结构化答案项字段，也没有由 Go 统一生成的仅答案复制文本。
- 题库的权威边界仍为 `Session -> Records -> Questions`；一次正式提交对应一个 Record/round，完整 Answer 是必须保留的原始结果。
- 当前 Session schema version 为 2；计划新增字段必须是可选字段，以保持旧 v1/v2 JSON 向后兼容。

## 2. 当前答案汇总与复制行为

- Change 019 只要求多图多题时建议输出“最终答案汇总”，其格式可为列表或表格；单题不保证有汇总。
- 现有输出没有固定三列表格、稳定题型枚举、严格答案值规范和统一大小限制。
- 当前复制能力针对完整 Markdown，包含题干、解析和答案汇总，不能可靠得到用户应填写的答案值。
- Desktop 和 Mobile 尚无可共享的 `answerItems` / `answerCopyText` 结构化契约。
- 自由 Markdown 中可能出现选项字母、伪标题或不完整汇总；从正文正则猜测答案会产生误复制风险。

## 3. 三套历史与一次 round/Record 边界

- Solver 历史位于 `pkg/solution.Solver.chatHistory`，用于模型对话，不是持久化题号账本；其既有裁剪窗口不能保证覆盖整个题库 Session。
- 题库持久化历史位于 Session/Record；Record 的完整 Answer 和 ConversationSnapshot 属于同一完成边界。
- 桌面展示历史由 `frontend/src/stores/solution.js` 派生和选择；Change 020 已建立 typed `epoch + sessionId + recordId` 写入目标，浏览旧轮时后台新流不应覆盖所选轮。
- Mobile 使用独立 Bridge/State、Snapshot、SSE 和内存历史；`seq`、`clearSeq`、`controlSeq` 与 display epoch 已有独立语义，不能因答案复制重定义。
- Change 021 的“开始下一套”在成功后清理桌面与 Solver 运行态，下一次正式提交才懒创建新 Session；Mobile 当前显示保持不变。

## 4. 当前编号来源与续接边界

- 同一题库 Session 可跨多次提交、`KeepContext=false`、应用重启和 ended/exported/paused 分段续接。
- Solver 最近对话不是稳定编号账本；`KeepContext=false` 会清模型历史，但题库 Session 业务编号仍需要连续参考。
- 真正续接历史时，旧 Answer 只能作为 assistant turn 恢复，不得提升为 system 或 user 指令；恢复使用 replace，不使用 append。
- 当前没有仅包含题号的受限、持久化编号摘要，也没有“显式截图题号优先、无题号时才参考建议”的统一传输契约。
- 旧记录可能只有自由格式“最终答案汇总”；若用于编号参考，只能从明确汇总段保守提取 Label，不能由此生成可复制答案。

## 5. KeepContext 引用一致性现状

- `App` 同时维护 Solver history、`conversationRefs` 和 `conversationEpoch`。
- 现有 `KeepContext=false` 路径会清 Solver history，但持久化引用与运行态引用需要在同一 `conversationMu` 边界内统一处理。
- 已批准计划明确要用测试重现 `true -> false -> true` 且中间不提交 false 请求的场景，防止旧 `conversationRefs` 重新进入后续快照。
- 已完成旧 epoch 的 in-flight 请求也必须被阻止重新追加旧引用。
- 编号账本不得依赖 `conversationRefs`，否则关闭上下文、重启或续接时仍会不稳定。

## 6. Mobile 快捷键提示现状

- Mobile 快捷键数据流使用桌面当前保存的 `KeyName`，并通过白名单选择可展示动作。
- `app/mobile.go` 的显示白名单尚未包含 Change 016 的 `clear_mobile_display` 和 Change 021 的 `start_next_exam`。
- 当前页面标题为“快捷键”，没有明确说明按键实际应在 Desktop 答题端触发；浏览器页面不具备 Windows 全局 Hook 语义。
- 现有提示只应传递 `{label,keyName}`；空 KeyName 应跳过，action ID、ComboID、VK、Token 和配置路径不应进入 Mobile payload。

## 7. 关键文件与当前职责

| 文件/目录 | 当前职责 | Change 022 关注点 |
|---|---|---|
| `pkg/domain/prompts.go` | 统一系统行为与答案输出要求 | 收紧为单题/多题统一固定汇总契约 |
| `pkg/solution/solution.go` | 组装一次 multipart 模型请求和 Solver 历史 | 当前 user multipart 前置受限编号摘要，保持一次请求 |
| `pkg/questionbank/types.go` | Session/Record 持久化类型 | 新增可选 `AnswerItems`，保持 schema version 2 |
| `pkg/questionbank/store.go` | Record 完成、原子持久化与 pending completion | 完整 Answer、Items、Snapshot 同一提交边界 |
| `pkg/questionbank/continuation.go` | 分段续接与会话恢复 | 编号摘要、旧 Label 保守提取、上下文清理 |
| `app/solve.go` | 建 Record、发起解题、维护 conversation refs | Record 后读取编号上下文；修复 KeepContext 切换一致性 |
| `app/questionbank_solve_capture.go` | capture 解题终态与题库落盘 | 最终答案只解析一次，保存成功后再发布终态 |
| `app/questionbank_history.go` | 历史 DTO | 从持久化 Items 重建 CopyText；旧记录只读临时解析 |
| `app/mobile.go` | Mobile 路由与快捷键提示 | 补齐两项提示并保持脱敏白名单 |
| `pkg/mobile/state.go`、`bridge.go` | Mobile Snapshot/历史/发布兼容 | 携带 Items/CopyText，保持序号和旧一参数 Publish 兼容 |
| `frontend/src/stores/solution.js` | 桌面实时轮、归档轮和浏览目标 | 每轮绑定结构化结果，不让后台流改变选择目标 |
| `frontend/src/components/ActionBar.vue` | 完整答案操作入口 | 保留复制全文，新增可靠复制答案 |
| `frontend/src/components/SolveView.vue` | 解题结果与 ActionBar 展示 | 只对 completed 轮展示终态操作 |
| `pkg/mobile/web/index.html` | Mobile 内嵌页面 | 答案复制与“答题端快捷键”文案 |
| `frontend/wailsjs/go/models.ts` | Wails 生成 DTO | 最终以固定 CLI 全量生成并审计累计差异 |

## 8. 配置、依赖与构建基线

- 固定 `GOTOOLCHAIN=go1.25.13`。
- 固定 Wails CLI v2.11.0，不升级 Wails runtime。
- 不新增 Go 或前端依赖，不修改用户配置，不读取或输出 API Key、Mobile Token。
- 工作区叠加 Change 016～021 的未提交修改；不得 reset、clean、批量 restore/checkout 或覆盖式恢复。
- 正式 `build/bin/Q-Solver.exe` 与旧验证程序必须保持不变；后续验证构建使用独立名称。
- Windows race 在 Change 021 时于启动阶段出现 `0xc0000139`；若 Change 022 复现只能如实记为 `BLOCKED`，普通测试不能替代。

## 9. 已知限制与未验证项

- 真实模型是否稳定遵循固定表格必须由用户授权后人工验收，自动测试只能验证 Prompt 与传输契约。
- Windows GUI clipboard、历史轮选择、快捷键提示和 Mobile 浏览器 fallback 需要人工验收；当前未执行。
- 当前尚无 Change 022 业务实现、测试结果、bindings 差异或独立验证程序。
- 本快照来自已批准计划、项目规则和 Change 021 治理资料；本次建档未修改或运行任何业务内容。
