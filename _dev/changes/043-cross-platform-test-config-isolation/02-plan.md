# 实施方案 — 02

## 1. 方案概述

在 `app` 测试包新增单一 helper `isolateUserConfigDirs(t)`：创建一个 `t.TempDir()`，将 Windows、Linux 和 macOS 使用的配置/用户目录环境变量统一指向该根目录。先用红灯测试锁定 `os.UserConfigDir()` 必须位于隔离根内，再替换所有只设置 `APPDATA` 的 app 测试和现有 settings 测试的重复隔离代码。生产代码不变。

## 2. 分步实施计划

| 步骤 | 操作 | 涉及文件 | 预期产出 | 完成标志 |
|---|---|---|---|---|
| 1 | 写跨平台配置目录合同测试 | `app/test_config_env_test.go` | 调用尚不存在 helper 的编译红灯 | 失败原因仅为 helper 未定义 |
| 2 | 最小实现 helper | 同上 | 五个环境变量指向同一临时根 | 合同测试通过 |
| 3 | 迁移解题测试 | `solve_preflight_test.go`、`solve_persistence_test.go` | 两个 rc.4 失败测试独立隔离 | 聚焦测试通过 |
| 4 | 迁移其他 app ConfigManager 测试 | `mobile_commands_test.go`、`questionbank_test.go`、`settings_test.go` | 消除同类跨平台污染点 | `go test ./app` 通过 |
| 5 | 全量验证 | 全仓 | Go、前端、Vite、diff、diagnostics 结果 | 必要门禁真实记录 |
| 6 | 独立功能/测试提交 | 上述测试文件 | `fix: isolate test config directories across platforms` | 显式白名单提交 |
| 7 | 准备后续 RC | 发布治理文档 | rc.5 前进发布 | 不移动 rc.4 Tag |

## 3. 允许修改的测试文件

- 新增 `app/test_config_env_test.go`
- 修改 `app/solve_persistence_test.go`
- 修改 `app/solve_preflight_test.go`
- 修改 `app/mobile_commands_test.go`
- 修改 `app/questionbank_test.go`
- 修改 `app/settings_test.go`

生产文件不在白名单中。若实现需要生产代码，必须停止并重新审计范围。

## 4. 测试与验证计划

### TDD

1. `go test ./app -run '^TestIsolateUserConfigDirsControlsPlatformConfigPath$' -count=1`
   - 红灯：`undefined: isolateUserConfigDirs`。
2. 实现 helper 后重跑，预期 PASS。
3. 聚焦 rc.4 两个失败测试：
   - `TestKeylessSolveInternalReachesLocalTemplateServer`
   - `TestRetrySolveReusesBackendImagesAndSameQuestionBankRecord`
4. 连续运行 app 包，确保没有顺序污染。

### 全量门禁

- `go test ./app -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `go mod verify`
- 19 个 `frontend/*_test.mjs`
- `node pkg/mobile/web_dom_test.mjs`
- `npm --prefix frontend run build`
- VS Code diagnostics
- `git diff --check`

### 远程验收

后续 rc.5 workflow 必须由三个平台 job 和 Create Release 全部 PASS 才算发布成功。macOS arm64 聚焦失败必须消失；Intel 不得再处于 cancelled。

## 5. 回滚预案

- 测试 helper 导致本地回归时，仅定向还原 Change 043 测试文件。
- 已公开 rc.4 Tag 不移动、不删除；失败事实保留。
- rc.5 失败则继续保留日志，未经额外授权不删除公开对象，使用后续 RC 前进修复。

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 只修两个失败测试，其他测试继续污染 | 后续 macOS CI 偶发失败 | 迁移 app 中所有仅设置 APPDATA 的调用点 |
| helper 在某平台未控制 UserConfigDir | 隔离合同虚假通过 | 直接调用 `os.UserConfigDir()` 并验证路径位于临时根 |
| 路径字符串前缀误判 | 等价路径或前缀冲突 | 使用 `filepath.Rel` 判断目录包含关系 |
| 测试变更误触生产行为 | 范围扩大 | 白名单只含 `_test.go`；生产文件零改动 |
| rc.4 被覆写 | 发布历史不可追踪 | 使用 rc.5，不移动 rc.4 Tag |
