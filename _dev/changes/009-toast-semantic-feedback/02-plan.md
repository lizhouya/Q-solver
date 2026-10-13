# 实施方案 — 02 · 009-toast-semantic-feedback

## 1. 方案概述

ClickThrough 的三个结果改为在同一 `toast` 事件中发送 `map[string]string{"text": ..., "type": ...}`。Frontend Toast handler 先判断 payload：字符串沿用 `ui.showToast(payload)`；合法结构化对象调用 `ui.showToast(payload.text, payload.type)`。不修改 Store、template 或 CSS。

## 2. 分步实施计划

| 步骤 | 操作 | 涉及文件 | 完成标志 |
|---|---|---|---|
| 1 | ClickThrough 发送结构化 Toast | `pkg/shortcut/service.go` | success/info/error 类型准确 |
| 2 | 扩展测试断言文案与 type | `pkg/shortcut/service_test.go` | 三种结果和自定义绑定测试通过 |
| 3 | Frontend handler 双格式兼容 | `frontend/src/App.vue` | 字符串与对象均可显示 |
| 4 | 运行 Go/Frontend/Wails 验证及边界审计 | 无额外业务文件 | 构建测试通过，无生成污染 |
| 5 | 补齐 03～06 和 INDEX | Change 009 文档 | 停在 commit boundary |

## 3. 涉及业务源码的改动清单

仅允许修改：

1. `pkg/shortcut/service.go`
   - ClickThrough 开启：`type=success`。
   - ClickThrough 关闭：`type=info`。
   - ClickThrough 失败：`type=error`。
2. `pkg/shortcut/service_test.go`
   - `assertSingleToast` 改为断言结构化 payload 的 `text/type`。
   - 现有四个 ClickThrough 测试传入期望类型。
3. `frontend/src/App.vue`
   - 仅修改 `on('toast')` handler，兼容字符串和结构化 payload。
   - 禁止修改 template 与 scoped style。

## 4. 测试与验证计划

- 聚焦测试：`go test ./pkg/shortcut -run ClickThrough -count=1`。
- 重复测试：聚焦测试 `-count=50`。
- 全量测试：`go test ./... -count=1`。
- 构建：`go build ./...`、`npm --prefix frontend run build`。
- Windows 打包：`wails build -debug -windowsconsole`；审计并精确恢复已知 v2.16 生成噪音。
- 静态核对：Change 008 template/CSS 无 009 差异；旧字符串发送点仍存在；Frontend 字符串分支明确保留。
- `git diff --check`、status、name-only、staging 审计。

## 5. 回滚预案

恢复 ClickThrough 的三个字符串 payload，恢复测试字符串断言，将 Frontend handler 恢复为 `ui.showToast(msg)`。不回滚 Change 007/008。

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 结构化 payload 字段拼写错误 | 文案或 type 丢失 | Go 单测精确断言 map 的 `text/type` |
| 旧字符串调用失效 | 既有错误提示异常 | Frontend 先做 `typeof payload === 'string'` 分支 |
| 非法对象导致空 Toast | 运行时异常或空消息 | 仅识别带字符串 `text` 的对象，其他 payload 按旧兼容路径处理 |
| 误改 Change 008 视觉层 | 范围扩张 | 不修改 template、CSS 或 `Icon.vue`，最终 diff 核对 |
