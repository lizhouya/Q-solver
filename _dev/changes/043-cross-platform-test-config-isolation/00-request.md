# 变更请求 — 00

| 字段 | 值 |
|---|---|
| 变更编号 | `043-cross-platform-test-config-isolation` |
| 提出人 | 项目维护者 |
| 提出日期 | 2026-11-12 |
| 关联证据 | GitHub Actions run `43212159818` |
| 状态 | `进行中` |

## 1. 背景与问题

`v1.1.0-rc.4` 的 Release workflow 在 Windows 完整通过，但 macOS Apple Silicon 的应用构建成功后，`app` 包回归测试失败；Intel job 被取消，Create Release 被跳过。失败为：

- `TestKeylessSolveInternalReachesLocalTemplateServer`: `keyless solve did not complete`
- `TestRetrySolveReusesBackendImagesAndSameQuestionBankRecord`: `solve_retry_prepare_failed / 请确认是否忽略未解析的简历`

根因是 app 测试只设置 Windows 的 `APPDATA`。macOS 的 `os.UserConfigDir()` 使用 `HOME/Library/Application Support`，导致测试把配置写进 runner 的真实用户目录，后续测试再读到前序残留。Windows 使用 `APPDATA`，所以没有复现。

## 2. 目标

- 建立 app 测试共用的跨平台用户配置目录隔离 helper。
- 同时设置 Windows、Linux 和 macOS 相关环境变量，使 ConfigManager 永远落在每个测试自己的 `t.TempDir()`。
- 替换 app 测试中只设置 `APPDATA` 的调用点，消除跨测试配置污染。
- 不修改生产代码、配置格式、业务行为、依赖或网络边界。
- 在本地通过聚焦与全量门禁，并通过后续 macOS CI 证明修复有效。

## 3. 验收标准

- [ ] helper 设置 `APPDATA`、`LOCALAPPDATA`、`XDG_CONFIG_HOME`、`HOME`、`USERPROFILE` 到同一隔离根目录。
- [ ] 回归测试证明 `os.UserConfigDir()` 位于隔离根目录内。
- [ ] app 中所有仅设置 `APPDATA` 的配置测试改用统一 helper，不再污染 macOS runner 用户目录。
- [ ] 两个 rc.4 失败测试聚焦通过。
- [ ] `go test ./app -count=1`、`go test ./... -count=1`、`go vet ./...`、`go build ./...` 通过。
- [ ] 前端回归与 Vite build 通过，确认无跨层副作用。
- [ ] 后续 RC 的 Windows、macOS arm64、macOS amd64 和 Create Release 均成功；失败不标为 PASS。

## 4. 非目标 / 明确不做

- 不修改 ConfigManager 的生产路径逻辑。
- 不修改解题、题库、Mobile 或 Provider 业务行为。
- 不删除、移动或覆写已公开的 `v1.1.0-rc.4` Tag。
- 不手工创建 rc.4 Release 掩盖 CI 失败。
- 不新增依赖，不修改用户配置，不输出或提交 secret。

## 5. 影响面预估

- 测试文件：新增 `app/test_config_env_test.go`；修改直接构造 ConfigManager 的相关 `app/*_test.go`。
- 治理文档：Change 042 失败补录、Change 043 七份档案、审计、INDEX；验证完成后准备 rc.5 文档。
- 风险：环境变量隔离 helper 使用不完整，或某测试依赖真实用户目录而被暴露。

## 6. 附带约束

- 先写红灯测试证明跨平台隔离合同，再实现 helper 和迁移调用点。
- 不调用真实 API；本地 HTTP 测试继续只访问 `httptest.Server`。
- 功能/测试提交与治理提交分离；所有暂存使用显式路径。
- rc.4 保持失败 Tag 事实，修复采用后续 rc.5 前进发布。
