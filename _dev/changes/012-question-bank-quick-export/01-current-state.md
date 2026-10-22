# 现状快照 — 01

## 1. 涉及文件清单

| 文件 | 当前职责摘要 | 关键行为 |
|---|---|---|
| `pkg/questionbank/store.go` | Store、场次状态、增量记录和恢复 | 只有 active Session 才保存记录；固定追加 `求职笔试/会话记录` |
| `pkg/questionbank/export.go` | Markdown 输出 | 固定追加 `求职笔试/题库/按单位或题型/名称/时间_名称_短ID` |
| `pkg/questionbank/organize.go` | 按提交调用模型整理 | 支持失败重试和原答案保留；Question 没有独立 Category |
| `app/questionbank.go` | 偏好、Wails 接口、异步整理 | 偏好 v1 只存 Root；导出要求先 End；整理前后各导出一次 |
| `app/questionbank_capture.go` | 解题回调前归档 | Store 不处于 active 时不记录 |
| `frontend/src/components/QuestionBankPanel.vue` | 显式场次管理 | 路径、开始、继续、结束、整理集中在独立面板 |
| `frontend/src/components/ActionBar.vue` | 答案复制和导出 | 导出按钮实际为答案图片导出 |
| `frontend/src/components/SettingsModal.vue` | 主设置 | 没有题库标签 |
| `app/shortcut.go` | 滚动接口 | `ScrollContent` 只发送桌面事件 |
| `app/mobile.go` | outputMode 路由 | 只路由 solution 事件 |
| `pkg/mobile/bridge.go` | Mobile 快照 | 没有快捷键元数据或控制序号 |
| `pkg/mobile/web/index.html` | 手机 UI | 不显示快捷键，不处理远端滚动 |
| `pkg/shortcut/shortcut_windows.go` | Windows hook | callback 直接执行 `uintptr -> unsafe.Pointer` |
| `pkg/platform/platform_windows.go` | Win32 封装 | 已定义 hook 结构并加载系统 DLL |

## 2. 关键代码现状

- `Store.Start` 创建显式 active Session；没有 active Session 时 `Store.Begin` 返回空 ID。
- `Export` 只接受 ended Session，并使用 GroupBy 与 Name 构建多层目录。
- `OrganizeQuestionBank` 先导出原稿，再异步整理，结束后再次导出，产生两个版本目录。
- ActionBar 的“导出”调用 `solution.exportImage(...)`；历史项另有图片导出入口。
- `scroll_up` / `scroll_down` 默认快捷键已存在，Windows 为 Alt+PgUp / Alt+PgDn。
- Mobile 首帧和实时帧都是完整 Snapshot；一次性控制若无独立序号会在重连后重放。

## 3. 相关配置

- 主 Config 包含 OutputMode、Shortcuts、模型和截图设置，不包含题库偏好。
- 题库偏好位于用户配置目录的 `Q-Solver/questionbank-settings.json`，版本 1 仅存 Root。
- 本次不读取、打印或修改实际 API Key；文档状态只使用 SET / NOT SET。

## 4. 依赖与构建现状

- Go 模块使用 Wails v2.11.0；本机较高版本 Wails CLI 可能改写 bindings 或 go.mod 格式。
- 前端为 Vue 3 + Pinia + Vite，只有 dev/build/preview scripts。
- 不需要新增 npm 或 Go 依赖。

## 5. 运行行为现状

- desktop、mobile、dual 只影响答案输出事件；手机服务不可用时 mobile 降级桌面。
- 题库捕获发生在输出路由之前，因此手机独占答案也可归档。
- 顶部栏已有题库状态和部分快捷键提示，但没有手机滚动按钮。

## 6. 已知现状问题

- `go vet ./...` 报告 Windows hook 两处 `possible misuse of unsafe.Pointer`。
- race 测试二进制在本机启动阶段以 `0xc0000139` 退出；本 Change 不处理。
- 工作区已有 Change 010/011 与规则文档的未提交修改，暂存区为空。

架构依据详见 [`../../audits/012-question-bank-quick-export.md`](../../audits/012-question-bank-quick-export.md)。


## 7. 2026-10-20 Mobile 回修前现状

- 顶部上下箭头和全局 `scroll_up` / `scroll_down` 快捷键已通过 `App.ScrollContent` 按已保存的 `outputMode` 路由到 desktop/mobile/dual；鼠标滚轮尚未接入。
- Mobile 服务在 App 启动时无条件尝试初始化，不是用户临时选中 Mobile 卡片后才启动。
- `SettingsModal` 只在组件首次挂载时读取一次连接信息；读取失败和后端 `available=false` 被合并成同一条“端口冲突或被占用”提示，状态可能陈旧且不可诊断。
- 前端卡片修改的是 `tempSettings.outputMode`，只有保存后才成为实际路由；当前 UI 没有清楚区分草稿和已保存模式。
- `GetMobileOutputInfo` 只返回 `available/url/port`，服务启动失败后没有重试 API。
- `setupMobile` 每次调用都会新建 Bridge 并注册 config subscriber，不能直接作为重试入口。
- 手机端实时 `message` 先执行控制、再无条件执行完整 `applySnapshot`；控制帧与业务快照拥有相同 `seq`，仍会重建答案和历史 DOM，`done` 状态还会把 `selectedSeq` 重置为最新。
- `applyOutputModeWindowPolicy` 已有 Mobile 不可用时不隐藏桌面的保护；重试成功不得补偿触发隐藏。

## 2026-10-22 序号与终态排查

Bridge.Publish 的 thinking/chunk/solution/error 只递增 b.seq，没有赋值给 b.state.Seq；开始帧与后续帧共享旧序号。页面 applyBusinessSnapshotIfNew 拒绝不增序号，刷新则重置浏览器基线并应用最新全量快照。旧 DOM 测试手造递增序号，未覆盖真实 Bridge 输出。另：满队列允许丢普通帧，最终 done/error 也可能被丢弃，之后只有心跳而无新答案。
