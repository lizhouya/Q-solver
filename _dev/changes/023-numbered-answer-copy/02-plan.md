# Change 023 实施方案

日期：2026-10-25
状态：用户已确认双模式复制方案，按小范围增量实施。

## 实现范围与规则

- frontend/src/utils/clipboard.js：新增 formatNumberedAnswers(items, copyText)，仅在非空纯答案、items 全部 ready 且 values 按行拼接严格等于 copyText 时返回 `label + '. ' + value`，各题换行；不规范化标签、不重新编号、不解析正文。
- frontend/src/components/ActionBar.vue：新增复制题号＋答案按钮；复用 performCopy/clipboard helper，区分三个成功反馈；切换 answerItems 也使旧异步反馈失效。
- pkg/mobile/web/index.html：为相同结构化项增加同样的纯展示格式与按钮；复用现有剪贴板执行链，两个按钮共同忙碌、各自按可用内容恢复；clear、stream、history 保持原绑定。
- 测试：frontend/clipboard_regression_test.mjs、frontend/numbered_copy_regression_test.mjs、pkg/mobile/web_dom_test.mjs；先 RED 后 GREEN，覆盖非连续编号、章节号、待核对、字段不匹配、历史、复制失败与模式不回退。
- README.md/README_EN.md：补充新入口说明。
- 本 Change 的 00～06、审计及 INDEX：真实记录，不预写通过。

## 验证与交付

全量 Go test/vet/build、全部前端回归、Mobile DOM/SSE、Vite、固定 Wails v2.11.0（go1.25.13）独立构建。前端增量无需重生成 bindings，使用 -skipbindings -clean=false -nosyncgomod -m。
输出 Q-Solver-change023-numbered-answer-copy-validation.exe，复制到 build/bin/archive/validation 并追加 SHA256SUMS；旧 EXE、bindings、runtime、依赖先记录哈希并核验不变，不覆盖 Change 022。GUI NEEDS_USER；本次无 Go 并发逻辑变化，既有 race BLOCKED 不宣称已解决。

## 回滚边界

仅撤销上述格式化与新按钮/测试/说明增量；纯答案复制、全文复制、题库与旧产物保持不变。未经授权不进行 Git 写操作。
