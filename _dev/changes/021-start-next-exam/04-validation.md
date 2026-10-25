# Change 021 验证记录

日期：2026-10-24
状态：普通自动验证、正式 bindings、独立构建与统一归档完成；Windows GUI `NEEDS_USER`，race `BLOCKED`，未提交、未推送、未发布、未关闭。

## TDD 证据

| 阶段 | 命令/检查 | 结果 |
|---|---|---|
| Store RED | 新增 Store 专项测试后运行 | 预期编译失败：`s.SealCurrentForNext undefined` |
| Store GREEN | `go test ./pkg/questionbank -run 'Test.*SealCurrentForNext' -count=1 -v` | PASS |
| App RED | 新增 App 专项测试后运行 | 预期编译失败：`a.StartNextExam undefined` |
| App GREEN | `go test ./app -run 'Test.*StartNextExam' -count=1 -v` | PASS |
| 前端 RED | 新增确认链断言后运行 | 预期失败：`nextExamPending` 为 `undefined` |
| 前端 GREEN | `node frontend/questionbank_history_regression_test.mjs` | PASS |

Store 专项覆盖 active、paused、ended、exported、discarded、nil/empty、running、dirty、pending completion、write blocked、保存失败回滚和下一题懒创建。

App 专项覆盖未确认、active/paused 封存、ended/exported 幂等、discarded/empty、in-flight、Cancel 后未 Complete、capture、organizing/exporting、closing、服务未就绪、持久化失败零副作用、截图计数、epoch/refs/Solver 清理、Mobile 不变和下一题懒创建。

## JavaScript 回归

以下命令均在最终业务源码上运行并通过：

- `node frontend/questionbank_history_regression_test.mjs`
- `node frontend/history_round_selection_regression_test.mjs`
- `node frontend/questionbank_continuation_regression_test.mjs`
- `node frontend/round_screenshots_regression_test.mjs`
- `node frontend/settings_regression_test.mjs`
- `node frontend/mobile_store_regression_test.mjs`
- `node pkg/mobile/web_dom_test.mjs`（退出码 0，无输出）

覆盖确认层单例、防重、取消零副作用、失败保留现场、成功后 reset、typed/legacy 流目标清理、快捷键平台差异、安全迁移、Mobile store 和 Mobile DOM 兼容。

## Go 普通验证

固定 `GOTOOLCHAIN=go1.25.13`：

| 命令 | 结果 |
|---|---|
| `go test ./pkg/questionbank -run 'Test.*SealCurrentForNext' -count=1 -v` | PASS |
| `go test ./app -run 'Test.*StartNextExam' -count=1 -v` | PASS |
| `go test ./pkg/config ./pkg/shortcut -count=1` | PASS |
| `go test ./... -count=1` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |

## Frontend 与 Wails 构建

- `npm --prefix frontend run build`：PASS，Vite v4.5.14，128 modules transformed。
- 最终主 bundle：`assets/index-894b0c73.js`，850.40 kB，gzip 259.12 kB。
- 保留既有非阻断 warning：`Some chunks are larger than 500 kBs after minification.`
- 生成/首次构建：Wails CLI v2.11.0，`-clean=false -nosyncgomod -m`，PASS，15.565s。
- 独立构建：Wails CLI v2.11.0，`-clean=false -skipbindings -nosyncgomod -m`，PASS，10.427s。
- 治理收尾后以同一固定命令最终一致性重建：PASS，7.693s；程序大小和 SHA-256 与归档副本完全相同。

最终程序：

- 路径：`build/bin/Q-Solver-change021-start-next-exam-validation.exe`
- 大小：23,387,648 字节
- SHA-256：`95f4b3c17b96db0e25fa682ed337bb067458046784a09d3580d1eff4839e3c4c`

## bindings、依赖与正式程序审计

- bindings 已包含 Change 016～018 累积 API/DTO，并新增 `StartNextExam` 与 `StartNextExamResult`；Change 019～020 没有后端 bindings 变化。
- 事前只记录三份 bindings 的 SHA-256，没有保存文件副本；从当前生成文件逆向移除 `StartNextExam`/DTO 后未匹配事前哈希，故不能声称“除单接口外逐字节不变”。
- 可靠结论是：固定 Wails v2.11.0 根据当前累计 Go API 重新生成完整 bindings，既有公开 API 均存在，当前源码、bindings、Frontend 和 Wails 构建一致。
- `go.mod` SHA-256 保持 `6583908a30917f6864adc5cb7d36d36e56cb5ed6447889fa18e55e1a0784a298`。
- `go.sum` SHA-256 保持 `d6b12513771dd06b45257192185c07519cedf89a0eab1af7967cc44112d121a4`。
- 正式 `build/bin/Q-Solver.exe` SHA-256 保持 `5dcabb17933bfa9ab4dc15f289b4b8e50af99879c316b58831b09bbb4e39aa11`。
- 生成器产生的 `models.ts` 纯空白行 Tab 已清除；`git diff --check` 最终无空白错误，仅保留行尾转换 warning。

## Race

命令：

```text
GOTOOLCHAIN=go1.25.13 go test -race ./pkg/task ./pkg/solution ./pkg/questionbank ./app -count=1
```

结果：`BLOCKED`。四个包均在 Windows race 程序启动阶段返回：

```text
exit status 0xc0000139
FAIL Q-Solver/pkg/task
FAIL Q-Solver/pkg/solution
FAIL Q-Solver/pkg/questionbank
FAIL Q-Solver/app
```

普通测试没有替代 race，也未为绕过环境问题修改业务代码。

## 统一验证归档

- 已归档：`build/bin/archive/validation/Q-Solver-change021-start-next-exam-validation.exe`
- `SHA256SUMS` 保持 LF-only，共 11 条记录。
- `sha256sum -c SHA256SUMS` 对既有十份程序及 Change 021 新程序全部返回 `OK`。
- 未覆盖或删除任何旧验证程序；Change 019 既有归档阻塞记录不因本次归档成功而改写。

## 独立审查状态

只读审查代理因 API 使用限额返回 HTTP 429 并提前终止，状态不记为 PASS。已返回的局部检查未确认锁序、Wails 签名、配置迁移、快捷键分发或 Mobile 边界缺陷；本 Change 仍依赖上述自动验证及 Windows GUI 人工验收完成最终确认。

## Windows GUI 人工验收

状态：`NEEDS_USER`。使用归档验证程序执行：

1. active/paused 当前段点击“开始下一套”或按 `Ctrl+Shift+N`，只出现一个确认层；取消后内容不变。
2. 解题、capture、整理或导出进行中确认时被拒绝，桌面历史、待发送截图、模型上下文和题库状态不变。
3. active/paused 成功后当前段为 ended；ended/exported 可继续开始下一套，旧导出目录和快照不变。
4. 成功后 ScreenshotDock、桌面逐轮历史与流状态清空，界面回到待记录状态；下一次正式解题创建新 Session。
5. 新一套第一题不携带上一套 Solver 对话；旧题库 Record、归档截图、完整答案和导出仍可浏览。
6. Mobile 固定链接和当前显示不变；需清屏时仍独立使用现有 `Shift+C` 确认链。
7. `start_next_exam` 自定义快捷键可保存，冲突组合被拒绝；旧配置显式空/自定义绑定不被迁移覆盖。

GUI 与 race 问题未完成前，本 Change 不关闭。
