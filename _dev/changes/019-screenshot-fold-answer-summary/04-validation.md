# Change 019 验证记录

状态：普通自动验证 PASS；Windows GUI NEEDS_USER；真实模型效果 NEEDS_USER；race BLOCKED；验证程序归档复制 BLOCKED。

## TDD 证据

- 截图组件新回归在旧实现上按预期失败：旧组件没有本地展开状态、`scopeKey` 或可再次收起入口。
- Prompt 契约测试在旧 Prompt 上按预期失败：缺少当前轮多题、跨图合并、去重、待核对和最终汇总约束。
- 实现后上述测试转绿。

## 自动验证

以下命令均成功：

```text
node frontend/round_screenshots_regression_test.mjs
node frontend/questionbank_continuation_regression_test.mjs
node frontend/questionbank_history_regression_test.mjs
node frontend/settings_regression_test.mjs
node frontend/mobile_store_regression_test.mjs
node pkg/mobile/web_dom_test.mjs
npm --prefix frontend run build
GOTOOLCHAIN=go1.25.13 go test ./... -count=1
GOTOOLCHAIN=go1.25.13 go vet ./...
GOTOOLCHAIN=go1.25.13 go build ./...
git diff --check
```

Vite 结果：128 modules transformed，构建耗时 2.29s。主要 bundle 为 `assets/index-491ed2d5.js`，845.01 kB，gzip 257.56 kB。大于 500 kB 的提示是既有 chunk-size warning，不记为 Change 019 失败。

`git diff --check` 没有空白错误；仅有工作区既有 LF/CRLF 提示。

## 专项保护结果

- 三张截图只调用 Provider 一次。
- 当前 user 消息含三个 image part，内容和顺序保持不变。
- 当前请求仅有一个 system，且同时包含统一多题规则与当前 Scene Prompt。
- fake provider 返回的“逐题正文 + 最终答案汇总”完整进入 `solution` 事件和 KeepContext assistant 历史，没有第二次解析或改写。
- 已缓存历史图片再次调用 `loadRoundImage` 返回 false，不重复访问 API；缓存和 `round.userScreenshot` 保持不变。
- `RoundScreenshots.vue` 不直接赋值任何图片缓存字段。

## race

执行：

```text
GOTOOLCHAIN=go1.25.13 go test -race ./pkg/domain ./pkg/solution -count=1
```

结果：

```text
exit status 0xc0000139
FAIL Q-Solver/pkg/domain
exit status 0xc0000139
FAIL Q-Solver/pkg/solution
```

结论：`BLOCKED`。普通测试不能替代 race，也未为绕过该环境问题修改业务代码。

## Wails 独立验证构建

命令：

```text
GOTOOLCHAIN=go1.25.13 _dev/changes/015-rc-release/tmp/tools/wails.exe build -clean=false -skipbindings -nosyncgomod -m -o Q-Solver-change019-answer-summary-preview-validation.exe
```

结果：

- Wails CLI：v2.11.0。
- Platform：windows/amd64。
- Skip Bindings：true。
- Clean Bin Dir：false。
- 构建成功，耗时 8.571s。
- 文件：`build/bin/Q-Solver-change019-answer-summary-preview-validation.exe`。
- SHA-256：`cd4b354ef37613029d2bb0e39b9d62fd924480aefa0eafbf762ca60135d19cf7`。
- 大小：`23,366,656` 字节。

构建前后哈希检查全部 `OK`。构建前基线保存在 `pre-wails-protected.sha256` 和 `pre-wails-formal-exe.sha256`，作为本 Change 的只读验证证据保留：

```text
frontend/wailsjs/go/app/App.d.ts: OK
frontend/wailsjs/go/app/App.js: OK
frontend/wailsjs/go/models.ts: OK
go.mod: OK
go.sum: OK
build/bin/Q-Solver.exe: OK
```

## 归档状态

计划复制到 `build/bin/archive/validation/Q-Solver-change019-answer-summary-preview-validation.exe` 并重建 LF-only `SHA256SUMS`。包含复制和清单更新的组合命令在执行前被 Claude Code auto mode classifier 拒绝，因此：

- 归档目标当前不存在。
- `SHA256SUMS` 没有 Change 019 条目。
- 没有声称完整归档校验已通过。
- 未使用其他命令、脚本或工具绕过该拒绝。

源验证程序可直接用于人工验收；统一归档仍需获得相应本地文件操作权限后完成。

## Windows GUI

`NEEDS_USER`。需使用上述验证程序检查：

1. 当前轮和历史轮 1～3 张截图默认收起。
2. 每张图片独立展开/隐藏，点击图片本身可收起。
3. 已缓存图片重开不读取；未缓存历史图按需读取。
4. 加载中收起后，请求完成不自动展开。
5. 失败后收起再展开可重试；Session/Record 间展开状态不串。
6. Tab、Enter、Space、焦点轮廓和 `aria-expanded` 正常。
7. 折叠不影响答案、题库、截图缓存、导出或模型上下文。

## 真实模型效果

`NEEDS_USER`；未经额外授权未调用真实模型、未上传截图、未消费真实 API 额度。自动测试只证明 Prompt 契约和请求链，不能证明具体模型一定正确识别每一道真实题目。
