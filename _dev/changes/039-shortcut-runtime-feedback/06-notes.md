# 总结与注意事项 06

## 当前摘要

Change 039 已修复三层问题：新录制组合保存到 `vkCode`，历史错误版本留下的单 F 键 `keyName/vkCode` 错配在加载时安全修复到内存，截图 Dock 固定按 F8→F12 展示五项解题快捷键。正式配置只读验证由 5 项错配降至 0，普通自动门禁与用户 revision C GUI 复验通过，race 环境 BLOCKED 已披露；revision C 已晋升本地稳定版，功能提交 `fbae933` 已创建，治理提交与推送已授权，未发布。

## 长期约束

- 前端快捷键对象持久化字段必须使用后端 JSON schema 的 `vkCode`，事件内部变量可称 `comboID`，但不得直接进入配置 payload。
- 设置页显示的 `keyName` 不能作为运行态 ComboID 已应用的唯一证据；加载时必须校验两者一致性。
- 自动迁移只覆盖无歧义单键 `F1–F24`；含修饰键的名称丢失左右键细节，不得猜测。
- 迁移必须先规划再统一应用，支持旧 owner 迁走后的安全接管；目标重复或被非迁移 action 占用时保持原值。
- `Load` 只修复内存，不自动覆盖加密配置；正常保存后再持久化。
- 快捷键反馈只能确认 action 已分发；业务是否成功继续由对应业务事件决定。
- 截图 Dock 必须使用实际动态绑定并固定按 F8 截图、F9 发送、F10 快速解题、F11 中断、F12 删除排列；满三张不能隐藏 F8。
- revision C 根稳定版：`build/bin/Q-Solver.exe`，SHA-256 `DC8B0E248B82972F92A6EC45152F0D9B4F93450DC335A61185476129D0D357EC`。
- revision C 验证归档位于 `archive/validation/change-039/`；被替代稳定版位于 `archive/superseded/change-039/`；`testing/current/` 已清空。
- revision B 已归档到 `archive/transient/Q-Solver-candidate-change039-revision-b-before-dock-hints.exe`。
- 首版失败候选已归档到 `archive/failed/change-039/`。
- 根稳定版已更新为 revision C，Release 未修改。
- 用户已明确授权本次本地版本更新、Git 提交和推送到 `origin/dev/qsolver-custom`；Tag、Release 仍未授权。
