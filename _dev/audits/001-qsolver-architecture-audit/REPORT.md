# Q-Solver 架构审计报告

> **审计编号**：001-qsolver-architecture-audit
> **审计类型**：只读架构审计（read-only）
> **日期**:2026-09-26
> **基线**:`dev/qsolver-custom`（HEAD `834e9fe`，治理提交 `b34bb04` 之后）
> **审计原则**:所有结论来自①当前源码 ②Git history ③`_dev/baseline` 既有文档；不使用猜测/既往结论/未验证记忆。确定性分级：`Confirmed` / `High Confidence` / `Needs Verification`。
> **本报告为入口文档**，详细证据见同目录 `01`~`06`。

---

## 1. 审计范围与结论概要

| 项 | 结论 |
|---|---|
| 审计范围 | 全部后端 `pkg/` + `app/` + 前端 `stores/ / services/ / components / utils` + 入口 `main.go` |
| 只读 | **已遵守**：未修改任何业务源码（结束前 `git status` 验证，见第 9 节） |
| 主要产出 | 6 份分析文档 + 本报告 |
| 核心结论 | **系统架构整体稳健，但存在 1 个核心功能级失效（KeepContext 多轮无效）与若干平台级行为缺陷** |

### 关键结论（Top 发现）

1. 🔴 **F-001 `KeepContext` 后端无效**：`chatHistory` 从未被 `Solve` 读取，每轮都是「system + 新截图」独立请求，前端却展示多轮卡片 —— 产品形态与实际行为错位。
2. 🔴 **F-004 Windows 置顶不可控**：`SetWindowLevel` 空实现，「取消置顶」开关无效果。
3. 🟡 **F-002 macOS `solve` 快捷键未绑定动作**。
4. 🟡 **安全 2 项**：配置静态密钥（F-011）、dev 日志打印完整 system+简历（F-012）。
5. ✔️ **架构清晰**：配置订阅驱动（Subscribe→Provider/快捷键/Solver 重建）、事件驱动流式输出、模块解耦良好，二次开发基础良好。

---

## 2. 技术与架构总览

**技术栈**：Wails v2 + Go 后端 + Vue 3 (Pinia) + Vite + WebView2（Win）/ WKWebView（mac）

```
main.go ──> app/ (App struct)
  ├─ pkg/config    ConfigManager (AES-256-GCM 存 userConfigDir/Q-Solver/config, Subscribe)
  ├─ pkg/llm       Service/Provider/OpenAIAdapter (SDK 流式, thinking 解析)
  ├─ pkg/solution  Solver.Solve (SystemPolicy+Scene+Resume → GenerateContentStream)
  ├─ pkg/task      TaskCoordinator (context 取消)
  ├─ pkg/screen    CapturePreview (kbinani)
  ├─ pkg/ImageUtil CompressForOCR (长边≤1280, JPEG q≤90)
  ├─ pkg/shortcut  Windows 全局钩子 + KeyBinding
  ├─ pkg/platform  幽灵模式/焦点/置顶
  ├─ pkg/domain    70 场景 prompts + SystemPolicy
  ├─ pkg/resume    简历 PDF→LLM Markdown
  ├─ pkg/state     窗口可见/穿透/移动
  └─ pkg/common    常量/环形缓冲/日志(dev/prod 双构建签)
     (pkg/tools 死代码)

frontend/src
  ├─ services/api.js (wails 绑定) + events.js (事件总线)
  ├─ stores/ ui.js / settings.js / solution.js
  ├─ utils/ markdown-latex.js / modelCapabilities.js
  └─ components/ (~20 个, App.vue 接线)
```

- **配置流**：前端保存 → `UpdateSettings(json)` → `UpdateFromJSON` → 订阅者（llm 重建 provider / solver SetProvider+ClearHistory / shortcut 复制键）→ 即时生效。
- **解题流**：截图`CapturePreview`→ buffer(≤3) → `TriggerSend` → buffer 空则自动截图 → goroutine `Solve` → 流式 emit(thinks→content→solution)。

---

## 3. 模块职责（详见 02）

| 模块 | 一句话职责 | 健康度 |
|---|---|---|
| config | 加密持久化+变更订阅 | ✅ 但密钥硬编码(F-011) |
| llm | Provider 自适应+流式 | ✅ 但 PDF 仅当图片(F-003) |
| solution | 组装 Prompt + 流式调LLM | ⚠️ chatHistory 未用（F-001） |
| task | 单当前任务 context 取消 | ✅ 简洁够用 |
| screen | 截图预处理 | ✅ |
| shortcut | 全局快捷键/录制 | ⚠️ 缺 solve case（F-002） |
| platform | 幽灵/穿透/置顶 | ⚠️ Win 置顶空实现（F-004） |
| domain | 70 场景提示词 | ⚠️ 默认域 ID 不在集合（F-006） |
| resume | 简历 PDF→LLM Markdown | ✅ 但简历全文落盘（F-013） |
| state | 窗口隐身/置顶/移动 | ✅ |
| tools | **死代码** | 🔴 孤儿包（F-005） |
| frontend stores | UI/设置/解题状态机 | ✅ 架构清晰 |
| frontend components | 展示/交互 | ⚠️ 若干 UX/死代码（F-017~019） |

