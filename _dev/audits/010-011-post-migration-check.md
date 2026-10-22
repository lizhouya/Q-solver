# Q-Solver 迁移后独立复核

日期：2026-10-19。范围：迁移完整性、迁移前未提交改动、题库功能自动检查及待验收项。本轮不修改业务逻辑、不提交或推送、不调用真实模型。

## 一、结论

- 新项目位置：`E:/DATA/求职_AI/求职笔试/Q-Solver`。旧根不再含 `.git`，项目已独立于笔试分类父目录和输出目录。
- 分支仍为 `dev/qsolver-custom`，HEAD 为 `9efb96f041688c03014c59762725020c08195926`。11 条 Git refs、1,015 条索引条目与迁移前证据一致，暂存区为空。
- 迁移前的修改没有被自动提交：本轮检查起点为 15 个已跟踪文件修改、54 个未跟踪文件。后者包含功能源码、测试、治理档案和迁移证据；本轮报告另新增一个未跟踪文件。
- 迁移不是验证空白：此前与 Codex 接手后均已有普通自动验证证据；本轮独立复跑再次通过普通测试及构建，但静态检查有既有告警，race 启动受阻，真机验收仍未完成。
- 两个 Change 仍不 Close，也不把“可构建”表述为“功能已全面验收”。

## 二、迁移一致性

依据 `../changes/011-job-workspace-layout/evidence/20261019-094017/source-before.json` 重新逐文件计算 SHA-256，而非只采信原有 PASS 字样：

| 检查 | 本轮结果 |
|---|---|
| 基线文件数量 | 10,655 |
| 缺失文件 | 0 |
| 字节一致 | 10,641 |
| 发生变化 | 14，限迁移文档、验证记录、Git 索引缓存和新构建 EXE |
| 业务源码/依赖/配置范围 | 119 项全部一致，未发现迁移导致的业务文件缺失或修改 |
| Git 索引语义 | 全部条目与基线一致；不把 stat 缓存导致的 index 字节变化认定为源码变化 |
| Git refs | 使用相同的 `refname objectname` 格式比较，11 条全部一致 |
| Git 完整性 | `git fsck --full --no-reflogs` 退出 0；57 条 dangling 提示，不属于缺失或损坏对象 |

14 项差异路径：README.md、README_EN.md、CLAUDE.md、INDEX.md、.git/index、build/bin/Q-Solver.exe、Change 010 的 04-validation.md、Change 011 的 00～06 文档。它们与迁移过程更新及重新构建的职责相符。

## 三、未提交资产的归属

### 迁移/题库开发之前已有的规则修改

- AGENT.md、CLAUDE.md、_dev/README.md、_dev/changes/INDEX.md。
- 新增但未跟踪的 AGENTS.md。

其中 CLAUDE.md 与 INDEX.md 又叠加了迁移路径和新 Change 状态。未来提交前需要区分增量归属，不能整文件夹带。

### Change 010 功能修改

- 已跟踪文件：app/app.go、app/solve.go、main.go；frontend/src/App.vue、components/TopBar.vue、services/api.js；frontend/wailsjs/go/app/App.js、App.d.ts、models.ts。
- 未跟踪源码：app/questionbank.go、questionbank_capture.go、questionbank_lifecycle.go、questionbank_test.go；frontend/src/components/QuestionBankPanel.vue、stores/questionbank.js；pkg/questionbank/ 的实现与测试。
- README.md、README_EN.md 为功能说明及目录说明，按治理边界管理。

### 治理与迁移证据

Change 010/011 档案、迁移脚本、哈希及 Git 证据、审计记录均尚未跟踪。最终提交需要逐个明确路径，不使用 git add . 或 git add -A。本轮没有暂存、commit、push。

## 四、本轮验证结果

环境：Windows 11；Go 1.27.1；Node.js 24.14.0；npm 11.9.0；Wails CLI 2.16.0；项目模块继续使用 Wails v2.11.0。

