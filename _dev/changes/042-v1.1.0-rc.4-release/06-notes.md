# 总结与注意事项 06

## 当前摘要

Change 042 负责将 rc.3 之后 Change 024～041 的成果作为 `v1.1.0-rc.4` 预发布上线。当前已完成版本边界、自动发布合同与文件白名单审计，正在准备发布文档；尚未创建 Tag 或 Release。

## 长期约束

- race `BLOCKED` 与 macOS GUI `NEEDS_USER` 存在时，不标记稳定版或 Latest。
- Tag push 会触发自动 Release；发布正文必须在 Tag 对应提交中提前存在。
- 自动 Release 成功后不再手工执行 `gh release create`。
- 已公开 Tag 保持不可变；后续修复使用新 RC，不移动 Tag。
- Release 不代表所有历史 Change 已关闭。
- 发布说明不得包含 API Key、Token、Mobile URL、题库数据、个人配置或本机路径。

## 当前发布边界

- 目标分支：`origin/dev/qsolver-custom`
- 目标 Tag：`v1.1.0-rc.4`
- Release：prerelease、非 draft、非 Latest
- 平台：Windows amd64、macOS arm64、macOS amd64
- 产物：三个平台附件与 `SHA256SUMS`

## 遗留项

- 发布准备文档检查、提交与分支推送。
- rc.4 Tag 创建、显式推送及 Actions 监控。
- Release 属性、附件和校验和核对。
- 最终运行事实补录、治理提交与分支推送。

## 参考资源

- `_dev/audits/042-v1.1.0-rc.4-release.md`
- `_dev/releases/v1.1.0-rc.4.md`
- `.github/workflows/release.yml`
- `_dev/changes/040-mobile-shortcut-hint-sync/04-validation.md`
- `_dev/changes/041-choice-option-consistency/04-validation.md`
