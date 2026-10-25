# 实现过程记录 03

状态：实现与自动验证已完成；人工真机验收待用户执行。

## 实现内容

### Mobile 专用状态

- 新增 `pkg/config/mobile_state.go`，Token、固定模式、`clearSeq` 与 generation 不进入普通 `config.Config`。
- 默认启用固定链接；固定模式将 Token 以现有 AES-GCM 能力加密落盘，临时模式磁盘 Token 为空并在每次进程启动时重新生成。
- 使用 `mobile-state.0.enc` / `mobile-state.1.enc` 双槽和不参与启动选择的 `mobile-state.pending.enc`。pending 完成 Write、Sync、Close 与解密回读后，通过同目录 Rename 发布到非当前槽；Rename 是提交线性化点。
- 单槽损坏可回退，两个现有槽均无效、代次冲突或未知 schema 时 fail closed。计数溢出、随机源和持久化失败不改变 current。
- Reset 和固定模式切换会有限重试随机值，确保新 Token 不等于当前 Token；无法生成不同值时保持原状态。

### Bridge、Server 与浏览器协议

- Bridge 为每个进程生成 `sessionId`，所有快照携带 `sessionId + clearSeq`；`PublishIfCurrent` 在同一写锁中比较显示代际并发布。
- `ClearDisplay` 清空答案、分析、错误和一次性控制，保留模型、快捷键提示与 `controlSeq` 基线；订阅队列被排空后只保留唯一清屏屏障。
- Server 改为外部注入 Token，`/` 与 `/sse` 使用相同鉴权，`/health` 不暴露秘密；页面响应使用 `no-store` 与 `no-referrer`。
- Token 轮换通过认证代次和 SSE lease 主动撤销旧连接；鉴权和 lease 登记位于同一临界区，snapshot、message 与 heartbeat 写入前检查撤销状态。
- Mobile 页面统一验证协议身份。更高 `clearSeq` 会清理 DOM、宽/窄历史、历史选择、视觉状态和延迟滚动；已淘汰 `sessionId` 不能把页面切回旧进程。
- Markdown 外链只允许 `http`、`https` 和 `mailto`，新窗口链接带 `noopener noreferrer`；页面不使用浏览器存储记录协议状态。

### App 编排与桌面 UI

- `mobileOpMu` 串行 setup、Retry、Clear、Reset、模式切换和 shutdown；`mobileMu` 只保护短生命周期字段。
- 所有 mutation 均先由专用 manager 持久化，再无失败地更新 App、Bridge 和 Server 运行态。服务未监听时 Clear/Reset 仍可完成且不会隐式启动。
- `mobileClosing` 的所有读写统一经过 `mobileMu`；配置订阅在释放 `mobileMu` 后注册，避免 config lock 与 Mobile lock 顺序反转。
- 解题任务在启动 goroutine 前捕获显示代际。题库 Append/Finish 仍先于 Mobile 路由；旧任务可继续完成题库、桌面与模型上下文，但不能污染清屏后的 Mobile。
- 新增共享 Pinia Mobile store、根级单例确认层、设置页完整入口和 TopBar 高频清屏入口。读取和 mutation 使用请求版本号，迟到响应不能覆盖新 URL。
- 正式 Wails bindings 已生成：`ClearMobileDisplay`、`ResetMobileOutputLink`、`SetMobileStableLink`、扩展后的 `MobileInfo` 与 `MobileCommandResult`。
- 新增 `app/mobile_commands_test.go` 集中覆盖 Mobile mutation 与任务代际，并复用其中的 fake manager 适配 `app/mobile_test.go` 和 `app/phase4_test.go`；后者只调整新构造器/状态注入，既有窗口策略断言不变。

## 与计划的偏差

- 为避免“失败返回但重启后候选生效”，双槽提交增加了独立 pending 文件；pending 永不参与启动选择。这是对原计划事务边界的加固。
- 当前机器的生产端口 `38511` 在一次聚焦测试期间已被外部进程占用，因此依赖默认端口的既有 App/窗口策略用例按原设计 SKIP；核心命令、持久化、Bridge、Server 和随机端口 SSE 用例均已执行。端口占用与释放后的 GUI Retry 仍列入人工矩阵。

