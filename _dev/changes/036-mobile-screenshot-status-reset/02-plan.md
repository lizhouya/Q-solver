# 实施方案 — 02

## 1. 方案概述

在 Mobile Snapshot 增加安全的截图数量/上限元数据，由 Bridge 以幂等绝对值广播；App 在截图缓冲每次成功变化后同步权威数量。手机页新增始终可见的紧凑截图状态卡，复用 Snapshot.hints 中的实际截图、删除、发送快捷键。三个桌面清理入口成功后调用 Bridge.Reset 发布递增 seq 的 idle 快照；手机页把新的 idle 视为业务重置，清空答案和浏览器内历史但保留连接、提示和截图状态。

## 2. 分步实施计划

| 步骤 | 操作 | 文件 | 完成标志 |
|---|---|---|---|
| 1 | 编写 Bridge 截图元数据与 Reset 红灯测试 | `pkg/mobile/*_test.go` | 旧协议失败 |
| 2 | 扩展 Snapshot 与 Bridge | `state.go`、`bridge.go` | 聚焦测试通过 |
| 3 | 编写 App 截图/清理入口红灯测试 | `app/*_test.go` | 旧行为失败 |
| 4 | 接线 App 状态同步与成功清理 Reset | `mobile.go`、`solve.go`、`questionbank*.go` | App 聚焦测试通过 |
| 5 | 编写并实现 Mobile DOM 回归 | `web/index.html`、DOM 测试 | 数量提示和 idle 清历史通过 |
| 6 | 全量验证、候选构建和人工验收 | 测试与治理记录 | 结果真实归档 |

## 3. 允许修改的业务文件

- `pkg/mobile/state.go`
- `pkg/mobile/bridge.go`
- `pkg/mobile/web/index.html`
- 对应 `pkg/mobile` 测试
- `app/mobile.go`
- `app/solve.go`
- `app/questionbank.go`
- `app/questionbank_next_exam.go`
- 对应 `app` 测试

若必须超出此范围，先在 `03-implementation.md` 记录原因和新增边界。

## 4. 关键语义

### 截图元数据

- Snapshot：`screenshotCount`、`screenshotLimit`。
- Bridge 保存最后绝对值；相同值不广播。
- count/limit 负值归零，count 不超过 limit（limit 为 0 时 count 归零）。
- `baseSnapshot`、Reset、ClearDisplay 均保留当前截图元数据。

### 本地业务重置

- 新增 App helper，仅在 Bridge 存在时调用 `Bridge.Reset()`。
- 成功结束/清空/开始下一套后调用；失败路径不调用。
- Reset 只推进业务 `seq` 并发布 idle，不改 `clearSeq`、Token、Hints 或截图状态。
- 手机页收到比 `lastBusinessSeq` 更新的 idle 后清空业务内容和 `sessionHistory`。

### UI

- 紧凑卡片始终展示 `待发送截图 N/Limit`。
- 从 hints 中按标签提取“截图”“删除截图”“发送解题”；缺失绑定时只显示已有项。
- 完整快捷键折叠卡继续保留。

## 5. 测试计划

- Bridge：SetScreenshotState 幂等、克隆/重连、Reset/ClearDisplay 保留、无敏感字段。
- App：截图 1→2→3、删除、Clear、发送消费、自动捕获被运行任务拒绝、开始下一套归零。
- 清理：三个成功入口 Reset；未确认、Store 失败、in-flight、导出阻塞不 Reset。
- DOM：紧凑卡渲染实际快捷键和数量；idle 清答案/history；保留 hints/capture card；旧 seq idle 不回滚。
- 全量：Go test/vet/build/mod verify、全部前端回归、Vite、Wails 保护构建、diff-check。

## 6. 回滚预案

定向移除 Snapshot 两字段、Bridge setter、App 接线和 Mobile UI；恢复 Change 034 的 Mobile 保留语义。禁止使用 `git reset --hard` 清理用户工作区。

## 6.1 2026-11-05 用户验收后 UI 修订

初版把截图状态卡放在答案顶部。用户刷新确认功能可见后指出：答案自动滚到底部时顶部提示离开视野，且解题阶段不应常驻。经用户明确批准，UI 收敛为与桌面 `ScreenshotDock` 一致的临时底部 Dock：仅 `screenshotCount > 0` 时固定显示；1～2 张提示继续截图，3 张提示已满；发送或清空使数量归零后隐藏。协议、Bridge、App 接线和清理语义不变。

## 7. 风险与对策

| 风险 | 对策 |
|---|---|
| idle 帧把快捷键提示一起清掉 | 业务重置函数只清答案/history，提示与截图卡独立渲染 |
| 本地失败却误清 Mobile | Reset 只放在成功提交和解锁之后；失败测试验证零副作用 |
| 截图计数广播乱序 | 使用 `conversationMu` 下的权威缓冲长度和 Bridge 绝对值 setter |
| 与用户主动清屏混淆 | Reset 不推进持久化 clearSeq；ClearDisplay 语义不变 |
| 旧页面兼容 | 新 JSON 字段为可选增量；旧页面忽略未知字段 |
