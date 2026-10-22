# Audit 015 — v1.1.0-rc.1 发布边界

日期：2026-10-22。

## 授权与目标

用户已明确批准三组整合提交、推送 `origin/dev/qsolver-custom`、创建并推送 annotated Tag `v1.1.0-rc.1`，并允许在如实披露 race 阻塞与人工验收缺口的前提下发布 RC。该授权不覆盖 `upstream`、`main`、force push、稳定版/Latest、删除或覆写其他 Tag/Release。

## 提交边界

1. 维护规则：`AGENT.md`、`AGENTS.md`、`CLAUDE.md`、`_dev/README.md`。
2. 功能与测试：`app/`、`pkg/`、`frontend/src/`、前端回归脚本、Wails App bindings、`main.go`；不含 runtime shim、用户配置、输出或 EXE。
3. 发布与治理：`.github/workflows/release.yml`、`.gitignore`、`wails.json`、中英文 README、`CHANGELOG.md`、`_dev/audits/`、`_dev/changes/010`～`015`、`_dev/changes/INDEX.md`、`_dev/releases/v1.1.0-rc.1.md`。

每组只用明确路径暂存；提交前检查 `git diff --cached`。禁止 `git add .`、`git add -A`、`git reset --hard` 与 `git clean`。

## 公开内容审计

- `build/bin/`、配置文件、题库与输出目录由 Git 忽略，不进入提交。
- Change 011 的 `evidence/` 与 `migration-manifest.json` 含大型本机迁移快照和绝对路径，已按精确路径忽略，仅本地保留。
- 已扫描常见 API Key、GitHub Token、Mobile token URL 和私钥标记；未发现待提交凭据。发布前仍需对最终暂存内容复扫。
- Release 说明只描述临时连接地址的安全属性，不包含真实 Mobile URL、token、API Key 或用户配置值。

## 工具链与构建

- 项目 Wails runtime：`v2.11.0`；Release CLI 固定 `v2.11.0`。
- 系统 Go `1.27.1` 的旧 Wails 构建失败已记录；隔离 Go `1.25.13` 构建通过，CI 固定 Go `1.25`。
- 本地 RC SHA-256：`08b1419dfe24db0b59e8e7bff9779e7e27eb67a9004f032209736cf7a1b734ea`。
- race 状态为 BLOCKED，不宣称 PASS；macOS 与未完成 GUI 矩阵由 RC 说明明确披露。

## 推送与 Tag 规则

先推送三组提交并核对远端分支 SHA，再创建 annotated Tag。Tag 只能首次推送到 `origin`；如 CI 失败，需要新提交和新的 RC Tag，不 force 移动已公开 Tag。Release 必须为 prerelease，不能标记 Latest，附件为三个平台产物和 `SHA256SUMS`。
