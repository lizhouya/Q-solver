# Change 032：实施方案

## 状态通信

`ModelSelect` 新增 `open-change` 事件，并用统一的 `setOpen(value)` 更新内部状态和通知父组件。以下路径必须通过该函数：

- 输入框聚焦打开；
- 展开按钮切换；
- 手动输入打开；
- 选择模型关闭；
- 点击外部关闭；
- 组件卸载关闭。

`SettingsModal` 使用 `modelDropdownOpen` 保存状态，并在模型页根元素绑定锁定 class。关闭设置或切换离开模型页时主动复位，防止 Teleport 状态残留。

## 滚动隔离

- 下拉 portal 增加 `@wheel.stop` 和 `@touchmove.stop`，隔离输入事件冒泡。
- `.model-dropdown-portal` 增加 `overscroll-behavior: contain` 和 `touch-action: pan-y`，允许列表自身纵向滚动但阻断边界滚动链。
- `.model-tab.model-dropdown-open` 使用 `overflow-y: hidden` 锁定外层滚动位置。

不使用 `preventDefault` 手工计算滚动距离，保留浏览器原生滚轮、触控板和惯性滚动体验。

## TDD

先扩展 `frontend/model_layout_regression_test.mjs`，断言：

1. portal 存在 wheel/touchmove 隔离；
2. portal 使用 `overscroll-behavior: contain`；
3. `ModelSelect` 声明并发出 `open-change`；
4. `SettingsModal` 绑定 `modelDropdownOpen` 与锁定 class；
5. 锁定 class 设置 `overflow-y: hidden`；
6. 页签/窗口关闭路径会复位状态。

先运行测试确认失败，再实施最小修改并确认通过。

## 验证

- `node frontend/model_layout_regression_test.mjs`
- 全部 `frontend/*_test.mjs`
- `npm --prefix frontend run build`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./...`
- Wails 独立候选构建和启动/退出验证
- 构建前后依赖、bindings/runtime、Git index 保护检查

## 产物

- 测试前稳定版：`build/bin/testing/current/Q-Solver-stable-before-test.exe`
- 候选：`build/bin/testing/current/Q-Solver-candidate-change032.exe`
- 验证通过前不替换根目录 `Q-Solver.exe`。

## 回滚

删除 `open-change` 状态通信、portal 事件隔离、overscroll 样式、外层锁定 class 和对应回归断言即可恢复基线。二进制失败时不动根目录稳定版，并清理或归档候选。
