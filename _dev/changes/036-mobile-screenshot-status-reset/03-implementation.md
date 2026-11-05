# 实现过程记录 03

## 实现日志

| 时间 | 步骤 | 改动内容 | 涉及文件 | 说明 |
|---|---|---|---|---|
| 2026-11-04 | 事前归档 | 完成请求、现状、设计与审计 | `00/01/02`、审计、INDEX | 用户批准后创建，之后才修改业务源码 |
| 2026-11-04 | TDD 红灯 | 增加 Bridge 截图元数据、App 清理语义和 Mobile DOM 约束测试 | `pkg/mobile/screenshot_state_test.go`、`app/mobile_screenshot_reset_test.go`、既有相关测试 | 旧实现因字段、方法和截图卡缺失按预期失败 |
| 2026-11-04 | Snapshot 与 Bridge | 新增 `screenshotCount` / `screenshotLimit`，实现幂等归一化 setter，并使 Reset 成为可靠 idle 屏障 | `pkg/mobile/state.go`、`pkg/mobile/bridge.go` | Reset 排空每个 subscriber 的旧积压，只保留一个新 idle 快照；不推进 `clearSeq` |
| 2026-11-04 | App 接线 | 截图缓冲成功增删、清空和发送消费后同步权威数量；三个桌面清理入口成功后重置 Mobile 业务显示 | `app/mobile.go`、`app/solve.go`、`app/questionbank.go`、`app/questionbank_next_exam.go` | 失败路径零副作用；单独清截图不删除已显示答案 |
| 2026-11-04 | Mobile 页面 | 增加始终可见的截图状态卡，复用实际快捷键提示；新 idle 清除答案和浏览器内历史 | `pkg/mobile/web/index.html`、`pkg/mobile/web_dom_test.mjs` | 不使用浏览器持久化存储，不传输截图内容 |
| 2026-11-04 | 序号兼容修复 | 区分 `sessionId/clearSeq` 边界 idle 与同边界的本地 Reset idle | `pkg/mobile/web/index.html`、`pkg/mobile/web_dom_test.mjs` | `clearBoundaryPending` 防止持久清屏屏障占用新业务序号空间；`lastBoundaryIdleSeq` 去重重复屏障 |
| 2026-11-04 | 自动验证与候选构建 | 完成 Go、DOM、前端、Vite、Wails 保护构建、独立启动和产物保护验证 | 见 `04-validation.md` | 普通自动验证通过；race 因 Windows 启动错误 `0xc0000139` 标记 BLOCKED |
| 2026-11-05 | 首次 GUI 反馈诊断 | 用户反馈同步端没有截图和快捷键提示；核对当前进程和候选嵌入资源 | 候选进程、`pkg/mobile/web/index.html`、`pkg/mobile/server.go` | 当前进程确为 Change 036 候选，候选包含新版页面；既有浏览器标签页只重连 SSE、不自动替换旧 HTML，需刷新后复验 |
| 2026-11-05 | 底部 Dock 回修 | 用户刷新后确认提示可见，但常驻顶部且答案自动滚底后不可见；改为只在截图数大于 0 时固定显示在底部 | `pkg/mobile/web/index.html`、`pkg/mobile/web_dom_test.mjs` | 1～2 张显示“继续截图”，3 张显示“已满 3 张”；发送或清空归零后隐藏，并撤销答案底部安全留白 |
| 2026-11-05 | 回修验证与候选替换 | 完成聚焦与全量验证、Wails 保护构建、独立启动和哈希复核 | 见 `04-validation.md` | 清理本机较新 Wails CLI 生成的两个 runtime 行尾噪音；旧顶部卡片候选归档为 transient，新候选进入 `testing/current` |

## 与计划的偏差

- 计划中的 `Bridge.Reset()` 原拟复用普通非阻塞广播。实现审查发现 subscriber 队列满时 idle 可能被丢弃，因此收紧为“排空旧积压并强制放入唯一 idle 屏障”，并新增满队列回归测试。
- Mobile 页面需要兼容持久化 `clearSeq` 增长后业务 `seq` 从低值重新开始。新增边界状态区分，避免 clear 屏障阻止下一题渲染；不改变后端协议字段和 `clearSeq` 语义。
- 允许文件范围未扩展。`pkg/mobile/bridge_server_test.go`、`app/questionbank_delete_test.go`、`app/questionbank_next_exam_test.go` 均属于方案中“对应测试”的范围。

## 关键决策纪要

- “结束本段”“清空本段”“开始下一套”只在桌面操作成功后清空同步端答案和浏览器内历史。
- 单独清除待发送截图只把 `screenshotCount` 归零，不删除同步端已经显示的答案。
- `StartNextExam` 成功后截图数量归零；结束/清空当前段保留仍待发送的截图数量。
- Reset 保留 `clearSeq`、固定链接、Hints、截图状态、模型名和控制序号；不轮换 Token。
- Snapshot 只传数量和上限，不传图片、Base64、文件路径、Token、ComboID 或 VK。
- 截图 Dock 只从 Hints 中按固定顺序提取“截图”“删除截图”“发送解题”，并使用 `textContent` 渲染；仅在 `screenshotCount > 0` 时固定显示在同步页底部。
- Dock 在 1～2 张时显示继续截图，在 3 张时显示已满；数量归零后隐藏，因此解题等待、流式和完成阶段不占据答案区域。

## 遗留未完成项

- `go test -race` 在当前 Windows 环境启动阶段返回 `0xc0000139`，环境阻塞未解除并已如实披露。
- 功能与测试已提交为 `d90bbe3`；治理记录由配套治理提交归档。
- 用户已明确授权双提交和推送 `origin/dev/qsolver-custom`；不含稳定版晋升、Tag 或 Release。
