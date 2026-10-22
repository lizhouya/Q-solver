# 发布前现状

当前分支 dev/qsolver-custom，基线 9efb96f；origin 为个人 fork，默认分支也是 dev/qsolver-custom。远端存在 v1.0.0/v1.0.1，无 Release。工作树包含 Change 010～014 代码及早先规则改动，暂存区空。

build/bin 中三个版本：Q-Solver.exe（旧版）、Q-Solver-api-fix.exe（API 中间版）、Q-Solver-sync-fix.exe（用户已验证实时同步和远程滚动）。最终版本包含全部修复，旧输出必须可回退。

现有 release.yml 在 v* Tag 时构建 Windows/macOS 并自动创建 Release，Wails @latest 会漂移工具链，尚未自动区分 RC。项目 runtime v2.11.0，本地 CLI v2.16.0。wails.json 版本为 1.0.3.0，无 CHANGELOG。

原规则要求各 Change 分开提交、验收完成才能发布；用户本次已明确批准整合及 RC 例外，需先写入 AGENTS.md/CLAUDE.md，而非仅在 Change 文档自授权。
