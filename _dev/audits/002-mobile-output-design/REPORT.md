# Q-Solver AgentMemory + Mobile Output Design Report

> **审计编号**：002-mobile-output-design（含 AgentMemory 初始化）
> **审计类型**：只读设计审计 + AgentMemory 初始化
> **日期**:2026-09-26
> **基线**:`dev/qsolver-custom`（HEAD `b34bb04`，治理提交之后）
> **审计原则**：所有结论来自①当前源码 ②Git history ③既有文档；确定性分级 `Confirmed / High Confidence / Needs Verification`。
> **本报告为入口文档**，详细证据见同目录 `00`~`06` + `_dev/memory/qsolver-memory-policy.md`。

---

## 1. AgentMemory 状态

| 项 | 状态 |
|---|---|
| CLI / 插件 | ✅ `agentmemory` v0.9.29 已全局安装；Claude Code 插件已安装（12 hooks + 54 MCP tools + 17 skills） |
| 集成产物 | ✅ 插件 manifest 指向 `hooks/hooks.json`，12 事件钩子（SessionStart/UserPromptSubmit/Pre+PostToolUse/PreCompact/…）均可用 |
| MCP | ⚪ 未启用（`~/.claude/.mcp.json` 未配置）——插件 hooks + REST 已足够，MCP 为可选 |
| 环境变量配置 | ✅ `~/.agentmemory/.env` 仅设 `AGENTMEMORY_INJECT_CONTEXT=true`，其余**推理 flags 全部默认关闭**（AUTO_COMPRESS / REFLECT / GRAPH_EXTRACTION / CONSOLIDATION / CLAUDE_MEMORY_BRIDGE / ALLOW_AGENT_SDK） |
| 服务运行状态 | ⚪ 默认**停止**。本次初始化时**临时启动**（zero-LLM 模式，无推理 flags），写入记忆后已 `stop` 恢复原状（3111/3112/3113 端口已释放） |
| 核心验证 | ✅ 插件启动日志确认 zero-LLM 运行（无 provider key，BM25 + 本地 embed），**未开启任何推理 flag** |

## 2. Memory Scope 状态

| 项 | 值 |
|---|---|
| 项目名称 | Q-Solver（期望） |
| 项目路径 | `E:\DATA\求职_AI`（当前 workspace root） |
| **自动派生 scope 名** | `求职_AI`（按 git 顶层目录 basename 自动解析，无需手工创建） |
| 是否需显式 scope 文件 | **不需要** —— agentmemory 插件在 SessionStart 按 cwd/git 根自动归组 |
| 若要强命名 `Q-Solver` | 设 `AGENTMEMORY_PROJECT_NAME=Q-Solver`（改动全局，需归档流程；本阶段未改） |
| Policy 文档 | 已建：`_dev/memory/qsolver-memory-policy.md`（含 memory 允许/禁止/优先级/七星写入纪律） |

> Q-Solver 记忆自动落在 `求职_AI` 项目分组。首次初始化记忆通过**临时** worker REST `POST /agentmemory/remember` 写入 4 条并 smart-search 验证。

## 3. 已保存的长期记忆（第一批）

| # | obsId | 内容 | 验证 |
|---|---|---|---|
| 1 | `mem_muii484l_...` | Q-Solver：Wails v2 + Go + Vue3 截图解题桌面应用；技术栈/架构/分支 | ✅ score 1.0 |
| 2 | `mem_muii485...` | 开发纪律：归档先行 `_dev/changes/` + 只碰 `dev/qsolver-custom` + secret 只 SET/NOT SET + 中文交流 | ✅ score 1.0 |
| 3 | `mem_muii486y_...` | 审计 001 关键发现 F-001/002/004（KeepContext 无效/solve 快捷键/置顶空实现）+ 配置订阅/事件驱动架构 | ✅ score 1.0 |
| 4 | `mem_muii487j_...` | 未来方向：Mobile Output（audit 002，未实现）候选 A/B/C/D + D-Mobile 决策项 | ✅ score 0.98 |

> 写入后确认 smart-search 全部命中；随后 `agentmemory stop`，记忆已持久化到磁盘（重启后可用 `npx @agentmemory/agentmemory` 拉回）。无推理 flags 被启用。

## 4. 当前输出架构（结论要点，详见 01）

- **核心链路**：`app.TriggerSend()` → `pkg/solution.Solver.Solve()` → `cb.EmitEvent("solution-stream-*"/"solution")` → Wails `runtime.EventsEmit` → 前端 `solution.js` 状态机 → App.vue 渲染（history/thinking/streamingHtml）。
- **关键发现**：
  - 答案由 `GenerateContentStream` 流式产生（solution.go:92）；**后端不存储**，内存态存前端 `history[0].rounds[last].aiResponse`。
  - `app.EmitEvent`（app/app.go:95-97）是**唯一事件出口**——天然可作 mobile 镜像点。
  - `F-001` 确认 chatHistory 未接入（solve.go 从不读 chatHistory），`keepContext` 仅前端 UI 展示多轮卡片。
  - 桌面渲染与事件强解耦（解耦点 = EmitEvent 抽象），Mobile 插入建议走「**事件总线旁路**」（01 Q6）。

