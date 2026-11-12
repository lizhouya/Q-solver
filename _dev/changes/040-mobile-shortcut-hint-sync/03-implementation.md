# 实现过程记录 03

## 当前状态

同步端五项提示、满额键帽和动态底部空间已实现；普通自动门禁与用户 GUI 验收通过。Change 040 功能随 Change 041 叠加候选晋升为根稳定版，原 Change 040 候选按原哈希归档 transient。race 环境 `BLOCKED` 已披露；功能提交 `09dd485` 已创建，治理提交随本档案完成。未推送、未发布。

## 实现日志

| 时间 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-11-12 | 只读审计 | 确认后端已下发五项有序提示，遗漏位于同步端底部卡片旧三项筛选 | `app/mobile.go`、Mobile 页面与测试 | 不扩大到协议或配置层 |
| 2026-11-12 | 方案确认与事前归档 | 用户确认五项一致显示、满额保留键帽和动态底部空间方案 | Change 040 | 业务修改前完成 00/01/02 |
| 2026-11-12 | DOM 红灯 | 新断言期望实际卡片高度 146px + 32px 安全间距，旧实现返回固定 112px | `pkg/mobile/web_dom_test.mjs` | 红灯准确命中旧固定空间；后续五项断言尚未执行 |
| 2026-11-12 | 后端契约补强 | 用五项非默认键名严格锁定后端前五项标签和顺序，并验证 JSON 不泄露内部字段 | `app/mobile_test.go` | 后端实现无需修改 |
| 2026-11-12 | 最小实现 | 扩展为截图、发送、快速解题、中断、删除；满额只切换截图文案；按实测高度更新底部空间并监听 resize | `pkg/mobile/web/index.html` | 继续消费 Snapshot `hints`，不硬编码 F 键 |
| 2026-11-12 | 聚焦验证 | DOM 与 `go test ./app ./pkg/mobile -count=1` 通过 | Mobile 测试 | 真实 SSE→DOM 回归同时通过 |
| 2026-11-12 | 全量门禁 | 19 个前端回归、Vite、全量 Go、vet、build、mod verify、diagnostics 全部通过 | 全仓相关范围 | 无依赖升级或生成噪音 |
| 2026-11-12 | race | `app`、`pkg/mobile` 启动阶段均返回 Windows `0xc0000139` | race | `BLOCKED`，未进入测试体 |
| 2026-11-12 | 保护构建 | 固定 Wails v2.11.0、保护参数构建唯一候选；831 个受保护文件和 172 个测试区外产物不变 | `build/bin/testing/current/` | 根稳定版与 Release 未变 |
| 2026-11-12 | 隔离启动 | 隔离 Profile 启动未创建窗口；确认系统已有用户 Q-Solver 实例，未停止用户进程 | 候选运行 | 单实例边界导致 `BLOCKED`；失败 Profile 已清理 |
| 2026-11-12 | 用户验收 | 用户使用最终叠加候选确认同步端 F8–F12 五项提示验证通过 | Change 041 候选 | GUI PASS |
| 2026-11-12 | 稳定版晋升 | 最终叠加候选替换根 EXE，旧稳定版和验证候选双归档，测试区清空 | `build/bin/**` | 根 SHA-256 `288535…F15B`；Release 未变 |

## 与计划的偏差

- 首次构建后比较因 Windows PowerShell 5.1 将 JSON 清单序列化为“属性数组”对象而误报大量路径差异。未执行恢复或删除；改用按索引展开 Path/Size/SHA256 后重新核对，证明 831 个受保护文件和 172 个测试区外产物均无变化。
- 计划中的独立启动、WebView2、正常退出和残留进程验证未完成，因为用户原有 Q-Solver 实例不能由本任务擅自停止。

## 关键决策纪要

- 后端提示定义和配置订阅已正确，不修改 `app/mobile.go` 实现。
- 同步端继续只读；快速解题与中断只显示快捷键，不提供控制按钮。
- 底部空间使用 `getBoundingClientRect().height + 32px`，无有效高度时回退到原等价的 112px。
- 不为自动验证终止用户正在运行的 Q-Solver。

## 稳定版与归档

- 根稳定版：`build/bin/Q-Solver.exe`
- 大小：42,558,464 字节
- SHA-256：`2885358371510B827ACE560C354FEE8775BECEB47555D7CE654B0FBE3EB1F15B`
- 原 Change 040 候选归档：`build/bin/archive/transient/Q-Solver-candidate-change040-before-choice-consistency.exe`，SHA-256 `51228E51DD744D4E5CD74000FA059203464116DE396F0269CD98BA904178B2C5`。
- 验证归档：`build/bin/archive/validation/change-041/Q-Solver-candidate-change041.exe`。
- 被替代稳定版：`build/bin/archive/superseded/change-041/Q-Solver-stable-before-change041.exe`。
- Wails runtime：v2.11.0；`testing/current/` 已清空，Release 未改变。

## 遗留未完成项

- Git 功能与治理提交已授权，尚未执行；push、Tag、Release 未授权。
