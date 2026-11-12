# 实施方案 — 02

## 方案

采用两层最小兼容修复：

1. 保持 Go 配置 schema 的 `vkCode` 不变，在前端录制事件处把后端 `comboID` 事件值规范化为待保存对象的 `vkCode`，并删除临时 `comboID`。
2. 在 Windows 配置加载时检测 `keyName` 与 `vkCode` 的实际可读名是否一致。只对单个 `F1–F24` 解析目标 ComboID；先收集全部迁移提案，再验证目标唯一性和非迁移占用，最后统一应用，从而支持 F10 等组合在多个 action 间安全释放和接管。
3. 迁移只改变 ConfigManager 的内存快照，不在 `Load` 中写盘；设置页读取、运行态 Hook 和用户后续保存均使用修复后的同一快照。
4. 后端快捷键服务只增加准确的“动作已触发”反馈；不改变 action 名、默认组合或业务返回语义。

## 文件范围

### 允许修改

- `pkg/shortcut/shortcut.go`
- `pkg/shortcut/shortcut_test.go` 或现有快捷键测试
- `pkg/config/manager.go`
- `pkg/config/manager_test.go`
- `frontend/src/App.vue`
- `frontend/src/components/ScreenshotDock.vue`
- `frontend/screenshot_dock_regression_test.mjs`
- `frontend/shortcut_hint_sync_regression_test.mjs`
- `frontend/solve_retry_cancel_fast_regression_test.mjs` 或新建同级聚焦回归文件
- `pkg/shortcut/service.go`
- `pkg/shortcut/service_test.go`
- `pkg/shortcut/shortcut_windows_test.go`（新建，Windows build tag）
- Change 039 七份档案、对应审计、`_dev/changes/INDEX.md`

### 明确不修改

- `pkg/config` schema 与加密配置实现
- `app/solve.go` 的解题和截图缓冲业务
- Wails bindings、依赖、正式配置、`build/bin/releases/**`

## TDD 步骤

1. 前端红灯：断言录制映射更新 `vkCode` 且移除 `comboID`；保存 payload 不保留旧组合。
2. 修改 `App.vue` 的 `key-recorded` 映射，保持冲突回滚与 `shortcut-saved` 逻辑。
3. 后端红灯：断言 `send_fast` / `delete` 各发出一次结构化“已触发” Toast，且 delegate 各执行一次。
4. 修改 `service.go`，反馈只表示快捷键分发，不表示下游成功。
5. Windows 回归：以 F10/F12 单键 heldKeys 走 `onKeysChanged`，验证 action 匹配；不安装真实系统 Hook。
6. 历史迁移红灯：构造与正式配置同形的 5 项错配，证明逐项冲突场景下两阶段迁移全部恢复；另覆盖不可解析、目标重复、非迁移占用和磁盘不写。
7. 在 `pkg/shortcut` 提供无歧义单 F 键解析与 ComboID 可读名检查；在 `ConfigManager.Load` 的默认迁移后、Validate 前应用 Windows 内存迁移。
8. 截图 Dock 红灯：断言五项实际绑定始终存在且 DOM 顺序严格为 F8→F9→F10→F11→F12，满三张时不移除 F8 键帽。
9. 调整 `ScreenshotDock.vue` 的提示顺序、F11 动态绑定和紧凑间距，不改变发送或截图业务。
10. 运行聚焦测试、全量 Go、vet/build、前端回归和 Vite build。
11. 首版候选因用户 GUI FAIL 归档为 failed 或 transient，重建唯一 Change 039 候选并验证保护边界。
12. 更新实施、验证、回滚、总结和 INDEX；执行提交边界审计，但未经授权不 commit/push/晋升。

## 回滚

前端恢复原录制映射、后端移除新增反馈、删除 Windows 聚焦测试即可；配置 schema 未变，不需要迁移或重写用户配置。已经保存为旧 `vkCode` 的错误绑定需用户重新录制一次，修复不会猜测并自动改写。
