# 验证记录 04

## 当前状态

普通自动门禁、固定 Wails v2.11.0 保护构建和隔离 Profile 基础 GUI 验证已完成。race 因 Windows 启动错误 `0xc0000139` 为 `BLOCKED` 并已披露；用户于 2026-11-11 确认当前测试版本“没有什么问题”。结合自动证据与用户验收，Change 038 的产品验收通过；功能与治理提交已推送。用户随后明确要求更新本地版本，稳定版晋升已完成；发布不在授权范围。

## 自动验证

| 命令 | 日期 | 结果 | 说明 |
|---|---|---|---|
| SolveAttempt / Retry / Cancel / QuestionBank / Shortcut 聚焦测试 | 2026-11-11 | PASS | 按 TDD 逐任务运行；覆盖单槽、深拷贝、target/epoch、同 Record、原子失败、draining、迁移与 FIFO |
| `go test ./pkg/task ./pkg/solution ./pkg/questionbank ./pkg/shortcut ./pkg/mobile ./app -count=1` | 2026-11-11 | PASS | 相关 Go 包全量通过 |
| `go test ./... -count=1` | 2026-11-11 | PASS | 全仓 Go 测试通过 |
| `go vet ./...` | 2026-11-11 | PASS | 无输出，退出码 0 |
| `go build ./...` | 2026-11-11 | PASS | 退出码 0 |
| `go mod verify` | 2026-11-11 | PASS | `all modules verified` |
| `frontend/*_regression_test.mjs` | 2026-11-11 | PASS | 19 个现有前端回归全部通过 |
| `node pkg/mobile/web_dom_test.mjs` | 2026-11-11 | PASS | Mobile answer-copy、错误诊断、cancelled/fast DOM 通过 |
| `npm --prefix frontend run build` | 2026-11-11 | PASS | Vite 4.5.14，135 modules；仅既有大 chunk warning |
| `git diff --check` | 2026-11-11 | PASS | 无空白错误 |
| VS Code diagnostics | 2026-11-11 | PASS | Warning 及以上为空 |
| `go test -race ./pkg/task ./pkg/solution ./pkg/questionbank ./pkg/shortcut ./pkg/mobile ./app -count=1` | 2026-11-11 | BLOCKED | 六个目标包均在启动阶段返回 `exit status 0xc0000139`；未进入测试体 |

## bindings 与依赖保护

固定命令：

```text
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 generate module -nocolour
```

结果：

- `frontend/wailsjs/go/app/App.js` 与 `App.d.ts` 增加 `TriggerQuickSend`、`RetrySolve`、`CancelSolve`。
- `frontend/wailsjs/go/models.ts` 增加 `app.SolveControlResult`。
- `go.mod` SHA-256 保持 `EAD5D49037577845346967F41FAB9DC04138F75F500D09387B306CCCE5738595`。
- `go.sum` SHA-256 保持 `22174AB9FC52DDC6AC87F6862961A0C5DDDCB254BB05B0E30DB300EAEFBB7014`。
- `frontend/wailsjs/runtime/package.json`、`runtime.js`、`runtime.d.ts` 哈希不变。
- 未升级依赖或 Wails runtime。

## 保护候选

构建命令使用固定 Wails v2.11.0 与保护参数：

```text
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -s \
  -o testing/current/Q-Solver-candidate-change038.exe \
  -skipbindings -skipembedcreate -nosyncgomod -m -nocolour -v 1 \
  -ldflags "-s -w" -tags prod
```

| 项目 | 结果 |
|---|---|
| 晋升前候选路径 | `build/bin/testing/current/Q-Solver-candidate-change038.exe` |
| 晋升后根稳定版 | `build/bin/Q-Solver.exe` |
| 根稳定版大小 | 42,550,784 字节 |
| 根稳定版 SHA-256 | `8A868263F29E170F1204E8BABC8064055D5533B870ED1A023309C99F1758D220` |
| 被替代稳定版归档 | `build/bin/archive/superseded/change-038/Q-Solver-stable-before-change038.exe` |
| 被替代稳定版 SHA-256 | `8B19B242F7917D4BC0267E45BB6248077E2BA5BD2D3CE428AC9E1B836C8B4994` |
| 通过候选归档 | `build/bin/archive/validation/change-038/Q-Solver-candidate-change038.exe` |
| Wails runtime | `go version -m` 显示 `github.com/wailsapp/wails/v2 v2.11.0` |
| 归档清单 | superseded 与 validation 的 `SHA256SUMS` 均匹配 |
| 测试区 | `testing/current/` 已清空 |
| Release | rc.1、rc.2、rc.3 资产和清单哈希不变 |
| bindings/runtime/index | 构建及晋升后保护哈希不变 |

