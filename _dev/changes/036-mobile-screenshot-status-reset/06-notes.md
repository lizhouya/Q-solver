# 总结与注意事项 06

## 变更摘要

Change 036 已完成业务实现、普通自动验证、Wails 保护构建、候选独立启动和产物哈希校验：

- Mobile Snapshot 新增可重放的待发送截图数量与上限，不包含截图内容或敏感配置。
- 同步端在存在待发送截图时显示底部固定 Dock，包含 `N/3` 和实际截图、删除、发送快捷键；数量归零后隐藏，不占用解题答案空间。
- 截图缓冲成功增删、清空和发送消费后，Mobile 数量实时同步。
- “结束本段”“清空本段”“开始下一套”成功后，同步端答案与浏览器内历史清空；失败路径不改变 Mobile。
- 单独清除待发送截图只将数量归零，不删除同步端既有答案。
- Reset 采用可靠 idle 屏障，排空 subscriber 旧积压且不推进持久化 `clearSeq`。

## 经验与踩坑

| 现象 | 原因 | 结论 |
|---|---|---|
| 同步端有快捷键却不直观 | 全部提示默认折叠 | 截图工作流需单独紧凑展示，同时保留完整折叠卡 |
| 本地清理后远端历史残留 | idle 快照被页面忽略且清理入口未调用 Reset | 成功清理需发布新业务 idle，页面需清答案和内存历史 |
| 普通非阻塞广播可能漏掉 Reset | subscriber 队列满时中间 idle 会被丢弃 | Reset 必须排空旧积压并强制放入唯一屏障 |
| 持久清屏后下一题可能被旧序号规则拒绝 | 新 `clearSeq` 业务序号可从低值重新开始 | 边界 idle 只负责清屏，不占用新业务序号空间 |
| ClearDisplay 不适合普通本地清理 | 它是持久化、跨重连的用户主动清屏边界 | 本地业务清理使用不改 `clearSeq` 的 Bridge Reset |
| 截图状态可能与答案生命周期不同步 | 截图缓冲和解题显示是两个独立状态 | 单独清截图不得顺带删除答案；开始下一套才同时归零截图并清业务显示 |

## 当前状态

- 自动测试、静态检查、构建、候选启动和哈希校验：PASS。
- `go test -race`：当前 Windows 环境启动错误 `0xc0000139`，BLOCKED。
- Windows 双端 GUI：用户最终确认底部 Dock 回修候选验证通过。
- 候选：`build/bin/testing/current/Q-Solver-candidate-change036.exe`。
- 候选 SHA-256：`8E9543C9DA02B56C4C85099BAEA0D2621B47234DA218280C5D64DA0CEB1C80F2`。
- 功能与测试提交：`d90bbe3`；治理记录由配套治理提交归档。
- 用户已授权推送 `origin/dev/qsolver-custom`；根稳定版未覆盖，未创建 Tag/Release。

## 遗留项

1. 使用候选完成双端 GUI 人工验收，重点检查自定义快捷键、0/3～3/3、单独清截图保留答案、三个清理入口和 SSE 重连。
2. 用户验收通过后，重新执行最终 Git 预检、`git diff --check` 和提交边界审计。
3. 如用户明确授权，再按功能/测试与治理记录双提交；commit 授权不隐含 push 或稳定版晋升授权。
4. race 环境阻塞在获得可运行环境前保持如实记录，不标记 PASS。

## 参考

- `_dev/audits/036-mobile-screenshot-status-reset.md`
- `pkg/mobile/state.go`
- `pkg/mobile/bridge.go`
- `pkg/mobile/web/index.html`
- `app/mobile_screenshot_reset_test.go`
- `pkg/mobile/screenshot_state_test.go`