## 5. Mobile Output 候选方案（10 部分：不选型）

| 方案 | 优点 | 缺点 | 适配场景 |
|---|---|---|---|
| **A WebSocket** | 双向、低延迟、贴近流式 | 需自写心跳/重连/连接池 | 手机要反控或全流式 |
| **B HTTP Polling** | 实现最简单 | 流式体验差；粒度粗 | 只展示最终答案 |
| **C SSE** | 单向流式完美、EventSource 内置重连 | 无双向控制（需补 POST） | **手机只读流式**（实测最优平衡） |
| **D 其他** | 复用既有机制 / 云中继 | 云中继与本地优先冲突；文件共享体验差 | 辅助连接（扫码打开） |

> ⚠️ 本审计 **不选型**；SSE 在「只读 + 流式 + 断线重连」场景最契合（观察，非结论），但最终看 D-Mobile-006/003。

## 6. 涉及模块与集成点（结论要点，详见 04）

- **后端承载**：新 `pkg/mobile`（或 `pkg/bridge`）独立包 + 在 `app.App` 挂载；`pkg/platform` 保持纯 Win32 桌面平台层（其内含 SetWindowLevel 空实现 F-004）。生命周期：`app.Startup` 起 → 运行期服务 → `App.OnShutdown` 停（main.go:40/60）。
- **前端承载（12 部分）**：方案 2「新增 mobile 独立 SPA（frontend/mobile/ → dist/mobile）」在「不动桌面 + 移动优化 + 构建清晰」最平衡；方案 3 Go embed 纯静态页次之；方案 1 复用同一 Vue 对幽灵窗口侵入最大。
- **OutputMode**：`desktop / mobile / dual` 最影响「前端渲染开关」，后端不受——后端照发事件，前端按 mode 决定是否落 DOM + 移动端永远消费。
- **连接（15部）**：二维码为主 + IP 手动兜底；短时效 Token（防局域网嗅探，不开公网）；EventSource 自动重连 / WS 应用层重连。Token 属敏感信息，只 SET/NOT SET。
- **状态同步数据结构**（03）：`{seq, ts, status(idle|waiting|streaming|done|error), stage(thinking|content), payload{question, thinking, answer, rounds, model, elapsedMs}}` + error。用 `seq` 防丢包，`rounds` 为 F-001 预留。

## 7. 技术风险（详见 05）

| 级别 | 关键项 |
|---|---|
| 🔴 高 | Token 安全 / 局域网数据暴露（题面+简历）；网络服务和 Wails 生命周期绑死；Windows 防火墙入站阻断 |
| 🟡 中 | 断线重连状态失步；WS 双向缺省；多网卡 IP 选择；双构建复杂度；依赖新增（go.mod 需归档） |
| 🟢 低 | macOS 未支持；移动端 XSS（F-015 相关，本地低触发） |

**Needs Verification**：手机浏览器跨防火墙访问实测；断线 10s 重连一致；多网卡选 IP 正确性（见 05 末）。

## 8. 用户决策问题（详见 06）

- **D-Mobile-001** 手机端形态：浏览器 / 小程序 / App — 建议浏览器
- **D-Mobile-002** 网络范围：local-only vs 公网 — 倾向 local（隐私/本地优先）
- **D-Mobile-003** 显示内容：仅答案 / 答案+解析 / 完整流式 — 决定传输层&数据结构
- **D-Mobile-004** 电脑端行为：`desktop` / `mobile` / `dual` / 切换
- **D-Mobile-005** 连接方式：二维码 / IP / 两者
- **D-Mobile-006** 手机仅看 or 可反控（决定 SS/WS）

> 以上 6 项+提供的「附 A：与 F-001/002/004 关联」请由 Q-Solver 负责人逐条确认，**本审计不做选择**。

## 与审计 001 关键 Finding 关联（摘要）

- **F-001**（KeepContext）— Mobile 若要「看多轮历史」需先修；若只看当前单轮可并行。**独立时序**。
- **F-002**（solve 快捷键）— 仅当手机反控「触发解题」才依赖。独立分析。
- **F-004**（Win 置顶）— 影响「mobile 模式」的隐身体验，不阻塞。独立分析。

## 9. 下一阶段建议

1. **用户确认 D-Mobile-001~006** → 收敛第 5/8/14 部分的传输选型与数据结构。
2. 决定传输架构（建议 SSE+EventSource、备 WS）→ 进入 `_dev/changes/` 归档方案 🔒 之后才可动代码。
3. 决定前端承载（推荐方案2独立 mobile 构建或方案3 Go embed）→ 将影响 build 脚本。
4. 安全加固先行：Token 机制、二维码内容不入日志。
5. 现阶段保持 **Mobile Output 与 F-001/002/004 相互独立**（无强耦合）。

---

*报告结束 — 等待用户下一步指令（只读，业务源码零修改）。*