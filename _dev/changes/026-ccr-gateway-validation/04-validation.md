# Change 026 验证记录

日期：2026-10-25
状态：普通自动验证 `PASS`；Windows race `BLOCKED`；Windows GUI 与真实 CCR `NEEDS_USER`；未提交。

## 1. Go 验证

| 命令 | 结果 |
|---|---|
| `GOTOOLCHAIN=go1.25.13 go test ./pkg/llm/... ./pkg/config ./app -count=1` | `PASS` |
| `GOTOOLCHAIN=go1.25.13 go test ./... -count=1` | `PASS` |
| `GOTOOLCHAIN=go1.25.13 go vet ./...` | `PASS` |
| `GOTOOLCHAIN=go1.25.13 go build ./...` | `PASS` |
| `GOTOOLCHAIN=go1.25.13 go mod verify` | `PASS`，`all modules verified` |

全量测试覆盖 `app`、`pkg/config`、`pkg/llm`、`pkg/questionbank`、`pkg/mobile`、`pkg/resume`、`pkg/solution` 等现有包；无失败包。

## 2. 前端验证

以下 13 个回归文件逐个执行并全部 `PASS`：

1. `ccr_settings_lifecycle_regression_test.mjs`
2. `ccr_ui_regression_test.mjs`
3. `clipboard_regression_test.mjs`
4. `history_round_selection_regression_test.mjs`
5. `mobile_store_regression_test.mjs`
6. `model_bridge_timeout_regression_test.mjs`
7. `model_connectivity_regression_test.mjs`
8. `model_layout_regression_test.mjs`
9. `numbered_copy_regression_test.mjs`
10. `questionbank_continuation_regression_test.mjs`
11. `questionbank_history_regression_test.mjs`
12. `round_screenshots_regression_test.mjs`
13. `settings_regression_test.mjs`

`npm --prefix frontend run build`：`PASS`，Vite v4.5.14 转换 130 个模块。仍有既有“大于 500 kB chunk”警告，不是构建失败，本 Change 未新增依赖。

## 3. Wails bindings 与独立程序

- CLI：固定缓存可执行文件，版本实测为 Wails CLI v2.11.0；未使用 PATH 中的 v2.16.0。
- bindings：`PASS`。`app.ConnectionDraft` 与 `config.Config` 均包含 `connectionMode`。
- 方法清单：`App.js` 与 `App.d.ts` 均为 79 个导出方法，名称一致。
- Wails runtime：无内容漂移。
- 生成器尾随空白：仅对 `frontend/wailsjs/go/models.ts` 执行逐行 `rstrip`，随后 `git diff --check` 为 `PASS`。
- 独立程序：`build/bin/Q-Solver-change026-ccr-gateway-validation.exe`
  - 大小：`44,032,512` 字节
  - SHA-256：`5584d5448e6808a230a4d01e35e098938a92690beb3b1f3d149129b0d9771fe1`
- 事前记录的 11 个受保护 EXE 前后 SHA-256 完全一致；正式 `Q-Solver.exe` 与旧验证程序均未覆盖。

## 4. race

命令：

```text
GOTOOLCHAIN=go1.25.13 go test -race ./pkg/config ./pkg/llm ./app -count=1
```

结果：`BLOCKED`。

三个目标包均在启动阶段返回：

```text
exit status 0xc0000139
```

该结果与历史 Windows 环境阻塞一致，不记录为 `PASS`，也不归因于 Change 026 业务代码。

## 5. 关键自动断言

- typed nil 不 panic，成功 probe DTO 正常返回并结束 loading。
- CCR 工厂始终使用 OpenAI Chat adapter，不依据模型名、品牌、域名或残留协议选择 adapter。
- CCR 目录恰好一次 `GET /v1/models`，模型去重保序，零推理 POST。
- CCR text/vision probe 均在 Service 网络路径前拒绝，总请求数为 0。
- model 变化不清目录；目录 identity 变化使旧结果失效。
- CCR 自动同步同 revision 单发并复用在途 Promise；direct、保存和 API 页本身不隐式同步或推理。
- bridge deadline 只停止 UI 等待，不冒充后端取消，也不伪造 `requestSent=false`。
- 模型页只有一个纵向滚动容器，角色/场景列表可通过外层滚动访问。
- 目录刷新在 UI 和 Store 两层均要求 Key 与 BaseURL 完整。
- 已选模型与临时搜索词使用独立状态；第二次打开设置或点击箭头时展示完整目录，只有主动输入才过滤。

## 6. 依赖、网络、隐私与数据审计

- `go.mod`、`go.sum`、`frontend/package.json`、`frontend/package-lock.json` 的 SHA-256 与事前基线完全一致。
- Wails 保持 v2.11.0，OpenAI SDK 保持 v1.12.0，Anthropic SDK 保持 v2.3.0；无 Change 026 依赖新增或升级。
- 自动测试只使用 loopback `httptest`、fake provider 与合成图片；未调用真实 CCR 或真实模型，未消费真实 API 额度。
- Wails 构建期间只实例化配置管理器并输出配置路径，未调用配置 `Load`，未读取或输出真实密钥。
- 未修改题库 `_记录`、截图、旧导出、用户 API 配置或 Mobile Token。

## 7. 人工验收

以下状态均为 `NEEDS_USER`：

1. Direct/CCR 模式切换和 Direct 三协议。
2. 真实 CCR 地址、Key 与模型目录同步。
3. CCR 选择模型后目录不消失，刷新/保存不触发推理。
4. 真实 CCR 下用户主动执行主截图解题。
5. Direct 主动 probe 成功、失败和 timeout 后均退出 loading。
6. 小窗口中滚动到角色/场景并选择、保存。
7. 快速切模式与 A→B→A 不发生旧结果覆盖。

## 8. 结论

Change 026 实现与普通自动验证完成，可以进入用户人工验收；尚未获得 Git 提交、推送、Tag、Release 或发布授权。
