# Change 028 验证记录

日期：2026-10-26
结论：普通自动验证及固定 Wails 独立构建通过；全树空白检查和 prod 旧日志测试存在基线失败；race 环境阻塞；GUI 与真实上游未验证。

## 1. 实际执行环境

- Windows/amd64，直接运行缓存中的 Go 1.25.13。
- 全部 Go 命令使用 `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly`，不允许联网下载依赖或工具链。
- Node v24.14.0，npm 11.9.0，使用已有 node_modules。
- 固定缓存 Wails CLI 实测版本 v2.11.0；未使用 PATH 中其他 Wails。
- 下表中的 `go` 均指缓存的 `golang.org/toolchain@v0.0.1-go1.25.13.windows-amd64/bin/go.exe`；构建时将其 bin 临时放到命令 PATH 首位，不修改系统环境。

## 2. 验证结果

| 命令 / 项目 | 本次结果 | 证据与说明 |
|---|---|---|
| 改动前 `go test ./pkg/llm/... ./pkg/solution ./pkg/mobile ./app -count=1` | PASS | `evidence/baseline-tests-cached.log` |
| 新摘要/传播测试实现前 | RED，预期失败 | `diagnostic-red.log`，缺少 ErrorDiagnostic/DiagnoseError 等待实现符号 |
| Responses 分类矩阵修复前 | RED，预期失败 | `responses-red.log`，限流/无效请求等被误判为服务不可用 |
| 新前端回归实现前 | RED，预期失败 | `frontend-red.log`，没有 error.detail |
| Mobile DOM 实现前 | RED，预期失败 | `mobile-dom-red.log`，没有诊断显示区域 |
| 取消/空响应诊断实现前 | RED，预期失败 | `terminal-red.log`，缺少第二参数 |
| 最终 `go test ./... -count=1` | PASS | `full-go-test-final.log`，15 个有测试包通过，另有无测试包 |
| 最终 `go vet ./...` | PASS | `vet-final.log`，退出 0 |
| 最终 `go build ./...` | PASS | `build-final.log`，退出 0 |
| 14 个 `node frontend/*_regression_test.mjs` 逐个执行 | PASS | `frontend-regressions-final.log` |
| `node pkg/mobile/web_dom_test.mjs` | PASS | 同上；安全文字显示、旧快照、清屏及迟到帧 |
| `npm --prefix frontend run build` | PASS | `frontend-build.log`；既有大于 500 kB chunk 提示 |
| 固定 Wails 独立 prod 构建 | PASS | `wails-build-final.log` |
| prod 诊断专项 | PASS | `prod-diagnostics-tests.log`，4 包通过，证明诊断不依赖 logger |
| prod 全套相关包测试 | FAIL，基线测试假设冲突 | 见第 4 节；未修改旧测试 |
| 诊断相关 race 测试 | BLOCKED | `race-tests.log`，4 包启动失败 0xc0000139 |
| 全工作区 `git diff --check` | FAIL，原有空白问题 | 85 处，全部在未改动的 models.ts |
| 本次增量空白检查 | PASS | 逐文件入口版本差异检查及新增文件检查；见边界审计 |
| Windows WebView2 人工操作 | 未执行 / NEEDS_USER | 未启动程序，不以 SSR/DOM 测试冒充 GUI |
| 真实 GPT、Claude、CCR API | 未执行 / NEEDS_USER | 未消费真实 API 额度 |

## 3. 覆盖范围

