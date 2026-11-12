# 实施方案 — 02

## 1. 方案概述

复用既有 Snapshot `hints`，只修正同步端底部截图状态卡的筛选顺序和满额渲染。先用 DOM 回归构造五项非默认键名并获得红灯，再把本地工作流扩展为截图、发送、快速解题、中断、删除；截图满额时仍走统一 hint DOM，只动态切换文案。渲染完成后按固定卡片实际高度更新页面底部预留，隐藏时清除。后端协议、配置 schema 和桌面端均不改。

## 2. 分步实施计划

| 步骤 | 操作 | 涉及文件 | 预期产出 | 完成标志 |
|---|---|---|---|---|
| 1 | 增加五项顺序、非默认键名、满额键帽和动态底部空间红灯 | `pkg/mobile/web_dom_test.mjs` | 旧三项实现按预期失败 | 失败原因只指向缺失行为 |
| 2 | 补强后端提示前五项顺序与字段安全契约 | `app/mobile_test.go` | 防止页面与后端标签漂移 | 聚焦 Go 测试通过 |
| 3 | 扩展底部工作流并统一满额渲染 | `pkg/mobile/web/index.html` | 五项实际绑定按业务顺序显示 | DOM 聚焦回归通过 |
| 4 | 按卡片实际高度同步底部预留 | `pkg/mobile/web/index.html`、DOM 测试 | 窄屏换行不遮挡末尾内容 | 测试覆盖显示与隐藏 |
| 5 | 执行聚焦和全量门禁 | 不新增业务文件 | 真实测试记录 | 必要命令完成并记录 |
| 6 | 固定 Wails v2.11.0 构建保护候选并进行基础启动检查 | `build/bin/**`（ignored） | 唯一 Change 040 候选 | 保护哈希、运行和残留进程核对完成 |
| 7 | 更新 03–06、审计和 INDEX，执行提交边界审计 | `_dev/**` | 完整治理证据 | 保持待用户验收/待授权状态 |

## 3. 涉及业务源码的改动清单

允许修改：

- `pkg/mobile/web/index.html`：五项筛选顺序、满额文案、动态底部预留。
- `pkg/mobile/web_dom_test.mjs`：五项顺序、键名、满额、重置和高度测试。
- `app/mobile_test.go`：前五项提示顺序和字段安全契约。

禁止越界修改：

- `app/mobile.go`、`pkg/mobile/bridge.go`、Mobile 协议与 Token 逻辑。
- `frontend/src/**`、快捷键配置/迁移、Wails bindings、依赖文件。
- 正式配置、`build/bin/releases/**`。

## 4. 测试与验证计划

1. 红灯：`node pkg/mobile/web_dom_test.mjs`。
2. 聚焦：`node pkg/mobile/web_dom_test.mjs`。
3. 聚焦 Go：`go test ./app ./pkg/mobile -count=1`。
4. 全量：`go test ./... -count=1`。
5. 静态检查：`go vet ./...`。
6. 构建：`go build ./...`。
7. 依赖：`go mod verify`。
8. 前端：现有 `frontend/*_test.mjs` 回归与 `npm --prefix frontend run build`。
9. 差异：`git diff --check`，并核对没有配置、secret、依赖或生成噪音。
10. Wails：固定 v2.11.0、保护参数构建 Change 040 候选；验证独立启动、WebView2、无 Node/Vite 子进程、正常退出和无残留。
11. 用户 GUI：同步端有截图时显示五项实际绑定，顺序与桌面一致；满三张保留截图键帽。

## 5. 回滚预案

定向恢复 `pkg/mobile/web/index.html` 的工作流和底部空间逻辑，并同步恢复对应测试。若已有提交，仅在用户明确授权后使用 `git revert`；不使用 `git reset --hard`，不改正式配置或 Release。

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 五项在窄屏换行后遮挡内容 | 页面末尾不可见 | 使用实际卡片高度加安全间距，不依赖固定 112px |
| 标签名称与后端漂移 | 某项不显示 | Go 测试锁定后端前五项标签和顺序，DOM 测试使用同一标签 |
| 满额分支再次丢失键帽 | 与桌面不一致 | 不提前返回，始终创建统一 `capture-hint` |
| 配置更新后仍显示旧键名 | 提示不一致 | 继续使用每帧 Snapshot `hints`，不缓存键名或硬编码 F 键 |
