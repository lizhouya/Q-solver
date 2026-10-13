# Change 007 ClickThrough Toggle Feedback Report

> **变更**：`007-clickthrough-toggle-feedback`  
> **日期**：2026-10-13  
> **阶段**：Implementation Completed / Commit Boundary Audit  
> **提交状态**：未暂存、未提交、未推送

## 1. 原 clickthrough 调用链

```text
默认 F10 或用户自定义按键
  → Shortcut Manager 解析为 action="clickthrough"
  → shortcut.Service.handleTrigger
  → ServiceDelegate.ToggleClickThrough
  → app.App.ToggleClickThrough
  → state.StateManager.ToggleClickThrough
  → platform.SetClickThrough
  → 成功后更新 ClickThroughState
  → EmitEvent("click-through-state", enabled)
  → Frontend 同步 pointer-events
```

默认 F10 只存在于 Config 的初始 KeyBinding。反馈实现只识别 `clickthrough` action 和 State 返回结果，没有写死 F10。

## 2. 复用的现有提示机制

复用既有统一链路：

```text
Shortcut Service
  → delegate.EmitEvent("toast", message)
  → App.EmitEvent / Wails EventsEmit
  → frontend App.vue on("toast")
  → ui.showToast(message)
```

`ui.showToast` 默认 2000ms 后淡出，不抢焦点、不阻塞快捷键。未新增前端组件、第三方库或第二套 notification system。

## 3. 修改文件

业务与测试：

- `pkg/state/manager.go`
- `pkg/state/manager_test.go`
- `pkg/shortcut/delegate.go`
- `pkg/shortcut/service.go`
- `pkg/shortcut/service_test.go`
- `app/window.go`
- `app/phase4_test.go`（仅 fake delegate 接口适配）

治理：

- `_dev/changes/007-clickthrough-toggle-feedback/00-request.md`
- `_dev/changes/007-clickthrough-toggle-feedback/01-current-state.md`
- `_dev/changes/007-clickthrough-toggle-feedback/02-plan.md`
- `_dev/changes/007-clickthrough-toggle-feedback/03-implementation.md`
- `_dev/changes/007-clickthrough-toggle-feedback/04-validation.md`
- `_dev/changes/007-clickthrough-toggle-feedback/05-decisions.md`
- `_dev/changes/007-clickthrough-toggle-feedback/06-rollback.md`
- `_dev/changes/INDEX.md`

## 4. 开启提示行为

`platform.SetClickThrough(..., true)` 成功后，State 更新真实状态并返回 `true, nil`。Shortcut 随后发出：

```text
鼠标穿透：已开启
```

提示不会关闭、暂停或临时取消穿透；页面继续按既有方式渲染 Toast。

## 5. 关闭提示行为

`platform.SetClickThrough(..., false)` 成功后，State 更新真实状态并返回 `false, nil`。Shortcut 随后发出：

```text
鼠标穿透：已关闭
```

文案来自真实成功返回状态，不通过触发次数或旧状态推断。

## 6. 失败行为

HWND 未初始化或平台调用失败时：

- State 返回 error 和未改变的当前状态。
- 不更新 ClickThroughState。
- 不发 `click-through-state` 成功事件。
- Shortcut 不显示“已开启”或“已关闭”。
- 复用现有 Toast 显示 `鼠标穿透切换失败`。

## 7. 自动测试结果

| 验证 | 结果 | 说明 |
|---|---|---|
| T-CT-01 false → true | ✅ PASS | payload 精确为 `鼠标穿透：已开启` |
| T-CT-02 true → false | ✅ PASS | payload 精确为 `鼠标穿透：已关闭` |
| T-CT-03 底层失败 | ✅ PASS | 只发失败 Toast；State 失败时状态不变且无成功事件 |
| T-CT-04 单次调用 | ✅ PASS | `clickthrough` 只调用一次 `ToggleClickThrough` |
| T-CT-05 自定义快捷键 | ✅ PASS | 使用 `Ctrl+K` 测试绑定，反馈仍由 action/state 决定 |
| 聚焦测试重复 50 次 | ✅ PASS | `pkg/shortcut`、`pkg/state` 均通过 |
| `go test ./... -count=1` | ✅ PASS | 全部 Go package 通过 |
| `go build ./...` | ✅ PASS | exit 0 |
| `npm --prefix frontend run build` | ✅ PASS | Vite 118 modules；仅既有 chunk > 500 kB 警告 |
| `go vet ./pkg/state/... ./app/...` | ✅ PASS | exit 0 |
| `go vet ./pkg/shortcut/...` | ⚠️ 既有警告 | 未修改的 `shortcut_windows.go:122,155` 报 possible misuse of unsafe.Pointer |
| `go vet -unsafeptr=false ./pkg/shortcut/...` | ✅ PASS | 排除既有 unsafeptr 警告后，其余分析通过 |
| `git diff --check` | ✅ PASS | 无空白错误；仅工作区 LF→CRLF 提示 |

## 8. Windows 真机测试结果

当前会话运行在 Windows 11，但没有可替用户进行视觉确认和真实鼠标点击的交互式桌面验收能力，因此以下项目均**未执行 / 待人工真机验收**，不误报为 PASS：

| ID | 状态 | 待验收内容 |
|---|---|---|
| T-WIN-01 | ⏳ 待人工 | 启用穿透后穿透生效，Toast 仍可见 |
| T-WIN-02 | ⏳ 待人工 | 再次触发恢复可操作，并显示关闭提示 |
| T-WIN-03 | ⏳ 待人工 | 连续切换至少 5 次，无反转、重复或崩溃 |
| T-WIN-04 | ⏳ 待人工 | 改为非 F10 快捷键后提示仍正常 |

## 9. Git diff 边界