- 已知 ProviderError、多层包装、普通错误、typed nil、未知类别、未知状态、可/不可重试。
- 安全字符串白名单、固定友好文案、SDK cause 和 Message 哨兵不能进入摘要。
- Solver → questionSolveCapture → 带 epoch/sessionId/recordId 的桌面封套；第一参数始终为字符串。
- 题库仍保存 failed，不写入新详情或测试秘密；无 schema 迁移。
- Mobile Bridge 快照复制隔离、重连初始快照、旧字符串、成功/新题/清屏清错、旧代际拒绝。
- 真实 loopback HTTP/SSE 把 401 安全摘要传至 JSON，旧客户端只读取 Status/Error 仍正常。
- 原有 Bridge → SSE → DOM 的双题无刷新集成测试继续通过。
- 前端状态接受本轮错误，拒绝终态后重复、旧轮及迟到错误；旧事件缺详情显示暂无信息。
- Vue 编译器编译真实 ErrorDiagnostics 组件并 SSR 渲染；桌面两种错误区域均接入该组件。
- 手机 DOM 白名单和 textContent 输出；测试秘密和任意额外字段不进入诊断文本。
- Responses 16 组业务码 × 非流式/failed event/error event，共 48 场景；每次恰好一个请求。
- Responses 实际 HTTP 401/403/429/503 × 流式/非流式，共 8 场景；真实状态保留。
- 原 incomplete、refusal、empty、缺终态和流中断回归继续通过。

## 4. 失败、阻塞与处理

### 工具链选择器初次失败

初次设置 `GOTOOLCHAIN=go1.25.13` 且 `GOSUMDB=off` 时，Go 自动工具链验证返回：

```text
go: golang.org/toolchain@v0.0.1-go1.25.13.windows-amd64: verifying module: checksum database disabled by GOSUMDB=off
```

没有业务测试执行。改为直接调用已经缓存的 Go 1.25.13，`GOTOOLCHAIN=local`，保持离线；相关基线及最终测试通过。未下载或安装。

### prod 全套相关包测试

实际命令：`go test -tags prod ./pkg/llm ./pkg/solution ./pkg/mobile ./app -count=1`。

```text
ok Q-Solver/pkg/llm
--- FAIL: TestSolveLogsOnlySafeMetadata
    solution_test.go:614: missing safe log metadata "system 提示词长度" in ""
FAIL Q-Solver/pkg/solution
ok Q-Solver/pkg/mobile
ok Q-Solver/app
```

该旧测试与 logger_prod.go 均未被本次修改，测试要求日志出现，prod 则刻意关闭 logger。普通模式全量通过；不通过改旧断言伪造 prod 全套通过。

单独执行以下诊断专项，4 包均 PASS：

```text
go test -tags prod ./pkg/llm ./pkg/solution ./pkg/mobile ./app -run 'TestErrorDiagnostic|TestUnknownFailureDiagnostic|TestResponsesFailure|TestResponsesHTTPFailure|TestSolveErrorCarries|TestSolveDiagnostic|TestMobileErrorDiagnostic' -count=1
```

### race 环境阻塞

```text
go test -race ./pkg/llm ./pkg/solution ./pkg/mobile ./app -run 'TestErrorDiagnostic|TestResponsesFailure|TestSolveErrorCarries|TestSolveDiagnostic|TestMobileErrorDiagnostic' -count=1
exit status 0xc0000139
FAIL Q-Solver/pkg/llm
exit status 0xc0000139
FAIL Q-Solver/pkg/solution
exit status 0xc0000139
FAIL Q-Solver/pkg/mobile
exit status 0xc0000139
FAIL Q-Solver/app
```

未获得有效 race 结果，不记 PASS。与历史环境表现一致。

### 全树空白失败

`frontend/wailsjs/go/models.ts` 共 85 处 trailing whitespace，例如第 2、8、12、952、956 行。该文件 SHA-256 与任务入口一致；未生成或修改 bindings。保留原状，不为消除失败覆盖用户工作。

## 5. 独立构建命令

先单独执行 Vite build，再使用：

```text
<wails-v2.11.0> build -s -skipbindings -m -nosyncgomod -skipembedcreate -tags prod -o Q-Solver-change028-llm-error-diagnostics.exe
```

日志确认 Skip Bindings=true、Skip Frontend=true、Clean Bin Dir=false、Tags=[prod]。无新增公开 App API，无需重生成 bindings。编译成功不是 GUI 验收；本次未运行程序。

## 6. 隐私与网络

自动测试只使用 fake Provider、本机 httptest、合成数据、临时目录及临时 APPDATA。无真实模型请求、凭据读取、用户配置迁移或旧题库修改。没有新增含 SDK 原始文本的日志；生产诊断不暴露请求正文、截图、URL、Token 或 Authorization。
