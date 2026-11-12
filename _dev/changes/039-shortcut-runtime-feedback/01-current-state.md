# 现状快照 — 01

## 调用链

```text
Windows 低级键盘 Hook
  → heldKeys
  → GetComboID
  → Manager.Shortcuts[action].ComboID
  → Service.enqueueTrigger
  → FIFO dispatcher
  → handleTrigger(send_fast/delete)
  → App.TriggerQuickSend / App.TriggerDeleteScreenshot
```

录制与保存链：

```text
OnRecord(action, keyName, comboID)
  → key-recorded
  → App.vue 更新 tempShortcuts
  → settings.saveSettings
  → JSON payload
  → ConfigManager.UpdateFromJSON
  → subscriber 覆盖 Manager.Shortcuts
```

## 客观证据

- `KeyBinding.ComboID` 的 JSON tag 是 `vkCode`。
- `App.vue` 当前向草稿写 `comboID: data.comboID`，但旧对象仍保留 `vkCode`。
- Go `encoding/json` 忽略未知 `comboID`，因此保存后仍读取旧 `vkCode`。
- 设置页只显示 `keyName`，无法揭示 `keyName` 与 `vkCode` 已错配。
- `handleTrigger` 已具备 `send_fast` 和 `delete` 分支，业务入口并未缺失。
- `getKeyName` 与 `GetComboID` 支持 F10（121）和 F12（123）。
- 现有测试覆盖直接调用 `handleTrigger`，未覆盖 Windows 单功能键匹配与前端保存字段。
- `go test ./pkg/shortcut -count=1` 在首轮修复前通过，证明既有测试无法捕获本问题。
- 首版候选修复了后续录制字段，但没有迁移磁盘中的旧错配。用户 GUI 验收报告快捷键混乱。
- 只读正式配置一致性检查只输出 action 名称，确认 16 项中 5 项错配：`cancel_solve`、`clickthrough`、`delete`、`send_fast`、`toggle_window`；临时探针已删除，正式配置未修改。
- 当前设置页继续只按 `keyName` 显示，Hook 按 `ComboID/vkCode` 执行，因此首版候选仍会展示与运行态不一致的历史值。

## 风险

1. 仅改显示文案会继续保存旧组合，不能修复。
2. 仅在后端兼容 `comboID` 会扩大 schema，并保留前端双字段歧义。
3. 直接逐项迁移会因旧占用顺序失败，例如 `send_fast` 目标 F10 仍被待迁移的 `clickthrough` 占用，必须先规划后统一应用。
4. `keyName` 对 Ctrl/Shift/Alt 左右键有信息损失；自动迁移必须限定为无歧义的单个 `F1–F24`，其他错配不能猜测。
5. 自动加载迁移若直接写磁盘会覆盖用户配置；本轮只修复内存快照，正常保存设置后再持久化。
6. 快捷键命中反馈若写成业务成功，会在模型配置失败、任务运行中或无截图时误导用户。
7. 当前工作区另有 Change 038 晋升后的治理文档修改，必须保留并与 Change 039 业务边界区分。
