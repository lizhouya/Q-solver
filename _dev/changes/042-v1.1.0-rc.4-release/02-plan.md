# 实施方案 — 02

## 1. 方案概述

采用“发布准备提交 → 分支推送 → annotated Tag 单独推送 → 自动三平台 Release → 发布结果治理提交”的顺序。版本使用 `v1.1.0-rc.4`，以真实保留 race 与 macOS 人工验收缺口；不修改业务代码或工作流，不把预发布标为稳定版或 Latest。

## 2. 分步实施计划

| 步骤 | 操作 | 涉及文件 / 系统 | 预期产出 | 完成标志 |
|---|---|---|---|---|
| 1 | 建立事前档案和发布审计 | Change 042、audit、INDEX | 范围、风险、验收与回滚边界明确 | 00/01/02 完整 |
| 2 | 编写 rc.4 发布文档 | README、README_EN、CHANGELOG、release notes | 对外版本、功能、下载与限制一致 | 文档链接可解析 |
| 3 | 验证并创建发布准备提交 | Git 工作区 | `chore: prepare v1.1.0-rc.4 release` | 显式文件暂存、提交完成 |
| 4 | 推送开发分支 | `origin/dev/qsolver-custom` | Change 040/041 和发布准备提交上线 | 远程 HEAD 与本地一致 |
| 5 | 创建并推送 Tag | `origin` | annotated Tag `v1.1.0-rc.4` | 远程 Tag 指向发布准备提交 |
| 6 | 监控并核对自动发布 | GitHub Actions / Release | 三平台构建、测试、附件与校验和 | 工作流成功、Release 属性正确 |
| 7 | 补录发布结果并提交 | Change 042、audit、INDEX | 可追溯 run、URL、附件与最终状态 | 结果提交推送、工作区 clean |

## 3. 涉及业务源码的改动清单

无。允许写入范围仅为：

- `README.md`
- `README_EN.md`
- `CHANGELOG.md`
- `_dev/releases/v1.1.0-rc.4.md`
- `_dev/audits/042-v1.1.0-rc.4-release.md`
- `_dev/changes/042-v1.1.0-rc.4-release/00-request.md` 至 `06-notes.md`
- `_dev/changes/INDEX.md`

不得修改 `.github/workflows/release.yml`、业务源码、依赖、配置、bindings 或 `build/bin/**`。

## 4. 测试与验证计划

### 发布准备

- 检查所有新增和修改 Markdown 链接目标存在。
- 检查 rc.4 正文文件名与 workflow `body_path` 完全一致。
- 执行 `git diff --check`。
- 执行 Git 规定预检，并检查暂存区差异与显式文件白名单。
- 因本 Change 只改文档，不在本地重复全部业务测试；Tag 工作流会重新运行三平台构建和回归。

### 远程发布

- 核对 Tag 只推送至 `origin` 且指向发布准备提交。
- 监控四个工作流 job，失败时读取实际日志，不以手工 Release 掩盖失败。
- 核对 Release：非 draft、prerelease、非 Latest、Tag/target 正确。
- 核对附件：Windows amd64 EXE、两个 macOS ZIP 和 `SHA256SUMS`。
- 下载/读取远程附件元数据及校验和，确认 `SHA256SUMS` 覆盖三个二进制附件。

## 5. 回滚预案

- Tag 推送前：使用新提交执行 `git revert` 修正文档，不改写历史。
- Tag 推送后但工作流失败：保留失败事实，停止宣称发布完成；未经额外授权不删除 Tag/Release，采用后续 RC 修复前进。
- Release 成功后发现文档小错误：以新治理提交修正文档；Tag 内容保持不可变，必要时发布后续 RC。
- 不使用 `git reset --hard`、force push 或移动已公开 Tag。

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 把已知缺口包装为稳定版 | 误导用户 | 使用 rc.4、prerelease，明确 race/macOS 限制 |
| Release 正文文件缺失 | Create Release 失败 | Tag 前验证 `_dev/releases/v1.1.0-rc.4.md` 已提交 |
| 跨平台 CI 失败 | 无法创建完整 Release | 监控真实日志；不重复手工创建 Release |
| 附件不完整或校验和不匹配 | 下载不可验证 | 核对四个附件及 `SHA256SUMS` 内容 |
| 推送错误远程或分支 | 污染 upstream/main | 每次 Git 操作前预检，命令显式使用 `origin dev/qsolver-custom` 或单一 Tag |
| 发布后治理记录不在 Tag 内 | Tag 不含最终运行事实 | 使用第二个治理提交补录并推送分支，不移动 Tag |
