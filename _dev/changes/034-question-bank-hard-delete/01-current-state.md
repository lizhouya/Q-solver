# Change 034：现状快照

## 当前界面语义

- `QuestionBankPanel.vue` 的“清空本段”确认文案明确写着“仅标记清空，原始文件不会物理删除”。
- `questionBank.discardCurrent` 成功提示为“桌面历史已清空；题库原始记录和模型上下文仍保留”。
- `questionBank.splitCurrent` 只调用后端封存并提示下一题进入新段，不调用 `solution.resetLiveHistory()`。
- 导出弹窗根据用户选择执行原稿导出或模型整理，成功后自动封段。

## Store 现状

- `Store.Begin/BeginWithContext` 在模型调用前创建当前分段，并写入截图 assets 和 `session.json`。
- `Store.DiscardCurrent` 只把当前分段写为 `discarded`，保存 `DiscardedAt`；不会删除目录。
- 历史读取仍允许查看 `discarded` 分段和截图，这是旧数据兼容行为。
- `Store.SealCurrent` 将 `active/paused` 分段改为 `ended` 并保留全部文件。
- Store 已记录 `currentBase`，可定位主记录目录或 legacy 目录中的当前分段。

## App 现状

- `DiscardQuestionBankSegment` 调用 `Store.DiscardCurrent`，成功消息为软清空。
- 当前清空不会修改 Solver 历史、`conversationRefs` 或 `conversationEpoch`。
- `StartNextExam` 已建立安全锁顺序和“持久化成功后再清理运行时状态”的模式，可作为物理清空的事务边界参考。
- `bankAvailableLocked` 会拒绝 captures、整理和导出，但物理删除还需 Store 层拒绝 dirty、pending completion、running、exported 等状态。

## 风险

- 物理删除不可恢复，必须保留二次确认并严格限制到当前分段目录。
- 删除失败不能先清除前端、Solver 或 current 指针。
- 当前分段可能来自 legacy 记录根，不能假定总在主 `_记录` 目录。
- 旧 `discarded` 历史必须继续可读，不能因新增物理删除而追溯清理。

## 工作区基线

Change 034 开始前已有 Change 033 的预期未提交文件：Mobile Web 页面、Mobile DOM/嵌入页测试、Change 033 档案与审计、INDEX。暂存区为空。Change 034 不修改这些 Change 033 功能文件，验证和提交边界分别记录。
