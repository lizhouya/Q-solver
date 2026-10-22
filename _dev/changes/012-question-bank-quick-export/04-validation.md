# 验证记录 04

## 构建验证

| 命令 | 结果 | 备注 |
|---|---|---|
| `go test ./pkg/questionbank ./pkg/mobile ./pkg/shortcut ./app -count=1` | PASS | Change 012 聚焦包全部通过。 |
| `go test ./... -count=1` | PASS | 全量 Go 测试通过。 |
| `go vet ./...` | PASS | 原两处 Windows hook `unsafe.Pointer` 告警已消除。 |
| `go build ./...` | PASS | 全部 Go 包构建通过。 |
| `npm --prefix frontend run build` | PASS | Vite 4.5.14，121 modules；保留既有单 chunk 超过 500 kB 警告，主 JS 约 806.81 kB。 |
| 提取 `pkg/mobile/web/index.html` 内联脚本后执行 `node --check -` | PASS | 手机页面脚本语法通过。 |
| `wails build -clean -platform windows/amd64 -o Q-Solver.exe -trimpath -m -skipbindings -nosyncgomod` | PASS | 完整生产构建成功，输出 `build/bin/Q-Solver.exe`。 |
| `wails build ... -s ...` | PASS | 前端完整构建后再次编译最终 Go 改动，输出同一可执行文件。 |
| `git diff --check` | PASS | 无空白错误或冲突标记；仅显示 Git for Windows 行尾提示。 |
| `go test -race ./pkg/questionbank ./pkg/mobile ./app -count=1` | BLOCKED | 三个包均在启动阶段 `exit status 0xc0000139`，与 Change 012 前已知环境问题一致。 |

## 运行验证

| 检查项 | 预期 | 实际 | 通过 |
|---|---|---|---|
| 自动建段与分段隔离 | 第一次 Begin 懒创建；封段后下一题进入新段 | 自动测试覆盖并通过 | 是 |
| 首次保存失败 | 保留稳定 record ID、截图和答案内存状态，恢复目录后可 Flush | 故障注入测试通过 | 是 |
| dirty 终态保护 | 保存成功前不得被下一次 Begin 覆盖 | 故障注入测试通过 | 是 |
| 软清空 | 原始文件保留，下一题进入新段；Exported 不可误清空 | 自动测试通过 | 是 |
| 一键导出 | 当前段封存并异步导出；普通按钮不重复处理 Exported | App 测试通过 | 是 |
| 历史重导出 | Ended/Exported 可显式生成新版本 | App 测试验证目录不同并通过 | 是 |
| 部分成功报告 | 文件生成、状态保存失败时仍保留结果路径 | manifest 故障注入测试通过 | 是 |
| 路径布局 | 直接位于 ArchiveDir，固定文件名，冲突加 `_02` | 自动测试通过 | 是 |
| 旧版实际目录迁移 | 旧“求职笔试/题库/按单位/未命名”层级不再保留 | 5 份导出、3 个会话逐目录哈希核对；Store 实际读取 3 个会话 | 是 |
| v1/legacy | 可枚举并打开旧记录，不移动或覆盖 | 自动测试通过 | 是 |
| Markdown 分类 | 按 Category → Type，原答案回链 | 自动测试通过 | 是 |
| 模型字段安全 | HTML/外链被本地中和，原答案不改写 | 自动测试通过 | 是 |
| 截图大小一致 | 写入端拒绝导出端不能读取的 >20 MiB 内容 | 边界测试通过 | 是 |
| Mobile 快捷键 | 仅发送标签和显示名，不泄露内部值 | 自动测试通过 | 是 |
| SSE 首帧 | 只同步 `ControlSeq`，不重放旧命令 | 端到端测试通过 | 是 |
| SSE 满队列控制 | 滚动命令不被 64 帧积压静默丢弃 | 满队列测试通过 | 是 |
| 输出路由 | desktop/mobile/dual 和不可用回退符合约定 | 路由矩阵测试通过 | 是 |
| Windows GUI 交互 | 设置、做题、分段、清空、导出可见行为正确 | 未启动 GUI，需用户验收 | NEEDS_USER |
| 手机真机 | 隐藏桌面时提示清晰，按钮/快捷键滚动真实页面 | 未连接真实手机 | NEEDS_USER |
| 真实模型整理 | 只在开关明确启用后上传当前段数据 | 未调用用户真实模型、未消费 API 额度 | NEEDS_USER |
| 退出保护 | 标题栏、托盘、前端退出均显示正确保护提示 | 代码与构建通过，弹窗需人工确认 | NEEDS_USER |

