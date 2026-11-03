# Change 035 截图捕获恢复审计

## 审计结论

Change 035 的实现范围与已批准方案一致。Windows Hook 命中后的业务执行已从底层回调线程迁移到 `shortcut.Service` 的单 worker FIFO；截图上限、截图捕获实现、模型请求、题库和配置均未修改。ScreenshotDock 恢复张数、继续截图、删除和发送提示，并通过 `storeToRefs` 绑定实际快捷键。

## 根因与修复链路

### 修复前

`keyboardHookProc → onKeysChanged → Manager.OnTrigger → Service.handleTrigger → App.TriggerScreenshot → screenshot.Capture + 图片压缩`

Windows 低级 Hook 回调同步等待完整截图链路，存在明显阻塞和 Hook 超时静默移除风险。

### 修复后

`keyboardHookProc → onKeysChanged → Manager.OnTrigger → Service.enqueueTrigger → FIFO`

Hook 回调只做内存入队。单独 worker 在 Hook 外依次执行 `Service.handleTrigger`，保持 `截图 1 → 截图 2 → 截图 3 → 发送` 顺序。

## 文件边界

### 功能代码与测试

- `pkg/shortcut/service.go`
- `pkg/shortcut/dispatcher.go`
- `pkg/shortcut/dispatcher_test.go`
- `frontend/src/components/ScreenshotDock.vue`
- `frontend/screenshot_dock_regression_test.mjs`

### 治理记录

- `_dev/audits/035-screenshot-capture-recovery.md`
- `_dev/changes/INDEX.md`
- `_dev/changes/035-screenshot-capture-recovery/00-request.md`
- `_dev/changes/035-screenshot-capture-recovery/01-current-state.md`
- `_dev/changes/035-screenshot-capture-recovery/02-plan.md`
- `_dev/changes/035-screenshot-capture-recovery/03-implementation.md`
- `_dev/changes/035-screenshot-capture-recovery/04-validation.md`
- `_dev/changes/035-screenshot-capture-recovery/05-rollback.md`
- `_dev/changes/035-screenshot-capture-recovery/06-notes.md`

### 明确排除

- `app/solve.go`、`pkg/screen/**`、模型 Provider、Prompt、题库、Mobile、配置和依赖文件。
- `frontend/dist/**` 构建产物不进入 Git。
- `build/bin/**`、测试区 EXE 和 `SHA256SUMS` 保持 ignored，不使用 `git add -f`。
- 仓库外预构建哈希清单、用户配置、日志、CCR 数据和 secret 不进入提交。

## 验证证据

- 调度器红灯：缺少 `newSerialActionDispatcher`，符合预期。
- Dock 红灯：缺少 `storeToRefs` 与继续截图提示，符合预期。
- 调度器聚焦、Shortcut 全包、三图 Solver、15 份前端回归、全量 Go test/vet/build、Go module verify、Vite build、Wails v2.11.0 独立候选构建均通过。
- 调度器聚焦测试连续 50 次通过。
- `go test -race ./pkg/shortcut -count=1` 因环境错误 `0xc0000139` BLOCKED，未记为通过。
- 晋升后根稳定版独立启动验证通过：8 秒存活、WebView2 正常、无 Node/Vite 子进程、正常退出码 0、无残留进程。
- 用户已明确确认 Change 035 候选 GUI 验收通过，包括本轮三图截图、上限、删除补拍、发送和快捷键提示验收范围。

## 构建保护

- 当前稳定版：`build/bin/Q-Solver.exe`
- 候选归档：`build/bin/archive/validation/change-035/Q-Solver-candidate-change035.exe`
- 当前稳定版与候选大小：44,029,440 字节
- SHA-256：`29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE`
- 测试前稳定版已归档至 `build/bin/archive/superseded/change-035/Q-Solver-before-change035.exe`，SHA-256：`C033EAB4844721EB233841915C82583D213095965357E77091738D9FBC21B0B8`。
- 候选已晋升为根稳定版，根与验证归档 SHA-256 均为 `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE`。
- `testing/current/` 已清空；根目录唯一 EXE 为 `Q-Solver.exe`。
- 候选实际链接 Go 1.25.13、Wails runtime v2.11.0。
- 构建前后 `go.mod`、`go.sum`、`frontend/wailsjs/**`、`frontend/dist/**`、Git index 和 `build/bin/releases/**` 聚合哈希不变。

## 提交结果

- 功能与测试提交：`e23f86a fix: restore reliable multi-screenshot shortcuts`
- 治理记录由本审计所在的 `chore: add change 035 governance records` 提交落地，不要求记录自身哈希。

## 提交规划（已授权并按边界执行）

### Commit 1：功能代码与测试

文件：

- `pkg/shortcut/service.go`
- `pkg/shortcut/dispatcher.go`
- `pkg/shortcut/dispatcher_test.go`
- `frontend/src/components/ScreenshotDock.vue`
- `frontend/screenshot_dock_regression_test.mjs`

完整 message：

```text
fix: restore reliable multi-screenshot shortcuts

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

### Commit 2：治理记录

文件：本审计、Change 035 七份档案和 `_dev/changes/INDEX.md`。

完整 message：

```text
chore: add change 035 governance records

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

用户已明确授权本次双提交、稳定版晋升和向 `origin/dev/qsolver-custom` 推送；不授权 `main`、`upstream`、Tag 或 Release。
