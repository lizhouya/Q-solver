# 04 — 发现汇总（Findings）

> 只读审计。所有发现均基于当前 `dev/qsolver-custom` 分支源码（含前端组件扫描子代理结果）。
>
> **编号规律**：F-001 起连续编号，唯一。
> **确定性分级**：`Confirmed`（源码直接证明）/ `High Confidence`（沿调用链推断，逻辑确定）/ `Needs Verification`（需运行或二次开发时验证）。
> **本文件仅记录现状 + 影响，不提出解决方案**（按审计指令）。

---

## 一、功能级发现

### F-001 `chatHistory` 未参与解题请求 → `KeepContext` 后端无效
- **模块**：`pkg/solution`
- **证据**：[solution.go:27](pkg/solution/solution.go#L27) 定义字段 `chatHistory`；[solution.go:84-86](pkg/solution/solution.go#L84-L86) 组装请求时新建 `messagesToSend`，从不读取 `chatHistory`；[solution.go:137](pkg/solution/solution.go#L137) 返回前强制清空 `s.chatHistory`
- **当前行为**：`ensureSystemPrompt`（141-159）、`trimChatHistory`（162-181）、`ClearHistory`（41-43）、`MaxConversationRounds=10`（14）全部实现但从未被 `Solve` 调用，为死代码
- **影响**：`KeepContext` 配置无效。前端按 `/keepContext` 追加 round 卡片（App.vue:152 / solution.js:125-148）显示「多轮」，但模型每轮只看到 system + 新截图，无上一轮记忆
- **确定性**：`Confirmed` | 类别：`Bug Candidate`

### F-002 快捷键动作分发缺 `solve` case — macOS 默认解题快捷键失效
- **模块**：`pkg/shortcut`
- **证据**：[service.go:69-99](pkg/shortcut/service.go#L69-L99) `handleTrigger` 的 switch 仅含 screenshot/send/delete/toggle/clickthrough/move_*/scroll_\*；[delegate.go:5](pkg/shortcut/delegate.go#L5) 声明 `TriggerSolve()`，但 handleTrigger 从不触发；Windows 默认键表（config.go:91-103）无 `solve` 条目
- **影响**：macOS `getDefaultShortcuts()` 配置 `solve: Cmd+1`（config.go:77）但动作到达 switch 时无匹配 → 静默丢弃。Windows 无 `solve` 快捷键定义，仅 F8（screenshot）可用
- **确定性**：`Confirmed` | 类别：`Bug Candidate`

### F-003 PDF 内容部分被统一转成图片发送（非真正 PDF 支持）
- **模块**：`pkg/llm`
- **证据**：[openai.go:75-77](pkg/llm/openai.go#L75-L77) `toOpenAIParts` 将 `ContentImage` 与 `ContentPDF` 统一转为 `openai.ImageContentPart`（`ImageURL: part.Base64`）；[types.go:88-91](pkg/llm/types.go#L88-L91) `PDFPart` 生成 `data:application/pdf;base64,` 前缀
- **影响**：若未来以 `PDFPart` 传入资料，模型收到的是「PDF data URL 被当 image_url」，多数 OpenAI 兼容接口不支持 → PDF 有效输入不可行。当前实际解析简历走 `resume.ParseResume`（非 `PDFPart`），故仅二次开发引入 PDF 输入时暴露
- **确定性**：`Confirmed`（代码形态）/ 影响面 `Needs Verification` | 类别：`Behavior Limitation`

### F-004 Windows `SetWindowLevel` 空实现 — 置顶开关无效
- **模块**：`pkg/platform` / `app`
- **证据**：[platform_windows.go:248-251](pkg/platform/platform_windows.go#L248-L251) 返回 nil（注释「Windows 在 ApplyGhostMode 中处理」）；[app/window.go:37-39](app/window.go#L37-L39) 依赖它切置顶
- **影响**：Windows 端「取消/恢复置顶」无实际效果（窗口始终 TOPMOST，由 Ghost 模式 `SetWindowPos` 设定）。_dev 基线报告亦记录
- **确定性**：`Confirmed` | 类别：`Bug Candidate`

### F-005 `pkg/tools` 死代码（工具注册系统零引用）
- **模块**：`pkg/tools`
- **证据**：全仓 grep `pkg/tools` 无 import；`registry` + `screenshot.go` 的 `get_exam_question` 工具从未注入任何 LLM 调用
- **影响**：推测早期「截图→专用工具」设计遗存，现为孤儿包。二次开发接「截图+工具调用」时可能误用为可支持的特性
- **类别**：`Maintenance Risk` | `Confirmed`

### F-006 `domainId` 默认值 `general-assistant` 不在 70 场景 ID 集合
- **模块**：`pkg/domain` + `frontend settings.js` / `config.go`
- **证据**：[config.go:47](pkg/config/config.go#L47) / [settings.js:16,173](frontend/src/stores/settings.js#L17) 默认 `general-assistant`；prompts.go `promptMap` 中无此 key（通用键实为 `gen-solver`）
- **影响**：`GetPrompt('general-assistant')` 走 fallback（[prompts.go:142](pkg/domain/prompts.go#L142)）返回 `gen-solver` 提示词 → 功能正常，但「配置值不在合法枚举内」是数据不一致；若未来新增 domain 校验/Dropdown 高亮会错位
- **确定性**：`Confirmed` | 类别：`Needs Verification`（数据不一致，低危）

---

## 二、配置 / 字段

### F-007 前端 `prompt` 硬编码 `''` — 后端 `Config.Prompt` 不可编辑、保存即清空，成为死字段
- **模块**：`frontend/src/stores/settings.js` + `config.go`
- **证据**：[settings.js:263](frontend/src/stores/settings.js#L263) `buildConfigToSave` 中 `prompt: ''`；`applyConfig` 读取 `config.prompt`（172）但 `SettingsModal` 无任何 Prompt 输入 UI；后端字段 `Config.Prompt`（[config.go:13](pkg/config/config.go#L13)）
- **影响**：系统提示词 `prompt` 无用户入口，且一次保存即被清空。若未来做场景/自定义 Prompt 需整套重接
- **类别**：`Missing` | `Confirmed`

### F-008 `assistantModel` 死字段（注释指向未实现功能）
- **模块**：`config.go` + `settings.js`
- **证据**：[config.go:27-28](pkg/config/config.go#L28) 注释「辅助模型（用于总结对话生成问题导图）」；[settings.js:171](frontend/src/stores/settings.js#L171) 仅读取，全仓无使用
- **影响**：dead field。二次开发「对话总结 / 导图」若依赖此预留在字段，需确认后端生成器不存在
- **类别**：`Missing` | `Confirmed`

### F-009 前端默认截图参数与后端默认不一致 + `||` 兜底可能覆盖后端保存值
- **模块**：`frontend/src/stores/settings.js`
- **证据**：前端默认 `screenshotMode:'window'`、`compressionQuality:80`、`sharpening:0`、`grayscale:true`（[settings.js:19-24](frontend/src/stores/settings.js#L19-L24)）；后端默认 `fullscreen/92/0.3/false`（[config.go:51-55](pkg/config/config.go#L51-L55)）。`applyConfig` 用 `config.compressionQuality || 80`、`config.sharpening || 0` 等（settings.js:174-181）
- **影响**：若后端返回 0（如用户将锐化调到 0、灰度 false），前端 `|| 0` 仍落到前端默认 → 已保存的配置与实际生效值「同步抖动」；`grayscale` 属布尔因 `!= undefined` 判断已规避，但 `sharpening=0` 会被误回退
- **类别**：`Implementation Issue` | `Confirmed`

### F-010 窗口尺寸存在硬编码边框（min/max 上限 1024×768）
- **模块**：frontend `ResizeHandle.vue` + `main.go`
- **证据**：[ResizeHandle.vue:13-16](frontend/src/components/ResizeHandle.vue#L13-L16) MIN 840×700 / MAX 1024×768 硬编码；[main.go:29-34](main.go#L29-L34) 同步设 min/max 同值（840-1024 × 700-768）
- **影响**：窗口界≥1024×768 或 <840×700 不可达；`config.WindowWidth/Height`（app.go:51-54）恢复尺寸与限制一致。属有意的产品限制，但存硬编码等待出处
- **类别**：`Improvement Opportunity`（低）

---

## 三、安全 / 敏感

### F-011 配置 crypto 静态硬编码密钥（仅形式混淆）
- **模块**：`pkg/config`
- **证据**：[crypto.go](../..//pkg/config/crypto.go)（`init()` 对硬编码字符串 `"JTWZwHaMq3g55PfVgxkah1Edst59LoyF"` SHA256 → AES-256-GCM key）
- **影响**：同源码任何分发版本可解密本地配置（含 API Key）。因配置仅存 userConfigDir 且无远程同步，威胁中等，但注释、责任要知悉
- **类别**：`Security Concern`（中等）

### F-012 dev 构建日志打印完整 system prompt（含简历内容）
- **模块**：`pkg/solution` + `pkg/common/logger`
- **证据**：[solution.go:76](pkg/solution/solution.go#L76) `logger.Println("system 提示词:", ...)`；systemPrompt 内包含 `<CandidateProfile>`（简历，见 67-73）；dev 构建（默认无 `-tags prod`）启用日志，prod 静默（logger_prod.go）
- **影响**：调试构建会把整段 system + 简历发布到控制台/日志文件；若日志被收集会污染含个人简历。次要但需aware
- **类别**：`Security Concern`（低）

### F-013 简历内容全量存本地配置文件（明文-DE 加密、静态密钥）
- **模块**：`config.go` + `resume`
- **证据**：`Config.ResumeContent`（config.go:24）保存解析后的简历 Markdown；经 crypto.go 加密（静态密钥）
- **影响**：简历全文落在本地配置文件（userConfigDir/Q-Solver/config），EF 安全取决于本机文件权限；同 F-011 密钥问题
- **类别**：`Security Concern`（低-中）

---

## 四、前端组件（含子代理扫描）

### F-014 历史摘要 `v-html` 渲染（`HistoryItem.vue`）
- **证据**：[HistoryItem.vue:10](frontend/src/components/HistoryItem.vue#L10) `v-html="previewHtml"`，来自父级 `summary`（可能含 OCR/模型输出）
- **影响**：若 `summary` 含不可信 HTML（模型输出/OCR），有存储型 XSS 风险；需确认数据源净化
- **类别**：`Security Concern`（需验证）

### F-015 简历 Markdown `marked.parse` 结果 `v-html`（`ResumeImport.vue`）
- **证据**：[ResumeImport.vue:119-126](frontend/src/components/ResumeImport.vue#L119-L126) 渲染后 `v-html`；`marked` 默认不剥离原始 HTML
- **影响**：本次为**本地用户自己的 PDF**，风险低；若未来支持云端/他人简历则升为中危
- **类别**：`Security Concern`（低，数据自持）

### F-016 `ModelSelect.vue` 未用能力过滤（supportsVision/supportsFile 定义未用）
- **证据**：`modelCapabilities.js:355-369` 定义 `supportsVision/supportsFile/supportsPDF` 但全仓无调用；[ModelSelect.vue:63](frontend/src/components/ModelSelect.vue#L63) 原样渲染全部 models，能力仅作徽标
- **影响**：视觉解题场景下可选择到不支持图片的纯文本模型，潜在无效请求；过滤函数为「预留未接」
- **类别**：`Improvement Opportunity` | `Confirmed`

### F-017 `EmptyState.vue` 快捷键硬编码 F8 / Ctrl+J，与设置脱节
- **证据**：[EmptyState.vue props 默认值](frontend/src/components/EmptyState.vue) 硬编码 `'F8'` / `'Ctrl+J'`；`SolveView.vue` 未传 props → 展示恒为默认值；而 `WelcomeView` 用 store 动态读（WelcomeView.vue:32-44）
- **影响**：用户自定义快捷键后 EmptyState 提示与实际不符 → 可见 UX bug
- **类别**：`UX Limitation` | `Confirmed`

### F-018 `ThinkingBlock.vue` 的 duration computed 恒 `null` — 头部时长永不显示
- **证据**：[ThinkingBlock.vue:64-67](frontend/src/components/ThinkingBlock.vue#L64-L67) 注释「duration 由父组件处理」，模板 `:16` 引用但永远 null
- **影响**：展示位功能未实现即「隐性空状态」，无错误但延长开发预判
- **类别**：`Improvement Opportunity` | `死代码`

### F-019 `InitLoading.vue` / `VersionUpdateModal.vue` 全仓零引用，含其他项目残留
- **证据**：两个组件在 `frontend/src` 无 import；`InitLoading` 文案 `'GHOST SLOVE'`（拼写错误）且与 `Q-Solver` 命名不符（疑似其它项目残留）
- **类别**：`Maintenance Risk` | `死代码`

### F-020 场景提示词语言不统一（英文 PersonaCard + 中文 SystemPolicy）
- **证据**：prompts.go 的 70 条提示词以英文 `<PersonaCard><Role>...` 为主；`GetSystemBehaviorPrompt` 为中文 XML SystemPolicy
- **影响**：模型可能受提示词语言混合影响稳定性/输出一致性，面向中文用户的场景建议确认语义
- **类别**：`Improvement Opportunity` | `Confirmed`

---

## 五、汇总速查

| # | 模块 | 摘要 | 类别 | 确定性 |
|---|---|---|---|---|
| F-001 | solution | chatHistory 未用，KeepContext 无效 | Bug Candidate | Confirmed |
| F-002 | shortcut | 无 endpoint 无 `solve` case | Bug Candidate | Confirmed |
| F-003 | llm | PDF 转 image 发送 | Behavior Limitation | Confirmed |
| F-004 | platform | Windows SetWindowLevel 空实现 | Bug Candidate | Confirmed |
| F-005 | tools | 整个包死代码 | Maintenance Risk | Confirmed |
| F-006 | domain/settings | domainId 默认值不在集合 | Needs Verification | Confirmed（数据） |
| F-007 | settings | prompt 死字段 + 保存清空 | Missing | Confirmed |
| F-008 | settings | assistantModel 无 UI/无调用 | Missing | Confirmed |
| F-009 | settings | 截图参数默认值不符 + `\|` 覆盖 | Implementation Issue | Confirmed |
| F-010 | window | 窗口尺寸硬编码 min/max | Improvement Opp | 低 |
| F-011 | crypto | 静态密钥 | Security Concern | Confirmed |
| F-012 | solution/logger | dev 日志含简历/system | Security Concern | Confirmed |
| F-013 | config/resume | 简历全文落盘（加密但静态钥） | Security Concern | Confirmed |
| F-014 | HistoryItem | summary v-html XSS | Security Concern | 需验证 |
| F-015 | ResumeImport | marked v-html XSS | Security Concern | 低 |
| F-016 | ModelSelect | 能力过滤函数未接 | Improvement Opp | Confirmed |
| F-017 | EmptyState | 快捷键硬编码不一致 | UX Limitation | Confirmed |
| F-018 | ThinkingBlock | duration 死逻辑 | Improvement Opp | Confirmed |
| F-019 | InitLoading/VerModal | 死代码 + 残留文案 | Maintenance Risk | Confirmed |
| F-020 | prompts.go | 提示词语言不统一 | Improvement Opp | Confirmed |

*（本节完成）*