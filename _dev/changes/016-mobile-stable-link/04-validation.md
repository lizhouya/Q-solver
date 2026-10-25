# 验证记录 04

状态：普通自动验证 PASS；用户确认的 Windows GUI 人工验收 PASS；race BLOCKED；未提交、未推送、未发布。

## 自动验证

| 项目 | 结果 | 说明 |
|---|---|---|
| `go test ./pkg/config -run Mobile -count=1 -v` | PASS | 默认固定模式、双槽恢复、临时 Token、事务失败、未知 schema、溢出和 Token 去重 |
| `go test ./pkg/mobile -count=1 -v` | PASS | Bridge 屏障、epoch、队列、全路由鉴权、SSE 撤销、Stop 与实时 DOM 链路 |
| `go test ./app -run 'Mobile\|DisplayEpoch\|Settings' -count=1 -v` | PASS（含 11 个既有固定端口用例 SKIP） | 命令确认、持久化失败不切运行态、未监听 mutation、shutdown 栅栏、任务显示代际和设置读取；默认端口被外部进程占用的 11 个既有监听/窗口策略用例按设计 SKIP |
| `node pkg/mobile/web_dom_test.mjs` | PASS | 清屏、历史清理、旧帧、旧 rAF、session 淘汰和安全外链 |
| `node frontend/settings_regression_test.mjs` | PASS | 普通设置不包含 Mobile 专用状态，Clipboard 失败有反馈 |
| `node frontend/mobile_store_regression_test.mjs` | PASS | 确认门、共享 busy、请求版本和失败保留 URL |
| `go test ./... -count=1` | PASS | 全量 Go 测试通过 |
| `go vet ./...` | PASS | 无 vet 错误 |
| `go build ./...` | PASS | 全量 Go 构建通过 |
| `npm --prefix frontend run build` | PASS | Vite 4.5.14，126 modules；仅既有大 chunk 警告 |
| Wails v2.11.0 + Go 1.25.13 生产构建 | PASS | 唯一验证版 `build/bin/archive/validation/Q-Solver-change016-validation.exe`，未覆盖统一入口或 RC 文件 |
| `git diff --check` | PASS | 仅 Git 的 CRLF 提示，无空白错误 |
| `go test -race ./pkg/mobile ./app ...` | BLOCKED | 两个 package 均在启动阶段返回 Windows `0xc0000139`；普通测试不替代 race |

原实现验证构建 SHA-256 为 `4594b8d1fd9d828021da3592e38235c09569fe4f81a1f666ba2e2b54cdd6e5e0`；2026-10-23 的补充构建均保留在验证归档中，最终人工复验文件与哈希见下文和 `build/bin/archive/validation/SHA256SUMS`。构建后 `build/bin` 顶层仍只保留 `Q-Solver.exe`。

## 安全与数据边界检查

- PASS：Token 未进入普通 `Config`、Snapshot、README、Change 文档、Toast、浏览器存储或前端 Mobile DTO。
- PASS：完整 URL 只通过 `MobileInfo.URL` 进入设置页显式展示/复制链路；TopBar、Toast 和日志不显示完整 URL。
- PASS：日志只使用脱敏 `SafeURL()`；测试断言不输出测试 Token 或完整链接。
- PASS：Clear/Reset 代码路径不调用 `CancelRunningTask`、`solver.ClearHistory`、题库删除、截图删除或窗口隐藏策略。
- PASS：Wails 生成未改变 `go.mod`、`go.sum`；runtime shim 构建行尾噪音已恢复。

## 人工验收矩阵

