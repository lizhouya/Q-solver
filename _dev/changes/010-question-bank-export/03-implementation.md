# 实现过程

## 2026-10-18

1. 先完成 00/01/02 事前归档和场景选择审计，再创建业务测试。
2. 最初 `go test ./pkg/questionbank -count=1` 按预期因 Store 等接口尚未实现失败（TDD 红灯）。
3. 新增 pkg/questionbank 的场次模型、逐题落盘、恢复、PNG/JPEG 校验、Markdown 版本化导出、隔离的结构化整理。新模块不依赖前端历史或 Solver 上下文。
4. 接入 app/questionbank.go、questionbank_capture.go、questionbank_lifecycle.go；采集在输出路由之前，按独立记录 ID 关联回调。用户额外批准 main.go 的 OnBeforeClose 注册。
5. 新增 Vue 题库面板、Pinia store 和 API 封装，顶部显示录制/保存状态。开始场次、历史恢复、原稿导出、额外调用确认、取消和重试均有入口。
6. 新增 11 个题库模块测试与 3 个编排层测试（含表驱动子用例）。输出目录丢失的回归测试先发现旧路径可能被重建，已修复并重测通过；补充并发写入/读取的确定性测试。
7. 生成 bindings 并完成前端、Go、Wails 构建。CLI 2.16.0 重生成了 v2.11.0 范围外 runtime 通知 API，核对后定向移除，go.mod 恢复到构建前 SHA-256 完全一致；不升级依赖。
8. 最终打包使用 `wails build -skipbindings -m -nosyncgomod`，避免改写依赖；CLI 仍会同步 runtime，已在最终构建后再次定向去除 runtime 噪音。新增 bindings 保留，仅去除新增行的末尾空白。
9. 同步中英文 README，说明本 fork 边界、目录、归档步骤、额外调用费用、明文隐私与人工验收限制。

## 范围补充

- 用户已明确批准 main.go 的退出回调注册。
- 编排代码按职责拆成三个 questionbank 文件；没有修改角色提示词、手机协议、原 PNG 导出、个人配置或 agentmemory 配置。
- 未调用真实模型；测试图片是程序生成的 2×2 空白图片。真实屏幕、对话、简历均未用于测试。

## 状态

功能实现与普通自动验证完成。并发检测运行环境阻塞；Windows 真机及真实整理 NEEDS_USER。未提交、未推送、未正式关闭。

## 最终路径保护复核

将输出根目录校验统一放入 Store：拒绝空值、相对路径及 Git 仓库内目录，覆盖偏好文件损坏/手动改写的情况；不能默默退回当前源码目录。新增对应测试，且目录对话框复用该校验，去除重复校验逻辑。
