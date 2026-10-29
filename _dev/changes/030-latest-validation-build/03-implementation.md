# 实现过程记录 03

## 实现日志

| 时间 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-10-28 | 事前归档 | 创建 Change 030 七份治理文档 | 本目录 | 获得用户设计批准后实施 |
| 2026-10-28 | 构建前保护 | 确认目标 EXE 不存在、相关进程为 0，并记录 11 个关键文件及 `.git/index` 的 SHA-256 | 工作树（只读检查） | 基线为 53 个 tracked 修改、87 个 untracked、0 个 staged |
| 2026-10-28 | 前端构建 | 使用现有 `node_modules` 执行 Vite 生产构建 | `frontend/dist/` | 132 个模块转换成功；未执行 install/ci |
| 2026-10-28 | 独立构建 | 使用当前 `frontend/dist` 编译唯一命名的生产模式 EXE | `build/bin/Q-Solver-change030-latest-validation.exe` | 跳过前端重建、bindings、embed 文件创建、mod tidy 和 go.mod 同步 |
| 2026-10-28 | 独立启动验证 | 直接启动新 EXE，观察 8 秒后经窗口关闭路径退出 | 新 EXE | WebView2 正常初始化；无 Node/Vite、无 CCR 连接；退出码 0，无强制清理 |
| 2026-10-28 | 完整性复核 | 对比保护文件、Git 语义索引、工作树计数及产物哈希 | 工作树（只读检查） | 11 个保护文件保持不变；Git index 的 stage listing 与保护副本一致、staged 为 0，但二进制 stat cache 被只读 Git 检查刷新；旧产物未覆盖 |

## 与计划的偏差

- 全树 `git diff --check` 返回 2，定位到构建前已存在的 `frontend/wailsjs/go/models.ts` 尾随空白。该文件构建前后 SHA-256 均为 `4BF59CCEDF31CB81D513FC159AB5EF2910EBEDA629F74C52EE365EF942929B17`，本 Change 不越权修改既有差异。
- 一次只读 `git diff --check` 未设置 `GIT_OPTIONAL_LOCKS=0`，导致 `.git/index` 的 stat cache 被刷新，二进制 SHA-256 从 `FE0948DA...` 变为 `5EC93489...`。当前与保护副本的 1,194 条 `git ls-files --stage` 条目哈希同为 `DB01E541F4E2FD18173AE578904C77CE07E01F9F3B79D838016A07A3568324A5`，且 staged 为 0；未改变索引语义，不执行越权回写。
- 其余实施步骤与计划一致。

## 关键决策纪要

- 不修改业务源码，只构建当前工作树快照。
- 产物使用唯一文件名，避免覆盖任何现有 EXE。
- 不安装 Wails v2.11.0；使用本机 v2.16.0 并记录版本限制。
- 自动运行验证只确认独立启动、WebView2 初始化、无开发服务器依赖和正常退出，不调用模型。

## 遗留未完成项

- 待用户直接双击新 EXE 进行 Windows GUI 人工验收。
- 既有 `frontend/wailsjs/go/models.ts` 尾随空白不属于本 Change，保持原样。
