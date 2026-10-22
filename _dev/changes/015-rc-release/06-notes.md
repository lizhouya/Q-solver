# 发布说明与状态

目标为 `v1.1.0-rc.1` 预发布，不将测试缺口改成已验收，也不借此关闭 Change 010～014。

- 发布分支仅为 `origin/dev/qsolver-custom`；不操作 `upstream` 或 `main`，不 force push。
- 本地 RC 已用 Go `1.25.13` + Wails `2.11.0` 构建并设为 `build/bin/Q-Solver.exe`；旧 EXE 已备份且校验和可用。
- 原始迁移证据和机器清单只在本地保留，不进入公共仓库。
- 仍在运行的 `Q-Solver-sync-fix.exe` 未被强制终止。
- 公开 Release 的最终状态以 GitHub Actions 和 Release 元数据为准；失败时不删除或移动现有 Tag，不静默改写历史。

首次 RC1 workflow 的应用构建通过但回归测试失败，未创建 Release。现有 RC1 Tag 保留不动；本地 RC2 回修按代码/测试与发布治理拆分提交，任何推送和 RC2 Tag 仍需另行明确授权。
