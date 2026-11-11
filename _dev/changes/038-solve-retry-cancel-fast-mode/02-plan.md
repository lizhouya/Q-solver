# 实施方案 — 02

## 当前状态

权威设计见 `design.md`，逐文件 TDD 步骤见 `implementation-plan.md`。用户已批准设计、书面规格和实施计划；2026-11-11 已按批准范围完成实现、普通自动门禁和保护候选验证，用户随后确认当前测试版本“没有什么问题”。race 因 Windows `0xc0000139` 保持 `BLOCKED` 并已披露；功能提交为 `5564a45`，Change 按用户授权进入治理提交和关闭阶段。

## 1. 方案概述

建立一个后端权威、单槽、有界的 `SolveAttempt` 生命周期。普通与快速发送共享同一启动管线；失败/取消后保留原图和稳定轮次目标，重试时由后端验证并原位重启同一题库 Record。中断通过既有 context 传播到 Provider，旧任务完全退出前禁止新请求。快速模式只在请求副本中覆盖 `DomainId=gen-speed`，不修改持久配置。Desktop 提供重试与中断按钮，Mobile 只同步安全状态和实际快捷键提示。

## 2. 分步实施计划

| 步骤 | 操作 | 主要范围 | 预期产出 | 完成标志 |
|---|---|---|---|---|
| 1 | 解题尝试状态 TDD | `app/solve_attempt*.go`、`app/solve.go` | 单槽状态机、稳定目标、失效规则 | 并发/过期/重复调用测试通过 |
| 2 | 题库原位重启 TDD | `pkg/questionbank/**`、捕获层 | terminal Record 安全恢复 running | 同一 Record、图片不重复、失败原子性通过 |
| 3 | 取消语义收敛 | `pkg/task`、`pkg/solution`、`app` | cancelling/cancelled 终态和 in-flight 屏障 | context 传播、迟到事件和 KeepContext 测试通过 |
| 4 | 普通/快速请求模式 | `app`、`pkg/solution`、`pkg/domain` | normal/fast 单次模式 | Config/subscriber 不变，题库场景准确 |
| 5 | 快捷键与迁移 | `pkg/config`、`pkg/shortcut` | send_fast / cancel_solve | 默认值、冲突、旧配置和 FIFO 测试通过 |
| 6 | Desktop UI | Pinia、SolveView、ErrorView、控制组件/API | 重试、中断、快速状态 | typed 轮次复用、按钮幂等和迟到隔离通过 |
| 7 | Mobile 与提示 | `app/mobile.go`、`pkg/mobile/**` | cancelled 状态、两项提示 | SSE/DOM/Reset 回归通过 |
| 8 | bindings、文档与全量验证 | Wails bindings、README、Change | 固定 v2.11.0 候选 | 自动门禁、保护构建和人工验收如实记录 |

## 3. 允许的业务源码范围

计划允许修改或新增以下范围；详细计划不得无理由越界：

- `app/app.go`
- `app/solve.go`
- 新 `app/solve_attempt.go` 及对应测试
- `app/questionbank_solve_capture.go` 及相关测试
- `app/questionbank_next_exam.go`、`questionbank_continuation.go`、`questionbank.go`（仅接入 retry snapshot 失效）及对应现有测试
- `app/mobile.go` 及相关测试
- `pkg/task/coordinator.go` 及测试（仅在证明现有接口不足时）
- `pkg/solution/solution.go` 及测试
- `pkg/questionbank/store.go`、`continuation.go` 及测试
- `pkg/config/config.go`、`manager.go` 及快捷键迁移测试
- `pkg/shortcut/delegate.go`、`service.go` 及测试
- `pkg/mobile/state.go`、`bridge.go`、`web/index.html` 及测试
- `frontend/src/stores/solution.js`
- `frontend/src/services/api.js`
- `frontend/src/App.vue`
- `frontend/src/components/SolveView.vue`
- `frontend/src/components/ErrorView.vue`
- 新的聚焦控制组件（如详细计划证明拆分必要）
- 设置、欢迎、TopBar、ScreenshotDock 的快捷键提示相关文件和回归测试
- `frontend/wailsjs/go/app/App.js`、`App.d.ts`、`models.ts`
- `README.md`、`README_EN.md`

明确禁止修改 Connection Profile schema、题库导出格式、依赖、Wails runtime、正式用户配置和 `build/bin/releases/**`。

## 4. 测试与验证计划

### 聚焦测试

- App attempt 状态、失败快照、同轮重试、双击/过期拒绝、成功失效。
- QuestionBank terminal Record restart、图片引用、原子落盘、题号上下文。
- Task/Solver context cancellation、in-flight 屏障、KeepContext 不污染。
- 普通/快速场景覆盖和 Config/subscriber 不变。
- 快捷键默认值、冲突迁移、FIFO 和动态提示。
- Desktop typed event、按钮幂等、迟到事件、部分答案和错误诊断。
- Mobile cancelled Snapshot、SSE 重放、DOM、截图 Dock 与 Reset。

### 全量命令

```bash
go test ./pkg/task ./pkg/solution ./pkg/questionbank ./pkg/shortcut ./pkg/mobile ./app -count=1
go test ./... -count=1
go vet ./...
go build ./...
go mod verify
for test_file in frontend/*_test.mjs; do node "$test_file" || exit 1; done
npm --prefix frontend run build
git diff --check
```

相关 race 继续执行并单列；若复现 `0xc0000139`，记录 BLOCKED。Wails 候选固定 v2.11.0，按构建产物政策冻结稳定版、Release、依赖、bindings、dist 和 Git index 哈希。

### 人工验收

- 网络失败原图重试；中断前/中/后；普通与快速快捷键；自定义绑定和冲突。
- 题库开启/关闭；KeepContext 开启/关闭；Desktop/mobile/dual。
- 0～3 张截图 Dock、开始下一套、分段续接、Mobile 清屏与重连。
- 不执行真实收费 API，除非用户另行明确授权并提供隔离配置。

## 5. 回滚预案

- 业务失败时定向回滚 attempt/retry、Record restart、快捷键 action、Desktop 控制和 Mobile cancelled 状态；不回滚 Change 035/036/037 已验证能力。
- 配置迁移只新增 action，回滚代码时旧版本自然忽略未知快捷键键值，不删除用户绑定。
- 候选未满足必要门禁前不覆盖根稳定版；二进制回滚只使用哈希通过的 superseded 归档。
- 已提交后的 Git 回滚仅在授权后使用 `git revert`，禁止 `git reset --hard`。

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| cancel 后立即新请求 | 两个 Provider/题库 capture 短暂重叠 | 新发送和重试检查 `HasInFlightTask`，等待旧 goroutine Complete |
| 任意旧轮重试 | 错题、重复收费、越权读取图片 | 后端单槽目标校验，不接受前端 Base64 |
| 题库重复记录 | 同题出现多轮且编号变化 | terminal Record 原位 restart，Record ID 和图片引用不变 |
| KeepContext 重复提交 | 历史污染 | 失败不提交；重试成功只提交一次；会话代际变化使快照失效 |
| 快速模式污染默认场景 | 后续普通题错误使用 gen-speed | 只修改请求 Config 副本，不调用 ConfigManager 保存 |
| 自动重试产生费用 | 重复收费和服务端重复处理 | 只允许用户显式单次重试并提示费用 |
| Mobile 扩大控制面 | 局域网操作风险 | Mobile 继续只读，不新增命令端点或重试 token |
| 新默认快捷键冲突 | 覆盖用户绑定 | 迁移前检查现有 ComboID；冲突则保持未绑定 |