| 场景 | 状态 |
|---|---|
| 固定模式下相同 IP 完整重启复用链接，新进程低 seq 正常显示 | NEEDS_USER |
| Retry 保持链接、模式和内容，且不隐藏桌面窗口 | NEEDS_USER |
| 关闭固定模式后立即换链接并清屏；重启再换；同进程 Retry 不换 | NEEDS_USER |
| 重新启用后生成全新持久链接，且不恢复关闭前链接 | NEEDS_USER |
| Reset 拒绝旧页面和 SSE、断开旧连接、新链接首帧为空 | NEEDS_USER |
| 多条历史且选中旧答案时 Clear，宽/窄历史与显示均清空，本地数据保留 | NEEDS_USER |
| 断网期间 Clear，重连和刷新后旧内容不恢复 | NEEDS_USER |
| 流式解题中 Clear，旧任务不污染 Mobile，题库/桌面继续完成，新题正常显示 | NEEDS_USER |
| 占用并释放 `38511` 后 Retry 使用最新 Token | NEEDS_USER |
| GUI、日志、Toast、终端无真实 Token/完整 URL，非 Mobile 数据未删除 | NEEDS_USER |

## 结论

实现和普通自动验证已完成，可交付验证版进行人工矩阵；在用户完成真机/GUI 验收且 race 环境可用前，Change 016 不标记为关闭。未提交、未推送、未发布。

## 2026-10-23 补充修复验证

### 自动验证结果

| 项目 | 结果 | 说明 |
|---|---|---|
| `GOTOOLCHAIN=go1.25.13 go test ./pkg/config -count=1 -v` | PASS | Windows 默认绑定、非空 ComboID 唯一、macOS 不含 action；action 缺失、旧占用、旧版持久化空绑定加冲突、显式默认/自定义/空绑定边界 |
| `GOTOOLCHAIN=go1.25.13 go test ./pkg/shortcut -count=1 -v` | PASS | 新 action 只发送一次无 payload 请求事件；录制预览不提交、冲突不污染、成功完成才提交并发送 `shortcut-saved` |
| `node frontend/settings_regression_test.mjs` | PASS | Windows/macOS 清单、空绑定显示、成功录制结束、冲突回滚，以及快捷键事件不能绕过确认 |
| `node frontend/mobile_store_regression_test.mjs` | PASS | refresh、busy、pending、确认门及 TopBar 层级源码回归 |
| `GOTOOLCHAIN=go1.25.13 go test ./... -count=1` | PASS | 全量 Go 测试通过 |
| `GOTOOLCHAIN=go1.25.13 go vet ./...` | PASS | 无 vet 错误 |
| `GOTOOLCHAIN=go1.25.13 go build ./...` | PASS | 全量 Go 构建通过 |
| `node pkg/mobile/web_dom_test.mjs` | PASS | 既有 Mobile 页面协议回归继续通过 |
| `npm --prefix frontend run build` | PASS | Vite 4.5.14，126 modules；仅既有大 chunk 警告 |
| `git diff --check` | PASS | 仅 Git 的 CRLF 提示，无空白错误 |
| `GOTOOLCHAIN=go1.25.13 go test -race ./pkg/mobile ./app -run 'Mobile\|Bridge\|SSE\|DisplayEpoch' -count=1` | BLOCKED | `pkg/mobile` 与 `app` 均在启动阶段返回 Windows `0xc0000139`；普通测试不替代 race |
| Wails v2.11.0 + Go 1.25.13 生产构建 | PASS | 最终人工复验文件为 `build/bin/archive/validation/Q-Solver-change016-shiftc-final-validation.exe`；旧验证版保留 |
| 归档 `sha256sum -c SHA256SUMS` | PASS | 归档内五份历史/当前验证 EXE 全部校验通过；人工复验只使用 `shiftc-final` 文件 |

最终人工复验版 SHA-256：`a1480ab36c86bdff794669f362e71e5d29b5a5d96b73221bed187dad5b944659`。此前的 `Q-Solver-change016-validation.exe` 和中间 `shiftc-fix` 构建均保留为历史验证文件，不用于本轮复验。

构建后核对结果：

