# 回滚预案 05

## 回滚策略

- [ ] **A. 功能级定向回滚**：移除重试状态和 API，恢复当前发送/错误行为。
- [ ] **B. 题库级定向回滚**：移除 terminal Record restart，不删除或重写已有题库记录。
- [ ] **C. 快捷键兼容回滚**：停止注册新增 action；保留配置中的未知键值供旧版本自然忽略。
- [ ] **D. UI/Mobile 定向回滚**：移除控制按钮和 cancelled 展示，恢复既有 error Snapshot 兼容读取。
- [ ] **E. Git/产物回滚**：经授权使用 `git revert`；稳定版只从哈希通过的 superseded 归档恢复。

## 回滚触发条件

- 重试调用截图服务、错误图片、任意旧轮或成功轮。
- 重试创建重复题库 Record、改变图片引用或污染题号上下文。
- 取消后仍持续调用 Provider、提交 KeepContext，或旧任务覆盖新任务 UI。
- 快速模式修改持久 Config、活动 Connection Profile、默认场景或 subscriber 状态。
- 新快捷键覆盖用户绑定、阻塞 Hook/FIFO 或导致现有快捷键失效。
- Mobile 新增控制面、泄露图片/Record ID/配置，或破坏清屏代际与 Reset。
- 必要自动/人工验收 FAIL 且无法在批准范围内收敛。

## 回滚边界

- 不使用 `git reset --hard`，不丢弃用户已有改动。
- 不主动删除用户配置中的新 action；旧版本应忽略未知 Map key。
- 题库 schema 若仅新增兼容字段，回滚时保留字段，不批量改写用户文件。
- 不回滚 Change 035 的快捷键 FIFO、Change 036 的 Mobile 状态、Change 037 的 Connection Profiles。

## 实际回滚记录

截至 2026-11-11 未触发功能回滚。普通自动门禁与用户测试版本验收通过；race 为环境 `BLOCKED` 并已披露。候选保留在 `build/bin/testing/current/`，根稳定版仍为测试前哈希 `8B19…4994`，因此无需执行二进制回滚或修改 Release。
