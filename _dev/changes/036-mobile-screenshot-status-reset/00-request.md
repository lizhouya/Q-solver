# 变更请求 — 00

| 字段 | 值 |
|---|---|
| 变更编号 | `036-mobile-screenshot-status-reset` |
| 提出人 | 用户 / Q-Solver 维护者 |
| 提出日期 | 2026-11-04 |
| 关联 Issue | 无 |
| 状态 | `方案已批准 / 实施中` |

## 1. 背景与问题

用户提出两项关联需求：

1. 手机同步端需要显示答题电脑当前待发送截图数量，并突出显示截图、删除截图、发送解题三个实际快捷键。
2. 桌面端成功执行本地清理后，同步端仍保留旧题目和浏览器内历史，需要同步清空且不能残留。

只读核查确认：Mobile Snapshot 已携带全部快捷键提示，但没有截图数量；手机页只在折叠卡片展示完整快捷键。手机页的 `sessionHistory` 仅在 `sessionId` 或持久化 `clearSeq` 变化时清空。桌面“结束本段”“清空本段”“开始下一套”成功后只重置桌面历史，没有发布 Mobile 空闲重置帧，Change 034 测试还明确要求“清空本地分段不修改 Mobile 当前显示”。

## 2. 目标

- Mobile Snapshot 安全同步待发送截图数量和上限，不同步图片内容。
- 同步端始终显示紧凑的截图状态与截图/删除/发送实际快捷键。
- 截图新增、删除、清空、发送消费和开始下一套时数量实时对齐。
- “结束本段”“清空本段”“开始下一套”成功后清空同步端答案和浏览器内题目历史。
- 清理失败、未确认或被运行任务拒绝时保持同步端零副作用。
- 清理同步端业务内容时保留连接、固定链接、模型、快捷键提示和当前待发送截图数量。

## 3. 验收标准

- [ ] Snapshot 只增加 `screenshotCount` / `screenshotLimit` 数值字段，不携带图片、路径、Token 或键码。
- [ ] Bridge 截图状态更新幂等、可重放，重连首帧为权威值。
- [ ] 截图、删除、清空、发送消费和开始下一套分别同步正确数量。
- [ ] 同步端紧凑卡片显示 `待发送截图 N/3` 及实际截图、删除、发送快捷键。
- [ ] 自定义快捷键更新后紧凑提示同步刷新。
- [ ] 成功“结束本段”“清空本段”“开始下一套”后同步端答案和历史记录清空。
- [ ] 三个操作失败时 Mobile Snapshot 和浏览器历史不变。
- [ ] Mobile Reset 不推进固定链接 `clearSeq`，不修改 Token 或连接地址。
- [ ] 聚焦测试、全量 Go test/vet/build、全部前端/Mobile 回归、Vite 和 Wails 构建结果如实记录。
- [ ] Windows GUI 与手机/浏览器人工验收明确记录为 PASS、FAIL 或 NEEDS_USER。

## 4. 非目标 / 明确不做

- 不向同步端传输截图内容或缩略图。
- 不修改 `MaxScreenshots = 3`、模型请求、Prompt、题库 schema 或导出格式。
- 不修改 Mobile Token、固定链接、端口或清屏确认流程。
- 不让单独“清除待发送截图”删除同步端已显示答案；该操作只将截图数量归零。
- 不新增依赖，不升级 Go、Wails、Vue 或 Pinia。

## 5. 影响面预估

- 业务文件：`pkg/mobile/state.go`、`pkg/mobile/bridge.go`、`pkg/mobile/web/index.html`、`app/mobile.go`、`app/solve.go`、`app/questionbank.go`、`app/questionbank_next_exam.go`。
- 测试：Mobile Bridge/SSE/DOM、App 截图状态与三个清理入口。
- 治理：本 Change 七份档案、审计、INDEX。
- 风险：idle 快照误清提示、失败路径误清远端、截图状态与缓冲区竞态、旧 Snapshot 兼容。

## 6. 附带约束

- 本次批准只覆盖本地实现与验证，不包含 commit、push、稳定版晋升、Tag 或 Release。
- Mobile 不可用时桌面清理仍应成功；Mobile 重置是无错误的内存镜像收敛。
- 所有新增字段保持 JSON 向后兼容；旧页面忽略未知字段，旧快照缺省为 `0/0`。
