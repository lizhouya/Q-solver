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
