# Change 024 验证记录

日期：2026-10-25
结论：普通自动测试、静态检查、Go 编译、Vite 构建、固定 Wails v2.11.0 bindings/集成构建、独立验证程序与数据保护验证通过；Windows GUI 与真实 provider 为 `NEEDS_USER`；未提交。

## 1. 聚焦验证

### 1.1 题库与 App

```text
GOTOOLCHAIN=go1.25.13 go test ./pkg/questionbank ./app -count=1
```

结果：`PASS`

```text
ok  Q-Solver/pkg/questionbank  6.112s
ok  Q-Solver/app               6.934s
```

公开失败重试入口另行聚焦运行：

```text
GOTOOLCHAIN=go1.25.13 go test ./app -run 'TestRetryFailedQuestionBankRecords' -count=1
```

结果：`PASS`，`ok Q-Solver/app 2.008s`。

测试验证无失败项、organizing、closing、缺模型配置等前置拒绝，并通过本地 `httptest` 端点确认异步 worker 只重试失败 Record、保留既有成功结果、成功后清错且汇总转为 complete。

### 1.2 Change 024 前端回归

```text
node frontend/questionbank_history_regression_test.mjs
```

结果：`PASS`，输出 `Question bank history regression tests passed`。

覆盖全失败、部分失败、显式部分导出、失败重试防重复、no_eligible、RawOnly 不修改长期偏好、刷新保持弹窗选择，以及模板按钮和 API 封装。

## 2. 全量普通验证

### 2.1 Go

```text
GOTOOLCHAIN=go1.25.13 go test ./... -count=1
GOTOOLCHAIN=go1.25.13 go vet ./...
GOTOOLCHAIN=go1.25.13 go build ./...
```

结果：全部 `PASS`。全量测试覆盖 `app` 与所有可测试 `pkg/*` 包；`go vet` 和 `go build` 均无输出并以 0 退出。

### 2.2 前端脚本

依次运行 `frontend/*_regression_test.mjs`，结果全部 `PASS`：

1. `clipboard_regression_test.mjs`
2. `history_round_selection_regression_test.mjs`
3. `mobile_store_regression_test.mjs`
4. `numbered_copy_regression_test.mjs`
5. `questionbank_continuation_regression_test.mjs`
6. `questionbank_history_regression_test.mjs`
7. `round_screenshots_regression_test.mjs`
8. `settings_regression_test.mjs`

### 2.3 Vite

```text
npm --prefix frontend run build
```

结果：`PASS`，Vite 4.5.14 转换 129 个模块并完成生产构建。存在既有 `index` chunk 超过 500 kB 的警告，不影响构建退出状态，本 Change 未增加依赖或调整分包策略。

### 2.4 工作树格式

```text
git diff --check
```

结果：`PASS`，无空白错误。Git 仅提示工作区文件未来可能由 LF 转为 CRLF。

## 3. Race

```text
GOTOOLCHAIN=go1.25.13 go test -race ./... -count=1
```

结果：`BLOCKED`。多个测试包均在 Windows 启动阶段返回：

```text
exit status 0xc0000139
```

这与此前 Change 的已知环境问题一致；测试逻辑未运行完成，因此不得写成 PASS，也没有把该环境问题误判为本 Change 的竞态缺陷。

## 4. Wails 与 bindings

项目要求固定 Wails CLI v2.11.0。用户明确授权后执行：

```text
GOTOOLCHAIN=go1.25.13 go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 version
```

结果：`v2.11.0`，`PASS`。PATH 中的 v2.16.0 全程未用于生成或构建。

随后执行固定版本生产构建，未使用清理参数：

```text
GOTOOLCHAIN=go1.25.13 go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -o Q-Solver-change024-organize-failure-recovery-validation.exe
```

结果：`PASS`，Wails 报告 `Skip Bindings=false`、`Clean Bin Dir=false`，在 37.316 秒内完成 bindings、前端、应用资源和 Windows/amd64 程序构建。

