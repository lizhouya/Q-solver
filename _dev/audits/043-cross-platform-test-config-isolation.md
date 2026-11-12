# Change 043 跨平台测试配置隔离审计

## 结论

rc.4 的 macOS Apple Silicon 失败是 app 测试配置目录未跨平台隔离，不是生产应用构建失败。测试只设置 Windows `APPDATA`，macOS 仍使用 `/Users/runner/Library/Application Support/Q-Solver/config`，导致前序“未解析简历”配置污染后续 keyless solve 与 retry solve 测试。

## 证据

- Run：`43212159818`
- Windows：Build App / Regression Tests / Artifact 全部 PASS。
- macOS arm64：Build App PASS；Regression Tests 中 `Q-Solver/app` FAIL。
- 失败一：`TestKeylessSolveInternalReachesLocalTemplateServer`，本地模板 server 未完成调用。
- 失败二：`TestRetrySolveReusesBackendImagesAndSameQuestionBankRecord`，收到 `solve_retry_prepare_failed / 请确认是否忽略未解析的简历`。
- Intel：CANCELLED；Create Release：SKIPPED；rc.4 Release 不存在。

## 根因链

`config.NewConfigManager()` 在构造时调用 `os.UserConfigDir()`。现有 app 测试多处只设置 `APPDATA`：Windows 有效，macOS 和 Linux 不完整。macOS runner 因而共享真实 HOME 配置文件，测试顺序形成状态泄漏。

## 设计边界

- 新增测试专用 helper，同时设置 `APPDATA`、`LOCALAPPDATA`、`XDG_CONFIG_HOME`、`HOME`、`USERPROFILE`。
- helper 返回隔离根，合同测试用 `filepath.Rel` 验证 `os.UserConfigDir()` 确实位于根内。
- 迁移 app 中所有仅设置 `APPDATA` 的 ConfigManager 测试；settings 的重复三变量设置也统一收敛。
- 不修改生产 ConfigManager、业务代码、依赖或 workflow。

## 本地验证与提交

- 配置目录合同先按预期因 helper 未定义红灯，最小实现后转绿。
- 两个 rc.4 失败测试、`go test ./app`、全仓 Go、vet、build、mod verify、19 个前端回归、Mobile DOM 与 Vite build 通过。
- race 仍在 Windows 启动阶段返回 `0xc0000139`，状态 `BLOCKED`。
- 功能/测试提交：`532c828 fix: isolate test config directories across platforms`。
- 生产代码、依赖、workflow、frontend/dist 和二进制均未变化。
- rc.5 body、相对链接、敏感信息、diagnostics、`git diff --check` 与 Tag 占用检查通过；本地/远程 rc.5 Tag 均不存在。

## rc.5 远程结果

- 功能提交 `532c828` 与治理/发布准备提交 `8734b3e` 已推送；annotated Tag `v1.1.0-rc.5` 指向 `8734b3e`。
- run `43232814169` 与 `43246684317` 的 Windows job 均完整 PASS。
- 两个 run 的 macOS arm64/amd64 job 均未获分配 runner：`runner_id=0`、runner name 为空、steps 为空，约 15 分钟后 CANCELLED。
- Create Release 均 SKIPPED；rc.5 Release、附件和 `SHA256SUMS` 均不存在。
- 该状态是 GitHub-hosted macOS runner 阻塞，不是 Change 043 测试失败；也不是 macOS PASS。
- 当前实际公开 Release 仍为 Windows-only `v1.1.0-rc.3`。

## 发布边界

rc.4 Tag 已公开且 run 失败，保持不可变。Change 043 使用后续 `v1.1.0-rc.5` 前进发布，不手工创建 rc.4 Release，不删除或移动 rc.4 Tag。rc.5 正文必须在 Tag 对应提交中存在。

## 提交边界

功能/测试提交仅允许：

- `app/test_config_env_test.go`
- `app/solve_persistence_test.go`
- `app/solve_preflight_test.go`
- `app/mobile_commands_test.go`
- `app/questionbank_test.go`
- `app/settings_test.go`

治理提交仅允许 Change 042/043、对应 audit、INDEX 和后续 rc.5 发布文档。二进制、配置、secret 和用户路径不得进入提交。
