# Change 032：实施记录

## TDD 红灯

先扩展 `frontend/model_layout_regression_test.mjs`，当前实现首次运行失败：

```text
AssertionError: model dropdown must stop wheel propagation
```

证明测试覆盖了实际缺口，而不是在既有实现上直接通过。

## 实现

### ModelSelect

- 新增 `active` prop，在模型页隐藏或设置窗口关闭时关闭 Teleport 下拉。
- 新增 `open-change` 事件。
- 新增 `setOpen(value)`，统一处理聚焦、展开按钮、输入、选择、点击外部、失活和卸载路径。
- 下拉 portal 增加 `@wheel.stop` 和 `@touchmove.stop`。
- 下拉样式增加 `overscroll-behavior: contain` 和 `touch-action: pan-y`。

### SettingsModal

- 新增 `modelDropdownOpen` 状态。
- 模型页绑定 `model-dropdown-open` class。
- 接收 `ModelSelect` 的 `open-change`。
- 下拉打开时将外层 `.model-tab` 的 `overflow-y` 设为 `hidden`。
- 设置窗口关闭或切换离开模型页时主动复位锁定。

### 回归测试

新增静态契约覆盖：

- wheel/touchmove 隔离；
- overscroll 边界隔离；
- 原生纵向触摸滚动；
- open-change 状态通信；
- 子组件失活关闭；
- 外层锁定 class；
- 设置窗口/页签复位路径。

补强测试时曾出现一次断言字符串引号语法错误，已立即修正为模板字符串；修正后聚焦测试和全部 14 份前端回归通过。该错误仅存在于测试断言，不影响组件实现或候选应用代码。

## 构建产物

- 当前稳定版保持不变。
- 测试前稳定版复制到 `build/bin/testing/current/Q-Solver-stable-before-test.exe`。
- 候选生成于 `build/bin/testing/current/Q-Solver-candidate-change032.exe`。
- 使用 Wails CLI v2.16.0 保护参数构建，候选实际链接 Wails runtime v2.11.0。
- CLI 生成的未使用 Notification runtime 噪音已精确清除，`frontend/wailsjs` 恢复构建前哈希。

## 人工验收与晋升

- 用户运行 `Q-Solver-candidate-change032.exe` 完成真实界面验证，并明确反馈“测试通过”。
- 候选关闭后，旧根目录稳定版先复制归档并可逆重命名保留，再将候选复制为根目录 `Q-Solver.exe`。
- 根目录哈希核对通过后，对晋升版本执行独立启动验证：8 秒存活、WebView2 正常、无 Node/Vite、正常关闭、退出码 0、无残留进程。
- 测试前稳定版、旧根目录原件、候选规范副本、候选测试区原件和测试期清单均已归档。
- `testing/current/` 已恢复为空，根目录最终只保留 `Q-Solver.exe`。