## 验收标准对照

- [x] 路径不再重复追加，最终目录由设置直接指定。
- [x] 题库设置与常规、模型、截图、输出、简历、API 并列。
- [x] 不再要求手动开始场次；答案旁按钮一键处理当前段全部记录。
- [x] 手动分段和软清空控制后续导出范围，不删除旧原始文件。
- [x] 模型整理默认关闭；启用后逐条处理并按大类/题型输出。
- [x] 部分失败/取消仍导出原稿和已成功结果；历史入口可显式重试。
- [x] 手机端显示实际快捷键，电脑按钮/快捷键可按输出模式控制滚动。
- [x] 两处 Windows hook vet 告警真正修复，没有关闭检查或添加忽略规则。
- [x] 没有新增依赖，没有自动调用真实模型，没有修改密钥或 agentmemory。
- [ ] Windows GUI、手机真机、真实模型整理和退出弹窗人工验收。

## 回归与副作用检查

- `HistoryItem.vue` 未移除旧答案图片导出；`SolveView.vue` 仍连接 `solution.exportImage(idx)`。
- 归档发生在 desktop/mobile/dual 输出分流之前，三种模式的原始记录一致。
- Wails CLI 2.16.0 构建时改写的 2.16 runtime notification shim 与项目 2.11.0 不匹配，已逐文件检查并恢复；`go.mod`、`go.sum` 与业务 bindings 构建前后哈希未变化。
- 构建后暂存区为空；没有 commit、push、Tag 或 Release。
- 未删除、移动或覆盖任何旧题库记录和用户输出。

## 结论

状态：**实现与普通自动验证完成；人工验收 `NEEDS_USER`，race 检测环境 `BLOCKED`，未提交、未关闭。**

## 2026-10-20 Mobile 人工反馈回修验证

| 命令 / 检查 | 结果 | 备注 |
|---|---|---|
| `node pkg/mobile/web_dom_test.mjs` | PASS | 首帧不重放、控制只执行一次、70% 视口距离、历史选择保持、业务新帧与 hints 增量更新均通过。 |
| `go test ./app ./pkg/mobile ./pkg/shortcut -count=1` | PASS | 覆盖状态码、端口释放后重试、运行中幂等、窗口不隐藏、路由矩阵与既有 Bridge/Server 契约。 |
| `go test ./... -count=1` | PASS | Mobile 回修完成后全量 Go 回归通过。 |
| `go vet ./...` | PASS | 未发现新的静态检查问题。 |
| `go build ./...` | PASS | 全部 Go 包构建通过。 |
| `npm --prefix frontend run build` | PASS | Vite 4.5.14，123 modules；主 JS 816.87 kB，只有既有的大 chunk 警告。 |
| Wails 正式 bindings 再生成并编译 | PASS | `RetryMobileOutput` 与 `MobileInfo.status/retryable` 由工具链生成并通过桌面应用编译。 |
| `wails build -s -o Q-Solver-change012-mobile-final.exe -skipbindings -nosyncgomod -m` | PASS | 使用最终 `dist` 生成资产并编译桌面应用；备用验证 EXE 随后删除，原 `Q-Solver.exe` 未触碰。 |
| Wails 生成噪音检查 | PASS | `go.mod`、`go.sum` 无变化；CLI 2.16.0 生成的 runtime notification shim 已定向恢复，项目仍使用 runtime 2.11.0。 |
| 未跟踪文本空白/冲突标记扫描 | PASS | 71 个未跟踪文本文件未发现行尾空白或冲突标记。 |
| `go test -race ./app ./pkg/mobile -count=1` | BLOCKED | 测试进程启动即返回 `exit status 0xc0000139`；未使用普通测试冒充 race 通过。 |

### 回修契约对照

