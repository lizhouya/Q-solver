# 现状快照 01

## Git 基线

- 日期：2026-10-28
- 分支：`dev/qsolver-custom`
- HEAD：`43022e10cd8fb1b38b72b56d95bc39b8bb4d201d`
- 事前状态：53 个 tracked 修改、87 个 untracked、0 个 staged。
- 工作树包含多个既有 Change 的未提交修改，本 Change 不恢复、不覆盖、不混入无关文件。

## 当前导出链路

1. 导出弹窗通过 `QuestionBankExportRequest` 提交名称、元数据、一次性目录和 `AutoOrganize`。
2. `app/questionbank.go` 将请求转换为 `questionbank.ExportOptions`。
3. `pkg/questionbank/export.go` 在临时目录中渲染文件，再原子重命名为新的版本目录。
4. 始终生成 `原始记录.md` 和 `assets/`；启用整理时再生成 `<题库名称>.md`。
5. 整理稿当前先按分类/题型集中输出所有题目，然后在 `## 参考答案与解析` 下再次遍历全部题目。

## 当前数据能力

每道 `Question` 已包含：

- `Title`
- `Stem`
- `Options`
- `ReferenceAnswer`
- `KnowledgePoints`
- `KeyPoints`
- `ReviewNote`
- 对应原始提交索引

因此本 Change 只需重排本地已有结构化数据，不需要重新调用模型或扩展 schema。

## 当前兼容契约

- `ExportResult.Markdown` 指向单一整理稿。
- `ExportResult.RawMarkdown` 指向 `原始记录.md`。
- 关闭模型整理时只生成原始记录和 assets。
- 每次导出创建新版本目录，旧导出不覆盖。
- 整理稿不引用图片；原始记录保留全部截图引用。
- 原答案使用自适应代码围栏，不改写内容。

## 已知基线限制

- 本地 Wails CLI v2.16.0，Go 模块 Wails v2.11.0；独立构建必须跳过 bindings、mod tidy 和 go.mod 同步。
- 全树 `git diff --check` 存在既有 `frontend/wailsjs/go/models.ts` 尾随空白失败；该基线不得归因于本 Change，也不在本次范围内修复。
