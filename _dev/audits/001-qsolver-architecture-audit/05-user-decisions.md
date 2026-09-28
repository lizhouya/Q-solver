# 05 — 产品决策问题（User Decisions）

> 只读审计。本节**仅整理**审计过程中发现的、**非技术**的产品设计决策问题 —— 它们需要 Q-Solver 的负责人/产品来做决定，本报告**不做选择**，仅列出问题与依据（源码证据）。
>
> 技术层面的问题见 04-findings.md；这里聚焦「要做什么产品」的决策。

---

## D-1 产品定位：截图解题工具 vs 考试辅助工具？

**证据（Observed）**：
- 70 个场景中 47 个为考试（编程/产设/市场/学术/商业），如 `dev-java-exam`（prompts.go:332）、`aca-gongkao-exam`（prompts.go:793）、`biz-pmp-exam`（prompts.go:1015）
- `gen-grader`「作业批改」（prompts.go:278）、`gen-similar`「举一反三自动出 3 道变式」（prompts.go:229）
- MBTI 16 型提示词（prompts.go:1103-1263）—— 与「解题」无关，像是「角色扮演」扩展

**决策点**：产品在「通用解题」与「求职/备考答题助手」两个定位上叠加了 MBTI 角色扮演，当前 MBTI 分类与核心场景的产品逻辑无关。二次开发需明确目标市场。

---

## D-2 「保持上下文」（KeepContext）的预期行为

**Observed**：
- 前端枚举 `keepContext`（settings.js 无 UI 开关？实际在 SettingsModal general tab 无；但 App.vue:152 使用 `settings.keepContext`）
- 后端 `KeepContext` 配置存在（config.go:20），`chatHistory` 也实现了但未接（F-001）
- 前端 `solution.js:125` `keepContext && history[0].rounds` 追加 round 卡片

**决策点**：产品期望的「多轮对话」到底是：
a. 模型能看到前面的对话（真正的多轮语义）
b. 仅 UI 上把一个问题的多次发送并排展示（当前形态）
后者是当前实际行为（后端不携带历史）。需产品明确意图，否则 keepContext 设置对用户是误导。

---

## D-3 隐身模式（幽灵模式）与「低打扰」的边界

**Observed**：
- 主窗口默认透明（main.go:38/45-47）
- `toggle-visibility` 事件提醒「隐身模式开/关」（App.vue:165-169）
- `SettingsModal` 顶部 banner「打开设置会获取焦点，录屏期间请避免操作」（SettingsModal.vue:5）

**决策点**：
- 用户可以接受的「隐身」边界到哪里？（彻底不入屏 vs 可被录屏但不可见）
- 打开设置时的焦点行为（当前会打断隐身状态，靠文案提醒）是否可接受
- 这是一个 UX + 功能期望问题

---

## D-4 场景的维护模式

**Observed**：
- 70 条 OpenAI 提示词硬编码在 Go（`prompts.go`），修改需重新编译（不是配置化/动态读取）
- 前端 `DomainSelector` 通过 `GetCategoryCategories` 动态展示，但提示词内容在后端固定

**决策点**：
- 场景库应作为**编译期**的（跟随版本）还是**运行时**（配置文件/远程）可维护？
- 决定直接影响二次开发的迭代成本与协作方式

---

## D-5 简历能力的边界

**Observed**：
- `ResumeImport` 支持导入 PDF → LLM 解析 Markdown → 存入配置作为 `<CandidateProfile>` 注入（solution.go:67-73）
- 只有本地单份简历；PDF ≤ 5MB 限制（resume/service.go）
- 解析结果可编辑（ResumeImport.vue 编辑区）

**决策点**：
- 简历是「一次性解题上下文」还是「持久的候选画像」？（当前每次解题都会注入）
- 是否应支持多简历切换、脱敏、本地加密问题

---

## D-6 模型策略：

**Observed**：
- 默认 BaseURL 为 OpenAI，但支持用户自定义 Provider（ProviderSelect）
- ModelSelect 列出用户 API key 下全部模型，不做能力过滤（F-016）
- `assistantModel` 字段预留（F-009）

**决策点**：
- 默认模型策略：保持 OpenAI 默认 vs 适配国内常用模型（qwen/deepseek 等）作为开箱默认？
- 是否按能力（vision）过滤模型列表，避免选到图片不支持的模型？（这是 UX 决策）
- 是否利用 `assistantModel` 做轻量总结/导图（字段已预留但未实现）

---

## D-7 提示词的「英语工作流」问题：

**Observed**：
- 场景提示词为人称固定英文 `<PersonaCard>`，而 SystemPolicy 中文（F-020）
- 提示词中大量使用「OCC reference」「RICE」等框架名词，如 `prod-pm-exam` 用 CIRCLES/SWOT/RICE/MoSCoW（prompts.go:488-504）

**决策点**：面向中文用户的场景提示词以**混合语言**驱动模型，影响输出的稳定性和术语准确度。产品要选择：统一中文、统一英文、还是保留英文框架术语？

---

## D-8 输入形态：截图数量与文本

**Observed**：
- 截图最多 3 张（`MaxScreenshots=3`，app/solve.go:9）
- 无文本输入框（只有截图触发解题）
- 简历解析用的是「PDF→LLM markdown」而非视觉传 PDF（F-003）

**决策点**：是否增加纯文本输入模式（如粘贴题目）？当前产品「只能截图解题」，对无截图的场景（打字题、代码粘贴）有局限

---

## D-9 「发送即解题」的行为

**Observed**：
- `TriggerSend` 在 buffer 空时**自动截图**（solve.go:91-105）— 用户无需 F8 即可发送，会自动补一张全屏
- 这可能在用户未预期时抓了全屏

**决策点**：自动补图的默认行为是否符合预期？（保护隐私考虑：无意触发时全屏被抓）

---

## D-10 首次体验 / 欢迎屏

**Observed**：
- 欢迎屏 WelcomeView + 初始化 InitLoading/VersionUpdateModal（后两者死代码 F-019）
- 前端对首次配置的引导：`require-api-key` 事件自动弹设置 + 跳到 api tab（App.vue:247-251）

**决策点**：首次启动流程（配置 API Key → 选择模型 → 选择场景）的产品引导力度、是否需要 step-by-step，取决于目标用户技术程度

---

## D-11 付费/计费考虑（模型成本）

**Observed**：
- 每个截图（≤1280 长边 JPEG q≤90）经 base64 发给模型，多张会显著增加 token（视觉）
- 思考链流式输出（无需额外成本但对用户可见）

**决策点**：是否要做用量/成本提示（当前无）；是否做截图尺寸/压缩的策略以控制成本 — 是产品取舍。

---

> 以上 D-1~D-11 共 11 项为**产品决策问题**，均非技术债。二次开发前请由产品/负责人逐条拍板方向。（本审计不做选择。）

*（本节完成）*