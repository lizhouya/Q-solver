# 验证记录

环境：Windows 11，Go 1.27.1 / windows-amd64，Node.js 24.14.0，npm 11.9.0，Wails CLI 2.16.0；项目 Wails 模块仍为 2.11.0。日期：2026-10-18。

## 自动检查（迁移前）

| 命令 / 检查 | 结果 | 说明 |
|---|---|---|
| 初次 `go test ./pkg/questionbank -count=1` | 预期 FAIL | 尚无 Store/NewStore/ExportOptions 等实现，编译失败；随后实现后通过。 |
| `go test ./pkg/questionbank ./pkg/solution ./app -count=1` | PASS，退出 0 | 多次复测通过。 |
| 输出根目录消失回归 | 初次 FAIL，修复后 PASS | 不再静默重建旧输出根目录，内存记录保留可重试。 |
| `go test ./...` | PASS，退出 0 | 全部包通过；本次未出现标准测试 vet 告警。 |
| `go build ./...` | PASS，退出 0 | Go 全量编译成功。 |
| `wails generate module` | PASS，退出 0 | 新增接口 bindings；DTO 时间字段改为 RFC3339 字符串后不再有 time 类型查找警告。 |
| `npm --prefix frontend run build` | PASS，退出 0 | Vite 提示 chunk 大于 500 kB，非构建失败。 |
| `wails build` | PASS，退出 0 | Windows exe 构建成功；CLI 与模块版本不同，出现已知生成噪音。 |
| `wails build -skipbindings -m -nosyncgomod` | PASS，退出 0 | 保留项目 runtime 与依赖后再次构建成功。 |
| 依赖哈希 | PASS | go.mod、go.sum、frontend/package.json、package-lock.json、wails.json 与构建前完全一致。 |
| `go test -race ./pkg/questionbank ./app -count=1` | FAIL / BLOCKED，退出 1 | 两个测试二进制都以 `exit status 0xc0000139` 退出，没有可用 race 结果。 |
| `go test -race ./pkg/questionbank -run '^$' -count=1` | FAIL / BLOCKED，退出 1 | 不运行任何用例也以 `0xc0000139` 退出，属于测试二进制启动阶段阻塞；没有修改系统工具链来绕过。 |

## 覆盖内容

- 12 次提交、共 36 张 PNG/JPEG 全部导出；原答案以最终响应替代部分流；文件夹可整体搬移。
- 重启后未结束场次暂停、处理中记录标为中断；取消/迟到回调不覆盖下一题；多场次隔离。
- 写盘失败保留 ID、截图与内存答案，恢复可写后重试；无效图像、目录穿越、Windows 名称校验。
- 原稿离线导出、重复导出不覆盖；结构化 JSON、多题来源校验、非法返回、取消以及跳过已成功整理的记录。
- 确定性编排测试确认终态落盘先于输出回调，并核查 desktop/mobile/dual 路由决策；不等同于真实手机 SSE 和 GUI 联调。
- 原始记录没有 API Key、Token、简历和完整配置；新模型请求只含整理规范与本条截图、原答案。

## 必要人工验收（均 NEEDS_USER）

1. Windows WebView2 面板布局、键盘/鼠标操作、目录对话框、重启恢复。
2. 托盘及系统关闭窗口时的未保存/处理中提醒。
3. desktop/mobile/dual 真机连续做题、取消、整场导出与图片显示。
4. 用户明确确认后真实模型整理，核对题干、分类及原答案保留。
5. macOS 未验证，不宣称跨平台通过。

## 结论

普通自动检查通过；race 被运行环境阻塞，人工验收未执行。不得 Close。迁移后验证将在本文件继续记录。

## 最终复核补充

- Change 010/011 七份档案完整，本次本地 Markdown 链接全部有效。
- `git diff --check` 已通过；暂存区为空；现有用户改动保留。
- 迁移受阻并完整回退后，再次 `go test ./...`、`go build ./...` 均退出 0。
- 恢复项目原 runtime 后，最终 `npm --prefix frontend run build` 退出 0，仍有非阻断的大 chunk 警告。
- 新增显式输出目录/仓库隔离保护测试；最后一次 `go test ./...`、`go build ./...`、`wails build -s -skipbindings -m -nosyncgomod` 全部退出 0；其中 `-s` 使用此前已验证的最新前端产物。

最终构建后已复核并定向保留原 runtime，五份依赖/配置文件哈希不变。当前仅有已说明的预期未提交修改，不称为 clean；没有 commit 或 push。

## 迁移后新路径复核（2026-10-19）

- 从 `E:\DATA\求职_AI\求职笔试\Q-Solver` 运行 `go test ./pkg/questionbank ./pkg/solution ./app -count=1`、`go test ./...`、`go build ./...`、`npm --prefix frontend run build` 和标准 `wails build`，均退出 0。
- Wails CLI 2.16.0 生成噪音已按构建前 SHA-256 定向恢复，项目模块仍为 v2.11.0，依赖未升级；新 EXE 构建产物保留。
- race 聚焦检查及零用例启动探针仍以 `0xc0000139` 失败，状态保持 `BLOCKED`；没有把普通测试通过当作 race 通过。
- 用户配置和题库偏好仍使用系统用户配置目录。旧根 `.git` 尚存，因此计划输出目录 `E:\DATA\求职_AI\输出` 当前仍被仓库边界保护拒绝；没有修改代码绕过保护。
- Windows GUI、手机端、退出弹窗和真实模型整理仍为 `NEEDS_USER`。Change 010 不 Close。

### 旧副本清理后的输出根复核

- 用户授权清理旧副本后，`E:\DATA\求职_AI\.git` 已移除；计划输出目录的祖先链不再包含 Git 仓库。
- `E:\DATA\求职_AI\输出` 已通过本轮临时文件创建/删除探针，可作为题库输出根目录。该探针没有生成场次或调用模型，GUI 中的目录选择与真实导出仍属 `NEEDS_USER`。

## 迁移完成后的独立复验（2026-10-19，本轮）

- 在新项目根目录重新非缓存运行聚焦测试、全量 Go 测试及 Go/前端/Wails 构建，均退出 0；不是复用旧结果。
- 新增执行 `go vet ./...`，退出 1：pkg/shortcut/shortcut_windows.go 第 122、155 行的 unsafe.Pointer 告警。该文件与 HEAD 及迁移前基线一致，为既有实现问题；本轮没有擅自修复或禁用检查。
- race 聚焦及零用例探针仍以 0xc0000139 退出，保持 BLOCKED。
- 迁移前 10,655 项文件没有缺失，119 项业务源码/依赖/配置全部哈希一致，Git 索引条目和 11 条 refs 保持一致。
- 通过直接调用 NewStore 的只读探针，实际输出目录已被模块接受，源码目录继续被拒绝。没有创建场次或调用真实模型。
- 本轮 Wails 造成的 5 项生成/格式噪音已恢复至构建前备份，保留所有既有未提交修改；11 项基线文件最终哈希一致。
- GUI、手机、退出弹窗及真实模型整理仍为 NEEDS_USER；未暂存、未提交、未推送，不 Close。
- 详见 [独立复核报告](../../audits/010-011-post-migration-check.md)。
