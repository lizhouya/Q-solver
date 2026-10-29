# Change 033：实现记录

## 实现结果

### 快捷键区域

- 将 `#shortcut-card` 内部改为原生 `details/summary`。
- 默认不设置 `open`，折叠状态只显示标题、有效快捷键数量和“展开”提示。
- 展开后保留原说明文字和快捷键网格。
- `renderHints` 根据实际通过校验的提示项更新数量；提示变化不主动覆盖用户当前展开状态。
- 清屏或换会话时关闭详情、清零数量并隐藏卡片。
- 不使用 `localStorage`、`sessionStorage` 或新增依赖。

### 最新答案自动跟随

- 新增 `scheduleLatestAnswerScroll`，在下一动画帧滚动到文档底部。
- 只在实时最新答案的 `streaming` 与 `done` 路径调用；`selectHistory` 和选中历史期间的新帧不调用。
- `answerScrollPending` 合并同一动画帧内的高频流式更新。
- 回调同时校验 `renderEpoch`、调度版本和 `selectedSeq`；清屏、换会话或用户切换历史后，旧回调不会改变位置。
- 自动跟随使用 `window.scrollTo`，与远程键盘控制的 `window.scrollBy` 保持独立。

### 键盘滚动幅度

- `applyRealtimeControl` 的单次距离由视口高度 70% 调整为 40%。
- 保留 `controlSeq` 去重、`requestAnimationFrame`、平滑滚动和 `renderEpoch` 失效语义。
- 800px 视口的单次滚动从 560px 降为 320px。

### 测试契约

- 扩展 `pkg/mobile/web_dom_test.mjs`，覆盖默认折叠、计数、展开状态保持、最新答案滚底、流式滚底、历史浏览不跟随、清屏失效以及 40% 控制距离。
- `pkg/mobile/bridge_server_test.go::TestIndexHTML` 原先把 `viewportHeight * 0.7` 作为实际嵌入页完整性哨兵；将其同步为 `viewportHeight * 0.4`，并加入 `shortcut-details` 标记。

## TDD 过程

1. 先加入折叠结构与滚动行为断言。
2. 首次运行按预期失败：`快捷键区域必须使用可折叠 details`。
3. 实施页面最小改动后，测试曾因 Node VM 跨 realm 对象使用 `deepStrictEqual` 报“值相同但原型不同”；改为逐字段断言，不改变业务预期。
4. Mobile Go 聚焦测试首次发现旧的 70% 页面完整性哨兵；确认是与批准行为直接相关的测试契约后，将该测试加入白名单并更新。
5. 聚焦 DOM 与 Mobile Go 测试转绿。

## 构建过程

- 使用本机 Wails CLI v2.16.0 及全部保护参数构建候选。
- `go version -m` 证明候选实际链接 Wails runtime v2.11.0。
- CLI 向 `frontend/wailsjs/runtime/runtime.js` 与 `runtime.d.ts` 注入已知 Notification API 生成噪音；按差异精确删除后，两文件及其他受保护源码哈希恢复。
- `.git/index` 文件哈希因只读 Git 检查刷新 stat cache 发生变化；`git write-tree` 与 `HEAD^{tree}` 完全一致，且 staged diff 为空，证明 index 语义未变。

## 范围偏差

原批准范围预计两个 Mobile Web 文件。聚焦 Go 测试发现 `bridge_server_test.go` 对旧滚动比例有直接完整性断言，因此将该测试文件加入白名单，仅更新一行哨兵；没有修改 Go 生产代码、SSE 协议、配置、依赖或 Wails bindings。
