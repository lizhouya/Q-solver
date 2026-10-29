# 变更请求 — 00

| 字段 | 值 |
|---|---|
| 变更编号 | `030-latest-validation-build` |
| 提出人 | 用户 |
| 提出日期 | 2026-10-28 |
| 关联 Issue | 无 |
| 状态 | `规划中` |

## 1. 背景与问题

当前工作树包含尚未提交的最新二次开发成果。`build/bin/Q-Solver-dev.exe` 是 `wails dev` 的临时开发二进制，依赖 Wails DevServer 与 Vite，不能作为普通独立程序直接双击。目录中的 `Q-Solver.exe` 生成于 2026-10-22，早于当前工作树，不能代表本次远程 CCR 联调验证过的最新版本。

用户要求生成一个可独立双击运行、用于验证当前最新工作树的 Windows EXE，同时保留所有现有修改和历史构建产物。

## 2. 目标

- 使用当前工作树和现有依赖生成独立 Windows 验证 EXE。
- 新 EXE 不依赖 Vite 或 Wails DevServer，能够直接启动。
- 不覆盖 `Q-Solver.exe`、`Q-Solver-dev.exe` 或历史验证程序。
- 不修改业务源码、用户 API 配置或远程 CCR。
- 记录构建命令、产物 SHA-256、启动验证及构建前后完整性结果。

## 3. 验收标准

- [ ] `frontend/dist` 由当前前端源码成功构建。
- [ ] 生成 `build/bin/Q-Solver-change030-latest-validation.exe`。
- [ ] EXE 可直接启动，进程和 WebView2 初始化正常。
- [ ] EXE 启动不依赖 Vite/Wails DevServer。
- [ ] `wails.json`、`go.mod`、`go.sum`、package 文件和现有 Wails bindings 无非预期内容变化。
- [ ] 未安装或升级依赖，未覆盖既有 EXE，未执行提交或推送。

## 4. 非目标 / 明确不做

- 不修改 Q-Solver 业务源码或请求协议。
- 不重新设计启动器、安装器或自动更新机制。
- 不构建 Release、Tag 或对外发布产物。
- 不替换当前正式文件名 `Q-Solver.exe`。
- 不修改本机或远程 CCR，不发送模型请求。
- 不解决既有 race 环境阻塞或其他历史 Change 遗留项。

## 5. 影响面预估

- 涉及业务源码文件：无。
- 治理文件：本目录 00–06、`_dev/changes/INDEX.md`。
- 忽略产物：`frontend/dist/**`、`build/bin/Q-Solver-change030-latest-validation.exe`。
- 潜在风险：Wails CLI v2.16.0 与项目模块 v2.11.0 不一致；构建工具可能生成 runtime、资源或临时文件；现有工作区已有大量未提交修改。

## 6. 附带约束

- 仅在 `dev/qsolver-custom` 工作，不切换分支。
- 使用现有 `node_modules`，禁止 `npm install`、`npm ci`、`go mod tidy` 和依赖升级。
- Wails 构建必须使用 `-nosyncgomod -m -skipbindings` 等保护参数。
- 所有 secret 只以 `SET` / `NOT SET` 表示。
- 本 Change 不含 commit、push、Tag 或 Release 授权。
