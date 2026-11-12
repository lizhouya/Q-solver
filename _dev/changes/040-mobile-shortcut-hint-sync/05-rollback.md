# 回滚预案 05

## 回滚策略

采用定向还原：只恢复 Change 040 修改的 Mobile 页面与测试，不修改正式配置、Mobile Token、依赖或 Release。

## 回滚触发条件

- 同步端底部卡片顺序仍与桌面不一致。
- 五项提示导致窄屏严重遮挡或页面无法滚动到末尾。
- 满三张时键帽仍消失，或截图状态卡无法正常隐藏。
- Mobile Snapshot、SSE 重放、清屏代际或只读安全边界发生回归。

## 回滚步骤

1. 定向恢复 `pkg/mobile/web/index.html` 的 Change 040 差异。
2. 同步恢复 `pkg/mobile/web_dom_test.mjs` 和 `app/mobile_test.go` 的对应断言。
3. 重新运行 Mobile 聚焦测试和必要全量门禁。
4. 若已提交，仅在用户授权后执行 `git revert`；禁止 `git reset --hard`。
5. 若候选失败，保持根稳定版不变，并按产物政策归档有诊断价值的失败候选。

## 实际回滚记录

截至 2026-11-12 未触发回滚。普通自动门禁和用户 GUI 验收通过，功能已随 Change 041 叠加候选晋升本地稳定版；原 Change 040 候选保留在 transient 归档。若后续需要回滚，只能在用户再次授权后使用 `archive/superseded/change-041/` 中哈希通过的旧稳定版；Release 未修改。
