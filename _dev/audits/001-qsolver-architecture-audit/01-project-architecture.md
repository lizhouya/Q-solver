# 01 — 项目整体架构

> 审计范围：只读。本节描述 Q-Solver 的整体分层、目录结构、模块归属与关键入口。所有结论基于当前 `dev/qsolver-custom` 分支源码。

## 1. 技术栈

| 层 | 技术 | 备注 |
|---|---|---|
| 桌面壳 | Wails v2 (`wails/v2@v2.11.0`) | 后端 Go 提供 native 能力，前端经 WebView2 渲染 |
| 后端 | Go（`go.mod` module `Q-Solver`） | 业务模块全部位于 `pkg/` |
| 前端 | Vue 3 `<script setup>` + Pinia + Vite | 源码 `frontend/src/`，build 产物 `frontend/dist` 被 embed 进二进制 |
| 渲染 | WebView2（Windows）/ WKWebView（macOS） | 主窗口默认透明背景 |
| 构建 | `wails build` / `npm run build` | 见 CLAUDE.md |

---

## 2. 三层架构总览

```
┌────────────────────────────── 前端 (Vue 3) ──────────────────────────────┐
│  App.vue ← 事件接线总入口                                               │
│  stores/     ui.js / settings.js / solution.js   (Pinia)                 │
│  services/   api.js (绑定调用) / events.js (事件总线) / theme.js        │
│  components/ TopBar · SolveView · ScreenshotDock · SettingsModal · ... │
│  utils/      markdown-latex.js · modelCapabilities.js                   │
└───────────────┬─────────────────────────────────────────────────────────┘
                │  wailsjs 绑定 (GetSettings/TriggerSolve/...) + 事件(runtime)
┌───────────────┴────────────────────────── 绑定层 (app/) ──────────────┐
│  App struct 持有所有 Manager；EmitEvent → runtime.EventsEmit             │
│  app/solve.go  截图缓冲 + 解题触发                                     │
│  app/window.go 窗口操作 (置顶/移动/焦点)                                │
│  app/resume.go  简历选择/解析/清空                                       │
│  main.go  wails.Run(options{透明窗口, 单实例锁, AlwaysOnTop})          │
└───────────────┬─────────────────────────────────────────────────────────┘
                │ 依赖注入 (pkg 内部)
┌───────────────┴────────────────────────── 业务层 (pkg/) ───────────────┐
│  config     ConfigManager (AES-256-GCM 加密配置文件) → Subscribe       │
│  llm        Service / Provider 接口 / OpenAIAdapter (OpenAI-SDK 流式)  │
│  solution   Solver.Solve (SystemPrompt 组装 + GenerateContentStream)   │
│  task       TaskCoordinator (context 取消)                             │
│  screen     CapturePreview (kbinani/screenshot)                        │
│  ImageUtil  CompressForOCR (长边≤1280, JPEG q≤90)                       │
│  shortcut   Windows 全局钩子 + KeyBinding                              │
│  platform   Windows/macOS 系统 API 封装 (幽灵模式, 置顶, 焦点)         │
│  domain     GetCategories() 70 场景 + promptMap + SystemPolicy         │
│  resume     SelectResume/ParseResume (LLM 非流式)                       │
│  state      窗口可见/穿透/焦点/移动                                     │
│  common     常量 / 环形缓冲 / 日志(dev/prod 双构建签)                  │
│  tools      (死代码)：registry/screenshot.go 无任何引用                 │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 3. 目录职责速查

| 路径 | 职责 | 关键文件 |
|---|---|---|
| `main.go` | Wails 应用入口：窗口尺寸/透明/置顶/单实例锁 | — |
| `app/` | Wails 绑定层（前端 JS 可直接调用的方法） | app.go, solve.go, window.go, resume.go, config.go |
| `pkg/config` | 配置模型 + 加密持久化 + 订阅 | config.go, manager.go, crypto.go |
| `pkg/llm` | Provider 抽象 + OpenAI 适配 + 模型/连通性测试 | service.go, openai.go, types.go, provider.go |
| `pkg/solution` | 解题主流程（组装 Prompt → 流式） | solution.go |
| `pkg/task` | 运行中任务协调（新任务自动取消旧任务） | coordinator.go |
| `pkg/screen` | 截屏 + 预览 Base64 | screen.go |
| `pkg/ImageUtil` | 图像压缩（OCR 友好） | Compress.go |
| `pkg/shortcut` | 全局快捷键钩子 + 录制 | service.go, shortcut_windows.go, delegate.go |
| `pkg/platform` | Windows / macOS 系统 API | platform_windows.go, platform_darwin.go |
| `pkg/domain` | 场景分类 + 70 条提示词 | categories.go, prompts.go |
| `pkg/resume` | 简历导入/解析 | service.go |
| `pkg/state` | 窗口隐身/穿透/聚焦/移动 | manager.go |
| `pkg/common` | 常量 / 环形缓冲 / 日志 | constants.go, ringBuffer.go, logger_dev.go, logger_prod.go |
| `pkg/tools` | **死代码**：tool 注册库，无任何调用方 | registry, screenshot.go, tool.go |
| `frontend/src/` | Vue 3 前端全部源码 | 见 02 节 |
| `wailsjs/` | Wails 自动生成的绑定 shim | `frontend/wailsjs/go/app/App.js` |

---

## 4. 运行时生命周期

### 4.1 后端启动

```mermaid
sequenceDiagram
    participant M as main.go
    participant A as App.NewApp
    participant C as configManager
    participant S as stateManager
    participant L as llmService
    participant Sol as Solver
    participant Sk as shortcutService
    participant SkM as shortcut.Manager

    M->>A: NewApp() 
    A->>A: configManager/sample managers（图 1）
    M->>M: wails.Run(Options)
    M->>A: Startup(ctx)
    A->>C: Load() 解密读配置
    A->>L: NewService(Get(), configManager)
    L->>L: updateProviderLocked() → NewOpenAIAdapter
    A->>Sol: NewSolver(llmService.GetProvider())
    A->>SS: NewService(App, shortcuts, subscribe)
    SS->>A> : Manager.Start() 装钩子（Windows）
    A->>C: onConfigChanged(Subscribe 注册)
    A->>S: UpdateInitStatus(StatusReady)
