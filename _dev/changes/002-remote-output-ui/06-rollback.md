# 回滚预案与遗留项 06 — 002-remote-output-ui

> 本文档含两部分：**A. 提交前治理检查报告**（2026-09-28 新增，Q-Solver 二次开发提交前治理）与 **B. 回滚预案与遗留项**（002 原始内容）。治理检查仅汇报工作树现状，**未执行 git add / commit / 回退**，未修改任何代码。

---

# 一、提交前治理检查报告（提交门禁检查）

## 1. 检查命令与依据

- `git status` / `git status --porcelain=v1`：工作树全量快照（2026-09-28，分支 `dev/qsolver-custom`）。
- `git diff (--stat)`：已修改文件的差异定位归属。
- `git ls-files --others`：未跟踪目录内文件明细。
- **未执行** `git add` / `git commit` / `git restore` / `git checkout`——纯只读检查。

## 2. 分类结果

### A. 001-mobile-output 必需变更（含新增与修改）

| 文件 | 状态 | 纳入理由 |
|---|---|---|
| `app/app.go` | M | 01 变更：`mobileBridge`/`mobileServer`、`setupMobile()`/`shutdownMobile()`、`EmitEvent` 按 `OutputMode` 分流 |
| `app/mobile.go` / `app/mobile_test.go` | ?? | 01 新增绑定与单测 |
| `pkg/mobile/bridge.go` / `server.go` / `state.go` | ?? | 01 引入的 Go mobile 服务（bridge/server/state） |
| `pkg/mobile/*_test.go`（bridge_server/lanip） | ?? | 01 新增单测 |
| `pkg/config/config.go` | M | `OutputMode` 字段 + validate（desktop/mobile/dual） |
| `frontend/src/App.vue` / `components/SettingsModal.vue` / `services/api.js` / `stores/settings.js` | M | 01 新增移动同步二维码 / token / LAN 设置 UI（002 未动前端） |
| `frontend/wailsjs/go/app/App.d.ts` / `App.js` / `models.ts` | M | 01 同步新绑定（`NewMobile`、mobile 类型） |
| `_dev/changes/001-mobile-output/`（6 文档） | ?? | 01 归档（00–05） |

### B. 002-remote-output-ui 必需变更

| 文件 | 状态 | 纳入理由 |
|---|---|---|
| `pkg/mobile/web/index.html` | ?? | 002 重构：Answer-first 响应式 + History + 隐藏 Analysis |
| `_dev/mobile-check/dom-mobile-page.js` | ?? | 002 重写：14 项行为断言 |
| `_dev/mobile-check/check-raw-syntax.js` | ?? | 002 新增：vm.Script 语法护栏 |
| `_dev/changes/002-remote-output-ui/`（00–06 共 7 文件） | ?? | 002 归档 + 本报告 |

### C. 治理文件（跨变更通用）

| 文件 | 状态 | 说明 |
|---|---|---|
| `_dev/memory/`（手机页经验 + memory 策略） | ?? | 002 完成后的 AgentMemory 记忆（按用户指示仅在 002 最终确认后才记录——本文件已含该记录） |
| `_dev/changes/INDEX.md` | M | 变更编号总表更新（002 已标记「已完成」） |

### D. 非本阶段文件（本次提交不纳入）

| 文件 | 状态 | 说明 |
|---|---|---|
| `go.mod` / `go.sum` | M | **Wails v2.11.0 → v2.16.0、go 1.24.1 → 1.25.0** 等依赖升级；非 001/002 必需，建议另立变更或回退（01 期间明确「不升级 Wails」红线） |
| `frontend/wailsjs/runtime/runtime.d.ts/js` | M | 随 Wails 升级生成的 runtime 绑定（非业务代码）；同 go.mod 处置 |
| `_dev/audits/`（01/02 审计） | ?? | 独立审计产出，非功能变更文件 |

> **注意（A 与 D 的边界）**：`app/`、`pkg/mobile/` 中 `web/index.html`（属 B）与 `pkg/mobile/*.go`（属 A）两者都存在；前端 `frontend/src` 未跟踪（001 新增），`frontend/wailsjs/go/app` 属于 A（随 001 绑定），`frontend/wailsjs/runtime` 属于 D（随 Wails 升级）。

## 3. 提交建议（仅供参考，**未执行** commit）

| 顺序 | 提交内容 | 说明 |
|---|---|---|
| 1 | **001 必需**（A 表）| 首次提交，严格按 `_dev/changes/001` 范围 |
| 2 | **002 必需**（B 表）| `pkg/mobile/web/index.html` + `_dev/mobile-check/` + `_dev/changes/002/` |
| 3 | 治理索引（C 表）| INDEX + memory |
| 4 | 依赖升级（D 表）| **另立变更**或决策是否回退 Wails 升级 |

## 4. 检查结论（无关项）

- 工作树与 git 索引一致：所有未提交改动均为 001 / 002 / 治理三类所需，外加一项未被红线允许的 `go.mod` 依赖升级（D）。
- 语法与验证链已绿（002/ 04）：`check-raw-syntax`（vm.Script）、`dom-mobile-page`（14 断言）、`go vet/test/build` 均通过。
- **本次未执行** git add/commit；提交时机与是否拆分子 commit 由用户决定。

---

# 二、回滚预案与遗留项（原始 002 实现）

> 本变更以「单文件前端重构」为特征，回滚成本极低。对应 00-request §6 与 02-plan §5。

## 1. 回滚场景与操作

| 场景 | 操作 | 失效风险 |
|---|---|---|
| `index.html` 回归（布局/JS 错误） | `git checkout -- pkg/mobile/web/index.html` 恢复 001 交付版 | 低：单文件，瞬时恢复 001 的「状态+analysis+答案」页面 |
| History 交互状态混乱 | 仅回退「会话历史」分支：`sessionHistory/latestSnapshot/selectedSeq` 取消，`applySnapshot` 回退为「仅渲染 latest」 | 低：隔离在 `index.html` 内 |
| 测试脚本改动误伤 001 断言 | `git checkout -- _dev/mobile-check/dom-mobile-page.js` 回滚脚本 | 低 |
| 后端被误动 | 本变更未动任何 Go 文件；若发现误动则 `git checkout` 对应文件 | 极低（红线约束，已核验 `git status` 无 Go 文件改动） |
| 真机验收发现断点不适配 | 调整 CSS 断点数值（非结构性），重跑两层护栏 | 低：断点集中在一个 `@media` 块 |

## 2. 执行记录

| 日期 | 事件 | 结果 |
|---|---|---|
| 2026-09-27 | 变更归档完成、验证全绿 | 未发生回滚 |

## 3. 遗留 / 后续（不阻塞本次）

- **真机验收**（用户执行）：`wails build -debug -windowsconsole` → 运行 → iPhone/iPad 打开 QR URL。重点：390（Phone 单列、strip）、768–899（Portrait 单列+strip）、1024（横屏双栏）、1366（大屏双栏钳宽）。
- **`_dev/mobile-check` shim 局限**：`dom-mobile-page.js` 只断言 DOM 行为，**不能做真实布局尺寸断言**（无 CSS 引擎）。响应式断点正确性依赖真机目检。
- **刷新丢更早历史**：按 D-UI-004 接受；若后续需持久化，新开变更（不动本变更协议）。
- **`payload.question` 后端占位为空**：历史卡摘要优先 `question`，为空时回落 answer 首行；待后端补充 question 后摘要自动更优。