# 实现过程记录 03

## 当前状态

前端字段修复、快捷键命中反馈、历史损坏配置内存迁移、F8–F12 Dock 顺序提示、普通自动门禁和 revision C 用户 GUI 验收均已完成。用户授权本地版本更新、Git 提交与推送；稳定版已晋升，功能提交 `fbae933` 已创建，治理提交与推送已授权，未发布。

## 实现日志

| 日期 | 步骤 | 结果 |
|---|---|---|
| 2026-11-11 | 根因定位 | 确认前端 `comboID` 与持久化 `vkCode` 字段错配 |
| 2026-11-11 | 前端红灯 | `solve_retry_cancel_fast_regression_test.mjs` 因缺少 `vkCode: data.comboID` 按预期失败 |
| 2026-11-11 | 后端红灯 | `send_fast` / `delete` 命中反馈断言按预期失败；Windows F10/F12 已能进入对应 action |
| 2026-11-11 | 最小修复 | `App.vue` 写入 `vkCode` 并删除临时 `comboID`；快捷键服务增加准确的“已触发”信息 Toast |
| 2026-11-11 | 旧契约回修 | `shortcut_hint_sync_regression_test.mjs` 从错误的 `comboID` 契约改为 `vkCode` 持久化契约 |
| 2026-11-11 | 首版候选 | 自动门禁和基础启动通过，候选 `E4C5…D5C2` 交付用户复验 |
| 2026-11-11 | 用户 GUI 验收 | FAIL：快捷键仍混乱，与设置显示不一致 |
| 2026-11-11 | 正式配置只读审计 | 16 项中 5 项历史错配：`cancel_solve`、`clickthrough`、`delete`、`send_fast`、`toggle_window` |
| 2026-11-12 | 迁移红灯 | 单 F 键解析和两阶段历史迁移接口缺失，聚焦测试按预期编译失败 |
| 2026-11-12 | 历史迁移 | 只解析无歧义 `F1–F24`；先规划全部目标，再稳定剔除重复/非迁移占用，统一更新内存快照 |
| 2026-11-12 | 正式配置只读复核 | 加载时修复 5 项，修复后 `mismatched=0`，磁盘字节未改变；临时探针已删除 |
| 2026-11-12 | 普通门禁 | 全量 Go、19 个前端回归、vet/build、Vite、mod verify 和 diagnostics 通过 |
| 2026-11-12 | race | `pkg/shortcut`、`pkg/config` 启动阶段均返回 Windows `0xc0000139`，BLOCKED |
| 2026-11-12 | 候选重建 | 首版失败候选归档到 `archive/failed/change-039/`；固定 Wails v2.11.0 重建 revision B 候选 |
| 2026-11-12 | Dock 提示补充 | 按用户要求固定显示 F8 截图、F9 发送、F10 快速解题、F11 中断、F12 删除；满三张保留 F8 键帽 |
| 2026-11-12 | revision C | revision B 归档 transient；全量门禁重跑通过，固定 Wails v2.11.0 重建唯一最新候选 |
| 2026-11-12 | 用户 GUI 复验 | PASS：快捷键实际行为、历史迁移和 F8–F12 Dock 顺序通过 |
| 2026-11-12 | 本地稳定版晋升 | revision C 经哈希校验替换根 EXE；旧稳定版与验证候选双归档，测试区清空，Release 未变 |

## 关键实现

- 新录制事件仍接收后端 `data.comboID`，但配置草稿规范化为 `{ keyName, vkCode }`，并删除临时 `comboID`。
- `FunctionKeyComboID` 仅接受严格的单键 `F1–F24`，不猜测组合键、左右修饰键或未知名称。
- `ReadableNameFromComboID` 从数值 ComboID 计算实际显示名，用于识别历史错配。
- 两阶段迁移支持 `clickthrough` 从旧 F10 迁到 F3 后，将释放的 F10 交给 `send_fast`；重复目标、非迁移 owner、不可解析名称均不改。
- 迁移只改变 `ConfigManager.Load` 的内存 `cfg`，不调用 `Save`；用户正常保存后才持久化。
- `send_fast` 和 `delete` 在调用 delegate 前发出 info Toast，只确认 action 已被快捷键服务分发。
- 截图 Dock 始终使用 settings store 的实际绑定，按 F8→F9→F10→F11→F12 顺序显示；三张满额时仅切换 F8 文案，不移除键帽。

## 产物

### 首版失败候选

- 归档：`build/bin/archive/failed/change-039/Q-Solver-candidate-change039-revision-a.exe`
- SHA-256：`E4C5FBDD3F805CEC2386DBD5B89B13F5C9A0FF7A7331DD8614A5327A8594D5C2`

### revision B 中间候选

- 归档：`build/bin/archive/transient/Q-Solver-candidate-change039-revision-b-before-dock-hints.exe`
- SHA-256：`C9F7680BAEE94AFA2D0851065EE279192C4276BB59D2A9746512F4F2A8B0AC8E`

### revision C 最终稳定版

- 根稳定版：`build/bin/Q-Solver.exe`
- 大小：42,556,928 字节
- SHA-256：`DC8B0E248B82972F92A6EC45152F0D9B4F93450DC335A61185476129D0D357EC`
- 被替代稳定版：`build/bin/archive/superseded/change-039/Q-Solver-stable-before-change039.exe`，SHA-256 `8A868263F29E170F1204E8BABC8064055D5533B870ED1A023309C99F1758D220`。
- 验证归档：`build/bin/archive/validation/change-039/Q-Solver-candidate-change039.exe`，哈希与根稳定版一致。
- Wails runtime：v2.11.0。
- `testing/current/` 已清空；依赖、bindings、dist 和 Release 保护哈希未改变。
