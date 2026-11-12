# 变更请求 — 00

| 字段 | 值 |
|---|---|
| 变更编号 | `042-v1.1.0-rc.4-release` |
| 提出人 | 项目维护者 |
| 提出日期 | 2026-11-12 |
| 关联 Issue | 无 |
| 状态 | `进行中` |

## 1. 背景与问题

Change 040、041 已完成普通自动验证、用户 GUI/模型复验、独立双提交和本地稳定版晋升，但四个本地提交尚未推送。个人 fork 当前最新公开版本是 `v1.1.0-rc.3`；中英文 README 仍指向 rc.2，CHANGELOG 也只记录到 rc.2。用户已明确授权本次分支推送、Tag 推送和自动 GitHub Release 上线。

当前仍有两个必须如实披露的发布缺口：Windows race 进程启动持续返回 `0xc0000139`，macOS CI 附件没有人工 GUI 验收。按项目关闭与发布规则，这些缺口不能包装为稳定版，因此本次顺序发布 `v1.1.0-rc.4` 预发布，不标记稳定版或 Latest。

## 2. 目标

- 为 `v1.1.0-rc.4` 创建完整发布说明并更新中英文入口文档与 CHANGELOG。
- 将 Change 040、041 及本次发布准备提交推送至 `origin/dev/qsolver-custom`。
- 创建并显式推送单一 annotated Tag `v1.1.0-rc.4`，触发既有三平台自动发布工作流。
- 核对 Actions、Tag、Release 属性、四个附件和校验和，并把最终事实补录到 Change 042。
- 保持个人 fork、CC BY-NC 4.0、secret、配置和上游边界清晰。

## 3. 验收标准

- [ ] `_dev/releases/v1.1.0-rc.4.md` 在 Tag 对应提交中存在，内容不泄露个人配置、Token、URL 或本机路径。
- [ ] README、README_EN 与 CHANGELOG 均指向 rc.4，并明确其为个人 fork 预发布而非稳定版。
- [ ] 发布准备文档通过链接、空白和提交边界检查。
- [ ] `origin/dev/qsolver-custom` 包含 Change 040、041 与发布准备提交。
- [ ] `v1.1.0-rc.4` Tag 指向已核对的发布准备提交，且只推送到 `origin`。
- [ ] 自动工作流的 Windows amd64、macOS arm64、macOS amd64 和 Create Release 均成功。
- [ ] GitHub Release 为非 draft、`prerelease=true`、非 Latest，并包含三个平台附件与 `SHA256SUMS`。
- [ ] 发布后治理记录补录真实 run、Release URL、附件与最终 Git 状态；失败项不标记 PASS。

## 4. 非目标 / 明确不做

- 不发布 `v1.1.0` 稳定版，不把 rc.4 标为 Latest。
- 不修改 `main`、`upstream`、默认分支、历史 Tag 或历史 Release。
- 不删除或强制移动任何 Tag，不执行 force push。
- 不修改业务源码、依赖、Wails 配置或 GitHub Actions 工作流。
- 不提交 `build/bin/**`、用户配置、API Key、Token、Mobile 连接 URL 或个人路径。
- 不补造缺失的 rc.3 本地发布说明。

## 5. 影响面预估

- 业务源码：无。
- 文档与治理：`README.md`、`README_EN.md`、`CHANGELOG.md`、`_dev/releases/v1.1.0-rc.4.md`、Change 042 七份档案、发布审计、INDEX。
- 外部影响：推送 Tag 会触发 GitHub Actions 并创建公开预发布 Release。
- 潜在风险：CI 平台差异、自动 Release 失败、附件或校验和缺失、发布文档与实际状态不一致。

## 6. 附带约束

- Git 操作前执行规定预检；暂存仅使用显式文件路径。
- 发布准备与发布结果分别提交，不 amend 既有提交。
- 自动工作流创建 Release 后不再手工重复执行 `gh release create`。
- 任何失败均保留真实证据并停止宣称发布完成；未经额外授权不删除失败 Tag 或 Release。
