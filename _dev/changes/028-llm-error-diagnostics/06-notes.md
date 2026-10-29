# Change 028 总结与下一步

日期：2026-10-26
状态：实现、普通自动验证与独立 Wails 构建完成；GUI 和真实上游 NEEDS_USER；race BLOCKED；未暂存、未提交、未关闭。

## 已交付

- 主解题错误的安全结构化摘要，支持包装错误、未知类别和缺失状态。
- 原 solution-error 字符串不变，新增可选第二参数；桌面封套与 Mobile SSE 同帧关联错误和详情。
- 桌面整页/分轮错误及手机错误卡中的可展开“诊断详情”。
- Responses failed/error 统一白名单分类，不再全部显示“服务暂时不可用”。
- 全量普通 Go test/vet/build、14 个前端回归、Mobile DOM、Vite、固定 Wails 构建与 prod 诊断专项通过。

## 验证程序

- 路径：`build/bin/Q-Solver-change028-llm-error-diagnostics.exe`
- 大小：44,012,032 字节。
- SHA-256：`370beea9dcff40f4776238f18ef0d4c3c34ac1b711bd3fb3829f3b470644fee7`。
- 生产构建包含 prod 标签，logger 关闭；诊断通过事件到 UI，不依赖日志。
- 未启动验证程序；没有覆盖正式或旧验证程序。

## 边界和遗留

- **尚未获得本次 GPT/Claude 实际失败的上游证据，不宣称已恢复正常。**
- 工厂、CCR Chat 入站、协议、模型目录、用户配置、图片、上下文和 Prompt 未改。
- 诊断只在当前桌面会话和 Mobile 当前快照存在，不写入题库；应用退出、重新加载历史、Mobile 清屏/新题等会使对应详情不再可用。
- HTTP 状态是 Q-Solver 当前连接边界观测到的状态，不能自动归因于最终模型服务商。
- 未知类别、未知状态或重试建议显示“未知/暂无信息”；可重试仅为手动建议，不触发自动请求。
- 全树 diff-check 保留 models.ts 的 85 处原有尾随空白；本次增量无空白错误。
- prod 全套相关包的旧日志测试仍 FAIL（要求输出日志，与 prod 空 logger 冲突）；专项诊断 PASS，两者不混淆。
- race 四包启动时 0xc0000139，BLOCKED；Windows GUI 未验证。

## 下一轮最小复现（仅由用户主动执行）

1. 确认旧实例已正常退出，然后手动启动本次独立 EXE；不要误用旧正式程序。
2. 保持现有 CCR、API 地址、凭据和协议不变；在模型页选择原来失败的现有模型 ID，保存设置。不要因名称切换协议。
3. 准备一张不含私人信息的简单测试图，手动触发一次正式解题。真实请求可能产生费用，本次自动阶段未执行。
4. 若失败，立即展开错误区“诊断详情”；mobile 输出模式在手机错误卡查看，dual 模式两端均可查看。
5. 提供：EXE 文件名/哈希、失败时间及时区、入口（正式解题）、code、stage、httpStatus（或未知）、protocol、retryable、友好 message。
6. 另从实际 CCR 实例取得同一时间的脱敏请求关联记录和路由错误类别。如需说明模型路由，用用户可接受的别名标签即可，不需给出 Key、完整 URL 或请求正文。
7. 仅在有需要且愿意承担费用时，再对另一模型重复一次；不要批量重试，也不要把目录加载成功当作推理成功。

不得提供 API Key、Token、Authorization、Prompt、截图/Base64、简历、完整 SDK 错误体或私人地址。若无法失败复现，只能记“该次完成”，不能推断所有原故障已消失。

## 停止条件

本次实施完成后暂停，等待用户检查验证程序和安全诊断，再决定是否有证据支持第二轮业务修复。未执行 git add、commit、push、Tag、Release。

## 2026-10-29 整合提交状态

- 用户批准 Change 024～031 一次性跨 Change 整合。
- 功能/测试提交：`2a8a506636496a1eb6c38662c6ef6eaccc1fe958`（Change 030 无业务源码，仅归档治理记录）。
- 本 Change 的治理档案纳入 `chore: archive changes 024-031 and define artifact lifecycle`。
- 本轮全量自动验证通过；原有 `NEEDS_USER` 与 race `BLOCKED` 项保持，不因提交自动关闭。
- 不创建 Tag、GitHub Release 或新 RC。
