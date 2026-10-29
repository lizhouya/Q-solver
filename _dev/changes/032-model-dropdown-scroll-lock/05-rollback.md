# Change 032：回滚方案

## 源码回滚

如滚动锁定引发新问题，仅回滚以下内容：

- `ModelSelect.vue` 的 `active` prop、`open-change`、`setOpen`、wheel/touchmove 和 overscroll 样式；
- `SettingsModal.vue` 的 `modelDropdownOpen`、active 绑定和锁定 class；
- `model_layout_regression_test.mjs` 的对应契约断言。

不涉及 Go、CCR、配置、题库、依赖或 Wails bindings。

## 产物回滚

Change 032 已晋升。旧稳定版位于：

- `build/bin/archive/superseded/change-032/Q-Solver-before-change032.exe`
- `build/bin/archive/superseded/change-032/Q-Solver-pre-change032-root-original.exe`

旧稳定版 SHA-256：`82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46`。

回滚时：

1. 关闭所有 Q-Solver 实例；
2. 在 superseded 目录运行 `sha256sum -c SHA256SUMS`；
3. 将当前 Change 032 最终版移入失败/回滚归档，不直接删除；
4. 将旧稳定版复制为根目录临时文件并核对哈希；
5. 替换 `build/bin/Q-Solver.exe`，再执行独立启动验证。
