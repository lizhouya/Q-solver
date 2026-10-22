# RC 发布实施计划

## 文件与步骤

1. 规则同步：AGENTS.md/CLAUDE.md 写入仅针对 v1.1.0-rc.1 的用户授权例外，其他规则不变。AGENT.md/_dev/README.md 原有规则整理单独提交。
2. 发布配置：.github/workflows/release.yml 固定 Wails v2.11.0 和版本化缓存键，显式平台，-nosyncgomod -m；RC 设置 prerelease 且不置为 latest；上传三个平台附件与 SHA256SUMS，使用已审核的发布说明。
3. wails.json Windows 数字版本更新为 1.1.0.1（表示本次 RC1），不新增运行依赖。
4. 新增 CHANGELOG.md、_dev/releases/v1.1.0-rc.1.md；同步中英文 README、INDEX、审计和本 Change 00～06。旧历史记录保留，追加当前状态说明，避免篡改历史测试结果。
5. 扫描本次新增和修改内容的凭据/私人绝对路径，必要时以通用路径占位符替代；不得将用户配置或 EXE 放入 Git。测试均用合成数据。
6. 最终本地 Go/Node/Vite 验证及 Wails 构建。将三个旧 EXE 备份到 build/bin/archive 的独立目录，逐个 SHA-256 核对，再整理 bin 根目录；若文件占用则不强停，记录未完成。
7. 精确路径暂存，依次按批准消息提交（均带 Co-Authored-By）；最后提交前写完发布记录，避免 Tag 后修改源码。全部检查通过后推送分支，再创建带说明 Tag 并单独推送。
8. 跟踪 Actions；成功后核对 prerelease、附件数量、Tag SHA、远端分支与工作区。若失败先诊断，禁止称为成功；不得自动改标稳定版、覆写 Tag 或发布错误附件。

## 回退与边界

不 reset/clean、不 force push、不覆盖既有 Tag。提交异常停止并报告；已发布版本问题通过新版本修复，删除 Release/Tag 须另行确认。旧 EXE 和数据不删除，规则例外仅本次有效。
