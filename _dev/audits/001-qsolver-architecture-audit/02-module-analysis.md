# 02 — 模块分析

> 只读审计。本节逐一分析后端 `pkg/` 与前端 `stores/`、`services/`、`components/` 各模块的职责、内部结构与对外接口。所有结论基于当前源码。

---

## 1. 后端模块

### 1.1 `pkg/config` — 配置管理

**职责**：配置模型、加密持久化、变更订阅。

| 文件 | 关键内容 |
|---|---|
| `config.go` | `Config` 结构体（全部带 `omitempty`）；`NewDefaultConfig()`；`Validate()` 校验 screenshotMode/opacity/compressionQuality；`getDefaultShortcuts()` 分平台 |
| `manager.go` | `ConfigManager`：`Load()`（AES 解密 + JSON）、`Save()`（加密 + 写文件）、`Patch()`、`UpdateFromJSON()`、`Subscribe()`、`Get()` |
| `crypto.go` | **`init()` 对硬编码字符串 `"JTWZwHaMq3g55PfVgxka1Edst59LoyF"` 做 SHA256 得到 32 字节 key → AES-256-GCM**。密钥固定、可逆，仅混淆非真正的保护 |

**关键设计**：
- 配置文件路径：`os.UserConfigDir()/Q-Solver/config`
- 前端 `UpdateSettings(json)` 走 `UpdateFromJSON` → 触发所有 Subscriber
- 订阅者：`llm.Service`（重建 Provider）、`solution.Solver`（SetProvider + ClearHistory）、`shortcut.Service`（更新键绑定）

**特点**：
- 配置缺省值由 `NewDefaultConfig` 保证；`Validate()` 只校验三个字段
- 无沙箱/距离校验；config 文件为本地明文（加密存储但密钥硬编码）

---

### 1.2 `pkg/llm` — LLM 层

**核心文件**：
- `types.go`：`Message`、`ContentPart`（Text/Image/PDF 三型）、`StreamChunk{Thinking|Content}`、构造器 `ImagePart`/`PDFPart`
- `provider.go`（interface）：`Provider` 含 `GenerateContentStream`、`GenerateContent`、`GetModels`、`TestChat`
- `service.go`：`Service` 持锁管理 `provider`；config 变更时 `updateProviderLocked()` 调 `NewOpenAIAdapter(&config)` 重建 adapter
- `openai.go`：OpenAI SDK 适配；`toOpenAIMessages` 把 `ContentImage`/`ContentPDF` **统一转成 `ImageContentPart`（图片 URL）**；`GenerateContentStream` 流式读 `delta.ExtraFields["reasoning"`] 或 `["reasoning_content"]` 解析 Thinking；`GetModels` 拉模型列表；`TestChat` MaxTokens=17

**关键设计**：
- Provider 靠 config.Subscribe 重建 → 换 API Key/BaseURL/模型即时生效
- Streaming 对 thinking 有 `handleThinkingChunk` (JSON raw → string)
- **PDF 无独立支持**：`ContentPDF` 在 `toOpenAIParts`（openai.go:75-77）被当**图片**发（base64 data URL 传给 image_url），语义上并不真正发送 PDF 内容 —— 见 F-xxx

---

### 1.3 `pkg/solution` — 解题主流程

**核心文件**：`solution.go`
- `Solver` struct：字段 `llmProvider`、`chatHistory []llm.Message`
- `Solve(ctx, req, cb)`:
  - 组装 system prompt：SystemPolicy + `<ScenePrompt>`（按 DomainId 注入）+ `<CandidateProfile>`（简历内容）
  - 组装 user messages：`ImagePart`(screenshot)（多图多 part）
  - `GenerateContentStream` 流式 emit `solution-stream-thinking` / `solution-stream-chunk`
  - 完成 emit `solution` / 错误 emit `solution-error`
- `ClearHistory()`、`ensureSystemPrompt()`、`trimChatHistory()` 均有实现

**关键发现（F01）**：`chatHistory` 字段 + 三个相关方法均实现，但 **`Solve()` 第 84-86 行直接新建 `messagesToSend`，从不读取 `s.chatHistory`**；第 137 行在返回前**强制 `s.chatHistory = []llm.Message{}` 重置**。`ensureSystemPrompt`/`trimChatHistory` 从未被调用。 → **KeepContext 后端完全无效**，每轮都是「system + 新截图」一次请求；`MaxConversationRounds=10` 成为死常量。

---

### 1.5 `pkg/task` — 任务协调

