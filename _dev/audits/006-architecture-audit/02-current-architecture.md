# Change 006 当前架构状态

> **审计日期**：2026-10-13  
> **基线**：`dev/qsolver-custom` / `c3db919`

## Window

- `WindowVisibilityManager` 以 `Visible/Hidden` 两态维护实体窗口可见性，平台操作成功后才更新状态，重复目标状态保持幂等。
- Windows `Show/Hide` 已落到 `platform.ShowWindow(SW_SHOW/SW_HIDE)`。
- Shortcut `toggle_window`、Tray Show/Hide、SecondInstance `app.Show()`、OutputMode 进入 mobile 时的 one-shot auto-hide 均经过该状态机。
- Windows `SetWindowLevel` 仍为空实现，但当前唯一前端调用位于 macOS 截图权限流程；Windows 用户路径不会调用该入口。
- GhostMode、WindowVisible 与 OutputMode 仍是三个正交状态，未发现混用。

## Shortcut

- Windows 默认键与 Settings 展示一致：F8、Ctrl+J、Ctrl+D、F9、Ctrl+Shift+H、F10 及移动/滚动键。
- 空配置/旧配置由 `NewDefaultConfig()` 先建立默认 map，再进行 JSON 解码；已有配置更新通过完整 settings JSON 回写。
- `toggle` 对应 GhostMode，`toggle_window` 对应实体窗口显示/隐藏，delegate 路径独立。
- `Service` 使用 `maps.Copy` 写入 manager map；若未来支持“删除某一 action 的绑定”，旧键可能残留，但当前 UI 只支持录制覆盖，不提供删除绑定动作。

## Mobile Output

- `Bridge` 维护单一全量 `Snapshot`，状态为 idle/waiting/streaming/done/error，慢客户端允许丢中间帧且不阻塞解题。
- 多个 SSE 客户端共享同一只读快照广播；每个客户端使用独立 buffered channel。
- OutputMode 路由只处理五个 `solution-*` 事件；截图采集、`start-solving`、toast 等仍只发桌面，这是 Change 001 已明确的协议边界。
- 手机状态可见性矩阵：

| 桌面/解题阶段 | 手机可见性 | 对应状态 |
|---|---|---|
| 截图采集 | 不可见 | 协议不传截图/题目 |
| Provider 请求即将开始 | 可见 | `solution-stream-start` → waiting |
| Thinking / answer chunk | 可见 | streaming + thinking/content |
| 最终答案 | 可见 | done |
| Provider/空响应/取消错误 | 可见 | error |

- mobile-only 模式下，桌面不渲染流式正文；终态通过 `solution-hidden` 复位 loading，并显示“已转发到手机端”或错误提示。
- `MobileInfo` available 分支已返回真实 URL 与真实运行端口，unavailable 分支返回 `Available=false, Port=0`。
- SSE handler 当前按“`GetState()` → 写出并 Flush 初始快照 → `Subscribe()`”执行，初始快照与订阅之间存在事件丢失窗口。

## Solver

- KeepContext OFF/ON 已完成后端接线；成功轮保存 current user + final assistant，失败轮不写入，System Prompt 唯一并按当前配置更新。
- history 最多保留 10 轮，multipart 图片 base64 原样保留，仅驻留内存。
- 产品解题入口由 `TaskCoordinator` 限制为单一运行任务，但 `Solver` 自身无锁；配置保存可调用 `SetProvider()`/`ClearHistory()`，与运行中的 `Solve()` 存在潜在竞态。
- 修复该竞态需要明确 provider 快照和 history 锁粒度，不能在持锁状态下覆盖长耗时网络流，因此不属于本轮最低风险目标。

## Frontend

- ModelSelect 展示 image/file/audio/video/context 标签，但不按 image 能力过滤模型。
- 应用始终以截图 multipart 请求模型；直接过滤会误伤 capability 数据库未知的新模型或自定义模型，因此不能在没有产品策略时直接实施。
- Settings 已包含 OutputMode 三态并通过 `syncSettings` 保存；Mobile URL、available 与 port 均有可见反馈。
- Solve UI 已处理 loading、thinking、answer、inline error、history 与 mobile-only 的隐藏终态。
- 快捷键提示读取真实 config，缺失时回退到平台默认显示。