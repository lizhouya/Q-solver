# 现状快照 — 01

## 1. 涉及文件清单

| 文件 | 当前职责 | 当前关键行为 |
|---|---|---|
| `pkg/mobile/state.go` | Mobile Snapshot 协议 | 有答案、错误、快捷键、控制字段，无截图数量 |
| `pkg/mobile/bridge.go` | Snapshot 权威状态与 SSE 广播 | `SetHints` 可重放；`Reset` 发布 idle 但页面当前忽略 idle |
| `pkg/mobile/web/index.html` | 同步端 UI 与浏览器内历史 | 完整快捷键折叠显示；`sessionHistory` 只随身份/clearSeq 清理 |
| `app/mobile.go` | Bridge 装配与输出路由 | 初始化 model/hints；没有截图状态同步 helper |
| `app/solve.go` | 待发送截图缓冲 | append/remove/clear/send 均只通知桌面 Wails |
| `app/questionbank.go` | 结束本段、清空本段 | 成功后清桌面历史；Mobile 保留旧答案 |
| `app/questionbank_next_exam.go` | 开始下一套 | 清桌面历史、截图、上下文；Mobile 保留旧显示 |
| `app/questionbank_delete_test.go` | 清空本段回归 | 明确断言 Mobile 当前显示不变 |

## 2. 关键调用链现状

### 截图状态

`TriggerScreenshot/RemoveScreenshot/ClearScreenshots/triggerSend` 修改 `a.screenshotBuffer` 并发送桌面事件。Mobile Bridge 不读取该缓冲，也没有对应 Snapshot 字段。

### 快捷键提示

`mobileShortcutHints` 已按顺序输出“截图、发送解题、删除截图”等显示名，`Bridge.SetHints` 写入 Snapshot。手机页把全部提示放在默认折叠的“答题端快捷键”卡片中，没有突出截图工作流。

### 本地清理

- `SplitQuestionBankSegment`：成功封存后前端调用 `solution.resetLiveHistory()`。
- `DiscardQuestionBankSegment`：成功物理删除并清模型上下文，前端重置桌面历史；现有测试要求 Mobile 保留。
- `StartNextExam`：成功封存、清截图与上下文，前端重置桌面历史；不触发 Bridge Reset。

### 同步端历史

`sessionHistory` 在 done 快照时追加。`applyBusinessSnapshotIfNew` 对 idle 直接返回 false，因此 `Bridge.Reset()` 的 idle 帧不会清除旧答案或历史。

## 3. 配置与安全边界

- 不读取或修改用户普通配置和 Mobile Token。
- 快捷键 Snapshot 只含 Label/KeyName，不含 ComboID/VK。
- Mobile 固定链接状态和 `clearSeq` 由专用状态管理器持久化。

## 4. 依赖与构建

- 无需新增依赖。
- Mobile 页面为 `pkg/mobile/web/index.html` 单文件嵌入资源。
- 现有 Go 测试包含 Bridge/SSE/DOM 集成及 App 清理行为。

## 5. 已知问题

- Change 034 的“Mobile 保留内容”测试语义与本次需求冲突，必须明确反转，而不是绕过测试。
- `ClearDisplay` 是用户主动清屏的持久化边界；本地清理解题界面不应滥用该命令或轮换 Token。
- race 测试在当前 Windows 环境历史上可能以 `0xc0000139` 启动失败，若复现须标记 BLOCKED。