**`coordinator.go`**：`TaskCoordinator` 用 context cancel 管理「当前任务」。
- `StartTask(name)` → 取消旧任务（Cancel）、返回新 ctx + id
- `CompleteTask(id)` → 标记完成（只认当前 id）
- `CancelCurrentTask()` → 取消
- `HasRunningTask()` → 是否在运行

实现简洁，无协程池/队列/超时，仅单当前任务。

---

### 1.6 `pkg/screen` + `pkg/ImageUtil` — 截图视觉链路

- `screen.go`：`CapturePreview(q,sharpen,grayscale,noCompression,mode)` → `screenshot.Capture`（kbinani），noCompression → PNG 否则 `CompressForOCR`
- `ImageUtil/Compress.go`：`ScreenshotMaxDimension=1280`，`CompressForOCR` resize 长边 ≤1280、gray、**sharpen**、JPEG quality **clamp 到 [1,90]**（即使 Config quality 传 92 也被截到 90）

---

### 1.7 `pkg/shortcut` — 全局快捷键

**`service.go`**：
- `NewService` 注入 `Manager` + `ServiceDelegate`，创建 `handleRecord`、`handleRecordingComplete`
- `handleTrigger(action)` —— **`switch` 只有 screenshot / send / delete / toggle / clickthrough / move_*/scroll_*，没有 `"solve"` case**（F02：macOS 默认的 `solve: Cmd+1` 不生效，前端 `solveShortcut` 却回退到 `screenshot` 或 `solve`）
- `handleRecordingComplete` 检查冲突并重启录制
- **`TriggerSolve` 在 delegate 接口存在（delegate.go:5），但 handleTrigger 从不调用**

**`shortcut_windows.go`**：`globalManager` 单例；`Start`（装 WH_KEYBOARD_LL/WH_MOUSE_LL 钩子 + Msg 循环）、钩子处理判定。

---

### 1.8 `pkg/platform` — 平台能力

**Windows (`platform_windows.go`)**：
- `GetWindowHandle`（轮询找 hwnd）
- `applyGhostMode`：移除 caption/thickframe，加 WS_EX_TOOLWINDOW/NOACTIVATE/TOPMOST；`WDA_EXCLUDEFROMCAPTURE`（失败降级 `WDA_Monitor`）
- `SetClickThrough`、`SetDisplayAffinity`、`setWindowClickThrough`
- `RestoreFocus/RemoveFocus`：`WS_EX_NOACTIVATE` 增删
- **`SetWindowLevel(hwnd, level)` → 天栈 `nil`（注释「Windows 在 ApplyGhostMode 中处理」）** —— `app/window.go:31` 的 `SetWindowAlwaysOnTop` 调用它，导致**置顶开关在 Windows 无效**（F08）
- `isWindows11OrLater`（build ≥22000），Win10 无透明渐变

**macOS (`platform_darwin.go`)**：`SetWindowLevel` 用 `C.SetWindowLevelC` 有效实现。

---

### 1.9 `pkg/domain` — 场景/提示词系统

- `categories.go` / `prompts.go`：`GetCategories()` 返回 7 大分类（通用/编程笔试/产设笔试/市场运营/学术考试/商业证书/MBTI），其中 **9+9+9+9+9+9+16 = 70 个场景**，每个有 `PersonaCard` 风格提示词
- `GetSystemBehaviorPrompt()` 返回 XML `<SystemPolicy>`（中文优先、Markdown、LaTeX 规则）
- `GetPrompt(id)` 未命中 fallback `gen-solver`
- 前端域选择只传 `domainId`，不直接拿 prompt 文本

---

### 1.10 `pkg/state` — 状态/隐身

`manager.go`：
- `initWindowHandle` 轮询 hwnd → `applyGhostMode`
- `ToggleVisibility`：`SetDisplayAffinity` false/true
- `ToggleClickThrough`：`setWindowClickThrough`
- `RestoreFocus`/`RemoveFocus`：去掉/加 `WS_EX_NOACTIVATE`
- `MoveWindow(dx,dy)`：`runtime.WindowGetPosition` + `SetWindowPosition`

---

### 1.11 `pkg/resume` — 简历

`service.go`：`SelectResume`（内存文件对话框）、`ParseResume` 用 `llm.NewOpenAIAdapter`（非流式 `GenerateContent`）解析为 Markdown，**PDF ≤ 5MB**，`resume.ParsePrompt`。

---

### 1.12 `pkg/common`

