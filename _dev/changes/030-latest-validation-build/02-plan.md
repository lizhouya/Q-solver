# 实施方案 — 02

## 1. 方案概述

不改业务源码。先用现有锁定依赖把当前前端源码构建到 `frontend/dist`，再通过 Wails 的安全参数跳过依赖整理、模块同步、bindings 生成和重复前端构建，输出唯一命名的独立 Windows 验证 EXE。构建后实际启动 EXE，核对进程、WebView2/配置初始化、直接运行能力及工作区完整性。

## 2. 分步实施计划

| 步骤 | 操作 | 涉及文件 | 预期产出 | 完成标志 |
|---|---|---|---|---|
| 1 | 建立 Change 030 归档和 INDEX 条目 | `_dev/changes/030-*`、INDEX | 事前治理记录 | 00/01/02 完整 |
| 2 | 记录 Git 状态和关键哈希 | 只读 | 构建基线 | 状态与哈希已保存 |
| 3 | 构建当前前端 | `frontend/dist/**` | 最新 Vite 静态资源 | `npm --prefix frontend run build` 退出 0 |
| 4 | 生成独立验证 EXE | `build/bin/**` | `Q-Solver-change030-latest-validation.exe` | Wails build 退出 0 |
| 5 | 产物与实际启动验证 | EXE、仓库外日志 | SHA-256、进程/启动证据 | 独立启动和退出正常 |
| 6 | 完整性复核与归档 | 03–06、INDEX | 最终验证记录 | 无非预期项目差异 |

## 3. 涉及业务源码的改动清单

无。不得修改 `app/**`、`pkg/**`、`frontend/src/**`、`main.go`、`go.mod`、`go.sum`、`frontend/package*.json` 或 `wails.json`。

允许写入：

- `_dev/changes/030-latest-validation-build/**`
- `_dev/changes/INDEX.md`
- ignored `frontend/dist/**`
- ignored `build/bin/Q-Solver-change030-latest-validation.exe`
- 仓库外脱敏启动日志

## 4. 测试与验证计划

1. `npm --prefix frontend run build`。
2. `wails build -s -o Q-Solver-change030-latest-validation.exe -skipbindings -skipembedcreate -nosyncgomod -m -nocolour -v 1 -ldflags "-s -w" -tags prod`。
3. 计算 EXE SHA-256、大小和修改时间。
4. 直接启动 EXE，不运行 Vite/Wails DevServer；确认进程存活、WebView2 和配置加载成功，随后正常关闭。
5. 比较构建前后 Git 状态及关键文件哈希。
6. 执行 `git diff --check`；不以 race 结果冒充普通构建验证。

## 5. 回滚预案

若构建失败或产物不可运行：

- 保留日志和现场，不修改业务源码补救。
- 产物使用唯一文件名，不影响旧 EXE；停止使用该新产物即可回滚。
- `frontend/dist` 为 ignored 生成物，暂不自动删除或恢复。
- 任何非预期 tracked 文件变化均保留并报告，未经确认不自动覆盖。

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| CLI v2.16.0 与模块 v2.11.0 不一致 | 构建行为可能与 CI 不完全一致 | 禁止同步 go.mod/生成 bindings，记录限制并实际启动验证 |
| 前端构建隐式安装依赖 | 违反授权 | 只运行 `npm run build`，前置确认 node_modules 与 Vite 已存在 |
| Wails 覆盖旧程序 | 丢失历史产物 | 使用唯一输出名，不使用 `-clean`，不覆盖 `Q-Solver.exe` |
| 生成 runtime/bindings 噪音 | 污染工作区 | 使用 `-skipbindings`，构建前后哈希核对 |
| GhostMode 造成“未打开”误判 | 误认为 EXE 失败 | 同时检查进程、日志、托盘/F1 行为 |
| 启动自动调用远程模型 | 费用风险 | 仅启动并读取配置，不触发模型目录或解题请求 |
