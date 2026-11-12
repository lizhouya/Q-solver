# 验证记录 04

## 当前状态

发布文档编写中。业务实现验证沿用 Change 040、041 的当日证据；本 Change 只修改 Markdown 治理与发布资料，不在 Tag 前重复本地业务构建。自动 Tag workflow 将重新执行三平台构建和回归。

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
| Build Windows | 待执行 | Tag push 后记录 |
| Build macOS (Apple Silicon) | 待执行 | Tag push 后记录 |
| Build macOS (Intel) | 待执行 | Tag push 后记录 |
| Create Release | 待执行 | Tag push 后记录 |
| Release 属性 | 待执行 | 非 draft、prerelease、非 Latest |
| Release 附件 | 待执行 | 三个平台附件 + `SHA256SUMS` |

## 验收标准对照

- [ ] rc.4 正文存在于 Tag 对应提交且无敏感信息。
- [ ] 中英文 README 与 CHANGELOG 指向 rc.4 并明确预发布。
- [ ] 分支和 Tag 仅推送至 `origin`。
- [ ] 自动三平台 Release 成功，附件与校验和完整。
- [ ] 最终治理事实已提交并推送。

## 结论

状态：**发布准备进行中**。在自动 workflow 和 Release 核对完成前，不报告发布成功或关闭 Change 042。
