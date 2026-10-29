# 总结与注意事项 06

## 变更摘要

本 Change 已把当前未提交工作树构建为唯一命名、可独立双击的 Windows 验证 EXE，不修改业务源码、不覆盖旧程序。自动化独立启动与正常退出验证通过，现交付用户进行 GUI 人工验收。

## 经验与踩坑

| 现象/坑 | 原因 | 结论/建议 |
|---|---|---|
| `Q-Solver-dev.exe` 双击无窗口 | 它是 `wails dev` 临时二进制，依赖 DevServer/Vite；GhostMode 也可能隐藏窗口 | 开发态使用 `wails dev`，独立验证必须使用 `wails build` 产物 |
| 本地 CLI 与模块版本不同 | CLI v2.16.0、Go 模块 v2.11.0 | 构建使用保护参数并记录限制，不自动升级模块 |

## 对本项目后续的影响

产物是构建时工作树快照，不会随源码自动更新。后续任何代码变化后，若要验证“最新版本”，必须重新执行受控构建并更新哈希。

## 产物与验证结论

- 产物：`build/bin/Q-Solver-change030-latest-validation.exe`
- SHA-256：`5924B7D00CE0B00A4BE5A0169EDDD8D900F9442E96B621ACE7F374DD28AB07EF`
- 自动验证：直接启动、WebView2 初始化、无 Node/Vite、无 CCR 连接、退出码 0、无残留进程。
- 完整性：关键配置、依赖文件与 bindings/runtime 哈希均未改变；Git 语义索引与保护副本一致且 staged 为 0。一次只读 Git 检查刷新了 `.git/index` 的 stat cache，二进制哈希变化已在验证记录单列，不执行回写。
- 基线限制：全树 `git diff --check` 仍被构建前已有的 `frontend/wailsjs/go/models.ts` 尾随空白阻塞，本次未修改该文件。

## 遗留项与 TODO

- 待用户直接双击新 EXE 进行 Windows GUI 人工验收。
- 本 Change 不自动成为 Release，也不替换正式 `Q-Solver.exe`。

## 参考资源

- `wails.json`
- `.github/workflows/release.yml`
- `_dev/changes/014-api-settings-recovery/04-validation.md`
- 项目 `AGENTS.md` 与 `CLAUDE.md`

## 2026-10-29 整合提交状态

- 用户批准 Change 024～031 一次性跨 Change 整合。
- 功能/测试提交：`2a8a506636496a1eb6c38662c6ef6eaccc1fe958`（Change 030 无业务源码，仅归档治理记录）。
- 本 Change 的治理档案纳入 `chore: archive changes 024-031 and define artifact lifecycle`。
- 本轮全量自动验证通过；原有 `NEEDS_USER` 与 race `BLOCKED` 项保持，不因提交自动关闭。
- 不创建 Tag、GitHub Release 或新 RC。
