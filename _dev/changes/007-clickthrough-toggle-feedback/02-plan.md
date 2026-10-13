# 实施计划 — 02 · 007-clickthrough-toggle-feedback

> **阶段**：设计完成，业务代码尚未修改

## 方案

### 1. State 返回真实结果

将 `StateManager.ToggleClickThrough()` 调整为 `(bool, error)`：

1. 在既有 `windowMu` 临界区内读取当前 ClickThroughState。
2. HWND 未初始化时返回当前状态和 error，不更新状态、不发成功事件。
3. 调用既有 `platform.SetClickThrough(hwnd, newState)`。
4. 平台失败时返回当前状态和包装后的 error，不更新状态、不发成功事件。
5. 平台成功后更新 `windowState.ClickThrough`，继续发出既有 `click-through-state`，返回真实 `newState, nil`。

不改变锁、平台调用、状态更新或既有 `click-through-state` 的成功语义。

### 2. App 与 Delegate 透传

- `App.ToggleClickThrough()` 原样返回 State 的 `(bool, error)`。
- `shortcut.ServiceDelegate.ToggleClickThrough()` 使用相同返回契约。
- 只适配受该接口影响的测试 fake；不改变其他 delegate 方法。

### 3. Shortcut 复用现有 Toast

`handleTrigger("clickthrough")` 仍只调用一次 delegate：

- `enabled, err := s.delegate.ToggleClickThrough()`。
- `err != nil`：记录日志并 `EmitEvent("toast", "鼠标穿透切换失败")`，不发成功文案。
- `enabled == true`：`EmitEvent("toast", "鼠标穿透：已开启")`。
- `enabled == false`：`EmitEvent("toast", "鼠标穿透：已关闭")`。

该逻辑只依赖 action 和真实返回状态，与 F10 或用户自定义按键无关。Toast 不修改 pointer-events，不为提示临时关闭穿透。

## 测试计划

| ID | 自动测试 |
|---|---|
| T-CT-01 | mock 返回 `true, nil`，断言一次调用及 toast payload `鼠标穿透：已开启` |
| T-CT-02 | mock 返回 `false, nil`，断言一次调用及 toast payload `鼠标穿透：已关闭` |
| T-CT-03 | mock 返回 error，断言无成功文案，只发失败 toast；State HWND 缺失时返回 error 且不发状态事件 |
| T-CT-04 | 既有 action 回归测试继续断言 `clickthrough` 只调用一次 |
| T-CT-05 | 用非 F10 自定义 KeyBinding 构造 Service 后直接触发 `clickthrough` action，断言提示仍由 action/state 决定 |

## 文件范围

| 文件 | 计划 |
|---|---|
| `pkg/state/manager.go` | 返回真实状态与失败 error |
| `pkg/state/manager_test.go` | 覆盖 HWND 缺失失败语义 |
| `pkg/shortcut/delegate.go` | 更新最小接口契约 |
| `pkg/shortcut/service.go` | 基于真实结果发既有 toast |
| `pkg/shortcut/service_test.go` | T-CT-01～T-CT-05 payload/call 测试 |
| `app/window.go` | 透传 State 结果 |
| `app/phase4_test.go` | 适配既有测试 fake 接口 |

Frontend 不修改：现有 `toast` listener 和自动消失 UI 已满足需求。

## 风险与控制

| 风险 | 控制 |
|---|---|
| 失败时误报成功 | error 分支先返回，只发失败文案 |
| 状态提示反转 | 文案只使用 State 成功后的返回值，不自行取反 |
| 快捷键重复触发 | 单次调用保存在局部结果中，测试严格断言一次 |
| 穿透后 Toast 不可见 | Toast 由既有页面渲染；不修改/取消穿透，真机验收确认可见性 |
| Wails 绑定噪音 | 不运行 `wails dev`，不修改或重新生成 `frontend/wailsjs/` |
| 范围蔓延 | 不修改 Frontend、Config 或其他窗口状态功能 |

## 验证

执行 `go test ./...`、`go build ./...`、`go vet ./pkg/state/... ./pkg/shortcut/... ./app/...`、`npm --prefix frontend run build`、`git diff --check`、`git diff --name-only` 和生成污染检查。Windows 真机 T-WIN-01～04 若当前自动环境无法交互操作，则明确标记待人工验收，不误报通过。

## 回滚

恢复三个层级的 `ToggleClickThrough()` 原签名，移除 Shortcut toast 分支和新增测试即可。没有配置、数据迁移、协议、依赖或持久化变化。
