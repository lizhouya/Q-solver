# Change 021 实现记录

日期：2026-10-24
状态：业务实现、正式 bindings 与独立验证构建完成；Windows GUI `NEEDS_USER`，race `BLOCKED`，未提交、未推送、未发布、未关闭。

## 事前状态

- 用户已明确授权实施“开始下一套”优化；未授权 commit、push、Tag 或 Release。
- `00-request.md`、`01-current-state.md`、`02-plan.md` 已在业务源码修改前完成。
- 工作区叠加 Change 016～020 的未提交修改；本 Change 未执行 reset、clean、批量 restore/checkout 或覆盖式重写。
- 构建前已记录 bindings、`go.mod`、`go.sum` 和正式 `build/bin/Q-Solver.exe` 的 SHA-256。

## RED：Store 与 App 事务

1. 新建 `pkg/questionbank/store_next_exam_test.go`，首先锁定 active/paused 封存、ended/exported 幂等、discarded/empty/running/dirty/pending/write-blocked 拒绝、保存失败回滚和下一次 Begin 懒创建。
2. 首次运行按预期因 `s.SealCurrentForNext undefined` 编译失败。
3. 新建 `app/questionbank_next_exam_test.go`，锁定确认、busy/服务不可用、Cancel 后未 Complete、Store 失败、截图/refs/Solver/epoch 清理和 Mobile 不变。
4. 首次运行按预期因 `a.StartNextExam undefined` 编译失败。

## GREEN：Store 原子封存

修改 `pkg/questionbank/types.go`、`pkg/questionbank/store.go`：

- 新增 `SealForNextResult` 和 `Store.SealCurrentForNext()`。
- active/paused 先构造候选 Session，仅在 `saveSessionLocked()` 成功后替换内存 current；保存失败不改 current、dirty 或磁盘状态。
- ended/exported 幂等成功，不改 `EndedAt`、`ExportedAt`、`ExportDirectory` 或 `Exports`。
- discarded、nil/empty、running、dirty、pending completion 和 write blocked 均拒绝。
- 成功后不预建下一段；下一次 `BeginWithContext` 通过既有 `ensureWritableLocked()` 懒创建新 Session。

## GREEN：App 提交边界

新建 `app/questionbank_next_exam.go`：

- 新增 `StartNextExamResult` 与 `App.StartNextExam(confirmed bool)`。
- 锁序固定为 `conversationMu → HasInFlightTask → questionBank.mu → Store → Solver`。
- 在 Store 提交前拒绝未确认、in-flight（包括已 Cancel 但未 Complete）、capture、organizing/exporting、closing 及服务未就绪状态。
- Store 成功是提交点；随后才执行不可失败的内存清理：`solver.ClearHistory()`、清 `conversationRefs`、`conversationEpoch++`、清 `screenshotBuffer`。
- `screenshots-cleared` 与 `question-bank-updated` 在两把 App 锁释放后发送。
- 不调用 Mobile API，不改变 Mobile bridge snapshot、display epoch、固定链接运行态或当前显示。

## GREEN：快捷键与旧配置迁移

修改配置、快捷键服务、设置页和相关测试：

- Windows 新增 `start_next_exam`，默认 `Ctrl+Shift+N`，Combo ID `78+160+162`；macOS 不显示也不建立默认 action。
- 通过持久化 JSON 中 action key 是否存在，区分“旧配置缺失”和“显式空绑定”。
- action 缺失且默认组合空闲时补默认；被其他 action 占用时保留占用者并让新 action 为空。
- 显式空绑定与自定义绑定原样保留；现有 `clear_mobile_display` 特殊迁移语义及测试保持通过。
- 快捷键触发只发送 `request-start-next-exam`，不直接执行题库、截图、Solver 或 Mobile mutation。
- 同步修复录制预览不能提前污染运行态绑定：冲突检查成功后才提交绑定并发送 `shortcut-saved`。

## GREEN：前端确认链

新增 `frontend/src/components/NextExamConfirm.vue`，修改 ActionBar、question-bank store、API 接线和 App：

- 主界面新增“开始下一套”按钮；按钮与快捷键事件共用 `requestStartNextExam()`。
- `nextExamPending` / `nextExamSubmitting` 防止重复弹层、长按和重复提交。
- 取消不调用后端；提交中禁用取消和确认。
- 后端失败保留弹层和错误，不清桌面历史；成功后才调用 `solution.resetLiveHistory()`、设置 `ui.hasStarted=false`、刷新题库状态并关闭弹层。
- 确认文案明确清理桌面历史、待发送截图与模型上下文，同时保留题库原始记录、归档截图、旧导出和同步端当前显示。
- 未调用 `clearMobileDisplay`，未新建第二个 Mobile store 或第二套 Mobile 清屏协议。

## bindings 与构建产物

- 使用固定 Wails CLI v2.11.0 正常生成 bindings，并生成 `StartNextExam` / `StartNextExamResult`。
- 正常生成同时把工作区中 Change 016～018 已存在但先前未全部进入生成文件的 App API/DTO 一并同步；该累计差异经编译和回归验证，不是 Change 021 新增业务范围。
- 事前只保存了 bindings 的哈希而非文件副本，因此不能从当前文件严格重建并逐字节证明“只增加 StartNextExam”；治理记录明确保留此限制，不伪称单接口字节级差异通过。
- 固定生成器在 `models.ts` 的纯空白行写入 Tab；仅清除了纯空白行，业务类型和字段未手改。
- 随后使用 `-skipbindings -nosyncgomod` 从已审计源码重建独立程序：`build/bin/Q-Solver-change021-start-next-exam-validation.exe`；治理收尾后再次用同一命令重建，哈希保持不变。
- 正式 `build/bin/Q-Solver.exe`、`go.mod`、`go.sum` 和旧验证程序未被覆盖。

## 与计划的偏差

- `SealForNextResult` 是 Store 内部结果，不经 App 方法直接暴露，因此 Wails 只生成 `StartNextExamResult`；没有额外暴露内部 Store DTO。
- bindings 是基于当前累计 Go API 的完整再生成，不满足计划中“仅接受单接口所需字节差异”的严格表述；当前 Go 源码、bindings、Frontend 与 Wails 构建一致，累计接口均保留，但缺少事前文件副本，无法补做精确旧内容比对。
- 独立只读审查代理因 API 429 限流未完成，不能记为独立审查 PASS；已返回的局部结论未发现锁序、DTO、配置迁移或快捷键分发缺陷。

## 未完成项

- Windows GUI 人工验收仍为 `NEEDS_USER`。
- Windows race 程序启动阶段仍以 `0xc0000139` 失败，状态为 `BLOCKED`。
- 未获得 commit、push、Tag 或 Release 授权，相关操作未执行。
