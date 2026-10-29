# Change 034：实施方案

## Store 物理删除

新增 `Store.DeleteCurrent`，保留旧 `DiscardCurrent` 供历史兼容测试和旧软清空数据语义使用。`DeleteCurrent`：

1. 在 Store 锁内拒绝 write-blocked、pending completion、dirty、空分段、running、exported 和 discarded；
2. 只根据经过校验的当前 Session ID 与 `currentBase` 定位当前分段目录；
3. 先将目录原子重命名到同级删除暂存名，再递归删除，避免删除开始前就改变内存 current；
4. 删除成功后才清空 `current/currentBase/pending/dirty` 并返回已删除 Session ID；
5. 删除失败时尝试恢复原目录并保持 current；若恢复失败则阻止后续题库写入并返回严重错误，不伪造成功。

不扫描或删除其他目录，不清理导出结果，不自动处理旧 `discarded` 分段。

## App 事务边界

`DiscardQuestionBankSegment` 保留现有 Wails API 名称，但内部改用 `DeleteCurrent`：

- 使用 `conversationMu → questionBank.mu → Store → Solver` 锁顺序；
- 拒绝进行中的解题、整理、导出和关闭状态；
- Store 删除成功后才清理 Solver 历史、`conversationRefs`，并递增 `conversationEpoch`；
- 不清除尚未发送的截图缓冲，不修改 Mobile 显示；
- 失败时运行时状态零副作用；成功后通知题库状态更新。

## 前端行为

- 清空确认改为永久删除警告，按钮文案改为“永久删除”。
- 清空成功提示明确本地分段和截图已删除；随后清理实时解题历史。
- “结束本段”成功后新增 `solution.resetLiveHistory()` 和 `ui.hasStarted = false`，但后端仍只封存，文件与模型上下文保持现有语义。
- 导出代码不修改，只用回归测试锁定现有整理/导出调用链。

## TDD

先增加失败测试：

- Store：目录与 assets 真删除、历史消失、下一题新建；running/dirty/pending/exported/discarded 拒绝；删除/恢复失败不伪造成功。
- App：确认门禁、成功后上下文清理、截图缓冲和 Mobile 不变、Store 失败时运行时零副作用。
- Frontend：永久删除警告与成功文案；清空和结束均清实时历史；导出仍调用原 API，未被清空逻辑替代。

再实施最小修改并运行：聚焦测试、`go test ./... -count=1`、`go vet ./...`、`go build ./...`、全部前端回归、Vite build 和 `git diff --check`。

## 回滚

恢复 `DiscardQuestionBankSegment → DiscardCurrent`、旧确认文案和成功提示；删除 `DeleteCurrent` 及新测试；移除 `splitCurrent` 的界面历史清理。已被物理删除的数据无法通过代码回滚恢复，因此必须在人工确认后才执行真实数据验收。
