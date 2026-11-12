# 实现过程记录 03

## 当前状态

Change 042 已完成发布准备提交、分支推送和 `v1.1.0-rc.4` annotated Tag 推送。Tag push 未自动生成 run，使用 workflow 已有的 `workflow_dispatch` 回退路径在同一 Tag/commit 上启动 run `43212159818`。Windows PASS，macOS arm64 回归 FAIL，Intel CANCELLED，Create Release SKIPPED；rc.4 Release 未创建，后续由 Change 043 修复并使用 rc.5 前进发布。

## 实现日志

| 时间 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-11-12 | 发布审计 | 核对个人 fork、权限、分支、已有 Tag/Release 与自动 workflow | Git / GitHub / release workflow | 最新公开版本为 rc.3，rc.4 未占用 |
| 2026-11-12 | 版本决策 | 将计划从稳定版收敛为 `v1.1.0-rc.4` | Change 042 | race BLOCKED 与 macOS GUI 未验收不能包装成稳定版 |
| 2026-11-12 | 事前归档 | 创建 00/01/02、发布审计和允许文件白名单 | Change 042 / audit | 不修改业务源码或 workflow |
| 2026-11-12 | 提交范围核对 | 读取 rc.3 到 HEAD 的实际提交清单 | Git log | 覆盖 Change 024～041 |
| 2026-11-12 | 发布文档 | 编写 rc.4 说明并同步 README、README_EN、CHANGELOG、INDEX | 发布文档 | 明确预发布、下载、验证和已知限制 |
| 2026-11-12 | 发布前验证 | 核对 rc.4 相对链接、占位符、VS Code diagnostics、Git 差异与本地/远程 Tag | 文档 / Git | 链接目标存在；diagnostics 与 `git diff --check` 通过；Tag 未占用 |
| 2026-11-12 | 发布准备提交与分支推送 | 创建 `36b4192` 并推送 `origin/dev/qsolver-custom` | Git | 本地/远程 HEAD 一致 |
| 2026-11-12 | Tag 推送 | annotated Tag `v1.1.0-rc.4` 指向 `36b4192` | origin | 远程 peeled commit 核对一致 |
| 2026-11-12 | workflow 回退调度 | Tag push 未生成 run，按同一 Tag 使用已有 `workflow_dispatch` | GitHub Actions | run `43212159818`，head 为 rc.4/`36b4192` |
| 2026-11-12 | 远程失败 | Windows PASS；macOS arm64 回归 FAIL；Intel CANCELLED；Release SKIPPED | Actions | rc.4 Release 未创建；进入 Change 043 |

## 与计划的偏差

- 原先考虑发布 `v1.1.0` 稳定版；依据项目“失败、BLOCKED 或必要 NEEDS_USER 不能冒充 PASS”规则，改为顺序 RC `v1.1.0-rc.4`。
- 不创建或补写缺失的 rc.3 本地发布正文，避免伪造历史。
- rc.4 Tag push 没有生成 run；采用 workflow 明示支持的 `workflow_dispatch`，ref 仍为 rc.4 Tag，未改变提交或 Release 逻辑。
- rc.4 CI 失败后不删除或移动 Tag，不手工创建 Release；使用 Change 043 和 rc.5 前进修复。

## 关键决策纪要

- Tag 必须指向包含 `_dev/releases/v1.1.0-rc.4.md` 的发布准备提交。
- 自动 workflow 负责创建 Release；不重复手工创建。
- 发布成功后的 run、URL 和附件事实通过第二个治理提交补录，不移动已公开 Tag。
- rc.4 保持 prerelease、非 Latest。

## 遗留未完成项

- Change 043 修复跨平台测试配置隔离并完成本地/远程验证。
- 使用 rc.5 前进发布；rc.4 Tag 和失败 run 保持不可变。
- rc.4 没有 Release 或附件，不得宣称发布成功。
