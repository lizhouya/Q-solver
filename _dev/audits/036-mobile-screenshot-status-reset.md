# Change 036 Mobile 截图状态与清理收敛审计

## 结论

本变更已在既有 Mobile Snapshot/SSE 架构内完成，无新增协议端点、依赖、配置项或网络暴露。Snapshot 只增加截图数量和上限；三个桌面清理成功入口通过可靠 `Bridge.Reset()` 清除同步端业务内容，用户主动 `ClearDisplay` 的持久化 `clearSeq` 语义保持不变。

普通自动验证、Wails 保护构建、候选独立启动、SHA-256 校验和用户双端 GUI 验收均通过。Windows race 检查因测试程序启动错误 `0xc0000139` 为 BLOCKED 并已如实披露。功能与测试提交为 `d90bbe3`；用户已授权配套治理提交和推送 `origin/dev/qsolver-custom`，Change 036 可在治理提交完成后关闭。

## 实现后链路

### 截图状态

1. `app/solve.go` 在截图缓冲成功 append、删除、清空、自动补截图和发送消费后取得权威长度。
2. `app/mobile.go` 的 helper 调用 `Bridge.SetScreenshotState(count, MaxScreenshots)`。
3. Bridge 归一化并幂等保存绝对值，通过全量 Snapshot 广播与重连重放。
4. `pkg/mobile/web/index.html` 在 `screenshotCount > 0` 时显示底部固定 Dock，从 Hints 中提取实际截图、删除、发送快捷键；数量归零后隐藏，避免解题阶段占用答案顶部空间。

### 本地清理

1. `SplitQuestionBankSegment`、`DiscardQuestionBankSegment`、`StartNextExam` 仅在本地事务成功并释放相关锁后调用业务重置 helper。
2. `Bridge.Reset()` 推进业务 `seq`，生成保留 Hints、截图状态、模型和控制序号的 idle Snapshot。
3. Reset 在 Bridge 锁内排空每个 subscriber 的旧积压，并强制放入唯一 idle 屏障，避免旧流式帧恢复已清内容。
4. Mobile 页面对同 `clearSeq` 的新 idle 清空答案、错误、复制目标、滚动任务和浏览器内 `sessionHistory`，但保留连接、截图卡和快捷键。
5. `StartNextExam` 额外把待发送截图数量归零；结束/清空当前段保留尚未发送截图。

## 协议与序号审计

- 新字段仅为 `screenshotCount` 和 `screenshotLimit`，旧页面可忽略未知字段。
- 不传输截图、Base64、文件路径、Token、ComboID、VK 或完整配置。
- Reset 不推进 `clearSeq`，不轮换 Token，不改变固定链接。
- 页面使用 `clearBoundaryPending` 区分新 `sessionId/clearSeq` 的清屏 idle 与同边界的本地 Reset idle。
- 边界 idle 不占用新业务序号空间，保证持久清屏后低 `seq` 的新题仍可渲染。
- `lastBoundaryIdleSeq` 防止重复边界屏障反复清理同一浏览器状态。

## 锁与并发审计

- App 读取 `screenshotBuffer` 时使用 `conversationMu`；创建 Bridge 时不持有 `mobileMu` 再反向获取 `conversationMu`。
- Bridge 的状态更新、seq 推进、队列排空和屏障写入均在 `b.mu` 下原子完成。
- subscriber channel 容量为 64；Reset 先排空，因此同步写入屏障不会阻塞。
- 本地清理失败、未确认或不可用时不调用 Reset，测试验证 Snapshot 完全不变。

## 数据与安全边界

- Mobile 浏览器历史继续只在内存中维护，不引入 `localStorage`、`sessionStorage` 或 IndexedDB。
- 不改题库持久化 schema、模型上下文、导出数据、用户配置或 Mobile 专用状态。
- 不新增依赖，不升级 Wails 或 Go 模块，不生成 bindings。
- 构建前后 `go.mod`、`go.sum`、`frontend/dist/**`、`build/bin/releases/**` 内容哈希不变；`frontend/wailsjs/runtime` 的较新 CLI 生成内容已定向恢复为 HEAD 语义且不在 Git 差异中。Windows checkout 将两个文件规范为 CRLF，因此 WailsJS 工作树原始聚合字节哈希变化，但未形成提交内容。Git index 仅刷新 stat cache，最终暂存区为空。
- `build/bin/**` 保持 ignored，候选不纳入 Git。

## 测试证据

- Bridge：截图状态幂等、归一化、重放、Reset/ClearDisplay 保留及满队列唯一 idle 屏障。
- App：删除/清截图数量同步、单独清图保留答案、三个清理入口成功/拒绝矩阵、固定运行态保持。
- DOM：底部固定 Dock 的 0 张隐藏、1～2 张继续截图、3 张已满、归零隐藏与安全留白，以及实际快捷键、同 `clearSeq` idle 清宽窄历史、边界低序号兼容。
- 全量：`go test ./... -count=1`、`go vet ./...`、`go build ./...`、`go mod verify`、15 份桌面前端回归、Mobile DOM、Vite build、Wails 保护构建、独立启动和哈希校验通过。
- race：`pkg/mobile` 与 `app` 均在启动阶段返回 `0xc0000139`，标记 BLOCKED，不冒充 PASS。

## 提交边界审计

### 功能与测试候选范围

- `app/mobile.go`
- `app/solve.go`
- `app/questionbank.go`
- `app/questionbank_next_exam.go`
- `app/mobile_screenshot_reset_test.go`
- `app/questionbank_delete_test.go`
- `app/questionbank_next_exam_test.go`
- `pkg/mobile/state.go`
- `pkg/mobile/bridge.go`
- `pkg/mobile/web/index.html`
- `pkg/mobile/screenshot_state_test.go`
- `pkg/mobile/bridge_server_test.go`
- `pkg/mobile/web_dom_test.mjs`

### 治理候选范围

- `_dev/audits/036-mobile-screenshot-status-reset.md`
- `_dev/changes/036-mobile-screenshot-status-reset/**`
- `_dev/changes/INDEX.md`

### 排除项

- `build/bin/**` 二进制与哈希清单。
- 用户配置、CCR 配置、日志、题库数据、导出目录和任何 secret。
- `go.mod`、`go.sum`、`frontend/wailsjs/**`、`frontend/dist/**`、Release 产物。
- 其他 Change 或无关工作区内容。

当前暂存区为空，未执行 commit 或 push。若后续获批，仍须在用户 GUI 验收后重新核对最终差异，并分别列明功能提交和治理提交的完整文件清单与 commit message。

## 授权边界

用户已批准本地设计、实施、自动验证和候选构建，并在最终 GUI 验收通过后明确授权双提交及推送 `origin/dev/qsolver-custom`。功能与测试提交为 `d90bbe3`；配套治理提交归档本审计。不授权稳定版晋升、Tag 或 Release。
