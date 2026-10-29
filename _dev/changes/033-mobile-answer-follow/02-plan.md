# Change 033：实施方案

## 方案概述

保持既有 Mobile Snapshot/SSE 契约不变，只调整嵌入式页面。快捷键卡片使用原生 `details/summary` 提供默认折叠的紧凑入口；答案渲染增加仅用于“跟随最新”的下一帧滚底辅助函数；现有远程滚动控制的比例从 0.7 调整为 0.4。

## 分步实施

1. 先扩展 `pkg/mobile/web_dom_test.mjs`，增加折叠结构、答案自动滚底、历史浏览不跟随和 40% 滚动幅度断言，运行并确认红灯。
2. 修改 `pkg/mobile/web/index.html` 的快捷键 HTML/CSS 和 `renderHints`，使提示默认折叠并显示数量。
3. 增加答案自动滚底调度；只在 `streaming`/`done` 的最新答案路径调用，历史选择不调用。
4. 把 `applyRealtimeControl` 的距离比例改为 0.4，保留去重、平滑滚动和 epoch 失效逻辑。
5. 运行聚焦测试、Mobile/Frontend 回归、Frontend build、Go 全量测试及构建，并记录真实结果。

## 业务源码白名单

- `pkg/mobile/web/index.html`
- `pkg/mobile/web_dom_test.mjs`
- `pkg/mobile/bridge_server_test.go`（仅同步 `TestIndexHTML` 的滚动比例完整性哨兵）

不得越过白名单修改其他 Go 后端、Vue 前端、Wails bindings、依赖或配置。

## 交互细节

- 快捷键摘要在有 hints 时显示“答题端快捷键（N）”，默认关闭；无 hints 时整个区域隐藏。
- 展开区域保留“以下按键在答题电脑使用，不是浏览器快捷键”和现有网格。
- 自动滚底使用下一动画帧，确保答案 DOM 和卡片布局已完成；滚动目标是当前文档底部，行为为 `smooth`。
- 自动跟随只由实时最新答案渲染触发。`selectHistory` 继续只切换历史内容，不触发滚底；选择历史期间到达的新帧也不滚动。
- 清屏/换会话递增 `renderEpoch`，使已排队的答案滚底与控制滚动失效。

## TDD 与验证

- 红灯：`node pkg/mobile/web_dom_test.mjs`
- 聚焦绿灯：`node pkg/mobile/web_dom_test.mjs`
- Mobile Go：`go test ./pkg/mobile -count=1`
- 全量：`go test ./... -count=1`、`go vet ./...`、`go build ./...`
- 前端回归：运行现有 `frontend/*_test.mjs`
- 前端构建：`npm --prefix frontend run build`
- 差异检查：`git diff --check`
- GUI：电脑浏览器确认折叠空间、流式/完成滚底、历史浏览和键盘滚动体感；若未执行则标记 `NEEDS_USER`。

## 风险与对策

| 风险 | 对策 |
|---|---|
| 高频流式帧排队多个滚动 | 复用单一待执行标记，每帧最多保留一个答案滚底回调 |
| 清屏后旧回调改变位置 | 回调捕获 `renderEpoch`，不一致时放弃 |
| 用户查看历史时被拉回 | 自动滚动调用仅放在实时最新答案的 `streaming`/`done` 分支，不放入通用 `showAnswer` |
| 自动滚底与远程控制混淆 | 两条路径分别测试；远程控制继续由独立 `controlSeq` 驱动 |

## 回滚

恢复原快捷键卡片结构和样式，删除答案滚底辅助函数与调用点，将滚动比例恢复为 0.7，并移除对应测试断言。回滚不涉及数据迁移或配置恢复。
