# 现状快照 01

## 数据与状态边界

- 题库模型是 `Session → Records → Questions`；`Record` 表示一次提交，一次可含 1～3 张截图和多道题。
- 桌面 `solution.history` 是内存实时答案历史；Solver `chatHistory` 是模型上下文；两者与持久化题库互相独立。
- `Store.SnapshotByID(id)` 返回深拷贝且不切换 current，是历史只读详情的正确入口。
- `Store.Open(id)` 会切换 current，并可能修改 Session/Record 后写盘，不能用于历史预览。
- `Store.List()` 全量扫描并按 `StartedAt` 倒序；启动恢复和每日稳定编号依赖全量结果，不能固定截断。
- `DiscardCurrent()` 只把当前段标为 `discarded` 并保存，保留 Records、答案、截图和 JSON；前端实时 history 当前不会同步清空。

## 当前导出与整理

- 导出草稿包含单位、题型、名称、目录和是否模型整理；目前没有出题方字段。
- 默认名称为 `<单位>_<题型>`；无单位时为 `笔试_<YYYYMMDD>_<NN>_<题型>`。
- 模型整理按每个未整理的 completed Record 一次请求，严格解析 `sourceRecordId + questions`；已有 Questions 的旧记录不会再次请求。
- 当前初始确认后才进行整理并导出，无法先展示模型元数据候选供人工核对。

## 当前前端与图片边界

- 题库历史列表读取完整 sessions，历史行不能打开只读详情。
- 现有 `openSession(id)` 调用有副作用的后端 Open，不能用于新浏览 UI。
- 流式回调始终写实时 history，归档详情不能复用该 store。
- 题库 JSON 中图片只保存恢复目录相对路径，不能直接把路径暴露给前端。
- 现有 `validID`、`safeChild`、`readLimited` 与图片解码可复用；限制为单图 20 MiB、data URL 约 28 MiB、80,000,000 像素、PNG/JPEG，并拒绝符号链接和非普通文件。

## 与 Change 016 的叠加边界

以下文件已有 Change 016 未提交修改，Change 017 必须逐段合并，不得覆盖：

- `frontend/src/services/api.js`
- `frontend/wailsjs/go/app/App.js`
- `frontend/wailsjs/go/app/App.d.ts`
- `frontend/wailsjs/go/models.ts`
- `_dev/changes/INDEX.md`

Change 016 的 Mobile API、共享 store、唯一确认层和快捷键链路必须完整保留。