- [x] 设置页每次打开或进入“输出”标签重新读取 Mobile 状态。
- [x] 状态读取失败与服务未启动、初始化失败、端口不可用、已停止、已就绪分开呈现。
- [x] 未保存的 `outputMode` 草稿与当前实际生效模式分开显示。
- [x] 可重试失败提供显式恢复入口；运行中重试幂等，重试成功不触发桌面窗口隐藏。
- [x] 顶部上下箭头继续作为滚动按钮，全局键盘快捷键继续复用统一后端路由。
- [x] wheel 只绑定箭头控制组，忽略横向为主输入，支持 delta 模式归一化、小幅累计、空闲重置与 140ms 节流。
- [x] 手机控制帧不重建答案/历史 DOM、不重置历史选择；首帧和断线重连不重放旧命令。
- [x] Mobile 不可用时滚动安全回退桌面；未修改 Windows 全局 Mouse Hook。
- [x] 未新增依赖、未升级 Wails runtime、未调用真实模型、未记录连接 URL 或 Token。

### 仍需人工复验

- [ ] Windows WebView2 中每次打开设置均显示准确状态；端口占用后释放并重试可恢复连接入口。
- [ ] 重试恢复时当前桌面窗口不会突然隐藏。
- [ ] 箭头组内的鼠标滚轮与触控板手感正常，设置弹窗、答案阅读区和窗口其他区域不被抢占。
- [ ] `desktop`、`mobile`、`dual` 三种已保存模式下，按钮、`Alt+PgUp/PgDn` 与局部 wheel 路由正确。
- [ ] 手机查看历史答案时滚动不跳回最新，断线重连不重放旧滚动，新答案仍正常进入历史。

### 回修结论

Mobile 回修实现、DOM 状态机、Go 回归、静态检查、前端构建、正式 bindings 和 Wails 桌面应用编译均已通过。Windows WebView2 与手机真机仍为 `NEEDS_USER`，race 仍为 `BLOCKED`；Change 012 保持待人工复验、未提交、未关闭。

## 2026-10-22 无需刷新实时同步回归

| 验证 | 结果 | 说明 |
|---|---|---|
| 修复前 TestBusinessSnapshotSequenceAdvancesOnEveryEvent | FAIL（已复现） | thinking snapshot seq=1, previous=1 |
| 修复前 TestTerminalSnapshotSurvivesFullQueue | FAIL（已复现） | done/error 均被满队列丢弃 |
| 修复后三项聚焦测试 | PASS | 序号逐帧递增、终态可达与控制保留、真实 SSE → DOM 连续两题无需刷新 |
| go test ./... -count=1 | PASS | 含真实 SSE → DOM 联合测试，非仅手造前端快照 |
| go vet ./... / go build ./... | PASS | 标准静态检查与构建 |
| node frontend/settings_regression_test.mjs | PASS | API 恢复功能未回归 |
| node pkg/mobile/web_dom_test.mjs | PASS | 控制去重、历史保持、重连基线仍通过 |
| npm --prefix frontend run build | PASS | 123 modules，818.21 kB 主 chunk，既有体积警告 |
| wails build -s -o Q-Solver-sync-fix.exe -skipbindings -nosyncgomod -m | PASS | 新程序保留，未覆盖旧程序，runtime 生成噪音已定向恢复 |

真实跨电脑/手机浏览器连续解题为 NEEDS_USER；race 本次未重跑，同日 Change 014 已复现 Windows 0xc0000139，继续 BLOCKED。没有调用真实模型，没有修改私人配置。

人工步骤：正常退出旧实例 → 启动 sync-fix → 确认已保存 dual → 从设置复制本次启动的新连接地址 → 远端保持页面打开，连续正常解题，确认不刷新即可更新。旧地址含上次进程的凭据，不能沿用。

## 2026-10-22 用户实测确认

| 人工验证项 | 结果 | 用户反馈依据 |
|---|---|---|
| 另一台电脑的同步页无需逐题刷新即可显示结果 | PASS（用户确认） | 实时同步修复后反馈“验证测试通过了” |
| 答题电脑控制同步端页面滚动 | PASS（用户确认） | 说明控制方式后反馈“验证通过” |

用户未逐项说明采用的输入方式或全部模式组合，故不据此勾选全部键盘/按钮/滚轮、三模式矩阵、手机真机、重连及端口恢复测试。上述两项替代先前对应的待验收状态；其他人工项和 race BLOCKED 保持不变。本次仅更新文档，未重跑业务测试，未提交或发布。
