# Change 026 总结与注意事项

日期：2026-10-25
状态：实现与普通自动验证完成；Windows GUI、真实 CCR 待用户验收；race 被环境阻塞；未提交。

## 1. 完成内容

- 修复成功模型实测返回 typed nil 时 App DTO mapper panic，避免界面永久停留“测试中…”。
- 为目录和 probe bridge 增加 20 秒前端等待上限，异常不 settle 时界面可以恢复交互。
- 修复模型页滚动和 `.domain-list` 样式目标，角色/场景恢复可访问性。
- 增加显式 `direct | ccr` 连接模式。
- Direct 保留 Change 025 三协议；CCR 固定使用 OpenAI Chat Completions 入站。
- CCR 只读取模型目录，不执行文字或图片付费 probe。
- 目录与 probe revision 分离，切换模型不再清空目录。
- 增加 CCR 模式 UI、中立能力说明、本地模型过滤和目录状态。
- 修复第二次打开设置时已保存模型误作搜索词、导致 8 个目录项只显示当前 1 个的问题。
- 使用固定 Wails v2.11.0 重生 bindings，并生成独立验证程序。

## 2. 独立验证程序

```text
build/bin/Q-Solver-change026-ccr-gateway-validation.exe
44,032,512 bytes
SHA-256: 5584d5448e6808a230a4d01e35e098938a92690beb3b1f3d149129b0d9771fe1
```

该程序未覆盖正式 `Q-Solver.exe` 或任何旧验证程序。事前记录的 11 个受保护 EXE 哈希前后完全一致。

## 3. 验证结论

- Go 全量 test/vet/build/mod verify：`PASS`。
- 13 个前端回归：`PASS`。
- Vite build：`PASS`，仅保留既有大 chunk 警告。
- 固定 Wails v2.11.0 bindings/build：`PASS`。
- `git diff --check`：`PASS`。
- race：`BLOCKED (0xc0000139)`。
- Windows GUI：`NEEDS_USER`。
- 真实 CCR：`NEEDS_USER`。

## 4. 使用与语义边界

- CCR 模式填写 CCR API root（根地址或 `/v1`）和 Key，进入模型页读取目录后选择模型。
- 模型目录读取成功不等于模型、图片、文件或额度验证通过。
- Q-Solver 不从模型名判断 provider、协议或多模态能力；实际路由和协议转换由 CCR 负责。
- CCR 模式中进入模型页、刷新目录、切模型和保存设置均不应触发推理请求。
- 前端 `frontend_wait_timeout` 只代表界面停止等待，后台请求不保证已取消。
- Direct 模式仍由用户显式选择三种协议并主动执行文字/图片实测。

## 5. 人工验收重点

1. 先启动独立 Change 026 验证程序，不要使用正式或旧验证程序。
2. 验证 Direct/CCR 切换以及 Direct 三协议仍可见。
3. CCR 模式填入真实地址和 Key，确认模型目录只同步一次并可搜索、手填、切换。
4. 确认 CCR 模式没有文字/图片实测按钮。
5. 确认选择模型后目录不消失，保存后设置仍正确。
6. 由用户主动执行一次真实截图解题，确认 CCR 实际上游路由。
7. 切回 Direct，确认 probe 成功或失败后均退出 loading。
8. 缩小窗口并滚动到角色/场景，确认可见、可选、可保存。

## 6. 授权与工作树

- 未读取或输出真实 API Key、Token、私人 URL 或题库内容。
- 未调用真实 CCR/provider，未消费真实 API 额度。
- 未暂存、未提交、未推送、未创建 Tag、未 Release、未发布。
- Change 024/025 未提交内容原样保留，不在本 Change 中清理或回退。

## 2026-10-29 整合提交状态

- 用户批准 Change 024～031 一次性跨 Change 整合。
- 功能/测试提交：`2a8a506636496a1eb6c38662c6ef6eaccc1fe958`（Change 030 无业务源码，仅归档治理记录）。
- 本 Change 的治理档案纳入 `chore: archive changes 024-031 and define artifact lifecycle`。
- 本轮全量自动验证通过；原有 `NEEDS_USER` 与 race `BLOCKED` 项保持，不因提交自动关闭。
- 不创建 Tag、GitHub Release 或新 RC。
