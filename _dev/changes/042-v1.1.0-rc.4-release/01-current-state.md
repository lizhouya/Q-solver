# 现状快照 — 01

## 1. 涉及文件清单

| 文件 | 当前职责摘要 | 当前事实 |
|---|---|---|
| `README.md` | 中文项目入口 | 当前公开版本仍指向 `v1.1.0-rc.2` |
| `README_EN.md` | 英文项目入口 | Current Prerelease 仍指向 `v1.1.0-rc.2` |
| `CHANGELOG.md` | 公开变更日志 | 只记录 rc.1 与 rc.2 |
| `_dev/releases/` | Tag 自动 Release 正文 | `v1.1.0-rc.4.md` 不存在；不补造 rc.3 文件 |
| `.github/workflows/release.yml` | Tag 构建与自动 Release | 监听 `v*`，三平台构建后自动创建 Release |
| `_dev/changes/INDEX.md` | Change 编号与状态索引 | 最新编号为 041 |

## 2. Git 与远程现状

- 当前分支：`dev/qsolver-custom`，`main` 继续作为官方基线保留。
- `origin`：个人 fork `lizhouya/Q-solver`；`upstream`：官方项目，只读同步用途。
- 发布审计时工作区和暂存区 clean。
- 本地 HEAD 为 `7bc240e518216a6a067b224fbc33c0e5bd51b743`，远程分支 HEAD 为 `d3ccbdff5143adca7173423bf449dcd0755eac32`。
- 本地领先四个提交：Change 040 功能/治理和 Change 041 功能/治理。
- 已有最新相关 Tag 为 `v1.1.0-rc.3`；`v1.1.0-rc.4` 与 `v1.1.0` 均未占用。
- GitHub 仓库为公开个人 fork，当前账户具备 `ADMIN` 权限，默认分支为 `dev/qsolver-custom`。

## 3. 自动发布现状

`.github/workflows/release.yml` 的当前行为：

1. 所有 `v*` Tag push 触发工作流。
2. Windows amd64、macOS arm64、macOS amd64 使用 Go 1.25、Node.js 22 和 Wails v2.11.0 构建。
3. 三个平台分别执行全量 Go 测试、`go vet`、设置回归和 Mobile DOM 回归。
4. Create Release 生成 `SHA256SUMS`，上传三个平台附件。
5. Release 正文读取 `_dev/releases/${tag}.md`；文件缺失会导致发布失败。
6. Tag 含 `-` 时 `prerelease=true`；`make_latest=false`。

## 4. 当前验证与产物事实

- Change 040：同步端 F8～F12 五项工作流提示、满三图保留键帽和动态底部间距已通过自动验证及用户 GUI 验收。
- Change 041：选择题映射优先、一致性复核与“待核对”策略已通过自动验证及用户实际模型复验。
- 最近全量门禁包括 `go test ./... -count=1`、`go vet ./...`、`go build ./...`、`go mod verify`、19 个前端 Node 回归和 Vite build，均通过。
- Windows race 在测试进程启动阶段返回 `exit status 0xc0000139`，状态 `BLOCKED`，未进入测试体。
- 本地根稳定版为 `build/bin/Q-Solver.exe`，大小 42,558,464 字节，SHA-256 `2885358371510B827ACE560C354FEE8775BECEB47555D7CE654B0FBE3EB1F15B`，链接 Wails v2.11.0。
- macOS 将由 CI 构建，但当前没有人工 GUI 验收证据。

## 5. 公开发布现状

- 最新公开预发布为 `v1.1.0-rc.3`，目标提交 `43022e10cd8fb1b38b72b56d95bc39b8bb4d201d`。
- rc.3 为 prerelease，现有 Release 只包含 Windows EXE 与校验和；本地仓库没有对应 rc.3 正文，不追溯伪造。
- 当前公开文档落后于 rc.3，必须在新 Tag 前同步到 rc.4。

## 6. 已知现状问题

- race 环境阻塞和 macOS GUI 未人工验证，不满足稳定版声明条件。
- 部分较早 Change 的人工验收状态仍是 `NEEDS_USER`；rc.4 不等同于关闭这些 Change。
- 模型输出只能作为参考；提示词约束降低选择题错配概率，但不构成绝对正确保证。
