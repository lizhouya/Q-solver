# Change 033：总结与注意事项

## 当前状态

业务实现、自动验证、候选构建和用户 GUI 人工验收完成。原 Change 033 候选已归档到 `build/bin/archive/transient/`，用户已在最新组合候选中确认 Change 033 行为验证通过；根目录稳定版未晋升。

## 结果摘要

- 快捷键区默认折叠为 42px 紧凑摘要，可展开查看完整提示。
- 最新流式答案和最终答案渲染后自动滚动到文档底部。
- 查看历史答案期间不自动拉回最新结果。
- 远程键盘控制由 70% 视口高度降为 40%。
- 不修改 Go 生产代码、SSE/Snapshot 协议、快捷键绑定、配置、依赖、用户数据或模型链路。

## 人工验证入口

原 Change 033 候选已归档到 `build/bin/archive/transient/`，原始 SHA-256：`19495E67DE3E84781E2E8598EE016EAB4AB4FD5599786817DD3F9EBE757016E9`。

当前最新组合验证入口：`build/bin/testing/current/Q-Solver-candidate-change034.exe`，同时包含 Change 033 与 Change 034。

启动前需正常退出当前正在运行的稳定版 Q-Solver，避免固定 `SingleInstanceLock` 将候选转交给旧进程后立即退出。不得通过强制终止破坏用户当前任务。

## 授权边界

用户已明确授权本轮 Change 033/034 提交并推送到 `origin/dev/qsolver-custom`；不含 Tag、Release、RC 或稳定版晋升授权。二进制继续由 Git 忽略，根目录稳定版保持不变。