```

### 4.2 关键入口点

| 入口 | 位置 | 触发 |
|---|---|---|
| `TriggerScreenshot()` | `app/solve.go:13` | 全局快捷键 F8 / 前端 Toolbar |
| `TriggerSend()` | `app/solve.go:77` | 快捷键 Ctrl+J / 按钮 |
| `TriggerSolve()` | `app/solve.go:127` | **仅委托接口；未绑定到任何快捷键 handler** |
| `UpdateSettings(json)` | `app/config.go` | 前端保存设置 |
| `SetWindowAlwaysOnTop(bool)` | `app/window.go:31` | 前端置顶开关 |
| `CancelRunningTask()` | `app/solve.go:155` | 前端中断按钮 |

---

## 5. 运行时数据流（一条解题链路）

```
用户按 F8
  → shortcut.Manager.keyboardHookProc 判定 → service.handleTrigger("screenshot")
  → app.TriggerScreenshot()
      · 检查 APIKey / Model / HasRunningTask / buffer<3
      · screen.CapturePreview(q,s,g,n,mode) → Base64 JPEG dataURL
      · screenshotBuffer append → EmitEvent("screenshot-taken")
用户按 Ctrl+J / 手动发送
  → app.TriggerSend()
      · buffer 为空则自动截图
      · 拷贝 buffer → 清空 buffer
      · EmitEvent("start-solving") + EmitEvent("user-message", first)
      · task.StartTask("solve") → goroutine → solveInternal
  → solution.Solve(ctx, req{Config, Screenshots})
      · 组装 system prompt（SystemPolicy + Domain + Resume）
      · 组装 user messages（多模态 image parts）
      · llmProvider.GenerateContentStream(messages, onChunk)
      · 流式 emit: solution-stream-thinking / solution-stream-chunk
      · 完成 emit("solution", fullContent)
→ 前端 events.js(runtime.On) 分发到 solution store
  → 渲染 markdown+Katex
```

（详见 03-call-chain.md 完整链路）

---

## 5. 默认值与配置首发差异（默认值设计）

| 配置项 | 后端默认 (config.go) | 前端默认 (settings.js) | 备注 |
|---|---|---|---|
| baseURL | `https://api.openai.com/v1` | 同 | 一致 |
| model | `""` | `""` | loadSettings 时若空自动 fetch 取第一 |
| prompt | `""` | `""` | **前端 buildConfigToSave 硬编码为 `""`，后端字段存在但 UI 不可编辑** |
| domainId | `general-assistant` | `general-assistant` | 一致 |
| screenshotMode | `fullscreen` | `window` | **前后端默认不一致**（影响首次配置未保存时的行为） |
| compressionQuality | `92` | `80` | **默认不一致** |
| sharpening | `0.3` | `0` | **默认不一致** |
| grayscale | `false` | `true` | **默认不一致** |
| opacity | `1.0` | `1.0` | 前端以 `transparency` 逆映射 |
| keepContext | `false` | `false` | 前端的轮次结构展示与后端一次性请求的错位 |

> 详见 findings（F-006 / F-007）。

---

## 6. 环境差异

| 项 | Windows | macOS |
|---|---|---|
| 快捷键 | VK 码全局钩子（`WH_KEYBOARD_LL`） | `Cmd+数字`（预置，不可录制） |
| 置顶 | `SetWindowLevel` 空实现（Ghost 模式中处理） | `C.SetWindowLevelC` 有效 |
| 截屏权限 | 无需请求 | 需 `ScreenCaptureKit`、权限设置 |
| 透明 | `WebviewIsTransparent/WindowIsTranslucent: true` | `WebviewIsTransparent: true, WindowIsTranslucent: false` |
| Ghost(隐身) | `WDA_EXCLUDEFROMCAPTURE` 降级 `WDA_Monitor` | 方案不同 |

---

*（本节完成。剩余展示组件细节补充到 02 或 04。）*