Bindings 审计：

- `App.js` 与 `App.d.ts` 新增 `RetryFailedQuestionBankRecords`；
- `models.ts` 新增 `organizeOutcome`、`eligibleOrganize`、`organizedRecords`、`structuredQuestions` 与 `allowPartial`；
- 既有 App API 未被移除；
- Wails runtime 文件无实际差异；
- 生成器在 `models.ts` 空行中加入的尾随制表符已定向清理，`git diff --check` 恢复通过，最终模型文件只有 10 行实际新增。

独立验证程序：

```text
build/bin/Q-Solver-change024-organize-failure-recovery-validation.exe
SHA-256: 1953484b5fb42576f2db013691bdc6190f7a60ce7e1aabbdadad539b3f35d274
大小：约 23 MiB
```

构建前后所有既有 EXE 逐文件 SHA-256 一致；正式程序和旧验证程序均未覆盖。

## 5. 依赖保护

下列哈希与实施前基线一致：

```text
6583908a30917f6864adc5cb7d36d36e56cb5ed6447889fa18e55e1a0784a298  go.mod
d6b12513771dd06b45257192185c07519cedf89a0eab1af7967cc44112d121a4  go.sum
0595e7f0957dd72c62d42dc1a310498d4ae2a3ab6a5e1efb198ac6a2f06b846a  frontend/package.json
827a54f1049d6d143ead365c00c6c0870d6f9eabde57f77ec0926c061f63d725  frontend/package-lock.json
```

结论：未新增依赖，未升级 Wails runtime、Go module 或 npm package。

## 6. TCL 数据保护后检

原目录与 Session：

```text
d2fe5cc342cceb4647bddd6c0548de95e8760316fad803aa7aebdf15b27392cd  旧 TCL_综合笔试.md
b938d2e732f1a948effb608d4d3453667fa720b3496b0da80050536992779fe8  旧 原始记录.md
f93e663bf9c45c3f8de156ac84a18dd25239f632c3c62534c6a010369cba3c83  _记录/a6ef78f5c602e1950b991e3d/session.json
```

恢复稿：

```text
1bc86cb087c489fedc0a47b07325ffa76f5033ba656555f5f18d95ea91d724a9  _02/TCL_综合笔试.md
```

验证结果：

- `_02/原始记录.md` 与旧原始记录逐字节一致；
- source/target assets 均为 25 个文件，文件名集合一致；
- 25 张截图逐文件字节一致；
- 旧整理稿、旧原始记录和 Session 哈希与预检一致；
- 目标前缀下只有原目录和新 `_02` 目录；
- `_02` 统计为语言 9、数字 9、图形 9；18 道文字化、6 道题图待补、3 道材料缺失。

结果：`PASS`。

## 7. 安全验证

- 自动测试仅使用 fake provider、本地 `httptest`、合成 JSON、临时目录和临时 `APPDATA`；
- 未调用真实模型，未上传真实截图，未消费真实 API 额度；
- 脱敏测试向 fake provider 错误中放入伪 secret、私有 URL、绝对路径和正文标记，确认它们不进入持久化用户文案；
- 未读取或修改用户真实 API 配置、密钥、Token 或快捷键配置。

结果：`PASS`。

## 8. 人工验收状态

以下项目未自动宣称完成：

- Windows GUI 中 `pending/complete/partial/all_failed/no_eligible` 的实际布局和按钮可达性：`NEEDS_USER`；
- 用户自愿配置的真实 OpenAI-compatible provider 上的失败分类与失败重试：`NEEDS_USER`；
- 使用新生成的 Change 024 独立 EXE 完成 Windows GUI 行为确认：`NEEDS_USER`。

## 9. Git 与发布状态

- 未暂存；
- 未提交；
- 未推送；
- 未创建 Tag；
- 未创建 GitHub Release 或其他发布。
