# 实施方案 — 02

## 1. 方案概述

在跨平台 `shortcut.Service` 层增加轻量的串行动作调度器：底层 Manager 的 `OnTrigger` 只负责将 action 快速放入内存 FIFO，专用工作协程在 Hook 回调之外依次执行现有 `handleTrigger`。这样既解除 Windows Hook 对截图/压缩的同步等待，又保持截图、删除、发送等动作的原始触发顺序。前端继续复用现有 ScreenshotDock，通过 `storeToRefs` 读取真实快捷键，并补充“继续截图”与满 3 张状态提示。

## 2. 分步实施计划

| 步骤 | 操作 | 涉及文件 | 预期产出 | 完成标志 |
|---|---|---|---|---|
| 1 | 先编写调度器红灯测试 | `pkg/shortcut/dispatcher_test.go` | 覆盖非阻塞入队、FIFO、停止后拒绝 | 旧代码因类型不存在而失败 |
| 2 | 实现串行动作调度器并接入 Service 生命周期 | `pkg/shortcut/dispatcher.go`、`pkg/shortcut/service.go` | Hook 只入队，业务在单 worker 中执行 | 聚焦 Go 测试通过 |
| 3 | 编写 Dock 红灯回归 | `frontend/screenshot_dock_regression_test.mjs` | 约束响应式快捷键、1/3～3/3 和三类指引 | 旧组件不满足断言 |
| 4 | 修复 Dock 展示 | `frontend/src/components/ScreenshotDock.vue` | 实际快捷键响应更新，继续截图/满额提示恢复 | 前端聚焦回归通过 |
| 5 | 运行全量验证 | 不新增生产文件 | Go、前端、Vite、Wails 结果真实记录 | 自动验证完成 |
| 6 | 更新实施、验证、回滚与总结记录 | Change 035 文档、INDEX | 留下真实证据与遗留人工项 | 提交边界可审计 |

## 3. 涉及业务源码的改动清单

允许修改：

1. `pkg/shortcut/service.go`
   - 将 `manager.OnTrigger` 从直接执行 `handleTrigger` 改为轻量入队。
   - 在 `Start`/`Stop` 中按正确顺序启动和停止调度器。
2. `pkg/shortcut/dispatcher.go`（新增）
   - 实现单 worker、FIFO、快速入队和幂等停止。
   - handler 在队列锁之外运行，耗时业务不阻塞后续入队。
3. `pkg/shortcut/dispatcher_test.go`（新增）
   - 验证阻塞 handler 不阻塞 enqueue、FIFO、停止边界和无并发 handler。
4. `frontend/src/components/ScreenshotDock.vue`
   - 使用 `storeToRefs` 获取 `solveShortcut`、`sendShortcut`、`deleteShortcut`。
   - 增加继续截图提示；3 张时显示已满状态。
5. `frontend/screenshot_dock_regression_test.mjs`（新增）
   - 静态约束组件响应性、计数和提示契约。

不在未更新本方案的情况下修改 `app/solve.go`、`pkg/screen`、配置、模型、题库、Mobile 或依赖文件。若测试证明必须调整，先在 `03-implementation.md` 记录理由和范围变化。

## 4. 测试与验证计划

### TDD 与聚焦验证

- `go test ./pkg/shortcut -run 'TestSerialActionDispatcher' -count=1`
- `node frontend/screenshot_dock_regression_test.mjs`
- `go test ./pkg/solution -run '^TestSolveThreeScreenshotsSingleCallInOrderAndPassesFullAnswer$' -count=1 -v`

### 全量自动验证

- `go test ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `go mod verify`
- 运行全部 `frontend/*_test.mjs`
- `npm --prefix frontend run build`
- `wails build`，构建前后核对 bindings、依赖和生成噪音
- Git 预检后执行范围内与全树 `git diff --check`

### Windows GUI 人工验收

1. 快速连续按 3 次截图快捷键，Dock 依次显示 `1/3`、`2/3`、`3/3`。
2. 截图过程中键盘不出现明显冻结；Hook 不因首张截图失效。
3. 第 4 次截图显示既有上限提示，不增加图片。
4. 按删除快捷键后变为 `2/3`，再次截图恢复 `3/3`。
5. 按发送快捷键后一次解题使用 3 张图片，Dock 清空。
6. 修改截图/删除/发送快捷键后，Dock 显示实际绑定。

## 5. 回滚预案

优先定向回滚 Service 调度器接线和 ScreenshotDock 展示，不回滚其他 Change。新增文件可单独移除，`service.go` 恢复 `manager.OnTrigger = s.handleTrigger`，Dock 恢复原模板与 setup 代码。禁止使用 `git reset --hard` 清理工作区。

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 每次 action 启动独立 goroutine导致乱序 | 发送抢在截图完成前执行 | 明确使用单 worker FIFO，不采用无序 goroutine |
| worker handler 持锁执行 | 后续入队仍阻塞 | handler 必须在队列锁之外执行 |
| Stop 与正在执行动作竞争 | 退出时残留动作或 panic | Stop 幂等；停止后拒绝新 action；测试正在执行和待执行边界 |
| 队列无界增长 | 极端按键风暴增加内存 | 正常快捷键频率下队列很短；停止时丢弃待执行项；实施时记录队列策略 |
| Dock 显示默认值而非实际绑定 | 用户按提示操作失败 | `storeToRefs` 绑定 Pinia computed，回归测试约束 |
| Wails 构建改写生成文件 | 提交混入噪音 | 构建前后对比并精确处理，不批量恢复用户改动 |
