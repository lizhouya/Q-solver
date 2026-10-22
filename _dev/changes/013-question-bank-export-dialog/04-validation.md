# 验证记录 04

## 自动化验证

| 命令 | 结果 | 备注 |
|---|---|---|
| `go test ./pkg/questionbank ./app -count=1` | 通过 | 覆盖试卷布局、仅原稿、一次性目录、草稿、单次请求和原答案安全围栏。 |
| `go test ./... -count=1` | 通过 | 2026-10-20 前端与回归测试补强后再次全量执行。 |
| `go vet ./...` | 通过 | 未发现新的静态检查问题。 |
| `go build ./...` | 通过 | Go 全量构建通过。 |
| `npm --prefix frontend run build` | 通过 | Vite 4.5.14，123 modules transformed；仍有既有 814.31 kB 主 chunk 警告。 |
| `wails build -clean` | 失败（环境占用） | bindings 和前端编译完成，但旧 `build/bin/Q-Solver.exe` 正被占用，清理时报 `Access is denied`；未强杀用户进程。 |
| `wails build -o Q-Solver-change013.exe -skipbindings -nosyncgomod -m` | 通过 | 生产应用实际编译成功，独立验证文件随后删除；旧 `Q-Solver.exe` 未触碰。 |
| `wails build -s -o Q-Solver-change013-final.exe -skipbindings -nosyncgomod -m` | 通过 | 使用已通过 Vite 构建的最终 `dist` 再次生成 assets 并编译应用；验证文件随后删除。 |
| Wails 生成噪音检查 | 通过 | `go.mod`、`go.sum` 和 v2.11 runtime shim 最终无差异；仅保留本次需要的 App bindings/models。 |
| `git diff --check` | 通过 | 清理 Wails 生成的 `models.ts` 行尾空白后通过；仅有 Git 的 LF/CRLF 提示。 |

## 已验证契约

- [x] 新命名组织稿采用 `<题库名称>.md`，包含“试题”和“参考答案与解析”两部分。
- [x] 组织稿不含 Markdown 图片语法或 `assets/` 引用；原始记录继续包含全部截图引用。
- [x] 完整原答案使用自适应代码围栏安全展示，内容本身不被改写。
- [x] 关闭本次模型整理只生成 `原始记录.md` 和 `assets/`，不生成空组织稿。
- [x] 一次性目录只改变最终输出位置，不移动恢复 manifest，也不创建目标目录下的 `_记录`。
- [x] 导出草稿只读取本地 Session/Question 数据，不调用模型。
- [x] 无单位默认名称使用当日稳定顺序编号，并显式返回 `NumberedPrefix`；同一分段重复读取名称不变。
- [x] 非法一次性目录在封存前失败，当前段仍保持 Active。
- [x] 本次目录不会写回长期偏好。
- [x] 旧长期单位不再暴露为 UI 默认值，也不再写回偏好。
- [x] 旧无参导出入口继续可用，并遵守关闭整理时只导出原稿的新语义；已导出历史段仍可显式重导出。
- [x] 显式 `AutoOrganize=false` 覆盖默认开启状态，开发测试未调用真实模型。
- [x] 纯本地导出期间状态为 `operation="exporting"`，完成后 `organizing` 和 `operation` 均清空。
- [x] Wails 正式 bindings 包含 `GetQuestionBankExportDraft`、`SelectQuestionBankExportDirectory` 和 `ExportQuestionBankSegment`。
- [x] Wails 生产应用使用备用输出名完成真实编译，不以 `go build` 或前端构建冒充桌面应用构建。

## 待验证与人工验收

- [ ] 点击答案旁“导出题库”只打开弹窗，取消无副作用。
- [ ] 单位、题型联动默认名称；用户手动改名后不被自动覆盖。
- [ ] 当前段与历史段均使用同一弹窗和一次性目录选择器。
- [ ] 本次目录修改不会改变设置中的默认目录。
- [ ] 顶部状态正确区分待记录、当前提交数和各终态。
- [ ] 题库面板不再重复显示设置目录和当前段主导出入口。
- [ ] “结束本段”在 ActionBar 与题库面板可见且语义一致。
- [ ] Windows WebView2 中弹窗布局、滚动、焦点和窄屏显示正常。
- [ ] 用户自愿启用后的真实模型整理结果符合参考答案提炼规则。

## 已知限制

- `go test -race` 在当前 Windows 环境仍因进程启动错误 `0xc0000139` 无法执行，状态为 `BLOCKED`，不能记为通过。
- Vite 生产构建有既有主 chunk 大于 500 kB 的警告；本 Change 未新增依赖，也未在收尾阶段扩展为代码拆分项目。
- Wails CLI 2.16.0 会生成高于项目 runtime 2.11.0 的 shim 内容；已定向恢复，未升级 runtime。

## 当前结论

后端、前端、静态检查、Go 构建和 Wails 生产应用构建均已通过；真实模型未被调用。Windows GUI、目录选择器与真实模型整理仍为 `NEEDS_USER`，race 为 `BLOCKED`，因此 Change 013 继续保持“进行中”，不能标记关闭或可发布。
