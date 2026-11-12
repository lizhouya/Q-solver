# 总结与注意事项 06

## 当前摘要

Change 043 已新增跨平台测试配置隔离 helper，统一控制 Windows/Linux/macOS 使用的目录环境变量，并迁移 app 中所有 APPDATA-only ConfigManager 测试。两个 rc.4 失败测试、app 全包和普通全量门禁均通过；生产代码零改动。功能/测试提交为 `532c828`，治理/rc.5 准备提交为 `8734b3e`。rc.5 两次 run 的 Windows 均通过，但 macOS job 均未分配 runner 后取消，Release 未创建。

## 长期约束

- ConfigManager 测试不能只设置 `APPDATA`；必须覆盖 Windows、Linux 和 macOS 的目录环境变量。
- 测试不得读写 runner 或开发者的真实 Q-Solver 配置。
- 本地 HTTP provider 测试只能访问 `httptest.Server`，不调用真实 API。
- 已公开 RC Tag 不因 CI 失败而移动或覆写；使用后续 RC 前进修复。

## 遗留项

- 用户决定继续等待三平台 runner，或明确批准 Windows-only RC 降级范围。
- rc.5 Tag、两次 run 和“无 Release”事实保持不可变。
- macOS GUI 与 Windows race 继续如实披露。

## 参考资源

- `_dev/audits/043-cross-platform-test-config-isolation.md`
- GitHub Actions run `43212159818`
- `app/solve_preflight_test.go`
- `app/solve_persistence_test.go`
- `pkg/config/manager.go`
