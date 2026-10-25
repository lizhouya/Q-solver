# 验证记录 04

状态：普通自动验证 PASS；Windows GUI 人工验收待执行；race BLOCKED。

## 自动验证结果

以下命令已于 2026-10-23 在固定 `GOTOOLCHAIN=go1.25.13` 下通过：

```text
go test ./pkg/questionbank -count=1 -v                 PASS
go test ./app -run 'QuestionBank' -count=1 -v          PASS
node frontend/questionbank_history_regression_test.mjs PASS
node frontend/settings_regression_test.mjs             PASS
node frontend/mobile_store_regression_test.mjs         PASS
node pkg/mobile/web_dom_test.mjs                        PASS
go test ./... -count=1                                  PASS
go vet ./...                                            PASS
go build ./...                                          PASS
npm --prefix frontend run build                         PASS
```

前端生产构建仅有既有的 Vite chunk size warning，没有构建错误。

## 覆盖的关键边界

- 软清空仅在后端成功后重置桌面实时历史；取消和失败保持原状，且不调用任务、截图、Mobile 或模型上下文清理接口。
- 最近列表的 `total/hidden/limit`、第 21 段详情/图片拒绝、`Store.List()` 全量语义和稳定编号不受 UI 截断影响。
- 详情读取不切换 current、不修改状态或磁盘；discarded 段可读但不可写、整理或导出；详情超过 8 MiB 整体拒绝。
- A→B 详情竞态、关闭面板迟到响应、跨段图片迟到响应和同段图片并发均按预期处理。
- 路径穿越、绝对路径、反斜杠、冒号、被篡改 manifest、非普通文件、扩展名/格式不一致、20 MiB 和 80M 像素限制均通过。
- v1/v2 Session JSON 兼容，严格模型 JSON 拒绝未知字段和超长候选；整理失败不覆盖旧建议或原答案。
- RawOnly 为单阶段且零模型准备；模型模式先准备、最终本地导出不再次 prepare 或调用模型。
- 自动名称包含单位、出题方、题型；手工名称受保护，“恢复自动命名”后才重新联动。
- 整理完成刷新会保留用户已编辑的单位、出题方、题型、目录和手工名称，同时接收模型建议与准备状态。
- 归档模板不使用 `v-html`，“重新导出”阻止冒泡，准备阶段文案为“正在整理并生成建议”。
- 历史分段点击会立即定位详情；加载、成功和失败标记互斥，零提交分段显示明确空状态并隐藏重新导出。
- Change 016 的设置、Mobile store、Mobile DOM 与生成接口回归通过。

## 权限型 SKIP

`TestReadRecordImageRejectsSymlinkAndNonRegularFile` 中的符号链接子测试在当前 Windows 账户下无法创建 symlink：

```text
A required privilege is not held by the client.
```

因此该子测试按预期 SKIP。目录等非普通文件子测试及其他图片安全边界均 PASS；生产代码的 symlink 拒绝校验未放宽。

## Wails bindings 与验证构建

使用固定工具：

```text
Wails CLI v2.11.0
GOTOOLCHAIN=go1.25.13
```

执行并通过：

```text
wails.exe generate module
wails.exe build -clean=false -o Q-Solver-change017-validation.exe
```

已核对生成 bindings 同时包含 Change 017：

- `PrepareQuestionBankSegment`
- `ListRecentQuestionBankSessions`
- `GetQuestionBankSessionDetail`
- `GetQuestionBankRecordImage`

以及 Change 016：

- `ClearMobileDisplay`
- `ResetMobileOutputLink`
- `SetMobileStableLink`

最终验证程序：

```text
构建路径：build/bin/Q-Solver-change017-validation.exe
统一验证目录：build/bin/archive/validation/Q-Solver-change017-history-view-fix-validation.exe
大小：23,291,392 bytes
SHA-256：3b85644beb8a726c7138d01ce843526538b4d028aea80fd75403a0844bd338c5
```

历史浏览修复版已复制到与 Change 016 相同的 `build/bin/archive/validation/`；因旧 Change 017 验证程序正在运行而被 Windows 锁定，使用新文件名保留旧版并避免中断，并加入该目录的 `SHA256SUMS`；清单内七个文件均通过 `sha256sum -c`。`build/bin/Q-Solver.exe` 未被覆盖。Wails 再生成后已清理 `frontend/wailsjs/go/models.ts` 尾空白；`git diff --check` PASS，仅输出既有 LF/CRLF 转换提示。

## race：BLOCKED

尝试：

```text
GOTOOLCHAIN=go1.25.13 go test -race ./pkg/questionbank ./app -run 'QuestionBank' -count=1
```

结果：

```text
exit status 0xc0000139
FAIL Q-Solver/pkg/questionbank
exit status 0xc0000139
FAIL Q-Solver/app
FAIL
```

失败发生在当前 Windows race 运行环境启动阶段，状态保持 BLOCKED；普通测试不能替代 race，也不能据此标记 race PASS。

## Windows GUI 人工验收：待执行

使用 `build/bin/archive/validation/Q-Solver-change017-history-view-fix-validation.exe` 验证：

1. 清空成功后实时历史和答案区归零；取消或后端失败时不变。
2. 清空后的下一题懒创建新段，不产生零记录段。
3. ended、exported、discarded 详情和截图正确，浏览不切换后台 current。
4. A→B 快速切换不串答案或截图，关闭面板后迟到响应不重现。
5. 第 20 段可见，第 21 段只在磁盘，不自动整理、导出或删除。
6. RawOnly 只生成原稿；模型模式先生成建议，人工核对后纯本地导出。
7. 手工字段和名称在整理完成刷新后保持，“恢复自动命名”才重新联动。
8. Change 016 固定链接、清屏确认、左 `Shift+C` 和同步端行为不回退。

人工验收完成前，Change 017 不标记关闭，不提交、不推送、不发布。
