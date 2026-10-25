# Change 023 验证记录

状态：实现、普通自动验证、独立构建与归档完成；GUI 用户已确认通过；未提交、未推送、未发布、未关闭。

## TDD

- Desktop RED：formatNumberedAnswers 尚不存在，`typeof` 为 undefined。
- Mobile RED：修正测试多行字符串夹具语法后，明确报告 `missing numbered-copy button`。
- GREEN：`node frontend/numbered_copy_regression_test.mjs`、`node frontend/clipboard_regression_test.mjs`、`node pkg/mobile/web_dom_test.mjs` 均通过。
- 格式用例明确保留 16、20、第二部分-3，不按数组索引重编号；空结果、待核对与纯答案不匹配返回空串。

## 最终验证

- `GOTOOLCHAIN=go1.25.13 go test ./... -count=1`、`go vet ./...`、`go build ./...`：PASS。
- 八份 frontend/*regression_test.mjs：PASS；Mobile DOM/SSE：PASS。
- Vite 构建：PASS；原有大包警告保留。
- Wails v2.11.0 独立生产构建与格式收尾后的最终重建：PASS（go1.25.13；未重新生成 bindings）。
- 35 项旧程序、bindings、runtime 和依赖保护哈希：全部不变。
- 本次不改变 Go 并发逻辑，未重跑 race；Change 022 的 Windows `0xc0000139` BLOCKED 状态不变，不能记为 PASS。

## 验证程序

- 路径：`build/bin/archive/validation/Q-Solver-change023-numbered-answer-copy-validation.exe`
- 大小：23,443,456 字节。
- SHA-256：`44fbb9381179b0d739003c53df1abe9c06847255447f87ff786fdbe8a711d339`。
- 构建与归档文件哈希相同；原 Change 022 与正式程序未覆盖。

## 人工验收（用户已确认通过）

正常退出旧版、启动本版，同步端重新加载后：

1. 点击“复制题号＋答案”，核对粘贴格式为实际题号如 `16. C`、`20. AB`，非连续和章节题号不被重排。
2. 点击原“复制答案”，仍仅有答案值；桌面“复制全文”仍包含完整回答。
3. 查看旧轮期间新题完成，复制保持对应旧轮；待核对时禁用，浏览器复制失败有错误提示。

没有调用真实模型、改写用户配置、删除题库/截图/导出或执行 Git 写操作。

## 2026-10-25 用户确认

用户确认本次带题号复制验收“通过”，并要求提交和推送。仅更新 Change 023 验收，不替其他 Change 或 race 作验收结论。提交范围正在核对，尚未执行提交或推送。