## 未执行操作

- 未调用真实模型、未上传截图、未消费 API 额度。
- 未修改真实 API 密钥、普通用户配置或 agentmemory。
- 未提交、未推送、未创建或移动 Tag、未发布 Release。

## 2026-10-23 补充修复

### 确认层遮挡

- `TopBar.vue` 同步监听共享 store 的 `pendingAction`；任何待确认操作建立后立即关闭运行状态面板。
- `.status-panel` 从 `--z-tooltip` 降至 `--z-dropdown`。即使 150ms leave 动画仍保留 DOM，残影也位于根级 `MobileActionConfirm` 下方。
- 状态面板鼠标入口继续调用 `mobile.requestClear()`；未新增确认层、清屏协议或 Mobile store。

### Windows `Shift+C` 清屏请求

- Windows 默认快捷键新增 `clear_mobile_display = 67+160 / Shift+C`，严格沿用现有 Hook 对左 Shift 的 VK 语义；macOS 默认表未增加该 action。
- `pkg/shortcut` 只发送无 payload 的 `request-clear-mobile-display` Wails 事件，不调用 App 清屏命令、不发成功 Toast。
- `App.vue` 收到事件后先检查共享 busy/pending，再刷新 Mobile 状态；只有读取成功后才调用 `mobile.requestClear()` 建立待确认操作。真正清屏仍只能由根级确认层在用户确认后调用既有 API。
- `setPending()` 和 `confirmPendingAction()` 均使用 `busy = loading || mutating` 门禁。长按、快速重复、读取中、mutation 中或已有 pending 时不会重复弹窗或提交。

### 设置与旧配置迁移

- “设置 → 常规”新增 Windows-only“清空同步端显示”，复用现有录制、冲突检测和保存流程；显式空绑定显示“未绑定（点击录制）”。
- 旧配置加载时额外读取原始 JSON 的 `shortcuts` key 集。action 缺失时把默认 `67+160` 迁给清屏，并清空占用该组合的旧 action；显式默认绑定同样清理重复占用。
- 早期验证版可能已持久化空 `clear_mobile_display`，同时让旧 action 继续占用 `67+160`。加载器会识别这一组合并纠偏，避免左 `Shift+C` 再次触发隐藏窗口；显式自定义清屏组合，以及默认组合无冲突时的显式空绑定保持不变。
- 原始 key 检查仍是必要的：目标 `Config` 先含默认非 nil map，`json.Unmarshal` 会合并旧键，不能仅从反序列化后的 map 判断 action 是否来自持久化配置。

### 快捷键录制运行态回修

- `handleRecord()` 现在只发送 `key-recorded` 预览，不再在按键尚未松开、冲突尚未检查时修改 Hook 运行态 map。
- `handleRecordingComplete()` 先扫描冲突；冲突时只发送 `shortcut-error`，目标 action 与占用 action 均保持原绑定，不再自动重新进入录制。
- 无冲突时才一次性提交 binding，并发送 `shortcut-saved`，使前端结束录制状态。这样运行态不会留下两个相同的非空 `ComboID`，Go map 遍历也不再随机触发隐藏窗口。

### 文档与验证程序

- README 中英文快捷键表与行为说明已同步：Windows-only、默认左 `Shift+C`、可重新录制、仅打开确认层，以及全局 Hook 命中后会吞掉组合键。
- 使用隔离 Wails CLI v2.11.0 与 `GOTOOLCHAIN=go1.25.13` 生成生产 EXE。旧验证版未被覆盖；最终人工复验文件为 `build/bin/archive/validation/Q-Solver-change016-shiftc-final-validation.exe`，SHA-256 为 `a1480ab36c86bdff794669f362e71e5d29b5a5d96b73221bed187dad5b944659`。
- 补充修复保持未提交、未推送、未创建 Tag、未发布。