- `build/bin` 顶层仍只有 `Q-Solver.exe`；临时 EXE 已移动到验证归档，不残留顶层临时入口。
- `go.mod`、`go.sum` 和前端 lockfile 均无变化。
- Wails runtime shim 与仓库差异在忽略行尾后为零，本次构建没有新增语义或行尾差异。
- 正式 bindings 含既有 `ClearMobileDisplay`、`ResetMobileOutputLink` 和 `SetMobileStableLink`，未新增绕过确认的清屏接口。

### 补充安全与边界审计

- PASS：未发现真实 API Key、Mobile Token、完整凭据链接或私人配置进入新增源码、测试、README、Change 文档、Toast 或日志；测试中的 Token 均为显式合成值。
- PASS：生产日志继续使用 `SafeURL()` 脱敏；完整 Mobile URL 只进入 `MobileInfo.URL` 的设置页显式展示与复制用途。
- PASS：普通 `config.Config`、Wails 普通设置 DTO 和设置保存 payload 未包含 Token、URL、generation、sessionId、clearSeq 或 stableLinkEnabled。
- PASS：`go.mod`、`go.sum`、`frontend/package.json` 与前端 lockfile 无差异；补充修复文件落在 `02-plan.md` 的批准白名单内。
- 注意：前端普通设置 store 会保留后端返回的完整普通配置快照，因此 Mobile 专用字段不进入 `config.Config` 仍是必须保持的后端边界。

### 补充人工验收矩阵

| 场景 | 状态 |
|---|---|
| 打开运行状态并点击清屏：面板立即关闭，确认层无遮挡；取消不清屏 | NEEDS_USER |
| 未打开状态面板时按左 `Shift+C`：只出现一个确认层；确认后才清屏 | NEEDS_USER |
| 快速重复或长按左 `Shift+C`：不重复弹窗，后端只提交一次 | NEEDS_USER |
| 首次启动直接按快捷键：先读取 Mobile 状态；读取失败不建立 pending | NEEDS_USER |
| 在“设置 → 常规”重新录制并保存后，新组合生效 | NEEDS_USER |
| 旧配置由隐藏窗口占用 `67+160`（含旧版持久化空清屏绑定）时，启动后旧 action 变为未绑定，左 `Shift+C` 只请求清屏确认 | NEEDS_USER |
| Clear 后桌面答案、题库、截图、导出和模型上下文保留；刷新/重连仍为空，新题正常显示 | NEEDS_USER |
| macOS 设置清单不显示该 action | NEEDS_USER |

补充修复的普通自动验证已完成；上述 GUI/平台行为未由自动构建替代。在人工矩阵完成且 race 环境可运行前，Change 016 继续保持未关闭、未提交、未推送、未发布。


## 2026-10-23 用户人工验收结论

用户已使用最终验证版完成并明确确认以下 Windows GUI 场景通过：

- 运行状态面板触发清屏时，面板先关闭，根级确认层无遮挡；取消不执行清屏。
- 左 `Shift+C` 只打开一个清屏确认层；快速重复和长按不会重复弹窗或重复提交。
- “设置 → 常规”重新录制快捷键后，新组合可生效。
- 清屏后同步端刷新或重连仍保持空白，下一题能够正常同步。
- 桌面答案、题库、截图、已有导出和模型上下文均保留。

另经现场诊断确认，用户曾按下的 `Shift+S` 被个人快捷键配置绑定为“向下移动”，导致桌面窗口连续移出屏幕；这与默认清屏组合 `Shift+C` 不冲突。窗口已通过 Win32 安全移回屏幕，用户随后把窗口移动恢复为 `Alt+方向键`。未修改用户私人配置文件。

本轮人工证据将上述补充 GUI 验收标记为 **PASS**。Windows race 仍因启动阶段 `0xc0000139` 标记 **BLOCKED**，普通测试不能替代；因此 Change 016 尚不满足 Close 条件，且仍未提交、未推送、未创建 Tag、未发布。