候选在用户验收通过后先保留于 `testing/current/`；用户随后明确要求更新本地版本，已按构建产物生命周期政策完成旧稳定版归档、同卷根 EXE 替换、验证候选归档和测试区清理。首版候选 `ABB231…67AB9` 在边界审计发现 Mobile 历史 cancelled 语义缺口后已移入 `archive/transient/`；回修后的全部普通门禁和最终候选验证均重新执行。

## 隔离 Profile GUI 验证

环境：Windows 11 Pro、WebView2 157.0.4341.42；`APPDATA` / `LOCALAPPDATA` 指向仓库外临时隔离目录；未读取或保存正式用户配置，未调用真实 API。

| 检查项 | 结果 | 证据 |
|---|---|---|
| 独立启动 | PASS | 候选创建 1024×768 可见窗口，主进程持续响应 |
| WebView2 | PASS | 候选进程树创建 `msedgewebview2.exe` 子进程 |
| Ghost Mode / 可见渲染 | PASS | 初始抓屏受 Ghost Mode 正常保护；基础验证通过 `F9` 关闭后取得可见 UI，最终候选另以 `SetWindowDisplayAffinity(WDA_NONE)` 临时关闭当前窗口抓屏保护并确认真实渲染，不写配置 |
| 快捷键提示 | PASS | TopBar/Welcome 同时显示 `F8`、`Ctrl+J`、`Ctrl+Shift+J`、`Ctrl+Shift+X`、`F9` |
| 中断无任务路径 | PASS | 按 `Ctrl+Shift+X` 显示“当前没有正在进行的解题”信息 Toast |
| 截图预检 | PASS | 未配置模型时按 `F8` 安全打开设置并提示选择模型；未发出请求 |
| Node/Vite 子进程 | PASS | 两次候选进程树均只有 WebView2 子进程，无 Node/Vite |
| 正常退出 | PASS | `CloseMainWindow` 后主进程正常退出；WebView2 短暂收尾后自行退出 |
| 最终残留进程 | PASS | 第二次验证等待最多 10 秒，候选及全部后代均不存在 |
| ScreenshotDock 快速按钮 | PASS（用户确认） | 隔离 Profile 未配置模型，自动 GUI 未形成待发送截图；用户随后确认当前测试版本没有问题 |
| 运行中“中断解题”按钮 | PASS（用户确认） | 自动测试覆盖 context 取消、draining 与 UI 状态；用户确认测试版本无问题 |
| 错误卡“重试本题” | PASS（用户确认） | fake Provider 与 typed UI 自动测试覆盖；用户确认测试版本无问题 |
| Mobile 真机 cancelled/fast | PASS（用户确认） | 自动 SSE/DOM 通过；用户确认测试版本无问题，未提供逐项真机日志 |

## 验收标准对照

| `00-request.md` 标准 | 自动证据 | 当前结论 |
|---|---|---|
| 网络错误、timeout、503、流中断、取消后同轮重试且不截图 | fake Provider、attempt 与 typed UI 回归覆盖同轮/原图；用户确认当前测试版本无问题 | PASS |
| 旧轮、成功轮、过期会话、重复点击拒绝 | attempt/retry/App/Pinia 测试 | PASS |
| 题库同 Record、图片引用不变 | `TestRestartRecord*`、`TestRetrySolveReusesBackendImagesAndSameQuestionBankRecord` | PASS |
| KeepContext 失败不写、成功只写一轮 | Solver/App 测试 | PASS |
| context 取消与 draining 屏障 | `TestCancelSolveWaitsForInFlightCompletionAndKeepsRetry` 等 | PASS |
| Desktop/Mobile cancelled、部分正文与不完整提示 | Solver/App/Pinia/Mobile Bridge/DOM 测试；额外覆盖 cancelled 条目进入 Mobile 页面内历史后仍显示“已中断”并保留 fast 模式；用户确认测试版本无问题 | PASS |
| normal 当前场景、fast 请求级 `gen-speed` 且 Config 不变 | App mode/config/subscriber 测试 | PASS |
| fast retry 保持 fast、题库记录实际场景 | retry/mode/capture 测试 | PASS |
| Windows/macOS 默认值、冲突不抢占 | Config/Manager 测试 | PASS |
| 全显示端动态提示、三图/FIFO/Reset 不回归 | 全部前端、Config/Shortcut/Mobile/App 回归、基础 GUI 提示及用户测试版本确认 | PASS |
| fake/httptest，不读取真实配置、不调用收费 API | 自动门禁与隔离 Profile | PASS |

## 结论

状态：**产品验收通过 / 普通自动门禁 PASS / 用户确认测试版本无问题 / race BLOCKED 已披露 / 功能提交 `5564a45` / 治理提交 `2502b31` / 本地稳定版已晋升 / 未发布**。

双提交已推送到 `origin/dev/qsolver-custom`。当前分支本身就是目标开发分支，无独立功能分支需要合并；不修改 `main`。用户后续授权的本地稳定版晋升已完成，Tag 与 Release 未执行。
