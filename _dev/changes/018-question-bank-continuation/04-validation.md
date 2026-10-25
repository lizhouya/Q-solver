# Change 018 验证记录

状态：普通自动验证 PASS；Windows GUI NEEDS_USER；race 因 `0xc0000139` BLOCKED；未关闭。

## 自动验证

| 项目 | 结果 | 证据 |
|---|---|---|
| `go test ./pkg/questionbank ./pkg/solution ./pkg/task ./app -count=1` | PASS | 四个核心包全部通过 |
| `go test ./... -count=1` | PASS | 全仓 Go 测试通过 |
| `go vet ./...` | PASS | 无输出，退出码 0 |
| `go build ./...` | PASS | 无输出，退出码 0 |
| Change 018 continuation Node 回归 | PASS | DTO、epoch/ID 过滤、A→B 迟到事件、图片请求和 panel 成败路径通过 |
| Change 017 history Node 回归 | PASS | 最近 20、只读详情、图片读取、固定操作列、纵向等宽按钮和窄屏单列约束通过 |
| Settings / Mobile store / Mobile DOM 回归 | PASS | 全部退出码 0 |
| `npm --prefix frontend run build` | PASS | Vite 4.5.14，128 modules；仅既有大于 500 kB chunk warning |
| Wails bindings | PASS | 固定 v2.11.0 生成；包含 `ContinueQuestionBankSession` 与 `ConfirmSolveWithoutResume` |
| Wails Windows 生产构建 | PASS | 固定 Go 1.25.13；未覆盖正式 `Q-Solver.exe` |
| `SHA256SUMS` | PASS | validation 目录 9 个 EXE 全部 `OK`；清单 LF-only |
| `git diff --check` | PASS | 清理生成 `models.ts` 行尾空白后通过；仅既有 LF/CRLF 提示 |
| `go test -race ./pkg/task ./pkg/solution ./pkg/questionbank ./app -count=1` | BLOCKED | 四个包均在启动阶段返回 `exit status 0xc0000139`，没有产生可用 race 结果 |

## 关键专项测试

- Store 激活：source pause 失败、target activate 失败、source rollback 成功、source rollback 失败与全写 API 阻断。
- Store 状态：same-current paused→active、discarded/running/dirty 拒绝、exported 元数据保留。
- 完成态 Retry：失败前不伪提交；Retry 后 Record、ConversationSnapshot、Solver 与 App refs 一致。
- App 拒绝矩阵：最近 20 之外、discarded、in-flight、取消但 goroutine 未完成、待发送截图、captures、organizing、exporting、closing、dirty、target running。
- App 补偿：Store 拒绝时 current、Solver、refs 与 epoch 保持旧值。
- 启动恢复：persisted 精确恢复、legacy 重建并警告、图片损坏整体拒绝。
- exported 集成：续接、追加、结束、再次导出 `_02`，旧导出文件字节不变，Exports 追加。
- 前端：稳定 DTO、完整答案与 1～3 图、只读查看期间后台写入、A→B 迟到事件、图片迟到失效、成功/失败 panel 行为。
- 布局：桌面固定 `7.5rem` 操作列、继续/重新导出纵向等宽、卡片信息上下排列、meta 安全换行、`560px` 以下操作区全宽；旧 CSS 先被新增断言拒绝，修复后通过。

## 验证程序

- 原续接验证入口：`build/bin/archive/validation/Q-Solver-change018-continuation-validation.exe`；SHA-256 `775196521b00528824e4620cc4df1ad80c8bbb78c7820d40c626b528f67fe8e0`。
- 最新布局验证构建原件：`build/bin/Q-Solver-change018-layout-validation.exe`。
- 最新统一验证入口：`build/bin/archive/validation/Q-Solver-change018-layout-validation.exe`。
- 最新 SHA-256：`853e64c3ccd59101317d0fc4e92414a10f1d12695deed9694b0d689ee2e07343`。

## Windows GUI 人工验收（NEEDS_USER）

使用最新布局验证入口执行：

1. 当前 active 有记录时继续旧 ended：当前段变 paused；主界面显示旧段全部截图、完整答案和旧 Model/Scene。
2. 切回此前 paused：题库 current、主界面和模型上下文同步切换；下一题追加到切回段。
3. 继续旧 exported：旧导出目录仍在；追加新题、结束、再导出后生成 `_02` 或更高后缀，旧目录不变。
4. legacy 显示“按题库记录重建上下文，可能与当时实际上下文不同”；新 persisted Session 重启应用后恢复且不显示 legacy 警告。
5. 修改当前 Model/Scene/Prompt/简历配置后续接，下一次请求使用当前配置；旧 Model/Scene 只作为历史标注。
6. solve 中、Cancel 后 goroutine 未退出、存在待发送截图、未落盘、整理或导出中时续接被明确拒绝，当前会话不变。
7. discarded 无“继续此分段”；最近 20 之外不出现在列表且不能通过旧 ID 续接。
8. 快速 A→B、取消与迟到事件不串段；查看只读会话时后台流只更新真实 current。
9. Change 016 的固定 Mobile 链接、同 SSE 连续显示、Reset Link、Clear Display、左 `Shift+C`、滚动控制与输出模式不回退。
10. Change 017 的软清空、只读详情、两阶段整理、待核对元数据和 RawOnly 不回退。
11. 同屏混合 0、1、2 个操作按钮时左侧卡片右边界一致；继续与重新导出纵向等宽；长名称和动态标签不覆盖；`560px` 以下操作区切换为全宽单列。

人工验收与 race 未解决前，Change 018 不得标记为关闭或发布。
