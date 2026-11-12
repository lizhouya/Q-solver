# 总结与注意事项 06

## 当前摘要

Change 042 已将 rc.3 后 Change 024～041 的内容准备为 `v1.1.0-rc.4`，完成发布准备提交 `36b4192`、分支推送和 annotated Tag 推送。Actions run `43212159818` 中 Windows 通过，macOS arm64 回归失败，Intel 取消，Create Release 跳过，因此 rc.4 没有 Release。失败根因由 Change 043 接续修复，后续使用 rc.5。

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

- Change 043 修复 APPDATA-only 测试导致的 macOS 配置污染。
- rc.5 完成三个构建 job 与 Create Release 后，再核对附件和校验和。
- rc.4 Tag 和失败 run 保持不可变；不得补造 rc.4 Release。

## 参考资源

- `_dev/audits/042-v1.1.0-rc.4-release.md`
- `_dev/releases/v1.1.0-rc.4.md`
- `.github/workflows/release.yml`
- `_dev/changes/040-mobile-shortcut-hint-sync/04-validation.md`
- `_dev/changes/041-choice-option-consistency/04-validation.md`
