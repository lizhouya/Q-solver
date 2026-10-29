# 现状快照 — 01

## 1. 涉及文件清单

| 文件 | 当前职责摘要 | 备注 |
|---|---|---|
| `wails.json` | Wails v2 项目配置与前端命令 | tracked，当前未修改 |
| `frontend/package.json` | Vite 前端构建脚本 | `dev` / `build` / `preview` |
| `frontend/package-lock.json` | 前端依赖锁文件 | 当前存在，不允许改写 |
| `frontend/dist/**` | Wails 嵌入的前端静态产物 | ignored，当前存在但早于最新工作树 |
| `frontend/wailsjs/**` | Wails JS bindings/runtime | 当前有既有未提交修改，不允许本次重生成 |
| `build/bin/**` | Windows 构建与历史验证程序 | ignored，不覆盖旧产物 |
| `_dev/changes/INDEX.md` | Change 总索引 | 本次登记 Change 030 |

## 2. 关键代码现状

- `main.go` 通过 `//go:embed all:frontend/dist` 嵌入前端，因此独立 EXE 必须在 Go/Wails 构建前生成最新 `frontend/dist`。
- `wails dev` 生成的 `Q-Solver-dev.exe` 由 CLI 配合 Vite 与 DevServer 启动，不是独立发行程序。
- 项目已有独立验证构建惯例：先运行 `npm --prefix frontend run build`，再使用 `wails build -s -o <唯一文件名> -skipbindings -nosyncgomod -m`。

## 3. 相关配置

- 项目根目录：`E:\DATA\求职_AI\求职笔试\Q-Solver`。
- 分支：`dev/qsolver-custom`。
- HEAD：`43022e10cd8fb1b38b72b56d95bc39b8bb4d201d`。
- 工作区：53 个 tracked 修改、80 个 untracked 文件、0 staged。
- 用户配置位于仓库外，本次不读取明文、不修改配置。

## 4. 依赖与构建现状

- Wails CLI：v2.16.0。
- Go 模块 Wails：v2.11.0。
- Go：1.27.1。
- Node.js：v24.14.0。
- `frontend/node_modules/.bin/vite.cmd`：存在。
- `frontend/dist/index.html`：存在，时间为 2026-10-26 22:01:31，不足以证明包含当前全部工作树修改。
- `frontend/package.json.md5`：存在。
- CI 正式构建使用 Go 1.25、Node 22、Wails CLI v2.11.0；本机不安装或切换 CLI，版本差异作为验证限制记录。

## 5. 运行行为现状

- `Q-Solver-dev.exe` 双击后不能独立工作；正确开发入口是 `wails dev`。
- 旧 `build/bin/Q-Solver.exe` 时间为 2026-10-22 20:03:44，不能代表最新工作树。
- 当前源码已通过远程 CCR 的真实文本、图片和人工快捷键联调，但尚无与该工作树同步的独立 EXE。
- Q-Solver 默认启用 GhostMode，独立启动后窗口可能隐藏，可通过 F1 或托盘恢复。

## 6. 已知现状问题

- 当前工作区不是 clean，构建结果必须明确标记为“当前未提交工作树快照”。
- 本机 Wails CLI 与项目模块版本不一致，不能冒充 CI 的 v2.11.0 构建复现。
- 历史存在 Windows race 启动错误 `0xc0000139`，本次不处理。
- 首轮 `wails dev` 曾改写两个 runtime 文件；本次以当前哈希为基线，不自动恢复历史内容。
