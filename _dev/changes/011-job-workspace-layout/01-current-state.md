# 现状快照


- 当前根目录就是 Q-Solver 仓库，分支为 `dev/qsolver-custom`；“自动网申”已核实为空。
- 已有修改：`AGENT.md`、`CLAUDE.md`、`_dev/README.md`、`_dev/changes/INDEX.md`；另有未跟踪的 `AGENTS.md`。保留这些内容，不覆盖、不清理、不擅自提交。
- `app/solve.go:113-152` 为一次请求复制完整截图，但桌面 `user-message` 只发送第 1 张。不能用桌面历史作为完整归档数据源。
- `frontend/src/stores/solution.js` 的历史仅在内存，包含覆盖、删除和多轮合并行为；已有导出是答案图片，并非整场 Markdown。
- `pkg/solution/solution.go` 的 `chatHistory` 是模型上下文，最多保留最近 10 轮；归档必须与 `KeepContext` 解耦。
- `app/mobile.go:35-49` 在手机独占模式下不把答案送到桌面；归档必须发生在输出分流之前。
- `pkg/config/manager.go:31-47` 正常使用系统用户配置目录存放加密配置。迁移仓库不要求迁移或重写密钥。
- 新 Change 编号已交叉核对 INDEX、实际目录及近期提交，下一编号为 010；003/004 已使用，不复用。


## 环境

Windows 11；Go 1.27.1，Node.js 24.14.0，npm 11.9.0，Wails CLI 2.16.0；项目 Wails 模块保持 v2.11.0。尚未运行业务测试。不读取用户密钥实值。

## 2026-10-19 接管复核快照

- 旧工作副本仍位于 `E:\DATA\求职_AI`，分支 `dev/qsolver-custom`，HEAD `9efb96f041688c03014c59762725020c08195926`；暂存区为空，15 个已跟踪文件有未暂存修改，31 个未跟踪文件条目必须保留。
- `git fsck --full --no-reflogs` 退出 0，仅报告 dangling objects；未发现 alternates、linked worktrees、modules 或 shallow 依赖。
- `求职笔试/Q-Solver` 在复制前为空；`求职网申`、分类输出目录存在；旧 `自动网申` 仍为空。
- 根项无重解析点，未发现 Git lock 文件或显式 ACL Deny。旧 PID 不再沿用；本轮未结束进程、未改 ACL、未取得所有权。
- 由于 `.git` 的目录移动已多次稳定失败，本轮不再运行旧移动脚本，改用用户已授权的完整副本复制和三方全量哈希验证。
