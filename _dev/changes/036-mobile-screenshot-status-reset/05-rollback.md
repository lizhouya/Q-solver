# 回滚预案 05

## 回滚策略

- [ ] 定向还原 Change 036 的 Mobile Snapshot、Bridge、App 接线、Web UI 和测试。
- [ ] 若已形成授权提交，使用 `git revert`；不使用 `git reset --hard`。

## 回滚触发条件

- 同步端泄露图片内容、路径、Token 或键码。
- 截图数量与桌面缓冲持续不一致。
- 本地清理失败时误清同步端。
- idle 重置导致连接、快捷键或固定链接失效。
- 全量测试或 Wails 构建出现无法在范围内修复的回归。

## 回滚步骤

1. 停止 Change 036 验证程序。
2. 恢复 Snapshot 和 Bridge 变更。
3. 恢复 App 截图/清理接线。
4. 恢复 Mobile Web 旧 idle 行为和 UI。
5. 重新运行 Mobile/App/全量测试并记录。

## 实际回滚记录

未发生。
