# Change 010/011 提交边界审计

日期：2026-10-18。当前未授权暂存、commit 或 push；以下仅是后续提交设计，不是提交操作。

## 本次范围

- Change 010：pkg/questionbank/ 及测试，app/questionbank*.go，app/app.go、app/solve.go、main.go，题库 Vue/Pinia/API 接入，新增 Go bindings。
- Change 010 治理：中英文 README、010 七份档案、题库与场景审计及 INDEX 的 010 增量。
- Change 011：完整仓库计划迁入“求职笔试/Q-Solver”，“求职笔试”作为多工具分类父目录，空“自动网申”改名“求职网申”，创建独立输出分类目录；仓库内部相对路径不变，不生成全量 Git 源码改名。
- Change 011 治理：011 七份档案、迁移清单、CLAUDE.md 本地仓库路径增量、INDEX 的 011 增量。

## 需要隔离的已有改动

任务开始前已修改 AGENT.md、CLAUDE.md、_dev/README.md、INDEX.md，并存在未跟踪 AGENTS.md。没有清理或覆盖这些内容。

CLAUDE.md 与 INDEX.md 存在用户原改动和本次增量，后续不能整文件直接夹带进提交；应在用户确认归属后重新审计和划分提交。不得以本审计为暂存授权。

## 生成噪音与依赖

- 保留新增 app bindings、models.ts；只修剪生成器新增行的尾随空白。
- 经 diff 核实后移除 Wails CLI 2.16.0 自动添加的 runtime 通知接口，保留项目 2.11.0 配套文件。
- go.mod、go.sum、前端 package 与 lock、wails.json 的构建前后 SHA-256 一致；没有新增或升级依赖。
- frontend/dist、build/bin、node_modules 不列入提交。
- 测试仅使用合成小图片及 fake provider；不含真实截图、API Key、Token、简历和个人偏好文件。实际运行输出位于仓库外。

## 拟议提交（需要另行确认文件增量归属）

### Change 010 功能与测试

目的：引入场次保存、题库导出和可选结构化整理。

```text
feat: add exam question bank archiving

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

### Change 010 治理

目的：保存设计、测试、使用文档和场景分析证据。

```text
chore: add change 010 governance records

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

### Change 011 治理

目的：记录工作区物理迁移与验证；不制造空的功能提交。

```text
chore: add change 011 governance records

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

## 验收边界

普通 Go 测试、前端构建、Wails 构建通过；race 在无测试执行时仍以 0xc0000139 退出，记录为阻塞。Windows GUI、手机真机、真实整理未验收，Change 不正式 Close。迁移后如有新的代码修改，需重新验证和审计。

## 实际目录状态

Change 011 尚未迁移：普通文件曾移动但已完整回退；`.git` 返回 WinError 5，单次获准的沙箱外执行也失败。原仓库仍在总目录根部，CLAUDE.md 的路径未改。只有分类目录已建立，旧空网申目录未移除；因此不应提交任何声称迁移完成的治理记录。

目录层级修订：已创建 求职笔试/Q-Solver 并同步当前方案；没有重试源码迁移，旧阻塞与回退证据保留。此次只更新文档与空目录，没有新的业务代码改动。

Change 011 新增外部执行辅助脚本 migrate-workspace.ps1：只读模式和 PowerShell 5.1 语法检查通过；-Apply 执行尚未验收，列入 011 操作资产，不混入 010 功能提交。
