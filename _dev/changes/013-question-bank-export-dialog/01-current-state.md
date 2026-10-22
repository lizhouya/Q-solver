# 现状快照 01

## 当前行为

- `SettingsModal.vue` 的题库标签同时保存最终目录、单位名称和模型整理开关。单位作为长期偏好容易跨考试残留。
- `ActionBar.vue` 的“导出题库”直接调用 `ExportCurrentQuestionBankSegment()`，没有单次确认或参数。
- `QuestionBankPanel.vue` 重复显示最终目录并提供“导出当前段”；同时包含“新建分段”，但只在 `active/paused` 状态启用，已导出时显示为灰色按钮。
- `questionbank.js` 顶部标签统一显示 `题库 · 当前段 N 次`，即使会话已是 `exported/ended/discarded` 终态。
- “N 次”代表原始提交次数；一次提交可能包含 1～3 张截图或多道题，并非整理后的题目数量。

## 后端现状

- `app/questionbank.go` 暴露无参数的当前段导出与仅带 ID 的历史导出 API，单次名称、位置和整理开关只能使用持久偏好。
- `pkg/questionbank/export.go` 以 `opts.Name`、单位或“笔试题库”决定目录名，组织稿固定为 `题库.md`。
- 最终输出直接写在 `ArchiveDir/<时间_名称>[_NN]`，恢复记录保存在 `ArchiveDir/_记录/<分段ID>`。
- `原始记录.md` 保存截图与原答案；当前 `题库.md` 也会按 `ImageIndexes` 再次插入截图。
- 组织结果已有 Category、Type、题干、选项、知识点、解题要点与核对说明，但没有独立的参考答案字段。
- 模型整理按记录逐条执行；只在持久偏好 `autoOrganize` 明确开启时调用模型。

## 分段语义

1. 第一次解题时懒创建可写段。
2. 后续解题继续追加到 Active 段。
3. 手动 Split 将 Active/Paused 段封存为 Ended；下一次解题懒创建新段。
4. 导出成功将段标记为 Exported；下一次解题同样懒创建新段。
5. 软清空将段标记为 Discarded，保留原始文件。
6. 终态会话仍可作为 Store 的当前快照返回，但不可继续写入，因此“当前段”文案容易误导。

## 兼容与风险

- Change 010～012 均存在未提交修改，本 Change 必须在同一工作树中精确限定文件，不能批量恢复或暂存。
- 旧前端和当前生成 bindings 仍调用无参数方法；修改签名会连带 `frontend/wailsjs/go/app` 和 `models.ts`。
- 历史输出使用固定 `题库.md`，新命名只能影响新导出，不能重命名旧文件。
- 自定义一次性目录不得改变恢复记录根，也不能静默写回持久偏好。
- 弹窗打开不能为“自动识别”提前调用模型；首次导出只能用本地元数据给出建议，整理完成后才有可靠 Category/Type。
