# 实现记录

## 发布工具链

- 将 GitHub Actions 的 Wails CLI 固定为 `v2.11.0`，与项目 runtime 一致；缓存键包含精确版本。
- 三个平台构建显式使用 `-nosyncgomod -m`，macOS 分别指定 `darwin/arm64` 与 `darwin/amd64`。
- 各平台构建后运行 Go 全量测试、`go vet`、设置回归和 Mobile DOM 回归。
- Release job 生成 `SHA256SUMS`，读取版本化发布说明，并按 Tag 是否含 `-` 自动设置预发布；`make_latest: false`。
- Windows 数字版本更新为 `1.1.0.1`；新增中英文预发布入口、`CHANGELOG.md` 和本次 RC 发布说明。

## 本地构建与版本整理

- 系统 Go `1.27.1` 配合 Wails `2.11.0` 首次构建失败，错误为 `internal error: package "errors" without types was imported from "Q-Solver/pkg/common"`。该结果未记为通过。
- 使用隔离的 `GOTOOLCHAIN=go1.25.13` 与隔离 Wails CLI `v2.11.0` 重建成功，未升级项目 runtime 或系统默认 Go。
- 正式本地 Windows 产物：`build/bin/Q-Solver-v1.1.0-rc.1-windows-amd64.exe`，大小 `23,126,528` bytes，SHA-256：`08b1419dfe24db0b59e8e7bff9779e7e27eb67a9004f032209736cf7a1b734ea`。
- Wails 构建将两个 runtime shim 的工作树行尾从 CRLF 改成 LF；已逐文件恢复构建前字节，`go.mod`、`go.sum` 和 runtime shim 无发布差异。
- 三个旧 EXE 已复制到 `build/bin/archive/pre-v1.1.0-rc.1-20261022/` 并生成独立 `SHA256SUMS`；未终止仍在运行的 `Q-Solver-sync-fix.exe`。
- 根入口 `build/bin/Q-Solver.exe` 已替换为本次 RC，且与版本化 RC 文件 SHA-256 一致。`build/bin/` 继续由 Git 忽略，不进入提交或 Release 源码历史。

## 公共提交边界

Change 011 的 `evidence/` 包含约 10 MB 的一次性目录快照、Git 状态和本机绝对路径，`migration-manifest.json` 也包含本机清单。这些原始材料保留在本地，并在 `.gitignore` 中按精确路径忽略；公共仓库只保留 Change 文档和迁移脚本，不发布冗余本机快照。

发布准备按批准边界拆为三组提交：维护规则；Change 010～014 功能与测试；发布工具链、Change/审计文档和 RC 说明。不得使用通配全量暂存。

## RC1 CI 回修

- workflow `39774286851` 的三平台 Wails 构建均成功；两处题库测试因系统路径别名直接比较字符串而失败，Apple Silicon 另有临时端口重绑竞争。
- 题库断言先解析目标目录的符号链接/系统别名，再与导出结果父目录比较；没有改变一次性导出目录的业务逻辑。
- `Server.Start()` 在请求端口为 `0` 时读取内核分配的实际 TCP 端口。生产默认端口保持 `38511`，仅测试从“先探测后重绑”改为原子监听。
- RC2 Windows 数字版本准备为 `1.1.0.2`，新增独立发布说明；RC1 Tag 保持原目标，不移动、不删除。

RC2 本地 Windows 产物为 `build/bin/Q-Solver-v1.1.0-rc.2-windows-amd64.exe`，大小 `23,126,528` bytes，SHA-256：`5dcabb17933bfa9ab4dc15f289b4b8e50af99879c316b58831b09bbb4e39aa11`。根入口 `Q-Solver.exe` 已更新为同一文件；版本化 RC1 仍保留，旧版备份目录未改动。
