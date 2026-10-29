# Change 024 总结与注意事项

日期：2026-10-25
状态：普通自动验证、固定 Wails v2.11.0 bindings 与独立验证构建完成；GUI 与真实 provider NEEDS_USER；未提交、未推送、未发布。

## 已完成

- 修复“全部整理失败仍显示准备完成”的状态误判；
- Store 在创建目录前拒绝零题整理稿；
- 全失败、部分失败、待处理、全部成功和无可整理记录拥有独立状态；
- 支持只重试失败记录，成功项与纯 pending 项不重复调用模型；
- 部分导出需要显式确认，RawOnly 始终可用；
- 保存稳定且脱敏的失败分类，不保存 provider 原始错误或无效响应正文；
- 对完整单一 JSON 围栏和 BOM 做有限兼容，其他严格校验保持不变；
- 创建 TCL `_02` 离线恢复稿，并验证旧数据、原稿和 25 张截图未变化；
- Go 聚焦/全量测试、vet、build、八份前端回归、Vite 构建和 diff-check 均通过。

## 证据边界

旧 TCL 的 25 条失败只保留了统一 `OrganizeError`，没有保存当时 provider 错误、空响应或具体解析阶段。因此不能把网络、额度、非流式接口或某一种 JSON 偏差写成已确认根因。Change 024 修复的是可确认的状态机、恢复操作、诊断留存和导出保护缺陷。

TCL 恢复稿属于离线证据整理，不是重新调用模型后的官方结果。所有答案只称为“参考答案”“原模型建议”或“待核对内容”。

## Wails 生成与验证程序

用户明确授权后，固定 Wails CLI v2.11.0 已核验并完成生产构建；PATH 中的 v2.16.0 未使用。

- bindings 已包含新重试 API 和 DTO 字段；
- Wails Windows/amd64 集成构建通过；
- 独立验证程序为 `build/bin/Q-Solver-change024-organize-failure-recovery-validation.exe`；
- SHA-256 为 `1953484b5fb42576f2db013691bdc6190f7a60ce7e1aabbdadad539b3f35d274`；
- 构建前后既有 EXE 和四个依赖文件哈希不变。

## 后续人工验证重点

1. 制造全失败状态，确认不能导出整理稿，可只重试失败项或改为仅原稿。
2. 制造部分成功状态，确认显示成功数、失败数和题目数；普通提交被拒绝，显式按钮可导出已有结果。
3. 确认失败重试不覆盖成功项，成功后错误提示清除。
4. 确认 RawOnly 只影响当前弹窗，不修改长期偏好。
5. 用户自愿使用真实 provider 时，确认失败分类文案不泄露第三方错误正文或配置。
6. 人工复核 TCL `_02` 中标记为待核对的语言第 4 题、数字第 5/7 题和全部图形题。

## 授权边界

Change 024 的实施批准不包含 commit、push、Tag 或 Release。当前工作树保持未暂存、未提交状态。

## 2026-10-29 整合提交状态

- 用户批准 Change 024～031 一次性跨 Change 整合。
- 功能/测试提交：`2a8a506636496a1eb6c38662c6ef6eaccc1fe958`（Change 030 无业务源码，仅归档治理记录）。
- 本 Change 的治理档案纳入 `chore: archive changes 024-031 and define artifact lifecycle`。
- 本轮全量自动验证通过；原有 `NEEDS_USER` 与 race `BLOCKED` 项保持，不因提交自动关闭。
- 不创建 Tag、GitHub Release 或新 RC。
