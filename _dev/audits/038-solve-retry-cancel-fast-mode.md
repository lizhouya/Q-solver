# Change 038 解题重试、中断与快速模式架构审计

## 审计结论

三项需求均可在现有架构上实现，但不能作为三个互不关联的按钮补丁。推荐引入统一、后端权威的解题尝试生命周期：失败/取消保存单槽 retry snapshot；重试原位复用轮次和题库 Record；中断通过既有 context 传播；快速模式只修改请求 Config 副本。

2026-11-11 已按批准的详细 TDD 计划完成业务实现、普通自动门禁、固定 Wails v2.11.0 bindings 和保护候选验证。实现遵守审计不变量：Retry API 不接收图片或配置，题库原位重启，中断等待 in-flight 完成，fast 不写持久配置，Mobile 保持只读。race 因 Windows `0xc0000139` 为 `BLOCKED` 并已披露；用户确认当前测试版本没有问题，功能提交为 `5564a45`。

## 当前调用链

```text
快捷键 FIFO / Wails API
  → App.triggerSend
  → 复制并消费 screenshotBuffer
  → TaskCoordinator.StartTask(context)
  → App.solveInternal
  → questionSolveCapture.BeginWithContext
  → Solver.Solve
  → Provider.GenerateContentStream(ctx)
  → DesktopSolveEvent + Mobile Bridge + QuestionBank Finish
```

## 可复用能力

1. `TaskCoordinator` 已持有 CancelFunc 和 in-flight Map。
2. 四类 Provider 均使用调用方 context；Custom JSON 使用 `http.NewRequestWithContext`。
3. Solver 失败/取消不提交 KeepContext，成功后才追加完整 user/assistant 对。
4. Desktop typed event 已有 epoch/sessionId/recordId 和迟到事件隔离。
5. 题库在请求前保存原图，Record 已区分 running/completed/failed/cancelled/interrupted。
6. 快捷键已通过单 worker FIFO，配置 Map 可扩展 action。
7. 已有 `gen-speed` 场景，无需新建快速 Prompt 或持久化第二场景。
8. Mobile 已有安全 Hints、结构化诊断、Snapshot 重放和显示代际。

## 现有缺口

| 缺口 | 后果 | 设计控制 |
|---|---|---|
| 发送即清截图缓冲 | 失败后只能重新截图 | App 单槽 retry snapshot 深拷贝原图 |
| 前端持图但后端无授权目标 | 回传图片会扩大信任面 | Retry API 只收目标 ID，后端验证并取图 |
| terminal Record 不可重启 | 重试会新增重复题目 | Store 原位 RestartRecord |
| Cancel API 无 UI | 用户无法主动停止慢请求 | Desktop 按钮 + cancel_solve 快捷键 |
| cancel 后 running=false、in-flight=true | 可能立即启动重叠请求 | 所有启动入口检查 HasInFlightTask |
| cancellation 走 error | Desktop/Mobile 语义错误 | 新增 solution-cancelled / StatusCancelled |
| 只有持久场景发送 | 限时题需先改设置 | 请求级 SolveModeFast 覆盖 gen-speed |
| 错误卡提示按截图键重试 | 实际会重新截图 | 改为显式“重试本题”按钮 |

## 关键不变量

- 一个活动 attempt；一个同源 retry snapshot；最多三张图。
- retry snapshot 不复制 Connection Profile、Key、Header 或 Token。
- Retry API 不接收 Base64、路径、Prompt 或 Config。
- 重试不新增题库 Record，不重复写 KeepContext。
- cancellation 必须等旧 goroutine Complete 后才允许新请求。
- fast 不写 Config、不通知 subscriber、不切换活动 Profile。
- Mobile 保持只读，不新增网络控制入口。
- 不自动重试，不隐藏潜在费用。

## 主要风险

### 1. 旧任务迟到写入

必须同时校验 taskID、desktop target、conversation epoch 和 Mobile display epoch；仅依赖当前 task 指针不足。

### 2. 题库落盘原子性

RestartRecord 必须 clone Session → 修改候选 → 完整保存 → 交换；保存失败不能把内存 Record 留在 running。

### 3. 快捷键迁移

新增默认组合前检查用户显式配置和现有 ComboID。冲突时新增 action 未绑定，不能夺取旧 action。

### 4. 快速模式承诺

`gen-speed` 可减少输出长度，但无法控制网络和上游排队。产品文案只能称“快速解题”，不能承诺固定秒数。

### 5. 取消与部分内容

取消后已流出的正文可显示，但不得作为 completed、答案摘要、KeepContext 或可直接导出的完整答案。

## 建议文件边界

业务范围以 `02-plan.md` 第 3 节为准。不得借本 Change 修改 Connection Profile、题库导出格式、依赖、Wails runtime、正式用户配置或 Release 资产。

## 验证底线

- fake Provider / `httptest` 覆盖失败、取消、迟到、503、timeout 和流中断。
- 题库 on/off、KeepContext on/off、desktop/mobile/dual 均有证据。
- Change 035 FIFO/三图、Change 036 Mobile Reset、Change 037 Connection Profiles 回归。
- 全量 Go、前端回归、Vite、Wails v2.11.0 保护构建和独立启动。
- 真实外部 API 只在用户单独授权后执行。

## 实施后边界审计

- **数据边界**：公开 `RetrySolve` 仅接收 `sessionId` / `recordId`；截图只存在后端 attempt 内存或既有题库资产；未向 Desktop/Mobile Snapshot 暴露图片、路径、Record 控制 token 或连接配置。
- **并发边界**：taskID、desktop target、conversation epoch、Mobile display epoch 分层校验；旧 task 不能 finish/cancel 新 attempt；draining 阻止旧 goroutine 退出前的新请求。
- **持久化边界**：QuestionBank RestartRecord 先 clone/save 后交换；fast 不写 Config；新增快捷键只按冲突安全迁移；未修改 Connection Profile schema。
- **Mobile 边界**：HTML/Snapshot 仅增加 `cancelled`、`solveMode` 和文字提示；无 RetrySolve、CancelSolve、POST、WebSocket 或新 URL 参数。
- **生成边界**：固定 v2.11.0 只更新三个 Go bindings；`go.mod`、`go.sum` 和 Wails runtime 哈希不变。
- **产物边界**：候选保留在 ignored 的 `testing/current`；根稳定版、Release、Git index 未改变。
- **secret 边界**：测试使用 fake Provider、`httptest` 和隔离 Profile；未调用真实收费 API，文档未记录 Key、Token、完整 URL、Header 或截图内容。

## 建议提交白名单（未授权）

1. 功能提交：本 Change 的 `app/`、`pkg/`、`frontend/`、`README.md`、`README_EN.md` 业务/测试/bindings 文件；明确排除 `build/bin/**`、`frontend/dist/**`、隔离目录和任何配置。
2. 治理提交：本审计、`_dev/changes/038-solve-retry-cancel-fast-mode/**`、`_dev/changes/INDEX.md`。
3. 在提交前必须重新执行 Git 预检并列出精确路径；当前不暂存、不提交。

## 当前授权边界

用户已批准 Change 038 设计、书面规格和详细实施计划，并于 2026-11-11 确认当前测试版本没有问题，明确授权本次提交和推送。功能提交 `5564a45` 已创建；治理记录按双提交规则随本文提交，并推送到 `origin/dev/qsolver-custom`。当前分支即目标开发分支，没有独立分支可合并，未创建 merge commit，也未修改 `main`。稳定版晋升、Tag、Release 和真实收费 API 不在本次授权中，均未执行。
