# 实现过程记录 03

## 实现日志

| 日期 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-10-20 | 基线确认 | 聚焦 Go 测试与前端生产构建均通过 | `pkg/questionbank`、`app`、`frontend` | 先区分既有问题与本 Change 引入的问题；前端仍有既有单 chunk 体积警告。 |
| 2026-10-20 | 契约测试 | 先增加命名试卷、仅原稿、一次性目录、导出草稿与单次请求测试 | `pkg/questionbank/segment_test.go`、`app/questionbank_test.go` | 新 App API 在实现前按预期编译失败，随后逐层实现。 |
| 2026-10-20 | 导出数据模型 | 为 `Question` 增加 `ReferenceAnswer`，扩展 `ExportOptions` 与 `ExportResult` | `pkg/questionbank/types.go` | 新字段兼容旧 JSON；原稿路径与组织稿路径分别返回。 |
| 2026-10-20 | 整理契约 | 模型 JSON 增加参考答案，缺失时本地回退“待核对” | `pkg/questionbank/organize.go` | 参考答案只能从原答案提炼，不重新作答或冒充官方答案。 |
| 2026-10-20 | 试卷导出 | 组织稿改为“试题 + 文末参考答案与解析”，不再嵌入截图 | `pkg/questionbank/export.go` | `原始记录.md` 和 `assets/` 继续完整保留；显式关闭整理时不生成空试卷。 |
| 2026-10-20 | 原答案安全展示 | 使用长度自适应的反引号围栏包裹完整原答案 | `pkg/questionbank/export.go`、`segment_test.go` | 不改写原答案内容，同时避免其中的 HTML、外链或内嵌围栏成为活动 Markdown。 |
| 2026-10-20 | 一次性目录 | 最终导出可使用本次目录，恢复记录仍留在默认根的 `_记录` | `pkg/questionbank/store.go`、`export.go` | 新增不创建恢复目录的预校验，所有请求先验证再封存。 |
| 2026-10-20 | App 单次请求 | 增加导出草稿、单次导出请求和目录选择 API | `app/questionbank.go` | 当前段与历史段共享请求；异步任务启动前冻结名称、单位、题型、目录和本次整理授权。 |
| 2026-10-20 | 本地建议名称 | 本地推断主要题型；无单位时按当日分段顺序生成稳定编号 | `app/questionbank.go` | 不为预填弹窗调用模型；同一分段重复打开草稿名称稳定。 |
| 2026-10-20 | 长期单位退场 | UI、保存文件和未来懒创建分段均不再使用旧长期单位 | `app/questionbank.go`、`frontend/src/stores/questionbank.js` | 旧字段只保留 JSON/API 兼容；单位由每次导出确认。 |
| 2026-10-20 | 根级单例弹窗 | 当前段与历史段统一进入同一个导出确认窗口 | `QuestionBankExportModal.vue`、`App.vue`、`questionbank.js` | 打开、预填和取消不封存、不上传、不调用模型；提交后才启动导出任务。 |
| 2026-10-20 | 前端职责收敛 | 设置只保留默认目录和默认整理开关；ActionBar 成为当前段主入口；面板聚焦分段管理和历史重导出 | `SettingsModal.vue`、`ActionBar.vue`、`TopBar.vue`、`QuestionBankPanel.vue` | 消除设置、面板和答案操作区的重复职责；状态文案明确“N 次提交”不是题目数。 |
| 2026-10-20 | 分段快捷操作 | ActionBar 和题库面板均提供“结束本段” | `ActionBar.vue`、`QuestionBankPanel.vue` | 只封存当前段；下一次解题自动懒建新段。 |
| 2026-10-20 | 弹窗并发保护 | 使用请求版本号阻止迟到草稿覆盖新弹窗，并区分读取、选目录和提交状态 | `frontend/src/stores/questionbank.js` | 关闭弹窗只清理临时状态；历史导出不调用 `OpenQuestionBankSession`。 |
| 2026-10-20 | 正式 bindings | 生成新 App 方法和模型类型 | `frontend/wailsjs/go/app/App.js`、`App.d.ts`、`models.ts` | 保留项目 runtime v2.11 shim，不接受 CLI v2.16 的 runtime 通知接口噪音。 |
| 2026-10-20 | 回归补强 | 增加 `operation` 生命周期、稳定编号前缀和旧长期单位失效测试 | `app/questionbank_test.go` | 本地导出显示 `exporting`，任务结束清空；无单位名称由后端稳定前缀驱动。 |
| 2026-10-20 | 文档同步 | 更新中英文题库使用流程 | `README.md`、`README_EN.md` | 说明确认弹窗、分段、输出文件、费用边界和历史重导出。 |

## 与计划的偏差

- `02-plan` 提到“名称是否由系统自动生成”字段。实现中没有把该 UI 状态加入后端契约：名称自动联动只属于弹窗本地交互，后端始终以最终提交的 `Name` 为准，减少持久契约复杂度。
- 关闭本次模型整理时按已批准语义设置 `RawOnly=true`，即使历史记录已有部分结构化结果，本次也只输出原稿，避免把“未调用模型”误解为重新生成完整试卷。
- 旧 `company` 字段保留 JSON 兼容，但不再作为未来自动分段的默认值；单位改由每次导出确认。
- Wails CLI 2.16.0 与项目 runtime 2.11.0 不一致。正式 bindings 保留，但构建产生的 `go.mod`/runtime shim 升级噪音定向恢复；生产验证使用独立输出名、`-skipbindings`、`-nosyncgomod` 和 `-m`，未升级项目 runtime。

## 关键决策纪要

1. 新增带参数方法，不改变旧 Wails 方法的参数数量；旧前端和旧调用继续可用。
2. 一次性目录只传给 `ExportSession`，Store 的恢复根不变，也不写回题库偏好。
3. 无效名称、目录、分段状态和模型配置全部在 `SealCurrent` 前验证。
4. 组织稿文件名使用题库名称；显式关闭模型整理时只生成 `原始记录.md` 和 `assets/`。
5. 所有真实模型调用仍只发生在用户提交且本次 `AutoOrganize=true` 后；开发验证未调用真实模型。
6. Go Promise resolve 只表示后台导出已接受；最终成功或失败继续通过题库状态事件、`message` 与 `lastExport` 呈现。
7. 当前段和历史段共用根级单例弹窗；历史重导出不会切换当前可写 Session。

## 遗留未完成项

- Windows WebView2 中的弹窗焦点、滚动、目录选择器和窄屏布局仍需人工验收。
- 真实模型整理只能由用户自愿开启后人工验证；自动验证未上传用户截图或答案，也未消耗真实 API 额度。
- Windows race 测试仍受环境启动错误 `0xc0000139` 阻塞，本 Change 不声称 race 通过。
- `OrganizeQuestionBank` 的旧异步路径、provider 获取方式、坏 Session 隔离和大量图片下的 I/O/内存优化属于后续独立 Change，不在本次收尾阶段扩展。
