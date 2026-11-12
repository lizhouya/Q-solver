# 总结与注意事项 06

## 当前摘要

Change 043 已新增跨平台测试配置隔离 helper，统一控制 Windows/Linux/macOS 使用的目录环境变量，并迁移 app 中所有 APPDATA-only ConfigManager 测试。两个 rc.4 失败测试、app 全包和普通全量门禁均通过；生产代码零改动。功能/测试提交为 `532c828`。race 仍受 Windows `0xc0000139` 阻塞，rc.5 远程验证待执行。

## 长期约束

- ConfigManager 测试不能只设置 `APPDATA`；必须覆盖 Windows、Linux 和 macOS 的目录环境变量。
- 测试不得读写 runner 或开发者的真实 Q-Solver 配置。
- 本地 HTTP provider 测试只能访问 `httptest.Server`，不调用真实 API。
- 已公开 RC Tag 不因 CI 失败而移动或覆写；使用后续 RC 前进修复。

## 遗留项

- 创建治理/rc.5 发布准备提交并推送开发分支。
- 创建并推送不可变 rc.5 Tag。
- rc.5 三平台构建、回归、Create Release、附件与校验和核对。
- macOS GUI 与 Windows race 继续如实披露。

## 参考资源

- `_dev/audits/043-cross-platform-test-config-isolation.md`
- GitHub Actions run `43212159818`
- `app/solve_preflight_test.go`
- `app/solve_persistence_test.go`
- `pkg/config/manager.go`
