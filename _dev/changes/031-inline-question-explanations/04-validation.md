# 验证记录 04

## TDD 证据

### 后端红灯

`go test ./pkg/questionbank -run '^TestNamedPaperPlacesEachExplanationDirectlyAfterItsQuestion$' -count=1` 首次返回 1。失败明确显示旧输出仍为：

- `## 试题` 下连续输出两道题；
- 文件末尾另有 `## 参考答案与解析`；
- 新契约要求的 `## 试题与解析` 不存在。

实现后同一命令通过。

### 前端红灯

`node frontend/questionbank_history_regression_test.mjs` 首次返回 1，失败于导出弹窗不包含“每道题紧跟参考答案和解析”。更新 Store 和弹窗文案后通过。

## 自动验证

| 命令 | 结果 | 备注 |
|---|---|---|
| 逐题解析聚焦 Go 测试 | PASS | 两题顺序、每题唯一解析、无集中答案章节、空字段回退、无图片、原稿截图均通过 |
| `go test ./pkg/questionbank ./app -count=1` | PASS | 两包分别约 6.0 秒、7.2 秒 |
| `go test ./... -count=1` | PASS | 最终状态 22 个包测试/无测试文件均成功 |
| `go build ./...` | PASS | 无输出，退出码 0 |
| `go vet ./...` | PASS | 无输出，退出码 0 |
| 14 份前端 `.mjs` 静态回归 | PASS | 全部输出 passed |
| `npm --prefix frontend run build` | PASS | Vite v4.5.14，132 modules transformed，2.81 秒 |
| 受控 Wails 独立构建 | PASS | Wails CLI v2.16.0，生产构建 6.003 秒 |
| Change 031 EXE 自动启动 | BLOCKED / NEEDS_USER | 用户现有 Change 030 实例持有单实例锁；新实例退出码 0，未停止用户进程 |
| Change 031 范围 `git diff --check` | PASS | 退出码 0；仅显示既有 LF/CRLF 提示 |
| Change 031 文档尾随空白扫描 | PASS | 无匹配 |

## 产物

- 路径：`build/bin/Q-Solver-change031-inline-question-explanations.exe`
- 大小：42,331,136 字节
- 修改时间：2026-10-29 00:03:52.248 +08:00
- SHA-256：`82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46`

## 验收标准对照

- [x] 每道题紧跟该题答案与解析。
- [x] 第一题解析位于第二题题干之前。
- [x] 不再输出末尾重复的全局答案章节。
- [x] 空参考答案、知识点、解析和核对说明安全回退为“待核对”。
- [x] 原始记录、安全围栏、截图引用和无图片整理稿契约保持不变。
- [x] `RawOnly`、文件名、ExportResult 与新版本目录契约保持兼容。
- [x] UI 预计输出和弹窗说明准确，未新增选择控件。
- [x] 无新增模型调用、Prompt、schema、持久化字段、Wails API、bindings 或依赖。
- [ ] Windows GUI 真实导出排版待用户人工验收。

## 完整性复核

- 构建前后 11 个保护文件 SHA-256 均不变：`wails.json`、`main.go`、`go.mod`、`go.sum`、前端包清单/锁文件和 Wails bindings/runtime。
- `.git/index` SHA-256 在本 Change 全程保持 `5EC93489E855B95724E4F90482A8A5452901F6F0C4877EE34B8F3B69E55A07B7`。
- 最终工作树：53 个 tracked 修改、94 个 untracked、0 个 staged；相较事前只增加 7 个 Change 031 文档，tracked 总数因本次白名单文件原已修改而保持不变。
- 全树 `git diff --check` 的既有 `frontend/wailsjs/go/models.ts` 尾随空白基线未在本 Change 重复标为通过或修复；本次白名单范围检查通过。
- Vite 仍有既有主 chunk 大于 500 kB 警告，本 Change 未新增依赖或扩大为拆包任务。

## 人工验收

1. 先正常退出当前运行的 Change 030 验证程序。
2. 双击 Change 031 验证版。
3. 对非敏感测试分段执行“本次调用模型整理”并导出。
4. 检查 `<题库名称>.md` 中每题后立即出现答案、知识点、解析、来源与核对说明；检查 `原始记录.md` 仍完整。

## 历史题库再生成验证

用户指定只处理有明确单位归属的 TCL 和中国移动；未处理其他无归属或名称不明确的导出。

### TCL

- 权威 Session：`a6ef78f5c602e1950b991e3d`，单位 `TCL`，状态 `exported`。
- Session 当前结构化题为 0，不能使用 `ExportSession` 生成整理稿；选择内容完整的离线恢复稿 `_02` 作为只读来源，不调用模型、不回写 Session。
- 新目录：`2026-10-25_095652_TCL_综合笔试_03`。
- 结果：27 个题位、27 个逐题解析，独立答案章节已移除。
- `原始记录.md` 与 `_02` 逐字节一致；25 个 assets 文件名和 SHA-256 集合完全一致。
- 新整理稿 SHA-256：`455B4DC16CCCDCD128507C7D6EB2A09E8A85B293D9ECE1136A8A6D1C7853A702`。

### 中国移动

- 权威 Session：`36df29f9bbb1b5314ada5cad`，单位和出题方均为 `中国移动`，状态 `exported`。
- 使用现有 Change 031 `Store.ExportSession` 对 106 道结构化题进行纯本地重导；未调用模型。
- 新目录：`2026-10-24_021627_中国移动_在线统一笔试_02`。
- 结果：106 道题、106 个逐题解析，独立答案章节已移除。
- `原始记录.md` 与原版本逐字节一致；95 个 assets 文件名和 SHA-256 集合完全一致。
- Session 导出快照由 1 条增加为 2 条，最新目录准确指向 `_02`。
- 新整理稿 SHA-256：`91F6D830FA297A9995A74783D646D321861800F222A2CF3969E5C3BDCAC4D9C4`。

### 保护与边界

- 写入前在仓库外 `E:\DATA\Q-Solver-reexport-backup-20261029-01` 保存两份 Session manifest 与 TCL `_02` 整理稿副本，并逐项验证 SHA-256。
- 一次性辅助程序位于仓库外 `E:\DATA\Q-Solver-reexport-helper-20261029-01`，未修改项目源码或依赖。
- 原 TCL、中国移动导出目录在再生成过程中均未覆盖。再生成并完成哈希验证后，用户明确选择删除已被 `_03` 取代的 TCL `_02` 目录；最初 TCL 原始导出和最新 `_03` 保留，`_02` 整理稿保护副本仍在仓库外。其他题库未处理。

## 结论

业务实现、TDD、Go/前端回归、Vite 与 Wails 独立构建均通过；保护文件和 Git 暂存状态符合边界。Windows GUI 真实导出仍为 `NEEDS_USER`，因此 Change 031 不 Close、不提交、不推送、不发布。
