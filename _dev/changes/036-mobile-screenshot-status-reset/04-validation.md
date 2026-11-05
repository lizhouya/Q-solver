# 验证记录 04

## 自动验证

验证环境：Windows 11 Pro。初版候选使用 Go 1.25.13 与 Wails CLI v2.11.0；2026-11-05 底部 Dock 回修使用当前 Go 1.27.1、本机 Wails CLI v2.16.0，并通过保护参数确保候选实际链接项目 Wails 模块 v2.11.0。

| 命令 / 检查 | 结果 | 备注 |
|---|---|---|
| Change 036 Mobile/App 聚焦测试 | PASS | 覆盖截图状态幂等与重放、满队列 Reset 屏障、截图增删清空、三个清理入口成功/拒绝语义 |
| `node pkg/mobile/web_dom_test.mjs` | PASS | 覆盖底部固定 Dock 的 0 张隐藏、1～2 张继续截图、3 张已满、归零隐藏、安全留白，以及 idle 清答案/宽窄历史和低业务序号兼容 |
| `go test ./... -count=1` | PASS | 全量 Go 测试通过 |
| `go vet ./...` | PASS | 标准静态检查通过 |
| `go build ./...` | PASS | 全量 Go 构建通过 |
| `go mod verify` | PASS | 模块内容校验通过 |
| 15 份 `frontend/*_test.mjs` | PASS | 全部桌面前端静态/DOM 回归通过 |
| `npm --prefix frontend run build` | PASS | Vite v4.5.14，132 modules transformed；仅保留既有大于 500 kB chunk 警告 |
| Wails v2.11.0 保护构建 | PASS | 使用 `-skipbindings -skipembedcreate -nosyncgomod -m -nocolour -v 1`、`-ldflags "-s -w"`、`-tags prod`；候选实际链接 Wails v2.11.0 |
| 候选独立启动与退出 | PASS | 启动 8 秒后存活；WebView2 direct child=1；Node direct child=0；正常关闭，ExitCode=0，ForcedCleanup=false，残留进程=0 |
| 保护文件/目录比较 | PASS | `go.mod`、`go.sum`、`frontend/dist/**`、`build/bin/releases/**` 内容哈希不变；较新 CLI 生成的两个 runtime 文件已定向恢复为 HEAD 语义且不在 Git 差异中。Windows checkout 将其规范为 CRLF，`frontend/wailsjs/**` 工作树聚合字节哈希由 `3AF48C…` 变为 `9C8F69…`，不作为业务或提交差异 |
| Git index 语义 | PASS | 只读 Git 与定向恢复刷新了 `.git/index` stat cache，原始文件哈希变化；最终 `git diff --cached` 为空，未产生暂存内容 |
| VS Code Problems | PASS | 已打开并检查的相关文件无诊断 |
| `git diff --check` | PASS | 最终业务差异检查无空白错误；治理文档补录后需在提交边界前再次执行 |
| `go test -race ./pkg/mobile ...` | BLOCKED | 测试程序启动阶段返回 `0xc0000139`，未进入测试逻辑 |
| `go test -race ./app ...` | BLOCKED | 测试程序启动阶段返回 `0xc0000139`，未进入测试逻辑 |

## 候选与产物保护

| 项目 | 路径 / 结果 |
|---|---|
| Change 036 候选 | `build/bin/testing/current/Q-Solver-candidate-change036.exe` |
| 候选大小 | 42,359,296 字节 |
| 候选 SHA-256 | `8E9543C9DA02B56C4C85099BAEA0D2621B47234DA218280C5D64DA0CEB1C80F2` |
| 被替换的顶部卡片候选 | `build/bin/archive/transient/Q-Solver-candidate-change036-top-card.exe`；SHA-256 `2B52AE48ED179E956FFFE27B5B28B9CB09AD992A9FF0E9A03481D2C09DC2D0A7` |
| 测试前稳定版副本 | `build/bin/testing/current/Q-Solver-stable-before-test.exe` |
| 稳定版副本 SHA-256 | `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE` |
| `SHA256SUMS` | 两项均重新计算并验证通过 |
| 根稳定版 | 保持 Change 035 版本不变，未覆盖、未晋升 |
| `build/bin/releases/**` | 聚合哈希不变 |

