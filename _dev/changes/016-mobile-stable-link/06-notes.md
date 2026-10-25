# 总结与注意事项 06

状态：实现、普通自动验证及用户确认的 Windows GUI 人工验收完成；race 运行环境 BLOCKED；未提交、未推送、未发布。

## 最终边界

- 固定的是 Token、端口 `38511` 和 URL 格式，不是网络 IP。DHCP、VPN、网卡优先级或局域网变化仍可能改变完整 URL。
- 固定模式默认启用；升级前 Token 从未落盘，因此 Change 016 首次启动会生成一次新链接。后续固定模式重启复用专用加密状态。
- 专用状态复用现有配置的 AES-256-GCM 能力，但应用内固定密钥不等同 DPAPI、Keychain 或硬件绑定密钥库。
- Retry 只恢复服务；Clear 只推进显示边界；Reset 和固定模式切换同时换 Token、撤销旧连接并清屏。
- Clear/Reset 不删除桌面、题库、历史分段、截图、导出、普通配置或 Solver 上下文，也不取消正在执行的任务。
- 单向 SSE 没有客户端 ACK；成功提示只表示服务端状态已提交并广播。已经复制、截图或离线保存的内容无法撤回。

## 实施注意事项

- Mobile Token 与普通 `config.Config` 分离；不要把专用字段加入设置 JSON 全量回写。
- 状态 mutation 的顺序必须保持“专用 manager 先提交，再更新 App、Bridge 和 Server”；不要把可能失败的 I/O 移到提交点之后。
- `PublishIfCurrent` 的代际比较与发布必须保持在同一 Bridge 写锁内，否则清屏和迟到任务之间会重新出现 TOCTOU。
- Server Token 轮换必须同步撤销旧 SSE lease；不得只修改新请求鉴权。
- 浏览器对 `sessionId + clearSeq` 的检查是协议边界。不能仅清 DOM 而保留历史、快照或待执行 rAF。
- `mobileOpMu` 必须继续串行 Mobile 生命周期；不要在 `mobileMu` 下执行配置锁操作、磁盘 I/O、socket shutdown 或 WaitGroup 等待。
- 临时模式磁盘 Token 必须保持为空；同一进程 Retry 复用内存 Token，完整重启才生成新的临时 Token。

## 验证与遗留项

- 普通 Go、Node、Vite、Wails、vet、build 与 diff 检查已通过；验证版及 SHA-256 记录在 `build/bin/archive/validation/`。
- 当前 Windows 环境的 `go test -race` 在 package 启动阶段返回 `0xc0000139`，状态为 `BLOCKED`，不能视为 race PASS。
- 固定链接跨完整重启、旧 SSE 真机撤销、断网重连、流式中途清屏、默认端口占用后 Retry 等 GUI/真机矩阵仍为 `NEEDS_USER`。
- 一次聚焦测试期间默认端口 `38511` 被外部进程占用，依赖该固定端口的 11 个既有 App 监听/窗口策略用例按设计 SKIP；没有停止或干扰用户进程。
- 未完成上述人工验收前，本 Change 不关闭；后续提交、推送、Tag 与 Release 均需单独授权。

## 2026-10-23 补充修复总结

- 遮挡根因已同时从状态和层级修复：pending 建立时同步关闭运行状态面板，面板及 leave 动画残留使用 dropdown 层级，始终低于根级确认层。
- Windows 新增可录制的 `clear_mobile_display`，默认严格为左 `Shift+C`（`67+160`）；macOS 不新增、不显示该 action。
- 快捷键只产生请求事件并建立现有确认层，不能直接调用已确认清屏 API。读取、mutation、pending 和长按重复均由共享 busy 门禁串行化。
- 旧配置迁移继续使用原始 JSON key 存在性判断。action 缺失、显式默认重复，以及“旧版已持久化空清屏绑定但旧 action 仍占用 `67+160`”都会把默认组合纠偏给清屏；显式自定义和无冲突的显式空绑定保持不变。
- 录制预览不再提前写 Hook 运行态；冲突不修改任何绑定，成功完成才提交并发送 `shortcut-saved`，避免重复 `ComboID` 让 Go map 非确定地命中隐藏窗口。
- 最终人工复验文件为 `build/bin/archive/validation/Q-Solver-change016-shiftc-final-validation.exe`，SHA-256 为 `a1480ab36c86bdff794669f362e71e5d29b5a5d96b73221bed187dad5b944659`；归档内五份历史/当前 EXE 哈希均通过，旧文件仅供追溯。
- 补充聚焦回归、Go 全量/vet/build、Mobile DOM、Vite、Wails 生产构建和 diff-check 均通过；race 仍因 Windows `0xc0000139` 为 `BLOCKED`。
- 遮挡、左 `Shift+C`、长按防重、重新录制、旧配置冲突和数据保留仍需用户使用 `shiftc-final` 验证版执行 GUI 人工验收。Change 016 未关闭，也未提交、推送、Tag 或发布。


## 2026-10-23 人工验收收尾

- 用户明确确认最终验证版的确认层、左 `Shift+C` 防重、快捷键重新录制、清屏后刷新/重连、下一题继续同步以及桌面/题库/截图/导出/模型上下文保留均通过。
- `Shift+S` 导致窗口消失的现场问题来源于用户当时把该组合绑定为“向下移动”；Windows Hook 对 `Shift+S`（`83+160`）与 `Shift+C`（`67+160`）严格区分，不是清屏快捷键冲突。
- 窗口坐标一度为 `X=-32768`，进程仍响应且窗口仍被系统标记为可见，因此托盘 Show 与重启没有重新定位。现场只执行窗口恢复位置，没有终止进程、删除数据或修改私人配置。
- 用户已将移动快捷键恢复为 `Alt+方向键`。窗口移动缺少屏幕边界保护是独立边界，不纳入 Change 016。
- Change 016 的 Windows GUI 人工验收结论更新为 PASS；race 仍为 BLOCKED，所以保持未关闭。所有 commit、push、Tag 与 Release 仍需新的明确授权。

## 2026-10-25 整合提交状态（最新）

用户明确批准本次将 Change 016～023 累积改动按功能/测试与治理两次提交整合，并连同既有 RC2 提交推送至 origin/dev/qsolver-custom；此授权只适用于本次。

- 功能/测试已提交：`e44bb6163f0cb1b88e2ec4b4da80a1446ddb1990`。
- 本治理档案随第二次整合提交归档；完整边界见 `../../audits/016-023-commit-boundary.md`。
- 原文“未提交/未推送”是当时记录，最新 Git 状态以分支历史和远端分支头为准；本次不创建 Tag 或 Release。
- 不扩大人工验收结论：用户明确确认 Change 016、023；其他未明确项目和既有 race BLOCKED 保留，不因提交自动关闭。
