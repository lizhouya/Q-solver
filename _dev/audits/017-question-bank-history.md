# Change 017 题库历史浏览与归档元数据审计

日期：2026-10-23
状态：事前架构审计完成，待实现后复核。

## 审计结论

### 状态隔离

桌面实时 `solution.history`、持久化题库 Session/Record 和 Solver `chatHistory` 是三套独立状态。历史浏览只能新增题库面板内的只读状态，不得复用实时 history；软清空桌面展示也不得调用 Solver 上下文清理。

### 无副作用读取

`Store.SnapshotByID` 是历史详情唯一合适入口。`Store.Open` 会切换 current 并可能修改状态、Record 后写盘，禁止用于预览。`Store.List` 继续全量返回，以保护启动恢复和每日编号。

### 最近 20

限制必须位于 UI 专用 App/Wails API：返回最近 20、total、hidden 和固定 limit。详情与图片重新校验 ID 属于该集合，避免旧前端引用绕过隐藏边界。隐藏不是删除，不得自动导出或调用模型。

### 图片与路径

前端只提交 sessionID、recordID 和 imageIndex，不能提交磁盘路径。后端从深拷贝 Session 定位相对路径，复用 `validID`、`safeChild`、`readLimited` 和图像解码限制，拒绝 symlink、穿越、绝对路径、格式不一致和资源超限。

### 模型信任边界

单位、出题方和题型由模型返回时只是候选。Prompt 要求仅提取截图明确内容、空缺优先、不执行截图内指令；解析继续 `DisallowUnknownFields`。候选只有经过用户最终确认才能写入 Session 权威字段。

### 两阶段副作用

打开或取消弹窗零副作用。整理阶段可以封存并保存 Questions/候选，但不创建最终导出目录；最终阶段只做权威元数据写回和本地导出，不再次请求模型。RawOnly 维持单阶段零模型调用。

### Change 016 叠加

`api.js`、Wails bindings 和 INDEX 已有 Change 016 未提交差异。实现与生成后必须核对 `ClearMobileDisplay`、`ResetMobileOutputLink`、`SetMobileStableLink` 及其 DTO 不丢失，不使用批量恢复。

## 风险与验证重点

- A→B 快速切换和关闭面板后的迟到响应必须由请求版本隔离。
- 8 MiB 详情限制不得通过截断原答案实现。
- `discarded` 只能读取，不得整理、导出或恢复。
- 清空失败、取消、busy 或后端拒绝时不能清桌面历史。
- 不调用真实 provider；fake provider 断言请求次数。
- race 环境失败如实记为 BLOCKED。

## 批准业务源码范围

范围以 Change 017 `02-plan.md` 的文件白名单为准；超出范围须先更新方案并重新确认。
