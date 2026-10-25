# 实现记录 03

状态：业务实现与普通自动验证已完成；Windows GUI 人工验收待执行；race BLOCKED；未提交、未推送、未发布。

## 时间线

- 2026-10-23：完成需求、现状和方案归档；用户批准“复用模型整理”和“仅隐藏并保留”方案。
- 2026-10-23：在 `dev/qsolver-custom` 的 Change 016 未提交改动之上实施 Change 017，没有删除、覆盖或迁移既有题库与导出数据。
- 2026-10-23：完成后端数据模型、模型建议协议、安全历史 API、两阶段导出、前端只读历史、实时历史软重置和专项回归。
- 2026-10-23：生成正式 Wails bindings 和独立 Windows 验证程序；最终源码复验后重新构建验证程序。

## 已实施行为

### 软清空与实时历史

- `DiscardQuestionBankSegment(true)` 成功后才调用 `solution.resetLiveHistory()`，清空桌面实时答案、流式缓冲、thinking 状态、定时器和内联错误，并回到待记录界面。
- 用户取消、后端拒绝或调用失败时不改桌面历史。
- 不调用模型上下文清理、任务取消、截图清理、Mobile 清屏或物理文件删除；下一次正式解题继续懒创建新段。

### 最近 20 段只读历史

- `Store.List()` 保持全量语义；App/Wails 专用接口仅向 UI 返回按时间倒序的最近 20 段及 `total`、`hidden`、`limit`。
- 详情使用 `Store.SnapshotByID`，不调用有副作用的 `Store.Open`，不切换后台 current，也不恢复历史段写入。
- ended、exported 和 discarded 均可只读查看；discarded 不提供整理、导出或恢复。
- 第 21 段及更早段拒绝详情和图片 API，但磁盘内容完整保留，不触发模型、导出或删除。
- 单段详情 JSON 上限为 8 MiB，超限整体拒绝，不截断完整原答案。

### 历史图片安全边界

- 前端只提交 `sessionID`、`recordID` 和 `imageIndex`，不提交文件路径；后端依据持久化记录重建安全路径与 data URL。
- 仅允许普通 PNG/JPEG；拒绝绝对路径、`..`、反斜杠、冒号、符号链接、非普通文件、扩展名/MIME/实际格式不一致。
- 原始文件上限 20 MiB，像素上限 80,000,000；详情 DTO 不暴露绝对路径、恢复目录或清单路径。

### 元数据建议与稳定命名

- Session 新增可选的 `Name`、`Issuer`、`NameCustomized`；Record 新增可选的 `MetadataSuggestion`，继续兼容 v1/v2 JSON，不批量重写旧文件。
- 模型整理的严格 JSON 响应可同时返回 Questions 与招聘单位、出题方、题型候选；候选只显示为“模型建议（待核对）”，不自动覆盖权威字段。
- 已有 Questions 的旧 Record 不为补候选重新调用模型；失败不覆盖旧建议、原答案或已成功结果。
- 自动名称按“招聘单位_出题方_题型”组成，忽略空值和大小写相同的重复项；单位和出题方均为空时使用每日稳定编号。
- 人工名称持久化保护；只有显式“恢复自动命名”才重新联动。

### 两阶段模型整理与最终导出

- `PrepareQuestionBankSegment` 只封存必要的活动段并整理尚未完成的 Record，保存 Questions 和待核对建议，不创建最终导出目录。
- `ExportQuestionBankSegment` 写回人工确认后的权威元数据并执行纯本地导出，固定不再次调用模型。
- RawOnly 保持单阶段、零模型调用，只生成 `原始记录.md` 与 `assets/`。
- 部分 Record 整理失败时保留成功结果和失败数，允许人工核对后导出已有内容。
- 整理完成刷新草稿时，保留用户已修改的单位、出题方、题型、导出目录和手工名称，同时接收新的模型建议、失败数和准备状态。

### 前端只读视图与竞态

- 题库面板区分“后台当前段”和“正在浏览”，归档详情与 `solution.history` 完全隔离。
- A→B 快速选择、关闭面板后的迟到详情、切换分段后的迟到图片均通过请求版本控制失效；同段不同图片可并发加载。
- 归档文本使用 Vue 文本插值与 `<pre>`，不使用 `v-html`；“重新导出”按钮阻止事件冒泡。
- 进度文案区分“正在整理并生成建议”和“正在导出原始记录”，不再把准备阶段描述成已导出。
- 历史分段点击后立即滚动并聚焦详情区；状态明确区分“正在读取”“正在浏览”和“读取失败”，零提交分段显示无记录说明且不再提供无效的重新导出入口。

## 主要文件

- 后端模型与安全存储：`pkg/questionbank/types.go`、`pkg/questionbank/organize.go`、`pkg/questionbank/history.go`、`pkg/questionbank/export.go`
- App/Wails API：`app/questionbank.go`、`app/questionbank_history.go`
- 前端状态与界面：`frontend/src/stores/solution.js`、`frontend/src/stores/questionbank.js`、`frontend/src/components/QuestionBankPanel.vue`、`frontend/src/components/QuestionBankExportModal.vue`、`frontend/src/services/api.js`
- 回归测试：`pkg/questionbank/history_test.go`、`app/questionbank_history_test.go`、`frontend/questionbank_history_regression_test.mjs`
- 生成 bindings：`frontend/wailsjs/go/app/App.js`、`frontend/wailsjs/go/app/App.d.ts`、`frontend/wailsjs/go/models.ts`

## 授权与数据边界

- 自动测试没有调用真实模型、上传截图或消费真实 API 额度。
- 未读取或输出 API Key、Mobile Token、完整 Mobile URL或私人快捷键配置。
- 未删除、覆盖或迁移 `_记录`、截图、旧导出目录。
- 未执行 commit、push、Tag 或 Release。