- Branch：`dev/qsolver-custom`
- Baseline HEAD：`aa35b55 chore: add change 006 governance records`
- 所有业务差异均属于 State/App/Shortcut 返回契约、Toast 反馈和直接测试。
- Frontend 源码未修改。
- `go.mod`、`go.sum`、`frontend/wailsjs/runtime/*` 无修改。
- 未运行 `wails dev`。
- 未执行 `git add`、`git commit` 或 `git push`。

## 10. 是否发现其他问题

发现一项既有静态检查问题：`pkg/shortcut/shortcut_windows.go` 两处 `unsafe.Pointer` 用法会使标准 `go vet ./pkg/shortcut/...` 返回非零。该文件未在 Change 007 中修改，此问题与反馈逻辑无关，未扩大范围修复。

未发现需要协议变更、新依赖、配置、数据库或持久化才能完成本目标的问题。

## 11. 是否达到 Commit Boundary Audit 条件

**自动化与差异边界已达到 Commit Boundary Audit 条件**：代码、测试、构建、Frontend build、范围和污染检查均完成；标准 Shortcut vet 的非零结果已定位为未修改文件中的既有警告并如实记录。

Windows T-WIN-01～04 仍需人工真机验收，因此产品层面的最终视觉/交互验收尚未关闭。当前可以停在提交边界等待后续明确指令，但不得把待人工项记录为已通过。

## Final Windows Validation

> **执行日期**：2026-10-13  
> **平台**：Windows 11 Pro  
> **Wails CLI**：v2.16.0  
> **测试构建**：`wails build -debug -windowsconsole`

### 构建与实例基线

- Branch：`dev/qsolver-custom`
- Baseline HEAD：`aa35b55 chore: add change 006 governance records`
- Wails build：**PASS**，生成 `build/bin/Q-Solver.exe`（34,910,208 bytes）。
- 启动日志确认 WebView2、配置加载、GhostMode、Mobile Server 和 System Tray 正常初始化。
- 用户当前配置：`clickthrough=F10`、`toggle_window=Ctrl+Shift+H`；仅通过项目 ConfigManager 提取这两个字段，未输出其他配置或 secret。
- 用户当前 `OutputMode=mobile`，实例启动后按既有策略自动隐藏；使用实际 `Ctrl+Shift+H` 恢复窗口后进入验证，不修改 OutputMode。
- 测试结束后通过正常 `CloseMainWindow()` 退出，最终无 Q-Solver 进程残留。

### Wails v2.16 生成污染

构建后检测到以下生成噪音：

- `go.mod`
- `frontend/wailsjs/runtime/*`
- `frontend/wailsjs/go/app/*`

差异确认为 Wails CLI v2.16 版本更新与 bindings 重生成。仅恢复上述明确生成文件，没有恢复或覆盖 Change 007 的 App/State/Shortcut 源码。最终 `git status` 中无 Wails 构建污染。

### 真机结果矩阵

| 项目 | 最终状态 | 真实观察结果 |
|---|---|---|
| T-WIN-01 开启穿透 | **NEEDS_USER** | Win32 `WS_EX_TRANSPARENT` 从 false 变为 true，窗口保持可见，DisplayAffinity 保持 `0x11`，前台窗口未改变，日志为“鼠标穿透已启用”；GhostMode 防录屏使当前会话无法可靠观察 Toast 和真实鼠标点击命中 |
| T-WIN-02 关闭穿透 | **NEEDS_USER** | Win32 `WS_EX_TRANSPARENT` 从 true 变为 false，窗口保持可见，DisplayAffinity 保持 `0x11`，前台窗口未改变，日志为“鼠标穿透已禁用”；Toast 视觉和真实鼠标点击恢复仍需人工确认 |
| T-WIN-03 连续切换 | **NEEDS_USER** | 连续 5 次真实状态为 ON/OFF/ON/OFF/ON；每次窗口可见、进程响应、DisplayAffinity=`0x11`，日志每次一组切换结果且无 panic；每次 Toast 文案、无重复和显示时序需人工观察 |
| T-WIN-04 自定义快捷键 | **NEEDS_USER** | 未修改用户配置；设置界面的冲突检查、临时保存、实际新按键触发及恢复原配置需要人工交互，未伪造为 PASS |
| CROSS-STATE TEST | **PASS** | ClickThrough ON 时 Ctrl+Shift+H 成功 Hide/Show，隐藏和恢复后 ClickThrough 仍为 ON；随后关闭 ClickThrough，WindowVisibility 仍为 Visible；GhostMode DisplayAffinity 全程保持 `0x11` |
| 失败提示路径 | **AUTOMATED PASS** | 自动测试确认 error 时只发送“鼠标穿透切换失败”，不发送开启/关闭成功文案 |

### 最终回归

| 命令 | 结果 |
|---|---|
| `go test ./... -count=1` | ✅ PASS |
| `go build ./...` | ✅ PASS |
| `npm --prefix frontend run build` | ✅ PASS，仅既有 chunk > 500 kB 警告 |
| `git diff --check` | ✅ PASS，仅 LF→CRLF 工作区提示 |
| 进程清理 | ✅ PASS，`Q-Solver process: NOT RUNNING` |
| Wails 污染检查 | ✅ PASS，无 `go.mod`、`go.sum`、`frontend/wailsjs/*` 残留修改 |

### 最终判定

- **确定性 Change 007 Bug**：未发现。
- **自动化 Commit Boundary Audit**：保持通过。
- **最终 Commit 条件**：**尚未完全满足**。T-WIN-01～04 中涉及 Toast 视觉、真实点击和设置 UI 的人工观察项仍为 `NEEDS_USER`，因此不标记 `READY FOR COMMIT BOUNDARY`。
- **Git 操作**：未执行 `git add`、`git commit` 或 `git push`。
