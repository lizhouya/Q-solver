# Change 019 实现记录

状态：业务实现与普通自动验证完成；Windows GUI NEEDS_USER；真实模型效果 NEEDS_USER；race BLOCKED；验证程序归档复制受权限阻塞；未提交、未推送、未发布、未关闭。

## 时间线

- 2026-10-23：完成事前探索、方案确认与 00/01/02 归档。
- 2026-10-23：先建立截图折叠和 Prompt 契约回归，确认旧实现按预期失败。
- 2026-10-23：实现主界面逐图默认收起、独立展开/隐藏、点击图片收起和按需读取。
- 2026-10-23：统一系统行为增加当前提交多图多题识别、跨图合并、重叠去重、题号映射、待核对和最终答案汇总规则。
- 2026-10-23：完成专项、全量普通验证和 Wails v2.11.0 独立验证构建；race 仍因 Windows `0xc0000139` 无法启动。
- 2026-10-23：构建后保护哈希全部通过；归档复制与 `SHA256SUMS` 更新的组合操作被 Claude Code auto mode classifier 拒绝，未绕过该拒绝。

## 截图折叠

- `RoundScreenshots.vue` 使用组件本地 `expandedImages` Set；初始为空，因此当前轮和历史轮截图均默认收起。
- 每张截图按 `image.index` 独立切换；主按钮使用 `aria-expanded`，已显示图片位于可聚焦按钮内，点击、Enter 或 Space 均可收起。
- `scopeKey` 由 Session 标识、conversation epoch 与 Record 标识组成；作用域变化时只清空展开集合，不清空图片缓存。
- 展开未缓存且未处于加载中的历史图片时，继续发出既有 `load(round, image)`；已缓存或加载中的图片不重复请求。
- 加载期间允许收起；迟到成功响应只写既有缓存，不会反向修改组件展开状态。
- 组件不写 `image.data`、`image.loading`、`image.error`、`image.requestVersion` 或 `round.userScreenshot`，图片缓存、竞态隔离和重试仍由既有 solution Store 负责。

## 多图多题输出契约

- 仅修改 `GetSystemBehaviorPrompt()`，新增独立 `<QuestionAnsweringRules>`；没有修改 Provider、`Solver.Solve`、流式事件或截图请求构造。
- 解题任务只处理当前 user 消息中的原题；历史轮只作上下文，不重复汇总。
- 多张截图按题号、题干、选项和上下文合并跨图同题；重叠题只回答一次。
- 保留原题号；无题号按首次出现顺序编号；无法可靠对应或内容冲突时标记“待核对”，不得猜测。
- 当前提交有两题及以上时，在逐题正文后追加 `## 最终答案汇总`；单题不强制重复汇总。
- 选择题使用“题号：选项”，非选择题汇总结果或结论；汇总只包含当前原题并与正文一致。
- OCR、纯翻译不强制解题或汇总；秒杀场景继续简洁；举一反三产生的变式题不混入原题汇总。

## 测试与保护

- 新增 `frontend/round_screenshots_regression_test.mjs` 和 `pkg/domain/prompts_test.go`。
- 扩展题库续接回归，锁定已缓存图片不会二次读取，也不会丢失 `userScreenshot`。
- 扩展 Solver fake provider 测试，证明三张图片按顺序进入一次 multipart user 消息，Provider 只调用一次，完整回答原样进入 solution 事件和 KeepContext assistant 历史。
- 构建后确认 `frontend/wailsjs/go/app/App.d.ts`、`App.js`、`models.ts`、`go.mod`、`go.sum` 和正式 `build/bin/Q-Solver.exe` 哈希均未变化。

## 生成物

- 使用 `_dev/changes/015-rc-release/tmp/tools/wails.exe` v2.11.0、`GOTOOLCHAIN=go1.25.13`、`-clean=false -skipbindings -nosyncgomod -m` 构建。
- 可运行验证程序：`build/bin/Q-Solver-change019-answer-summary-preview-validation.exe`。
- SHA-256：`cd4b354ef37613029d2bb0e39b9d62fd924480aefa0eafbf762ca60135d19cf7`。
- 文件大小：`23,366,656` 字节。
- 计划中的 `build/bin/archive/validation/` 复制和 LF-only `SHA256SUMS` 更新尚未执行：组合操作被权限分类器拒绝，未改用其他方式绕过。
- 未覆盖正式 `build/bin/Q-Solver.exe`，未停止或替换任何正在运行的旧验证程序。

## 与方案偏差

业务与测试方案无偏差。唯一未完成项是验证程序复制到统一归档目录及相应清单更新，原因是本地文件操作被权限分类器拒绝；源验证程序已成功生成并完成独立 SHA-256 计算。

## 实施约束

未调用真实模型，未上传测试截图，未读取或输出 API Key/Mobile Token，未删除、迁移或覆盖题库原始记录、截图和旧导出；未新增依赖、Wails API、题库字段、Mobile 协议或第二次模型调用；未执行 commit、push、Tag 或 Release。
