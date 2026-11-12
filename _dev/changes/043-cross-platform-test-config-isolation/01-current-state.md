# 现状快照 — 01

## 1. 涉及文件清单

| 文件 | 当前职责 | 当前问题 |
|---|---|---|
| `app/solve_persistence_test.go` | 提供 `configuredSolveManager` | 只设置 `APPDATA` |
| `app/solve_preflight_test.go` | 简历确认与无密钥本地模板测试 | 前置测试只设置 `APPDATA`，macOS 写入真实配置目录 |
| `app/solve_retry_test.go` | 原图重试与任务状态测试 | 通过受污染 helper 读到前序 Resume 配置 |
| `app/mobile_commands_test.go` | Mobile 命令/显示代际测试 | 一处只设置 `APPDATA` |
| `app/questionbank_test.go` | 题库整理与失败重试测试 | 两处只设置 `APPDATA` |
| `app/settings_test.go` | 设置加载/失败与连接 Profile 测试 | 已分别设置 `APPDATA`、`XDG_CONFIG_HOME`、`HOME`，但未复用 helper |

## 2. 关键代码现状

`configuredSolveManager` 当前仅执行：

```go
t.Setenv("APPDATA", t.TempDir())
manager := config.NewConfigManager()
```

`config.NewConfigManager()` 通过 `os.UserConfigDir()` 计算路径。平台差异：

- Windows：读取 `APPDATA`。
- Linux：优先读取 `XDG_CONFIG_HOME`。
- macOS：读取 `HOME` 并追加 `Library/Application Support`。

因此只设置 `APPDATA` 不是跨平台隔离。

## 3. rc.4 CI 证据

Run `43212159818`：

- Build Windows：PASS，包含 Build App 与 Regression Tests。
- Build macOS (Apple Silicon)：Build App PASS，Regression Tests FAIL。
- Build macOS (Intel)：CANCELLED。
- Create Release：SKIPPED。
- rc.4 Release：不存在。

失败日志显示 ConfigManager 使用 `/Users/runner/Library/Application Support/Q-Solver/config`，随后：

- keyless solve 未完成。
- retry start 返回 `solve_retry_prepare_failed`，文案要求确认忽略未解析简历。

## 4. 因果链

```text
macOS runner
  → APPDATA 对 os.UserConfigDir 无效
  → TestTriggerSend... 写入真实 HOME 配置（ResumePath 有值、ResumeContent 为空）
  → configuredSolveManager 后续从同一路径 Load
  → keyless/retry 测试继承残留配置
  → preflight 或 Provider 配置被污染
  → app 包失败
```

## 5. 依赖与构建现状

- 不需要新增依赖。
- rc.4 Windows 使用 Go 1.25、Node 22、Wails v2.11.0 构建和回归通过。
- macOS arm64 应用本体构建通过，失败仅位于 Go Regression Tests。
- 本地 Change 040/041 普通全量门禁此前通过。

## 6. 已知现状问题

- Windows race 仍因 `0xc0000139` BLOCKED，与本次 macOS 配置污染是不同问题。
- rc.4 Tag 已公开且必须保持不可变。
- Tag push 未自动产生 run，本次 rc.4 通过已有 `workflow_dispatch` 回退路径调度；该触发行为不在 Change 043 修改范围。
