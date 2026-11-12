# 回滚预案 05

## 触发条件

- 新录制组合仍在保存后回退。
- 快捷键反馈造成重复分发或误报业务成功。
- Windows Hook、FIFO、冲突检测或其他 action 回归。

## 回滚步骤

1. 定向恢复 `App.vue` 的录制映射改动。
2. 移除 `service.go` 的动作命中反馈。
3. 保留测试失败证据，不修改正式用户配置。
4. 如已有提交，仅在用户授权后使用 `git revert`，不使用 `git reset --hard`。

## 当前记录

首版候选 `E4C5…D5C2` 因用户 GUI 验收发现历史快捷键仍混乱，已停止使用并归档到 `build/bin/archive/failed/change-039/`；revision B 因后续新增 Dock 验收项归档 transient。revision C 用户验收通过并已晋升根稳定版；被替代的 Change 038 稳定版归档于 `archive/superseded/change-039/`，如后续需要回滚，必须再次获得授权并按该目录清单校验后执行。Release 未修改。