| 命令 / 检查 | 结果 |
|---|---|
| `go test ./pkg/questionbank ./pkg/solution ./app -count=1 -v` | PASS，退出 0；聚焦日志无 SKIP |
| `go test ./... -count=1` | PASS，退出 0；非缓存复跑 |
| `go build ./...` | PASS，退出 0 |
| `npm --prefix frontend run build` | PASS，退出 0；仍有大于 500 kB 的 bundle 提示 |
| `wails build` | PASS，退出 0；新路径下生成 build/bin/Q-Solver.exe |
| `go vet ./...` | FAIL，退出 1；Windows hook 两处 possible misuse of unsafe.Pointer |
| `go test -race ./pkg/questionbank ./app -count=1` | FAIL / BLOCKED，退出 1；两包均以 0xc0000139 退出 |
| `go test -race ./pkg/questionbank -run '^$' -count=1` | FAIL / BLOCKED，退出 1；无用例也启动失败 |
| 题库 10 个 API 的前端封装/生成绑定 | PASS |
| 本轮涉及的本地 Markdown 链接 | PASS |
| `git diff --check` | PASS |

### 静态检查问题的边界

```text
pkg\shortcut\shortcut_windows.go:122:38: possible misuse of unsafe.Pointer
pkg\shortcut\shortcut_windows.go:155:45: possible misuse of unsafe.Pointer
```

已将该文件与当前 HEAD 核对，内容一致，且与迁移前源码基线一致。这是原有 Windows hook 实现的告警，不是迁移或题库改动引入。不能因此把 go vet 记为通过，也不能单凭告警断言运行时一定崩溃。本轮检查没有擅自修复或禁用检查。

### race 的边界

零用例探针同样失败，说明当前未得到任何有效的 race 检测结果。没有使用普通测试通过来代替它，也没有修改工具链或系统配置来绕过。

### 功能测试覆盖

新增题库相关测试函数共 14 个，覆盖多图/多轮保存、完整原答案、可移动图片引用、版本化导出、恢复、写盘失败重试、取消与迟到回调、非法路径和图像、结构化字段与来源校验、多场次隔离和并发快照等。模型为 fake provider，截图为合成图片，不涉及真实题目或资料上传。

测试覆盖不等同于完整产品验收：未验证 Windows 界面实际交互、真实模型输出质量或手机完整链路。

## 五、输出与构建副作用

- 用仅调用 `questionbank.NewStore` 的临时探针直接检查真实 `E:/DATA/求职_AI/输出`：接受；同一探针确认源码目录仍被拒绝。没有开始场次、写个人偏好或调用网络。
- 原“自动网申”仍是空目录，仅是遗留清理项，不影响新仓库测试；本轮不删除。
- 构建前备份 11 个可能受影响的依赖/配置/生成文件。标准 Wails 构建造成 5 个文件变化：go.mod 的换行、runtime/package.json 的换行、runtime.js/runtime.d.ts 的新 CLI 通知 API、models.ts 的尾随空白。
- 已逐项比对后，仅将这 5 项恢复为本轮构建前的精确字节；没有使用 git restore 覆盖此前未提交内容。11 个文件最终哈希全部与本轮基线相同。
- 未新增或升级依赖；frontend/dist 和 build/bin 内的重新构建产物保留。

## 六、尚未完成的验收

| 项目 | 状态及建议操作 |
|---|---|
| 题库面板与目录选择 | NEEDS_USER：启动新 EXE，确认顶部入口、录制状态和输出目录选择 |
| 单图/多图与取消流程 | NEEDS_USER：使用允许处理的测试题目，检查截图数量、原答案和状态 |
| 整场 Markdown | NEEDS_USER：结束导出，核对图片、原答案、题序和重复导出不覆盖 |
| 重启恢复 | NEEDS_USER：未结束场次重新打开后为暂停，继续本场后才录制 |
| 退出保护 | NEEDS_USER：系统关闭/托盘退出的处理中或未保存弹窗 |
| 手机输出 | NEEDS_USER：desktop/mobile/dual 下真实联调 |
| 真实模型整理 | NEEDS_USER：用户明确确认后执行，核对文字题干、分类、来源和费用；本轮未调用 |
| macOS | NOT RUN：没有跨平台运行证据 |

建议先完成上述真机验收，并单独处理 go vet / race 问题，再决定是否提交。提交还须用户单独授权。

## 七、本轮修改范围

仅新增本复核报告、追加 Change 010/011 的验证说明及 INDEX 最新状态；保留全部已有业务源码和用户修改。构建噪音已定向撤销，测试/构建产物按原忽略规则保留。没有修改运行配置或密钥。

### 本轮命令耗时记录

- focused：退出 0，8.45 秒。
- all-tests：退出 0，4.11 秒。
- go-build：退出 0，2.5 秒。
- go-vet：退出 1，4.58 秒。
- frontend-build：退出 0，4.33 秒。
- wails-build：退出 0，16.66 秒。
- race：退出 1，4.2 秒。
- race-zero：退出 1，2.58 秒。
