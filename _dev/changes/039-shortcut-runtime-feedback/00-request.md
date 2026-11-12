# 变更请求 — 00

| 字段 | 值 |
|---|---|
| 变更编号 | `039-shortcut-runtime-feedback` |
| 提出人 | 用户 |
| 提出日期 | 2026-11-11 |
| 关联 Change | Change 038 |
| 状态 | 已完成 / revision C 用户 GUI 验收通过 / 本地稳定版已晋升 / 功能提交 `fbae933` / 治理提交与推送已授权 / 未发布 |

## 背景

用户在 Change 038 本地稳定版中把“快速解题”录制为 F10、“删除截图”录制为 F12。设置页显示新键名，但保存后两个按键均无动作反馈。

## 已确认根因

后端 `shortcut.KeyBinding.ComboID` 的 JSON 字段名为 `vkCode`。前端收到 `key-recorded` 后写入了 `comboID`，没有更新既有 `vkCode`。保存 payload 因而同时包含旧 `vkCode` 与新 `comboID`；Go 忽略未知 `comboID`，配置 subscriber 再用旧 `vkCode` 覆盖运行态绑定。设置页只显示新 `keyName`，形成“显示 F10/F12、实际仍是旧组合”的错配。

首版 Change 039 候选只修复后续录制，没有处理旧版本已写入磁盘的损坏配对。用户重新验证后报告快捷键混乱；只读一致性探针确认 16 项中有 5 项不一致：`cancel_solve`、`clickthrough`、`delete`、`send_fast`、`toggle_window`。因此首版候选 GUI 验收为 FAIL，必须增加历史迁移后重建候选。

## 目标

1. 录制后的新组合统一保存到 `vkCode`，删除前端临时 `comboID` 字段。
2. 加载旧配置时检测 `keyName` / `vkCode` 不一致，只对可无歧义解析的单个 `F1–F24` 做两阶段内存迁移。
3. 迁移允许旧占用者同时迁走，例如先规划 `clickthrough: F10→F3` 后再释放 F10 给 `send_fast`；目标冲突或不可解析时不猜测。
4. 保存并由 ConfigManager 发布后，运行态保持迁移后的功能键，不回退旧组合。
5. Windows 单键 F10/F12 能分别命中 `send_fast` / `delete`。
6. 两个动作被快捷键分发时提供“已触发”反馈，不冒充下游业务成功。
7. 截图 Dock 始终按 F8 截图、F9 发送、F10 快速解题、F11 中断、F12 删除的实际解题顺序展示五项动态绑定；满三张时保留 F8 键帽并显示满额状态。
8. 不修改配置 schema、默认快捷键、依赖、正式连接配置或 Release 资产。

## 验收标准

- [x] 录制 F10/F12 后，待保存对象只使用 `vkCode=121/123`，不含 `comboID`。
- [x] 含上述 5 项错配的旧配置加载后，在内存中分别恢复为其 `keyName` 表示的 F 键；磁盘文件在未保存前不变。
- [x] 两阶段迁移能处理 F10 从 `clickthrough` 释放后交给 `send_fast`，且不会产生重复 ComboID。
- [x] 不可解析、非单 F 键或目标被非迁移 action 占用时保持原值并返回可诊断状态，不猜测。
- [x] JSON 保存往返后 Go `KeyBinding.ComboID` 分别为 `121/123`。
- [x] Windows Hook 匹配层使用 `heldKeys={121}` / `{123}` 时分别分发 `send_fast` / `delete`。
- [x] `send_fast` / `delete` 分发产生安全的“快捷键已触发”反馈，原业务调用各执行一次。
- [x] 快捷键冲突回滚、默认组合、FIFO、截图三图上限和 Change 038 解题控制不回归。
- [x] 截图 Dock 按 F8、F9、F10、F11、F12 顺序始终显示截图、发送、快速解题、中断、删除五项实际绑定；满三张时 F8 不消失。
- [x] 聚焦测试、全量 Go 测试、Go build、前端回归和 Vite build 通过。
- [x] 新候选经用户确认设置显示与实际快捷键一致。

用户于 2026-11-12 确认 revision C 验证通过，并明确授权本地版本更新、Git 提交与推送；本地稳定版已按哈希晋升。

## 非目标

- 不更换用户选定的 F10/F12。
- 不自动保存其他设置，不读取或输出正式配置和凭据。
- 不引入新的全局快捷键框架或依赖。
- 不把“已触发”文案写成“解题成功”或“删除成功”。
- 事前授权不包含 commit、push、稳定版晋升、Tag 或 Release；用户后续已分别明确授权并完成本地稳定版晋升、创建功能提交 `fbae933`，并授权治理提交与推送。Tag 与 Release 仍未授权。
