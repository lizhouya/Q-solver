# Change 020 验证记录

日期：2026-10-23
状态：普通自动验证与独立构建完成；Windows GUI `NEEDS_USER`，race `BLOCKED`，未提交、未推送、未发布、未关闭。

## TDD 证据

| 阶段 | 命令/检查 | 结果 |
|---|---|---|
| RED | `node frontend/history_round_selection_regression_test.mjs`（实现前） | 预期失败：旧 Store 不存在 `historyEntries`，报 `TypeError: Cannot read properties of undefined (reading 'map')`。 |
| GREEN | 同一专项测试（实现后与最终复跑） | PASS：`History round selection regression tests passed`。 |

## 最终 JavaScript 回归

以下命令均在最终源码状态运行：

| 命令 | 结果 |
|---|---|
| `node frontend/history_round_selection_regression_test.mjs` | PASS |
| `node frontend/questionbank_continuation_regression_test.mjs` | PASS |
| `node frontend/questionbank_history_regression_test.mjs` | PASS |
| `node frontend/round_screenshots_regression_test.mjs` | PASS |
| `node frontend/settings_regression_test.mjs` | PASS |
| `node frontend/mobile_store_regression_test.mjs` | PASS |
| `node pkg/mobile/web_dom_test.mjs` | PASS，退出码 0，无输出 |

覆盖结果包括：每个 Record 独立 entry、Session 内最新轮优先、稳定 key、自身摘要、单轮选择、题库续接默认最新 Record、typed/legacy 浏览与后台写入隔离、删除/reset 修复、图片缓存与迟到响应保护，以及会话级菜单文案。

## Frontend 构建

命令：`npm --prefix frontend run build`

结果：PASS，Vite v4.5.14，126 modules transformed，最终构建 2.21s。

最终主 bundle：

```text
assets/index-d12def1d.js  847.76 kB  gzip 258.31 kB
```

保留既有非阻断 warning：`Some chunks are larger than 500 kBs after minification.` 本 Change 未引入依赖或展开无关代码分割重构。

## Go 普通验证

使用 `GOTOOLCHAIN=go1.25.13`：

| 命令 | 结果 |
|---|---|
| `go test ./... -count=1` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |

Change 020 没有修改 Go 后端、Wails API/bindings、Mobile、题库 schema、`go.mod` 或 `go.sum`。

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

该结果与此前环境阻塞一致；普通测试没有被记作 race PASS，也未为绕过环境问题修改业务代码。

## Wails 独立验证构建

工具与参数：

```text
GOTOOLCHAIN=go1.25.13 _dev/changes/015-rc-release/tmp/tools/wails.exe build \
  -clean=false -skipbindings -nosyncgomod -m \
  -o Q-Solver-change020-round-history-selection-validation.exe
```

结果：PASS；Wails CLI v2.11.0，windows/amd64，Frontend dependencies / compile / assets / application compile 均完成，构建耗时 8.41s。

产物：

- 路径：`build/bin/Q-Solver-change020-round-history-selection-validation.exe`
- 大小：23,369,216 字节
- SHA-256：`3ea25ab19009d373c304f440c126bfeb18ce7c0d4c218d8e9be8204abdaa75bf`

构建后执行 `_dev/changes/020-round-history-selection/pre-wails-protected.sha256` 校验，以下六项全部 `OK`：

- `frontend/wailsjs/go/app/App.d.ts`
- `frontend/wailsjs/go/app/App.js`
- `frontend/wailsjs/go/models.ts`
- `go.mod`
- `go.sum`
- `build/bin/Q-Solver.exe`

## 统一验证归档

- 已复制到：`build/bin/archive/validation/Q-Solver-change020-round-history-selection-validation.exe`
- 已以 LF-only 追加 `build/bin/archive/validation/SHA256SUMS`。
- `sha256sum -c SHA256SUMS` 对原有九份程序和 Change 020 新程序全部返回 `OK`。
- Change 019 的归档阻塞状态不因本次成功归档而被改写。

## 静态与范围检查

- `git diff --check`：PASS；只有工作区既有 LF→CRLF 转换 warning，没有空白错误。
- `git status --short --branch`：分支 `dev/qsolver-custom` 相对远端 ahead 1；工作区仍叠加 Change 016～019 与 Change 020，明确不是 clean。
- 最终 Git 状态与 Change 020 白名单审计见 `_dev/audits/020-round-history-selection.md`。

## Windows GUI 人工验收

状态：`NEEDS_USER`。必须使用本 Change 验证程序检查：

1. 连续提交两轮后，左侧出现两条独立记录，最新一轮在上，不再显示“2 轮”聚合徽标。
2. 点击任意轮，右侧只显示该轮截图、thinking、答案、错误和操作栏。
3. 查看同 Session 旧轮时，后台新轮继续生成；旧轮不显示新轮 loading/cursor，切回新轮可见累计内容。
4. 从题库续接包含多个 Records 的 Session 后，左侧逐条显示并默认选中最新 Record。
5. 切换轮次后截图展开状态不串轮，已缓存图片不重复读取。
6. 菜单明确说明删除/导出作用于整个会话。
7. legacy 取消正在追加的新轮后回到上一轮，不出现右侧空白。
8. 清空分段后桌面历史为空，题库原始记录、截图和旧导出仍保留。

在上述人工验收和 race 环境问题未解决前，本 Change 不关闭。
