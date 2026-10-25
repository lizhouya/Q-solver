# Change 019 总结与注意事项

状态：业务实现和普通自动验证完成；GUI 与真实模型效果待人工验收；race 与统一验证归档分别受环境和权限阻塞；未关闭。

## 已实现

- 当前轮和恢复历史轮的每张截图默认收起，可独立显示和隐藏。
- 点击已显示图片可再次收起；键盘可聚焦操作，提供 `aria-expanded` 与焦点轮廓。
- 折叠状态是组件本地状态，不等于释放图片缓存；切换 Session/Record 时重置显示状态但保留缓存。
- 多图多题只使用当前一次 multipart 模型请求；统一 system behavior 要求逐题识别、跨图合并、重叠去重和题号对应。
- 两道及以上原题追加“最终答案汇总”；选择题汇总选项，其他题型汇总结果或结论，模糊项标记“待核对”。
- 单题不强制重复汇总；OCR、翻译、秒杀、举一反三和 KeepContext 边界已写入 Prompt 契约。

## 已验证边界

- 没有修改 solution Store 的图片加载实现、Solver 请求链、Provider、题库 schema、Wails API、Mobile 协议或持久化格式。
- 没有新增依赖、第二次模型调用或答案后处理。
- Wails 构建没有重新生成 bindings，也没有改动 go.mod、go.sum 或正式 Q-Solver.exe。
- 普通自动验证通过；race 仍因 `0xc0000139` 标记 `BLOCKED`。

## 验证入口

- 文件：`build/bin/Q-Solver-change019-answer-summary-preview-validation.exe`
- SHA-256：`cd4b354ef37613029d2bb0e39b9d62fd924480aefa0eafbf762ca60135d19cf7`
- 大小：`23,366,656` 字节

统一 `archive/validation` 复制及 `SHA256SUMS` 更新尚未执行，原因是相关本地文件操作被权限分类器拒绝；不得声称该归档已完成。

## 人工验收重点

- 逐图默认收起、独立切换、点击图片收起。
- 加载中收起、缓存复用、错误重试和 Session/Record 隔离。
- 真实五题、多图重叠、跨图同题、混合题型和模糊题的正文与汇总一致性。
- `gen-ocr`、`gen-trans`、`gen-speed`、`gen-similar` 和 KeepContext 的场景边界。

## 注意事项

- 自动测试只能证明 Prompt 与请求链，不能证明真实模型一定正确识题。
- Prompt 输出仍受所选模型能力影响；无法可靠识别时应显示“待核对”，而不是猜测。
- 当前错误态会保留读取错误说明；收起后重新展开会走既有重试路径。
- Windows GUI 和真实模型效果未完成前，不关闭 Change 019。
- 未获得 commit、push、Tag 或 Release 授权。

## 2026-10-25 整合提交状态（最新）

用户明确批准本次将 Change 016～023 累积改动按功能/测试与治理两次提交整合，并连同既有 RC2 提交推送至 origin/dev/qsolver-custom；此授权只适用于本次。

- 功能/测试已提交：`e44bb6163f0cb1b88e2ec4b4da80a1446ddb1990`。
- 本治理档案随第二次整合提交归档；完整边界见 `../../audits/016-023-commit-boundary.md`。
- 原文“未提交/未推送”是当时记录，最新 Git 状态以分支历史和远端分支头为准；本次不创建 Tag 或 Release。
- 不扩大人工验收结论：用户明确确认 Change 016、023；其他未明确项目和既有 race BLOCKED 保留，不因提交自动关闭。
