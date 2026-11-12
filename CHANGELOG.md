# 变更日志

## v1.1.0-rc.5 — 2026-11-12（候选 Tag，Release 受 macOS runner 阻塞）

- 修复 app 测试仅设置 Windows `APPDATA`，导致 macOS/Linux 读取 CI runner 或开发者真实 Q-Solver 配置的问题。
- 新增跨平台配置目录隔离 helper，并迁移解题、重试、Mobile、题库和 Settings 测试。
- 本地配置隔离合同、两个 rc.4 失败测试、app/全仓 Go、vet、build、mod verify、前端与 Mobile 回归及 Vite build 通过。
- Windows race 仍因 `0xc0000139` 处于 `BLOCKED`；macOS GUI 仍未人工验证，因此保持 RC。
- run `43232814169` 与 `43246684317` 的 Windows job 均 PASS；两个 macOS job 均未分配 runner（`runner_id=0`、无 steps）并在约 15 分钟后取消，Create Release 跳过，因此 rc.5 目前没有 Release 或公开附件。

rc.5 的候选内容、安全边界与已知限制见 [RC5 发布说明](_dev/releases/v1.1.0-rc.5.md)。当前实际公开 Release 仍为 [v1.1.0-rc.3](https://github.com/lizhouya/Q-solver/releases/tag/v1.1.0-rc.3)。

## v1.1.0-rc.4 — 2026-11-12（候选 Tag，未创建 Release）

- 整合题库整理失败恢复、逐题解析、历史分段保护和当前分段物理清空。
- 增强多协议 Provider、CCR、Responses 与视觉失败诊断，统一 API Profile、模型刷新和连接测试。
- 建立构建产物盘点、候选验证、哈希晋升与回滚治理，继续固定 Wails v2.11.0。
- 优化模型下拉滚动、同步端答案跟随/清理/截图状态及五项快捷键提示。
- 恢复最多三张截图，新增原图重试、中断与快速解题，修复 F10/F12 和历史快捷键字段错配。
- 选择题先映射原始字母与选项文本，再回填答案；冲突或无法辨认时标记“待核对”。
- Windows race 仍因启动错误 `0xc0000139` 处于 `BLOCKED`，macOS GUI 未人工验证，部分较早人工验收仍待完成；因此保持 RC，不标记稳定版或 Latest。
- Tag 对应 run `43212159818` 中 Windows PASS，macOS arm64 回归 FAIL，Intel CANCELLED，Create Release SKIPPED；没有 rc.4 Release 或公开附件。

rc.4 的计划内容见 [RC4 发布说明](_dev/releases/v1.1.0-rc.4.md)；失败根因由 Change 043 修复并前进至 rc.5。

## v1.1.0-rc.2 — 2026-10-22（预发布）

- 修正跨平台测试对等价绝对路径的直接字符串比较。
- Mobile SSE 测试改用内核分配端口，消除 CI 中的端口重绑竞争。
- 保留 RC1 Tag：其三平台应用构建成功，但回归步骤失败，未创建 Release。

详细功能、使用方式与未验收限制见 [RC2 发布说明](_dev/releases/v1.1.0-rc.2.md)。

## v1.1.0-rc.1 — 2026-10-22（预发布）

- 整合题库自动归档、分段、导出确认弹窗和无图参考试卷。
- 修复 API 配置加载、草稿接口测试及普通设置保存。
- 修复跨电脑实时同步无需刷新、提供独立控制序号与键盘/局部滚轮操作。
- 明确原始记录保护、模型调用授权、局域网连接状态和失败重试。
- 固定发布 Wails v2.11.0，发布 Windows x64、macOS arm64/x64 附件及校验和。

详细功能、使用方式与未验收限制见 [发布说明](_dev/releases/v1.1.0-rc.1.md)。本版本为个人 fork 的 RC，不宣称所有 Change 验收完成。此前上游版本见原仓库历史，不追溯伪造本 fork 的发布记录。
