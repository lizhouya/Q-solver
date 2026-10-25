# Change 019 实现后审计

日期：2026-10-23
状态：业务范围审计与普通自动验证完成；GUI/真实模型 NEEDS_USER；race BLOCKED；验证程序归档复制 BLOCKED；未提交、未推送、未发布、未关闭。

## 结论

Change 019 按已批准设计完成截图折叠与多图多题答案汇总契约。实现复用既有图片缓存和单次 multipart 模型请求，没有扩张题库结构、Wails API、Mobile 协议或 Provider 链路。普通自动验证和构建通过；人工 GUI、真实模型效果、race 与统一验证归档仍需分别处理，不能将其标为已完成。

## 范围复核

### 截图显示状态

- 展开集合只存在于 `RoundScreenshots.vue`，不进入 Pinia、题库、API 或持久化。
- `scopeKey` 仅用于 Session/epoch/Record 显示隔离。
- 组件没有直接写 `image.data`、`image.loading`、`image.error`、`image.requestVersion` 或 `round.userScreenshot`。
- 历史图片继续由 `solution.loadRoundImage` 管理缓存、请求版本和迟到响应；Change 019 没有新增 Store 实现。
- 加载中收起只改变本地 Set；响应完成后不会自动展开。

### 模型请求与 Prompt

- 多题规则只加入统一 `GetSystemBehaviorPrompt()`。
- 未修改 `buildSystemPrompt`、`Solver.Solve`、Provider 参数、流式事件或图片 part 构造。
- fake provider 测试确认三图仍为一次调用、一个当前 system、三个按顺序排列的 image part。
- 完整模型回答原样进入 solution 事件和 KeepContext assistant 历史，没有第二次模型调用或本地二次总结。
- Prompt 明确限制为当前 user 消息中的待解原题，历史题不得重新汇总，变式题不得混入原题汇总。

### 场景边界

- OCR 和纯翻译不强制逐题解答或生成汇总。
- 秒杀或仅答案场景保持简洁。
- 举一反三场景的变式题答案与截图原题汇总分离。
- 无法可靠辨认或跨图冲突时标记“待核对”，不得猜测。

### 依赖与生成文件

构建后保护哈希检查全部通过：

- `frontend/wailsjs/go/app/App.d.ts`
- `frontend/wailsjs/go/app/App.js`
- `frontend/wailsjs/go/models.ts`
- `go.mod`
- `go.sum`
- `build/bin/Q-Solver.exe`

因此本 Change 没有新增 bindings、依赖或正式程序变化。

## 验证复核

PASS：

- 截图折叠专项 Node 回归。
- 题库续接、历史、设置、Mobile Store 与 Mobile DOM 回归。
- Frontend Vite build。
- `go test ./... -count=1`。
- `go vet ./...`。
- `go build ./...`。
- `git diff --check`（无空白错误，只有既有行尾提示）。
- Wails v2.11.0 独立 Windows 构建。

BLOCKED：

- `go test -race ./pkg/domain ./pkg/solution -count=1`：Windows 启动阶段 `0xc0000139`。
- 将新验证 EXE 复制到 `build/bin/archive/validation/` 并更新 `SHA256SUMS`：组合操作被 auto mode classifier 拒绝，未绕过。

NEEDS_USER：

- Windows GUI 截图折叠、缓存、重试、键盘和状态隔离验收。
- 真实模型的单题、多题、重叠、跨图、混合题型、模糊题和特殊 Scene 效果；未经额外授权未执行。

## 验证产物

- 当前可运行文件：`build/bin/Q-Solver-change019-answer-summary-preview-validation.exe`
- SHA-256：`cd4b354ef37613029d2bb0e39b9d62fd924480aefa0eafbf762ca60135d19cf7`
- 大小：`23,366,656` 字节
- 统一归档目标当前不存在，`SHA256SUMS` 尚无 Change 019 条目。

## 数据与安全边界

- 未调用真实模型、未上传真实截图、未消费真实 API 额度。
- 未读取、修改或输出 API Key、Mobile Token 或私人快捷键配置。
- 未删除、覆盖或迁移题库记录、截图、旧导出或旧验证程序。
- 未覆盖正式 `Q-Solver.exe`，未停止用户进程。
- 未执行 commit、push、Tag 或 Release。

## 关闭条件

Change 019 目前不得关闭。至少仍需用户完成 Windows GUI 验收；真实模型效果需要单独授权后验证；race 与统一归档阻塞必须继续如实记录，除非相应环境或权限问题被实际解决。
