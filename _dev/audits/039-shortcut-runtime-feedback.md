# Change 039 快捷键运行态与反馈审计

## 结论

故障根因是前端录制事件字段与 Go JSON schema 不一致：事件名为 `comboID`，持久化字段为 `vkCode`。界面更新 `keyName` 和临时 `comboID` 后保留旧 `vkCode`，保存时 Go 忽略未知字段并把旧组合重新发布到运行态。

## 修复边界

- 在前端配置边界规范化为 `vkCode`，不修改 Go schema。
- Windows 回归直接覆盖 F10/F12 的 ComboID 匹配。
- `send_fast` / `delete` 增加“快捷键已触发”反馈，不承诺业务成功。
- 不读取或修改正式用户配置；用户需要在修复版中重新录制受影响组合。

## 实施后结果

- `App.vue` 已把事件 `comboID` 规范化到配置字段 `vkCode`，并删除临时同名字段。
- Windows F10（121）/F12（123）匹配回归通过。
- `send_fast` / `delete` 均先发出准确的 action 命中反馈，再调用原 delegate。
- 首版候选用户 GUI 验收 FAIL 后，只读审计确认 5 项历史 `keyName/vkCode` 错配。
- revision B 增加单 `F1–F24` 两阶段内存迁移；正式配置只读加载后错配由 5 降至 0，磁盘字节不变。
- revision C 按用户补充要求将截图 Dock 固定为 F8 截图、F9 发送、F10 快速解题、F11 中断、F12 删除，满三张仍保留 F8 键帽。
- 全量普通自动门禁和固定 Wails v2.11.0 revision C 候选构建通过；race 为已知环境 `BLOCKED`。
- 首版候选归档 failed、revision B 归档 transient；revision C 用户 GUI 复验通过并已晋升本地稳定版。功能提交 `fbae933` 已创建，本治理提交与推送到 `origin/dev/qsolver-custom` 已获用户明确授权；未发布。

## 提交边界

功能提交仅包含前端映射、快捷键服务和对应测试，已创建为 `fbae933 fix: repair shortcut bindings and ordered hints`；治理提交仅包含本审计、Change 039 档案与 INDEX。用户已明确授权将本轮提交推送到 `origin/dev/qsolver-custom`；Tag 与 Release 未授权。
