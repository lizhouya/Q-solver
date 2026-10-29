# Change 032：当前状态

## 现有结构

- `ModelSelect.vue` 将下拉列表 `Teleport` 到 `body`，列表自身使用 `overflow-y: auto`。
- `SettingsModal.vue` 的模型页 `.tab-pane.model-tab` 同时是外层纵向滚动容器。
- 下拉列表没有 `wheel`、`touchmove` 事件隔离，也没有 `overscroll-behavior`。
- `ModelSelect` 没有向父组件报告下拉打开状态，外层无法在打开期间锁定滚动。
- 现有 `model_layout_regression_test.mjs` 只检查模型页布局和 DomainSelector，不覆盖滚动穿透。

## 根因

Teleport 列表和外层滚动页是两个独立滚动区域。WebView2 在列表边界或连续触控板滚动时会产生滚动链；由于列表没有阻断 overscroll，外层 `.model-tab` 继续响应并改变位置。

## 风险

- 只加 `@wheel.stop` 不能完整覆盖触摸和边界滚动链。
- 只加 `overscroll-behavior` 不能保证下拉打开期间在列表外误触时外层保持固定。
- 外层锁定若没有覆盖所有关闭路径，可能残留 `overflow-y: hidden`。
- 直接操作全局 `body` 会影响整个设置窗口，不符合最小范围。

## 基线

- 分支：`dev/qsolver-custom`。
- 基线 HEAD：`39411629d6f5a0e45a4c131b9852ac908d6b2b82`。
- 工作树与暂存区为空。
- 当前根目录稳定版 SHA-256：`82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46`。
- `build/bin/testing/current/` 为空。
