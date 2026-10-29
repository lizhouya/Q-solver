# Change 026 变更请求：CCR 网关接入与连接诊断修复

| 字段 | 值 |
|---|---|
| 变更编号 | `026-ccr-gateway-validation` |
| 提出人 | 用户 |
| 提出日期 | 2026-10-25 |
| 关联变更 | Change 025 多协议模型连接兼容 |
| 状态 | `进行中（事前治理已完成，待实现）` |

## 1. 背景与问题

用户在 Change 025 验证程序中确认：

1. 模型文字实测成功后按钮持续显示“测试中…”，没有结果反馈。
2. 模型页下方“角色设定/场景”内容被裁剪，无法查看和选择。
3. 用户实际通过 CCR 统一管理 provider、密钥与模型，希望在 Q-Solver 中只导入 CCR 地址和密钥，即可读取模型目录并切换模型，不需要逐模型执行付费测试。

只读核查已确认：成功 probe 返回的 typed nil 在 App DTO mapper 中触发 panic；模型页关闭外层滚动且场景区依赖剩余高度；CCR 模型目录没有足够信息让 Q-Solver 可靠推断每个模型的上游协议。

## 2. 目标

- 修复成功 probe 返回时的 typed-nil panic，使成功、失败和等待超时都能退出 loading。
- 修复模型页滚动和场景列表高度，使角色/场景始终可访问。
- 新增显式 `direct | ccr` 连接模式。
- Direct 模式完整保留 Change 025 的三种显式协议和手动文字/图片实测。
- CCR 模式固定通过 OpenAI Chat Completions 入站，实际 provider 和协议转换交给 CCR。
- CCR 模式只读取一次 `/v1/models` 目录，不自动调用推理 endpoint，不显示或执行文字/图片 probe。
- 模型切换只改变模型 ID，不清空已同步目录，也不改变地址、密钥或协议。

## 3. 验收标准

- [ ] typed nil DTO 映射不 panic；成功 probe 返回 `passed`、`error=nil`、`requestSent=true`。
- [ ] Direct 模式文字/图片测试无论成功、失败或前端等待超时都能恢复按钮。
- [ ] 旧配置缺少 `connectionMode` 时在内存解释为 direct，加载不主动重写磁盘。
- [ ] CCR BaseURL 只接受安全 root 形态并统一为 `/v1`；完整 endpoint 和额外 path 被拒绝。
- [ ] CCR 模式无论模型名或残留协议为何，都使用现有 OpenAI Chat adapter。
- [ ] CCR 目录同步恰好一次 `GET /v1/models`，无推理 POST。
- [ ] CCR text/vision probe 在后端发网前返回 `probe_disabled` 和 `requestSent=false`；前端也不调用 bridge。
- [ ] 进入 CCR 模型页自动同步一次；手动刷新可重试；保存、切模型、打开 API 页不隐式调用推理。
- [ ] 输入或选择模型不清空目录；目录身份变化和 probe 身份变化分别失效。
- [ ] 模型建议支持本地过滤，列表为空或不匹配时仍允许手填。
- [ ] CCR UI 不根据模型名显示推测的 provider/图片/文件/context 能力。
- [ ] 模型页可纵向滚动到完整角色/场景列表并保存选择。
- [ ] Go、全部 Node 回归、Vite、固定 Wails v2.11.0 构建和 diff-check 通过；race/GUI/真实 CCR 如实单列。

## 4. 非目标 / 明确不做

- 不实现 CCR 安装、启动、进程控制、内部路由管理或健康检查。
- 不新增 CCR adapter；不解析 CCR 私有配置。
- 不按模型名、品牌、域名、BaseURL 或目录内容猜 direct 协议。
- 不自动 fallback、双发、改 endpoint、改参数或自动推理测试。
- 不把目录读取成功称为模型、图片或协议验证通过。
- 不全面重构设置页、DomainSelector、题库、KeepContext 或日志框架。
- 不新增/升级依赖，不修改 Wails runtime。
- 不调用真实 CCR/模型，不读取真实配置，不消费额度。
- 不覆盖正式程序或旧验证程序，不提交、推送、Tag、Release 或发布。

## 5. 影响面预估

- Go：`pkg/config`、`pkg/llm`、`app/llm.go` 及对应测试。
- 前端：设置 store/UI store、Wails API wrapper、Provider/Model/Settings/Domain 组件及 Node 回归。
- 生成文件：固定 Wails v2.11.0 bindings。
- 治理：本 Change 七份档案、独立审计与 INDEX。
- 风险：旧配置兼容、CCR URL 拼接、隐式费用调用、异步迟到结果、Wails Promise 不 settle、场景区二次嵌套滚动。

## 6. 附带约束

- 自动测试只命中 loopback `httptest`、fake provider 与合成数据。
- API Key、Token、私人 URL、Prompt、截图、题库内容和 SDK 原始错误不得进入输出、日志或文档。
- 当前 Change 024/025 未提交工作树原样保留。
- 固定 `GOTOOLCHAIN=go1.25.13` 和 Wails CLI v2.11.0。
- Windows GUI 与真实 CCR 为 `NEEDS_USER`；race 若仍返回 `0xc0000139`，记为 `BLOCKED`。
