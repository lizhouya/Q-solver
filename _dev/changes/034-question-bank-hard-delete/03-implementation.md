# Change 034：实现记录

## Store 物理删除

- 新增 `Store.DeleteCurrent`，旧 `DiscardCurrent` 保持不变，用于兼容旧版本已经持久化的 `discarded` 历史。
- 删除前拒绝：write-blocked、pending completion、dirty/pending、空分段、running 记录、exported、discarded 和未知状态。
- 只允许删除 `recordsRoot` 或已注册 legacy root 中、ID 合法且不是符号链接的当前分段目录。
- 删除时先把当前目录原子重命名为同级 `.deleting-<session>-<random>` 暂存目录，再执行递归删除；删除完成前不修改 Store current。
- 删除成功后清空 `current`、`currentBase`、`pending`、`dirty` 和保存时间水位；下一次 Begin 懒创建新分段。
- 删除失败时先恢复原目录；恢复后逐一验证 `session.json` 和所有记录截图仍为普通文件。
- 如果恢复或完整性校验失败，设置 `writeBlocked` 并返回“停止继续操作、保留现场”，不伪造成功。

## App 事务边界

- 保留 Wails API `DiscardQuestionBankSegment`，内部由软清空改为 `DeleteCurrent`，因此不需要重新生成 bindings。
- 二次确认文案明确说明记录、答案和截图永久删除且无法恢复。
- 使用 `conversationMu → TaskCoordinator → questionBank.mu → Store → Solver` 顺序。
- 拒绝 running 和已取消但尚未 Complete 的 in-flight 任务、captures、organizing、exporting、closing、Store/solver 不可用状态。
- 只有 Store 物理删除成功后才清理 Solver 历史、`conversationRefs` 并递增 `conversationEpoch`。
- 尚未发送的截图缓冲和 Mobile 当前显示保持不变；成功后通知题库状态刷新。

## 前端

- 清空确认改为不可恢复警告，确认按钮改为“永久删除”。
- 成功提示改为“当前分段、答案和截图已从本地永久删除”。
- 后端失败时不重置实时历史；成功后才调用 `solution.resetLiveHistory()`。
- “结束本段”成功后清理实时解题历史并将主界面恢复为空闲，但后端仍只封存，本地记录和模型上下文保持现有语义。
- 导出代码未修改，继续使用既有整理、导出和自动封段流程。

## TDD 过程

1. Store 红灯：`DeleteCurrent undefined`、`removeAllHook undefined`。
2. App 红灯：分段目录仍存在，未确认提示仍说原始文件保留。
3. Frontend 红灯：缺少永久删除警告。
4. 实施最小功能后聚焦测试转绿。
5. 新增 canceled-but-incomplete 和 exporting 门禁，红灯证明旧实现会删除；补门禁后转绿。
6. 新增部分删除失败测试，红灯证明仅恢复目录名会误报完整；补逐文件完整性校验和写阻断后转绿。

## 范围

业务与测试文件严格限定为：

- `pkg/questionbank/store.go`
- `pkg/questionbank/store_delete_test.go`
- `app/questionbank.go`
- `app/questionbank_delete_test.go`
- `frontend/src/components/QuestionBankPanel.vue`
- `frontend/src/stores/questionBank.js`
- `frontend/questionbank_history_regression_test.mjs`

未修改题库 schema、导出逻辑、模型请求、CCR、Mobile、配置或依赖。Change 033 的 Mobile 文件是实施前已有差异，本 Change 未修改。