- `constants.go`：`AppName=Q-Solver`
- `ringBuffer.go`：环形缓冲
- `logger_dev.go` / `logger_prod.go`：**`//go:build !prod` / `//go:build prod`** —— dev 版真正打印日志，prod 版静默（即发布构建用 `-tags prod` 可去掉 API Key 等日志）。**注意 dev 构建会在日志中打印 system prompt 含简历内容**

---

## 2. 前端模块

### 2.1 `stores/ui.js` — UI 状态
toasts / activeButtons(flash) / isClickThrough / mainVisible / isStealthMode / hasStarted / showSettings / activeTab / availableModels / initStatus / showResumeWarning / quit

### 2.2 `stores/settings.js` — 设置状态机（核心）
- `settings` 反映已保存配置；`tempSettings` 为编辑草稿（打开设置时 `Object.assign` 拷贝）
- `loadSettings()`：并行 `getSettings()` + `getDomainCategories()` → `applyConfig()`
- `applyConfig(config)`：
  - `apiKey` / `model` / `domainId` / `keepContext` 等直接读取
  - **压缩/灰度/模式用 `||` 兜底默认**（如 `compressionQuality = config.compressionQuality || 80`），**若后端返回 0 会被前端默认覆盖**（F07）
- `buildConfigToSave()`：**`prompt: ''` 硬编码**（第 263 行）→ 前端没有 prompt 编辑 UI，保存时强制把后端字段清空（F07）
- `resetStatus()`：只有 api key 有无 → `未配置 / 已连接`
- 保存触发 `api.syncSettings(JSON)`；主题变更会静默保存

### 2.3 `stores/solution.js` — 解题状态
- history[{ time, rounds[{userScreenshot, thinking, aiResponse, error}] }]
- `handleStreamStart(keepContext)`：keepContext 时向 index 0 的 item 添加 round（**仅前端轮次展示**，后端不接历史）
- `handleStreamChunk` rAF 节流渲染 markdown；`handleThinkingChunk` 检测 stalling
- `exportImage`：`html2canvas` DOM 快照导出 PNG
- 关键设计：round 数据结构已是「多轮」形态，但后端每轮独立

### 2.4 `services/api.js`
wailsjs 绑定的薄封装：`triggerSolve`→`TriggerSolve()`（内部 = `TriggerSend`）、`getSettings`、`syncSettings`、`getModels`、`testConnection`、`cancelTask`、`triggerScreenshot`、resume 三件套、窗口控制、截图预览

### 2.5 `services/events.js`
`on/off/onMany` 事件分发（基于 `runtime.EventsOn`），注册隔离

### 2.6 前端组件层（扫描结果待补充部分见文件末）

*以下组件为**展示/交互**层，职责如下，详细发现见 04-findings.md*

| 组件 | 职责 |
|---|---|
| `TopBar.vue` | 标题栏 + 状态点 + 状态面板 + 主题切换 + 设置/退出 |
| `ScreenshotDock.vue` | 截图缩略图 dock（计数/删除/发送提示） |
| `SettingsModal.vue` | 5 tab 设置弹窗（常规快捷键/透明度、模型、截图、简历、API） |
| `ProviderSelect.vue` | 模型服务商选择（预设 URL / custom） |
| `ModelSelect.vue` | 模型列表下拉（含能力过滤） |
| `DomainSelector.vue` | 场景分类选择（70 选项） |
| `WelcomeView.vue` | 首页占位 |
| `SolveView.vue` | 主解题视图（历史/对话） |
| `EmptyState.vue` / `LoadingView.vue` / `ErrorView.vue` | 状态占位 |
| `HistoryItem.vue` | 历史单条 |
| `RoundDivider.vue` | 多轮分隔线 |
| `ActionBar.vue` | 操作按钮（发送/删除） |
| `ThinkingBlock.vue` | 思考链折叠块 |
| `ResumeImport.vue` | 简历导入/解析 UI |
| `ThemeToggle.vue` | 主题切换 |
| `Icon.vue` | SVG 图标 |
| `ResizeHandle.vue` | 调整大小手柄 |
| `VersionUpdateModal.vue` / `InitLoading.vue` | 更新/初始化弹窗 |

> ⚠️ 预研发现：`IncludesResumeImport.vue` 已在（`SettingsModal` 使用），`WelcomeView`/`EmptyState`/`LoadingV`/`ErrorView` 属于展示类。

---
*（本节完成。组件扫描子代理结果返回后，若有需要补充处，追加到对应小节。）*