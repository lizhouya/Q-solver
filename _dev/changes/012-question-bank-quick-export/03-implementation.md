# 实现过程记录 03

## 实现日志

| 日期 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-10-19 | Store v2 | 增加自动懒建段、按 ID 封存/整理/导出、软清空、legacy 根读取和 v2 路径 | `pkg/questionbank/types.go`、`store.go`、`export.go`、`organize.go` | 当前段只负责自动追加；终态段保持只读，后续解题懒创建新段。 |
| 2026-10-19 | 路径与 Markdown | 最终导出直接位于 `ArchiveDir`，固定 `题库.md`、`原始记录.md`、`assets/`，按“大类 → 题型”组织 | `pkg/questionbank/export.go` | 去除重复的“求职笔试/题库/按单位”等层级；顺序后缀避免覆盖。 |
| 2026-10-19 | App 接入 | 独立 v2 偏好、路由前归档、ActionBar 一键异步整理/导出、退出保护 | `app/questionbank*.go`、`app/solve.go`、`app/app.go`、`main.go` | 模型整理默认关闭；整理与正常解题分别使用独立任务状态。 |
| 2026-10-19 | 主设置与前端 | 将题库偏好移入主设置，改造题库面板、ActionBar、TopBar 和 Wails bindings | `frontend/src/**`、`frontend/wailsjs/go/**` | 取消日常“开始场次”步骤；选择目录只修改临时值，点击“保存”才持久化。 |
| 2026-10-19 | Mobile | 快照显示实际快捷键，增加独立 `ControlSeq` 和桌面到手机的滚动控制 | `pkg/mobile/**`、`app/mobile.go`、`app/shortcut.go` | 首个 SSE snapshot 只同步控制基线，实时 message 才执行一次性控制。 |
| 2026-10-19 | Windows hook | 将原生 hook 结构复制集中到 platform 层 | `pkg/platform/platform_windows.go`、`pkg/shortcut/shortcut_windows.go` | shortcut 层不再执行 `uintptr -> unsafe.Pointer`，真正消除 vet 告警。 |
| 2026-10-20 | 终态安全 | 已导出段拒绝软清空，已清空段拒绝模型整理；模型结构化字段中和 HTML 与外链 | `pkg/questionbank/store.go`、`organize.go`、`export.go` | 原答案仍逐字保留，仅净化模型生成的结构化字段。 |
| 2026-10-20 | 持久化故障修复 | 首次懒建段写盘失败仍返回稳定记录 ID 并保留内存记录；dirty 终态段禁止被下一次 Begin 覆盖 | `pkg/questionbank/store.go`、`segment_test.go` | 修复两条可复现的数据丢失状态序列；目录恢复后可用 Flush 重试。 |
| 2026-10-20 | 历史重试入口 | 新增显式历史“重试整理 / 导出”，普通答案旁一键按钮继续拒绝重复导出 | `app/questionbank.go`、前端 store/panel/API/bindings | 已导出段可只重试没有结构化问题的记录并生成 `_02` 等新版本。 |
| 2026-10-20 | 队列与大小边界 | 控制帧在 SSE 队列满时替换积压快照；截图写入与导出统一为解码后 20 MiB | `pkg/mobile/bridge.go`、`pkg/questionbank/files.go` 及测试 | 防止慢手机消费者静默丢失滚动命令，避免写入后无法导出。 |
| 2026-10-20 | 导出部分成功语义 | 文件已生成但状态清单保存失败时仍保留 `LastExport` 并明确提示 | `app/questionbank.go`、`app/questionbank_test.go` | 不再把“文件已生成、状态保存失败”误报成完全未导出。 |
| 2026-10-20 | 旧输出实际迁移 | 将 3 个 v1 会话移入 `_记录`，将 5 份旧导出移到最终题库根并改用固定 Markdown 文件名 | 仓库外 `输出/求职笔试/题库` | 用户明确要求修复资源管理器中的重复层级；逐目录哈希核对后仅删除空目录。 |

## 与计划的偏差

- 原计划只要求普通一键按钮防止重复导出；审查发现这会让“部分整理失败后只重试失败项”和历史重新导出不可达，因此增加了**显式历史重导出入口**。普通 ActionBar 的防重复语义不变。
- Mobile 初版使用非阻塞满队列丢帧；故障测试确认控制帧可能丢失后，改为仅对控制帧清理积压的全量业务快照，普通解题广播仍保持非阻塞。
- Wails CLI 为 v2.16.0，而项目模块为 v2.11.0。bindings 采用精确增量更新；构建使用 `-skipbindings -nosyncgomod`，构建产生的 runtime shim 版本噪音经审查后精确恢复。
- 未增加依赖，未修改模型角色/场景选择逻辑，未开发网申业务。

## 关键决策纪要

