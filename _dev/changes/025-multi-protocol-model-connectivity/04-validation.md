# Change 025 验证记录

日期：2026-10-25
状态：普通自动验证完成；Windows GUI 与真实 provider 验收 `NEEDS_USER`；race `BLOCKED`；未提交。

## 1. 自动验证结论

### 1.1 Go 全量测试

```text
GOTOOLCHAIN=go1.25.13 go test ./... -count=1
```

结果：`PASS`。

- `app`
- `pkg/config`
- `pkg/llm`
- `pkg/llm/protocol`
- `pkg/questionbank`
- `pkg/resume`
- `pkg/solution`
- 其余现有包

均通过；无测试包按 Go 正常输出 `[no test files]`。

### 1.2 Go 静态检查与编译

```text
GOTOOLCHAIN=go1.25.13 go vet ./...
GOTOOLCHAIN=go1.25.13 go build ./...
GOTOOLCHAIN=go1.25.13 go mod verify
```

结果：全部 `PASS`；模块校验输出 `all modules verified`。

### 1.3 前端 Node 回归

逐个运行 `frontend/*_regression_test.mjs`，结果全部 `PASS`：

- `clipboard_regression_test.mjs`
- `history_round_selection_regression_test.mjs`
- `mobile_store_regression_test.mjs`
- `model_connectivity_regression_test.mjs`
- `numbered_copy_regression_test.mjs`
- `questionbank_continuation_regression_test.mjs`
- `questionbank_history_regression_test.mjs`
- `round_screenshots_regression_test.mjs`
- `settings_regression_test.mjs`

连接专项覆盖：

- `apiProtocol` 加载、保存与旧配置默认 Chat。
- 打开、加载、保存、切品牌和切协议均不隐式调用模型。
- 模型列表使用完整草稿，不替换手填模型。
- 文字、图片 probe 独立调用、独立状态且不写入配置。
- 四个连接字段变化、A→B→A、关闭和重开后的迟到结果均失效。
- 同 origin 品牌预设保留草稿 Key，不同 origin 清除草稿 Key 和模型。
- 品牌与协议可独立修改。

### 1.4 Vite 构建

```text
npm --prefix frontend run build
```

结果：`PASS`。Vite 4.5.14 完成 129 个模块转换。仅保留既有的单 chunk 超过 500 kB 提示，不影响构建成功。

### 1.5 敏感日志回归

新增 `TestSolveLogsOnlySafeMetadata`，使用伪简历、伪模型答案和伪 SDK 错误 sentinel 验证：

- 不记录完整 system prompt。
- 不记录模型完整答案。
- 不记录原始 SDK 错误文本。
- 仅保留提示词长度、答案长度和固定失败事件等安全元数据。

结果：`PASS`。

## 2. Wails v2.11.0 bindings 与独立验证程序

使用已核验版本为 `v2.11.0` 的缓存 CLI 构建：

```text
GOTOOLCHAIN=go1.25.13 <wails-v2.11.0> build -o Q-Solver-change025-multi-protocol-validation.exe
```

结果：`PASS`。

- 输出：`build/bin/Q-Solver-change025-multi-protocol-validation.exe`
- 大小：`44017664 bytes`
- SHA-256：`589443f464f0e3f0c8adf5c6100beed81d5380d1c2ac98eb54b7f0a0bf6efa37`
- 未使用 PATH 中的 Wails v2.16.0。
- 未启用清理 bin 目录；未覆盖正式程序或旧验证程序。
- 构建前后正式 `Q-Solver.exe` 与 Change 017～024 验证程序哈希保持不变。
- bindings 已包含 `ConnectionDraft`、`ProviderErrorDTO`、`ModelListResult`、`ModelProbeRequest`、`ModelProbeResult`、`ListModelsForDraft` 和 `ProbeModelConnection`；旧连接方法仍存在。
- Wails runtime 文件无内容差异。

bindings 生成只构造 `App` 并解析方法；日志显示配置文件路径，但未执行 `Startup`，因此未调用 `ConfigManager.Load`，未读取配置文件内容。

## 3. race

```text
GOTOOLCHAIN=go1.25.13 go test -race ./pkg/llm ./pkg/config ./pkg/solution ./pkg/resume ./pkg/questionbank ./app -count=1
```

结果：`BLOCKED`。

所有目标包均在 Windows 启动 race 测试二进制时返回：

```text
exit status 0xc0000139
```

这与既有环境阻塞一致，未出现可归因于 Change 025 的 Go race 报告，因此不得记为 `PASS`，也不得写成发现了 Change 025 竞态。

## 4. 差异与依赖门禁

- `git diff --check`：`PASS`。Wails 生成的 `models.ts` 空行尾随空白已清理。
- 主模块 `go` directive：`1.25.0`。
- 固定执行工具链：`go1.25.13`。
- 无 `toolchain` directive。
- 唯一新增直接模块：`github.com/anthropics/anthropic-sdk-go/v2 v2.3.0`。
- `github.com/openai/openai-go` 保持 `v1.12.0`。
- `github.com/wailsapp/wails/v2` 保持 `v2.11.0`。
- npm 依赖版本未变化。
- indirect 变化已在实施记录中按 Anthropic SDK 模块图解释。

## 5. 数据、隐私与调用边界

自动测试和构建期间：

- 未调用真实 OpenAI、DeepSeek、Anthropic 或其他模型服务。
- provider 测试只命中 loopback `httptest`、fake provider，并使用合成 PNG/PDF。
- 未上传用户截图、简历、题库、Prompt 或历史会话。
- 未读取或输出用户 API Key、Token、Mobile Token 或真实配置内容。
- 未修改、删除或迁移题库 `_记录`、截图、旧导出和旧验证程序。
- 未执行 commit、push、Tag 或 Release。

## 6. 人工验收

### 6.1 Windows GUI

状态：`NEEDS_USER`。

使用：

```text
build/bin/Q-Solver-change025-multi-protocol-validation.exe
```

重点检查：

1. API 页品牌与协议为独立控件，协议含 Chat Completions、Responses API、Anthropic Messages。
2. 模型页允许手填模型；刷新失败或列表不含当前模型时不清空手填值。
3. 有 Key 时只显示“已配置 / 未验证”。
4. 文字与图片实测按钮和状态相互独立，并显示费用提示。
5. 切换不同 origin 的品牌预设会清除草稿凭据；同 origin 不清除。
6. 打开、保存、切品牌和切协议不会自动发起模型调用。
7. 模型目录能力明确标注为“模型目录参考，非当前通道实测”。
8. 快速切换、关闭与重开后无迟到结果覆盖。

### 6.2 真实 provider

状态：`NEEDS_USER`。

仅允许用户自行填写配置并主动点击测试。需要分别验证：

- OpenAI-compatible Chat Completions。
- OpenAI Responses API。
- Anthropic 原生 Messages API。
- 用户实际使用的自定义网关及其明确支持的协议。

文字通过不能替代图片通过；404 不能被写成已确认的唯一根因；任何失败都不应自动 fallback、双发或更换协议重试。
