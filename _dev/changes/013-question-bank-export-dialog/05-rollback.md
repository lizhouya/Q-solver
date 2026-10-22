# 回滚预案 05

## 回滚范围

本 Change 只能定向回退 013 新增的导出弹窗、单次请求、试卷模板和状态文案，必须保留 Change 010～012 已实现的题库 Store v2、自动分段、历史重导出、Mobile 路由和旧输出迁移结果。

## 回滚触发条件

- 新弹窗阻断当前段或历史段导出，且无法通过旧兼容入口临时恢复。
- 单次目录导致恢复记录被移动、覆盖或写入错误根目录。
- 导出前校验改变当前分段状态，造成未确认的封存或丢弃。
- 组织稿覆盖 `原始记录.md`、丢失截图/原答案，或重复导出覆盖既有目录。
- 新 Wails bindings 与项目 v2.11.0 runtime 不兼容，导致应用无法启动或调用。

## 定向回滚步骤

1. 停止应用，确认没有正在解题、整理或导出的任务。
2. 记录当前工作树和 Change 013 文件清单；不得使用 `git reset --hard`、`git clean`、`git add .` 或 `git add -A`。
3. 仅撤销以下 013 内容：
   - App 的导出草稿、单次请求和一次性目录选择接口；
   - `QuestionBankExportModal.vue` 及其 Store/API/bindings 接线；
   - ActionBar、TopBar、SettingsModal、QuestionBankPanel 的 013 文案与职责调整；
   - `ReferenceAnswer` 和新试卷渲染逻辑，以及对应测试。
4. 恢复旧无参 `ExportCurrentQuestionBankSegment()` 与 `ExportQuestionBankSession(id)` 前端调用；这些后端兼容入口在 013 实施中没有删除。
5. 重新执行聚焦测试、全量 Go 测试、前端构建和 Wails 构建。
6. 核对 `ArchiveDir/_记录` 与所有既有导出目录仍然存在且未改变。

## 数据保护

- 不删除任何 `原始记录.md`、组织稿、`assets/`、`session.json` 或 `_记录` 目录。
- 不重命名或覆盖已经由 013 生成的输出；即使代码回滚，新输出也作为用户资料保留。
- 不修改 API Key、Token、BaseURL、简历路径、主配置或 agentmemory 配置。
- 新 JSON 字段为兼容扩展；旧代码读取时会忽略未知字段，不需要迁移或清洗历史 manifest。

## 配置处理

- 旧 `company` 偏好字段继续可被读取，因此代码回退不要求改写用户配置。
- 一次性导出目录从未写回偏好，无需配置回滚。
- 如果 Wails CLI 构建产生 runtime shim 噪音，只按构建前后精确差异恢复生成文件，不升级或降级依赖。

## 实际回滚记录

本 Change 尚未发生回滚。若后续执行，需在此记录触发原因、精确文件、验证结果和保留的数据目录。
