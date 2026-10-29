# Change 032：总结与遗留项

## 已完成

- 修复模型下拉与外层模型设置页的滚动链穿透。
- 下拉打开期间锁定外层，列表保留原生滚轮、触控板和触摸纵向滚动。
- 所有常规关闭路径恢复外层滚动。
- TDD、14 份前端回归、Vite、Go 测试/vet/build 和 Wails 候选构建通过。

## 人工验收与最终版

- 用户运行 Change 032 候选并明确反馈测试通过。
- 候选已晋升为 `build/bin/Q-Solver.exe`。
- 最终大小：42,331,648 字节。
- 最终 SHA-256：`C1F1873F054520F0F45FADFCCBCC087056D1556381DD1EFC66D4CD3748085F7B`。
- 晋升后独立启动、WebView2、正常退出和残留进程验证通过。
- `testing/current/` 已清空，Release 资产未变化。

## 提交状态

- 功能/测试提交：`0d8b61b08ed3bd56708771ab0cbc1840151637ff`，消息 `fix: lock settings scroll while model dropdown is open`。
- 治理记录纳入 `chore: add change 032 governance records`。
- 用户已授权普通 push 到 `origin/dev/qsolver-custom`。
- 不创建 Tag、Release 或 RC。