## 人工验收

### 2026-11-05 首次反馈

用户反馈“同步端没有截图和快捷键提示”，本轮验收记为 **FAIL / 待复验**。只读诊断确认：

- 当前运行进程路径为 Change 036 候选，而非根目录旧稳定版。
- 候选二进制可检出 `capture-status-card`、`待发送截图` 和 `renderCaptureStatus`，证明新版 Mobile HTML 已嵌入。
- 新页面的 `/` 响应设置 `Cache-Control: no-store`，但已打开的旧浏览器文档在桌面进程重启后只会让 EventSource 自动重连，不会自动替换 DOM。
- 因此既有标签页仍能收到答案，却不会凭空出现新截图卡；需主动刷新或重新打开同一固定链接后再验收。

### 2026-11-05 刷新后反馈与底部 Dock 回修

用户刷新后确认截图数量和相关快捷键已正常出现，同时指出常驻答案顶部会在自动滚底后离开视野。经批准后按桌面 `ScreenshotDock` 语义回修：仅在有待发送截图时固定显示于页面底部；发送或清空使数量归零后立即隐藏。回修后的自动验证和候选独立启动均通过，等待第二轮 GUI 验收。

### 2026-11-05 最终人工验收

用户使用底部 Dock 回修候选完成复验并明确确认“验证通过”：截图时底部提示可见，解题时提示随截图数量归零而消失；Change 036 人工验收项均记为 PASS。

| 检查项 | 预期 | 结果 |
|---|---|---|
| 同步端截图数量 | 0 张时隐藏；1/3～3/3 时在页面底部固定显示 | PASS |
| 实际快捷键提示 | Dock 显示实际截图、删除、发送快捷键；1～2 张提示继续截图，3 张提示已满 | PASS |
| 截图状态实时变化 | 截图、删除、发送后数量立即对应变化 | PASS |
| 单独清截图 | 数量归零，但同步端既有答案仍保留 | PASS |
| 结束本段 | 成功后同步端答案及宽/窄屏历史均无残留 | PASS |
| 清空本段 | 成功后同步端答案及宽/窄屏历史均无残留 | PASS |
| 开始下一套 | 成功后同步端答案及历史无残留，截图数量归零 | PASS |
| SSE 刷新/重连 | 清理后不会恢复旧答案或旧历史 | PASS |
| 固定链接 | 上述操作不改变 Mobile 固定链接 | PASS |

## 提交边界审计

- 功能与测试提交：`d90bbe3`，精确包含 13 个 `app/`、`pkg/mobile/` 生产代码与测试文件。
- 治理提交范围：`_dev/audits/036-mobile-screenshot-status-reset.md`、`_dev/changes/036-mobile-screenshot-status-reset/**`、`_dev/changes/INDEX.md`。
- `build/bin/**` 保持 ignored，未纳入 Git。
- 未纳入依赖、bindings、配置、secret、用户数据或 Release 产物。
- 用户已明确授权双提交和推送 `origin/dev/qsolver-custom`；未授权稳定版晋升、Tag 或 Release。

## 结论

普通自动验证、底部 Dock 回修、候选保护构建、独立启动、SHA-256 校验和用户 GUI 验收均通过；race 检查仍受当前 Windows 运行环境 `0xc0000139` 阻塞并已如实保留。功能与测试提交为 `d90bbe3`，治理记录由配套提交归档。用户已授权推送 `origin/dev/qsolver-custom`。Change 036 满足关闭条件；不包含稳定版晋升、Tag 或 Release。
