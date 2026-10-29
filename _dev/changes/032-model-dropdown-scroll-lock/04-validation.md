# Change 032：验证记录

## 自动验证

| 验证项 | 状态 | 结果 |
|---|---|---|
| TDD 红灯 | PASS | 当前实现缺少 `@wheel.stop` 时按预期失败 |
| 聚焦布局回归 | PASS | `node frontend/model_layout_regression_test.mjs` |
| 全部前端回归 | PASS | 14/14 脚本退出 0 |
| Vite production build | PASS | Vite 4.5.14，132 modules transformed |
| 全量 Go 测试 | PASS | `go test ./... -count=1` |
| Go 静态检查 | PASS | `go vet ./...` |
| Go 全量构建 | PASS | `go build ./...` |
| Wails 候选构建 | PASS | 保护参数启用，3.302 秒 |
| Wails runtime | PASS | 候选实际链接 v2.11.0 |
| 依赖保护 | PASS | `go.mod`、`go.sum` 构建前后不变 |
| dist 保护 | PASS | Wails 构建前后 `frontend/dist` 不变 |
| Git index 保护 | PASS | stage listing 构建前后不变 |
| Wails bindings/runtime | PASS | v2.16 生成噪音清除后恢复构建前哈希 |
| 候选 SHA-256 | PASS | 目录清单校验通过 |
| 晋升后根目录启动 | PASS | 8 秒存活、WebView2、无 Node/Vite、正常退出码 0、无残留进程 |
| 最终目录不变量 | PASS | 根目录只有 `Q-Solver.exe`，`testing/current/` 为空，Release 哈希不变 |
| 候选 GUI 人工验收 | PASS | 用户运行 Change 032 候选并明确反馈测试通过 |
| 真实滚轮/触控板体验 | PASS | 用户确认外层滚动锁定修复通过 |

## 候选

- 路径：`build/bin/testing/current/Q-Solver-candidate-change032.exe`
- 大小：42,331,648 字节
- SHA-256：`C1F1873F054520F0F45FADFCCBCC087056D1556381DD1EFC66D4CD3748085F7B`

## 测试前稳定版

- 路径：`build/bin/testing/current/Q-Solver-stable-before-test.exe`
- SHA-256：`82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46`

## 人工验证步骤

1. 正常关闭当前运行中的根目录 `Q-Solver.exe`。
2. 双击 Change 032 候选。
3. 打开设置 → 模型，展开包含较多条目的模型列表。
4. 分别在列表顶部、中间、底部连续滚动鼠标滚轮和触控板。
5. 确认模型列表正常滚动，外层设置页的位置始终不变。
6. 选择模型、点击列表外部、切换页签和关闭设置，确认外层滚动恢复且下拉不残留。
## 最终晋升版本

- 路径：`build/bin/Q-Solver.exe`
- 大小：42,331,648 字节
- SHA-256：`C1F1873F054520F0F45FADFCCBCC087056D1556381DD1EFC66D4CD3748085F7B`
- 自动启动：PASS
- GUI 滚动锁定：用户确认 PASS