---

## 4. 核心调用链（详见 03）

| 链路 | 是否命中预期 | 说明 |
|---|---|---|
| 截图→buffer→发送→LLM→渲染 | ✅ | 完整可用 |
| 配置保存→订阅→即时生效 | ✅ | Subscription 驱动 |
| 快捷键触发→动作分发 | ⚠️ | `solve` 动作分发缺失 |
| KeepContext 多轮 | ❌ | 后端从不携带历史 |
| 简历注入 | ✅ | CandidateProfile 进 system |

---

## 5. 发现汇总（详见 04）

| ID | 摘要 | 类别 | 确定性 |
|---|---|---|---|
| **F-001** | chatHistory 未读 → KeepContext 无效 | Bug Candidate | Confirmed |
| **F-002** | handleTrigger 无 solve case | Bug Candidate | Confirmed |
| **F-003** | PDF 被统一当图片发送 | Behavior Limit | Confirmed |
| **F-004** | Windows SetWindowLevel 空实现 | Bug Candidate | Confirmed |
| **F-005** | pkg/tools 死代码 | Maintenance Risk | Confirmed |
| **F-006** | domainId 默认值不在枚举 | Needs Verify | Confirmed（数据） |
| **F-007** | prompt 死字段（保存即清空） | Missing | Confirmed |
| **F-008** | assistantModel 无 UI/调用 | Missing | Confirmed |
| **F-009** | 截图参数默认值不符 + `\|` 覆盖 | Impl Issue | Confirmed |
| **F-010** | 窗口尺寸硬编码 min/max | Improvement | 低 |
| **F-011** | 配置字符串静态密钥 | Security | Confirmed |
| **F-012** | dev 日志打完整 system+简历 | Security | Confirmed |
| **F-013** | 简历全文落盘（加密但静态钥） | Security | Confirmed |
| **F-014** | HistoryItem summary v-html | Security | 需验证 |
| **F-015** | ResumeImport marked v-html | Security | 低 |
| **F-016** | ModelSelect 能力过滤未接 | Improvement | Confirmed |
| **F-017** | EmptyState 快捷键硬编 | UX | Confirmed |
| **F-018** | ThinkingBlock duration 死逻辑 | Improvement | Confirmed |
| **F-019** | InitLoading/VersionModal 死代码 | Maintenance | Confirmed |
| **F-020** | 场景提示词语言不统一 | Improvement | Confirmed |

---

## 6. 风险评级（详见 06）

| 级别 | 项 |
|---|---|
| 🔴 高 | F-001（KeepContext 失效）、F-004（Windows 置顶）|
| 🟡 中 | F-002（solve 快捷键）、F-011（静态密钥）、F-012（dev 日志）、F-003（PDF 若引入）、F-014/015（v-html XSS，本地低触发）|
| 🟢 低 | F-005/006/007/008/009/010/017/018/019/020 |

**二次开发建议关注**：① 多轮对话的重新实现（F-001 是最大的重构点）；② 置顶问题解决（影响幽灵模式体验）；③ dev 日志脱敏。

---

## 7. 产品决策问题（详见 05）

D-1 定位（解题 vs 备考 vs MBTI 角色）
D-2 KeepContext 预期行为（真多轮 vs 仅并列展示）
D-3 隐身模式边界
D-4 场景库维护模式（编译期 vs 运行时）
D-5 简历能力边界
D-6 模型默认与能力过滤策略
D-7 提示词语言（中/英混）
D-8 是否支持文本输入模式
D-9 「发送即解题」自动补图
D-10 首次引导
D-11 成本/用量提示

---

## 8. AgentMemory 边界（详见 06§2）

- 确认：无 Q-Solver 记忆、本次未创建/未写入/未启服务。
- 将来应存：项目事实、`dev/qsolver-custom` 分支纪律、F-001/004/005 等关键架构约束、中文交流与目录操作用 PowerShell 等。

---

## 9. 只读验证

- ✅ `git status` 只新增 `_dev/audits/`（审计产物）。
- ✅ 业务源码零改动（`app/** pkg/** frontend/** main.go go.mod …`）。
- 产出目录：`_dev/audits/001-qsolver-architecture-audit/`

---

*报告结束 —— 等待用户下一步指令。*