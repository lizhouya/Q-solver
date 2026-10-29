# Change 029：验证记录

## 自动验证结果

| 验证项 | 状态 | 证据 |
|---|---|---|
| Git/远端/暂存区预检 | PASS | 分支 `dev/qsolver-custom`；设计提交为 HEAD；暂存区为空；远端为实施前基线 |
| 迁移前产物清单 | PASS | 仓库外记录 44 个文件 |
| 迁移前文件可恢复性 | PASS | 44/44 路径、大小、SHA-256 映射一致 |
| 当前目录级哈希清单 | PASS | 25 份 `SHA256SUMS` 逐项通过 |
| 历史 validation 清单原件 | PASS | SHA-256 `5FB58D43DEDF24D877983A1AF0E84DEB864DFAD4484F41212DCE089D1DAD4B68` |
| Release 保护 | PASS | 7 个文件前后哈希逐项一致 |
| 聚焦 Go 测试 | PASS | `go test ./pkg/questionbank ./pkg/config ./pkg/llm ./pkg/solution ./pkg/mobile ./pkg/resume ./app -count=1` |
| 全量 Go 测试 | PASS | `go test ./... -count=1` |
| Go 静态检查 | PASS | `go vet ./...` |
| Go 全量构建 | PASS | `go build ./...` |
| Go 模块缓存 | PASS | `go mod verify`：`all modules verified` |
| 前端回归 | PASS | `frontend/*_test.mjs` 共 14 份全部退出 0 |
| Vite production build | PASS | Vite 4.5.14，132 modules transformed |
| Wails 独立候选构建 | PASS | CLI v2.16.0 + 全部保护参数；实际链接 runtime v2.11.0；3.001 秒 |
| 依赖、dist、Git index 保护 | PASS | 构建前后哈希/语义一致 |
| Wails runtime 生成噪音 | PASS | Notification 噪音清除后 `frontend/wailsjs` 恢复构建前哈希 |
| 候选独立启动 | PASS | 8 秒存活；WebView2；无 Node/Vite；正常关闭；退出码 0；无残留 |
| 晋升后根目录启动 | PASS | 与候选相同的启动/退出条件全部通过 |
| 根目录唯一最终版 | PASS | 根目录 1 个文件、1 个 EXE：`Q-Solver.exe` |
| testing/current 收尾 | PASS | 0 个文件 |
| 差异空白检查 | PASS（源码与 Markdown） | 原始 `change028.patch` 证据保留其记录内容中的空白提示，不作为源码/文档格式失败 |
| 敏感模式检查 | PASS | 仅命中实施计划中列出的扫描关键词；无认证实值 |

## 最终 EXE

- 路径：`build/bin/Q-Solver.exe`
- 大小：42,331,136 字节
- SHA-256：`82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46`
- Go：go1.27.1 windows/amd64
- Wails runtime：v2.11.0
- Build tags：`desktop,prod,wv2runtime.download,production`

## 已知限制

- Windows race 测试既有 `0xc0000139` 环境问题保持 `BLOCKED`；本轮没有把它写成 PASS。
- Change 031 Windows GUI 真实导出没有新的人工证据，继续标记 `NEEDS_USER`。
- 本轮没有调用真实模型，不修改或验证 CCR/Provider 认证状态，不宣称间歇性 401/502 已解决。
