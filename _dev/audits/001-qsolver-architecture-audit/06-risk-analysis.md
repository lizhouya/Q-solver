# 06 — 风险分析 + AgentMemory 边界

> 只读审计。本节分两部分：
> ① 针对 04-findings 的风险评级与二次开发风险；
> ② **AgentMemory 使用边界**（按审计指令：确认记忆现状、不创建不写入、不启动服务，仅记录将来应存的长期记忆）。

---

## 一、风险分析

### 1.1 按影响分级（高 → 低）

| 风险 | 关联 Finding | 影响描述 | 级别 |
|---|---|---|---|
| KeepContext 无效 | F-001 | 产品宣称的「多轮」实际无记忆；用户勾选后被误导；二次开发若在此基础上做对话历史会推倒重做 | 🔴 高 |
| Windows 置顶不可控 | F-004 | 置顶开关无效；在「幽灵窗口」产品中置顶/穿透的交互不可靠 | 🔴 高 |
| 简历/提示词 dev 日志泄露 | F-012 | 个人简历全文可能写入日志；二次开发联调时更易泄漏 | 🟡 中 |
| Config 静态密钥 | F-011 | 本地配置形同明文；API Key 可被同源码提取 | 🟡 中 |
| PDF 以图片发送（若引入） | F-003 | 二次开发「传 PDF 给模型」的能力会无效 | 🟡 中（影响面待验证） |
| 前端 v-html XSS | F-014/F-015 | 历史摘要与简历预览若含不可信 HTML | 🟡 中（本地数据，低触发） |
| `pkg/tools` 死代码 | F-005 | 二次开发误用孤 Package | 低 |
| 截图参数默认值错位 | F-009 | 配置同步抖动 | 低 |
| `prompt`/`assistantModel` 死字段 | F-007/F-008 | 功能缺失，非故障 | 低 |
| 场景 ID 枚举不一致 | F-006 | 数据不合理，功能靠 fallback | 低 |

### 1.2 二次开发建档风险（制度）

- 分支纪律：审计仅基于 `dev/qsolver-custom`；二次开发必须继续走 `_dev/changes/` 归档 -> 动码 -> 构建验证 -> commit（需用户明确要求）。
- **AVOID 触碰 `main` 官方基线**（CLAUDE.md 规则）。
- `go build ./...` 与前端 `npm run build` 每次改动后验证。

### 1.3 需要运行验证的项（`Needs Verification`）

| 项 | 验证方式 |
|---|---|
| PDF 经 image_url 发送是否真失败 | 用一个支持 OpenAI image 但拒绝 PDF 的 provider 实测 |
| `keepContext` 用户可见行为 | 开启 keepContext 做两轮解题，观测第二轮模型是否记得第一轮（预期：不记得） |
| Windows 幽灵模式抓录屏 | 用 OBS 实测 WDA_EXCLUDEFROMCAPTURE 是否有效 |
| 前端 `||` 兜底实际覆盖的场景 | 手动改配置文件 quality=0 后打开应用重载截图预览 |

---

## 二、AgentMemory 边界

> 依据审计指令：「确认 Memory 使用边界：确认无 Q-Solver 记忆、不创建、不写入、不启动服务；只记录将来应存入长期记忆的信息」。

### 2.1 现状确认

| 检查项 | 结果 |
|---|---|
| AgentMemory（`E:\AI-Memory\agentmemory`）是否存在 Q-Solver 项目记忆 | **不存在**（审计会话仅能确认未创建；未探测出去前提着无记录） |
| 审计会话是否创建/写入任何 AgentMemory 笔记 | **否**。本次为只读审计，未调用任何 memory 写入 |
| 是否启动 agentmemory 服务 | **否**（且按 AGENT.md 规则不擅自启动） |
| 是否读取项目记忆辅助推断 | **否**（一切结论来自当前源码 / git history / `_dev/` 文档） |

### 2.2 审计完成后「将来应存入长期记忆」的信息（非现在写）

> 以下仅**记录建议**，供二次开发阶段存入 agentmemory；本审计不动 memory。

- 项目事实：Q-Solver 是 Wails v2 + Go + Vue3 截图解题桌面应用，开发分支 `dev/qsolver-custom`
- 关键架构约束：`KeepContext` 后端当前无效（F-001）；Windows 置顶不可控（F-004）；`pkg/tools` 是死代码（F-005）
- 配置加密为静态密钥（F-011）；dev 日志会打系统提示词（F-012）
- 归档/分支纪律：任何改动先 `_dev/changes/` 归档；不碰 `main`
- 工具偏好：中文交流、优先 Claude 内置文件工具、目录操作走 PowerShell（`E:\DATA\求职_AI` 中文路径）

### 2.3 运行边界提醒（二次开发时）

- **不要把 AgentMemory 当项目文档库**：本项目文档在 `_dev/`，agentmemory 只是跨会话辅记
- **不擅自启动 agentmemory 推理 flags**（AGENT.md 红线）
- secret 展示一律 `SET / NOT SET`

---

## 三、结论

本审计确认了 **1 个核心功能级缺陷（F-001：KeepContext 无效）**，若干平台级行为缺陷（F-002/004），以及一批死代码/占位（F-005 pkg/tools、F-007 prompt 死字段、F-008 assistantModel、F-016 能力过滤未接、F-018 ThinkingBlock duration、F-019 InitLoading/VersionModal）。**安全性方面**有静态密钥（F-011）与 dev 日志泄露（F-012）两点在二次开发阶段需重视。前端展示组件存在 XSS 风险点（F-014/015，数据本地所以低危）与 UX 不一致（F-017 EmptyState 快捷键）。

审计未修改任何业务源码，产物全部位于 `_dev/audits/001-qsolver-architecture-audit/`。

*（本节完成）*