1. `ArchiveDir` 表示最终题库目录，内部恢复数据固定放在 `_记录/`，用户可见导出直接放在 `ArchiveDir` 下。
2. 当前段在第一次解题时创建；封存、导出、软清空后不可继续写入。
3. Store 的 `Begin` 可以返回“非空记录 ID + 保存错误”：App 继续用该 ID 关联最终答案，同时向用户显示未落盘告警。
4. dirty 的 Ended/Exported/Discarded 内存段在成功 Flush 前不得创建新段。
5. ActionBar 只处理当前段且不重复导出 Exported；历史列表提供独立重试/重导出按钮。
6. 模型整理开关是持久化的明确授权，默认关闭；关闭时导出完全在本地完成。
7. 快捷键提示只包含动作标签和显示名；滚动控制使用独立序号，不包含 VK、Token、路径或配置。
8. 结构化模型字段由本地模板中和 HTML 和外链；原答案不修改。

## 遗留未完成项

- Windows GUI 全流程、手机真机滚动、关闭窗口提示和一次用户主动触发的真实模型整理仍需人工验收。
- `go test -race` 在本机启动阶段以 `0xc0000139` 失败，属于既有工具链/运行环境阻塞，未用普通测试替代宣称 race 通过。
- Change 010/011 与本 Change 均保持未提交；没有 commit、push、Tag 或 Release。

## 2026-10-20 Mobile 人工反馈回修

| 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|
| 生命周期拆分 | Bridge 与配置订阅在 App 生命周期内保持单例，Server 监听失败后保留实例并允许重试 | `app/app.go`、`app/mobile.go`、`pkg/mobile/server.go` | 避免重试时重复订阅或轮换运行中连接凭据；已运行服务的重试保持幂等。 |
| 稳定状态 | Mobile 信息增加 `status` 与 `retryable`，区分未启动、初始化失败、端口不可用、已停止和已就绪 | `app/mobile.go`、Wails bindings | 不把原始监听错误暴露给前端；不可用时不返回 URL 或端口。 |
| 显式恢复 | 设置页增加重新读取和服务重试；每次打开设置或进入“输出”标签刷新 | `SettingsModal.vue`、`api.js` | 服务状态与未保存的 `outputMode` 草稿分离，避免把临时选择误认为已经生效。 |
| 滚动入口 | 顶部上下箭头继续调用统一 `ScrollContent`，箭头组增加局部 wheel 输入 | `TopBar.vue`、`app/shortcut.go` | wheel 只在控制组内接管，不修改 Windows 全局 Hook，不影响设置弹窗或普通答案区。 |
| 控制帧隔离 | 手机页分别跟踪业务 `seq`、`controlSeq` 与 hints 指纹 | `pkg/mobile/web/index.html` | 相同业务序号的控制帧不重建答案或历史 DOM，不重置历史选择；断线首帧只同步控制基线。 |
| DOM 回归 | 使用 Node 标准库和手写 DOM/EventSource stub 验证控制与业务渲染分离 | `pkg/mobile/web_dom_test.mjs` | 无新增前端测试依赖；覆盖首帧、重复控制、历史选择、hints 与 70% 视口滚动。 |
| 窗口安全 | `RetryMobileOutput` 不调用输出模式窗口策略 | `app/mobile_test.go` | 端口释放后从设置页恢复服务时，不突然隐藏用户正在操作的桌面窗口。 |

### 本次回修保持不变的边界

- 继续复用 `ScrollContent → Bridge.PublishControl → SSE → window.scrollBy`，未新增第二套协议。
- 全局键盘快捷键继续使用已保存配置；默认 Windows 快捷键仍为 `Alt+PgUp` / `Alt+PgDn`。
- Mobile 不可用时，滚动和答案路由安全回退桌面；保存 `mobile` 模式也不会在服务不可用时自动隐藏桌面。
- 未新增依赖、未升级 Wails runtime、未调用真实模型，也未记录 Mobile URL、Token、内部 VK 或配置路径。

## 2026-10-22 实时同步补修

- Publish 在广播前统一把内部 b.seq 写入快照 Seq，修正开始帧以后 thinking/chunk/done/error 序号不变的问题。浏览器现有业务去重保留，不借重新渲染所有控制帧掩盖协议错误。
- done/error 队列满时替换陈旧业务快照、保留待发送控制；极端全控制积压时保留最新 cap-1 个命令，为终态留一格。生产者由 Bridge 锁串行，恢复队列不会超容量，解题线程不等待网络。
- 新增逐帧序号、满队列终态/控制保留测试；真实 Go HTTP/SSE 输出用临时 JSON 文件输入 Node DOM harness，同一连接连续两题均实时进入历史。仅使用合成数据，无 API 调用、无真实凭据。
- 新程序 build/bin/Q-Solver-sync-fix.exe 保留供用户启动，包含之前 API 恢复修改。没有覆盖旧 Q-Solver.exe 或 Q-Solver-api-fix.exe，没有启动程序或强停旧实例。
