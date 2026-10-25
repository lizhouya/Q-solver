# Change 022 实施后审计

状态：主会话源码自审、普通自动验证、最终构建与归档核验完成；GUI 与真实模型 NEEDS_USER，race BLOCKED，未提交、未关闭。

## 范围与依赖方向

- 独立 `pkg/answersummary` 仅使用标准库；App、Solver、题库与 Mobile 引用该包，没有逆向依赖或第二套答案解析。
- 题库 Record 新增可选 AnswerItems，schema 仍为 2；原 Answer、图片、旧导出和各 Session 状态语义保留。
- App 公开命令没有增加；历史 DTO 增加答案项和复制文本，生成器新增对应 TypeScript 类型。
- Task 6 恢复守卫实际位于 `app/questionbank_continuation.go`，已在方案补充到白名单。空快照采用非 nil 的 persisted 空 turns，避免触发旧数据重建。

## 核查结论

| 边界 | 结论 |
|---|---|
| 单次模型请求 | 编号摘要放当前 user text part，图片顺序不变；无额外模型调用 |
| 原子完成 | Answer、Items、ConversationSnapshot 同一候选保存；成功后才发布 solution |
| 失败降级 | 解析失败仅禁用结构化复制，完整回答保留；保存失败保持 pending completion 供 Retry |
| 上下文 | OFF 同锁清 Solver/refs，旧 epoch 不重新追加；启动/Retry/续接在 false 时不恢复 turns |
| 编号 | 同 Session completed Records 的受限 Label；长度/数量/字符集受限；显式截图规则由 Prompt 优先，不能宣称 OCR 已验证 |
| Desktop | selected round 与 stream target 分离，复制文本跟随 Record；错误与取消不提供最终答案复制 |
| Mobile | Snapshot 与订阅者各自复制切片，保持 seq/clearSeq/controlSeq；旧单参数终态不继承上轮复制数据 |
| 剪贴板 | Wails/Clipboard API/fallback 检查实际结果，仅成功显示已复制；不回退复制全文 |
| 快捷键 | 只传 label/keyName，空绑定跳过；未改 Hook 算法、私人配置或确认层 |
| 依赖和旧产物 | 构建前 28 项保护哈希最终全部一致，统一归档 12 条校验通过 |

## 自审修复

- “待核对”包含匹配，避免带说明文本被视为 ready。
- 取消后 provider 迟到成功不能发布 solution。
- 编号 Label 拒绝标记/引号/冒号等非法字符。
- 关闭上下文后的持久化失败不能错误返回解题成功，也不能回滚已清除历史。
- Mobile 测试用例与真实 SSE 夹具隔离，避免不同 VM 共享重建 DOM 造成假失败。

## 审查与测试限制

- 此处为主会话源码自审，不声称独立代理完整审查通过。先前代理受 429 中断；本轮主会话完成 Task 5～后续接线及验证。
- 普通 Go 测试/vet/build、七份前端回归、Mobile DOM 与 Wails 首次构建通过。
- race 五包均在启动阶段报 `exit status 0xc0000139`；GUI 与真实模型效果未执行。

## 提交边界与禁止项

- 工作树包含 Change 016～022 累计修改，不是 clean，不可用整文件盲目归集为本 Change 单独提交。
- 功能/测试与治理应分别审计后再请求授权；本次没有 commit、push、Tag 或 Release 授权。
- 不含真实 API Key/Token，不上传截图、不调用真实模型，不删除或覆盖用户题库、截图、旧导出、正式程序和旧验证程序。

最终验证程序 SHA-256：`5cde3df3a8bdd160f7740ffeebc6b68c7dcb3f87bbfc5ff7226ce6bf2950e751`。App.js/App.d.ts 与生成前副本逐字节相同，models.ts 差异仅结构化类型/字段及生成器空白清理。
