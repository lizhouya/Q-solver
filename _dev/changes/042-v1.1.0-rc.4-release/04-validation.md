# 验证记录 04

## 当前状态

发布准备、分支与 Tag 推送已完成。run `43212159818` 的 Windows job 通过，但 macOS Apple Silicon 回归失败，Intel job 取消，Create Release 跳过；rc.4 发布失败且没有 Release。失败由 Change 043 接续处理。

## 发布前验证

| 检查项 | 结果 | 说明 |
|---|---|---|
| Change 040 普通自动门禁与用户 GUI 验收 | PASS | 同步端五项快捷键提示与动态底部间距 |
| Change 041 普通自动门禁与用户模型复验 | PASS | 选择题字母、文本、解析与汇总一致性 |
| Windows race | BLOCKED | 启动阶段 `exit status 0xc0000139`，未进入测试体 |
| macOS GUI | NEEDS_USER | 仅由 CI 构建，无人工桌面验收 |
| Markdown 链接与 release body 文件 | PASS | rc.4 正文及引用的相对目标均存在；README 不再残留 rc.2 当前入口 |
| 占位符与 VS Code diagnostics | PASS | Change 042 无模板占位符；Warning 及以上为空 |
| `git diff --check` | PASS | 无空白错误 |
| 本地/远程 rc.4 Tag | PASS | Tag 均未占用，可创建 |
| 提交白名单与 secret 检查 | PASS | 仅 13 个批准的发布/治理 Markdown 文件；常见凭据与本机路径模式未命中 |

## GitHub Actions 验证

| Job / 检查 | 结果 | 证据 |
|---|---|---|
| Build Windows | PASS | run `43212159818`，Build App、Regression Tests、Upload Artifact 通过 |
| Build macOS (Apple Silicon) | FAIL | Build App 通过；Regression Tests 的两个 app 测试失败 |
| Build macOS (Intel) | CANCELLED | 无通过证据 |
| Create Release | SKIPPED | 依赖 job 未全部成功 |
| Release 属性 | NOT CREATED | `v1.1.0-rc.4` Release 不存在 |
| Release 附件 | NOT CREATED | 无公开附件或 `SHA256SUMS` |

## 验收标准对照

- [x] rc.4 正文存在于 Tag 对应提交且无敏感信息。
- [x] 中英文 README 与 CHANGELOG 指向 rc.4 并明确预发布。
- [x] 分支和 Tag 仅推送至 `origin`。
- [ ] 自动三平台 Release 成功：**未满足**，macOS arm64 FAIL、Intel CANCELLED、Create Release SKIPPED。
- [ ] 最终成功发布事实：**不存在**，由 Change 043/rc.5 前进处理。

## 结论

状态：**rc.4 Tag 已推送 / CI FAIL / Release 未创建 / 不关闭**。失败证据真实保留，后续由 Change 043 修复并发布 rc.5；不得把 Windows 单平台通过冒充完整发布成功。
