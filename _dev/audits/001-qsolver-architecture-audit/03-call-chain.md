# 03 — 核心调用链

> 只读审计。本节用调用链（时序/序列）描述系统的关键路径，重点：
> ① 解题数据流（截图→发送→LLM→渲染）
> ② 配置流（保存→后端生效→Provider/快捷键重建）
> ③ 快捷键流（全局钩子→动作分发）
> ④ 模型/Provider 流
> ⑤ 隐身/穿透状态流
> ⑥ KeepContext 历史流（含失效点）
> 每条链标注「实际行为 vs 形态设计」的差异。

---

## 1. 主链路：截图 → 发送 → LLM → 渲染

### 1.1 截图（快捷键 F8 / Toolbar）

```
[全局 HOOK] WH_KEYBOARD_LL
  → globalManager.keyboardHookProc()
  → shortcut.Manager.OnTrigger(action)   [判定 comboID 命中]
    → service.handleTrigger("screenshot")
      → delegate.TriggerScreenshot()        [delegate = App]
        → app.TriggerScreenshot()                       (solve.go:13)
            ├─ 校验: APIKey 非空, Model 非空, !HasRunningTask, buffer<3
            ├─ screen.CapturePreview(q,s,g,n,mode)      (屏全/窗)
            │     └─ kbinani/screenshot.Capture
            │     └─ (noCompression ? PNG : CompressForOCR→JPEG)
            ├─ screenshotBuffer = append(buffer, base64)
            └─ EmitEvent("screenshot-taken", b64, count)
                └─ frontend: ScreenshotDock.push(b64)   (第 43 行)
```

### 1.2 发送（快捷键 Ctrl+J / 按钮）

```
manager.OnTriggerPress("send")
  → service.handleTrigger("send")
    → delegate.TriggerSend()
      → app.TriggerSend()                                                   (solve.go:77)
          ├─ 校验 APIKey/Model
          ├─ buffer 空 → 自动 CapturePreview 补 1 张
          ├─ if HasRunningTask → toast 忽略
          ├─ copies := append([]string, buffer...); buffer = nil           (清空)
          ├─ EmitEvent("start-solving")
          ├─ EmitEvent("user-message", screenshots[0])
          └─ ctx, id := task.StartTask("solve")
             go func() { defer CompleteTask(id); a.solveInternal(ctx, copies) }()
```

### 1.3 解题（后端 Solver）

```
app.solveInternal(ctx, screenshots)
  → solution.Solver.Solve(ctx, req{Config, Screenshots}, cb)               (solution.go:45)
      ├─ systemPrompt = GetSystemBehaviorPrompt()  (XML SystemPolicy)
      ├─ 若 DomainId → 追加 <ScenePrompt> (GetPrompt)
      ├─ 若 ResumeContent → 追加 <CandidateProfile>
      ├─ userParts = for each screenshot: llm.ImagePart(screenshot)
      ├─ messagesToSend = [system, user]      ★ 从不读 chatHistory
      ├─ EmitEvent("solution-stream-start")
      ├─ provider.GenerateContentStream(ctx, messagesToSend, onChunk)
      │     └─ openai.go: NewStreaming(Chat.Completions)
      │          for stream.Next():
      │            ├─ delta.ExtraFields["reasoning"|"reasoning_content"] → handleThinkingChunk
      │            │    → EmitEvent("solution-stream-thinking", content)
      │            └─ delta.Content → EmitEvent("solution-stream-chunk", content)
      ├─ 错误: solution-error(err)/ "solution-error(context canceled)"
      └─ 成功: EmitEvent("solution", fullContent)
             → chatHistory = []  (第137行, F01)
```

### 1.4 渲染

```
frontend: solution-store
  on("solution-stream-start")   → handleStreamStart(keepContext)
  on("solution-stream-thinking")→ handleThinkingChunk(token)
  on("solution-stream-chunk")  → handleStreamChunk(token)  [rAF 节流 → renderMarkdownWithLatex]
  on("solution")               → handleSolution(data)
```

---

## 2. 设置保存流

```
前端 设置保存
  settingsStore.saveSettings()
    ├─ 若 model 空 → fetchModels
    ├─ buildConfigToSave(tempSettings, tempShortcuts)
    │     └─ prompt: ''                              (F07 硬编码)
    ├─ Object.assign(shortcuts, tempShortcuts)
    └─ api.syncSettings(JSON.stringify(config))
        → app.UpdateSettings(json)
            → configManager.UpdateFromJSON(json)
                → 保存解密写入文件
                → 触发 configManager.Subscribe 回调 (按注册顺序)
frontend 读取
  → settingsStore.loadSettings()
    → api.getSettings() → configManager.Get() (解密)
    → applyConfig(config)  处理默认值兜底（F07 的 `||` 掩盖问题）
```

### 2.1 配置变更的订阅树

| 订阅者 | 回调做什么 |
|---|---|
| `llm.Service` (service.go:26) | `updateProviderLocked()` → `NewOpenAIAdapter(&cfg)` 重建 provider |
| `app.onConfigChanged` (app.go:74) | `solver.SetProvider(new)`；若 `!KeepContext` → `solver.ClearHistory()` |
| `shortcut.Service` 注册的闭包 (service.go:30) | 复制 `newConfig.Shortcuts` 到 Manager |

> 事件顺序：`app.go:63` 先 `shortcut.subscribe`，后 app.go:70 `configManager.Subscribe(a.onConfigChanged)` → 保存后快捷键先于 solver 更新。均同步执行。

---

## 3. 快捷键：录制 / 触发 / 冲突

### 3.1 录制（Windows）

```
前端 设置-快捷键 → StartRecording(action)
  → app.StartRecordingKey(action)
    → shortcut.Service.StartRecording(action)
      → manager.StartRecording(action)
        → manager.mode = recording; OnRecord
   按键触发
      → manager.OnTriggerPress(action 命中)
      → service.handleRecord(action, keyName, comboID)
         → EmitEvent("key-recorded")
      → OnRecordingComplete → service.handleRecordingComplete
         ├─ 检查冲突
         └─ 无冲突 → EmitEvent("shortcut-saved")
前端: tempShortcuts[action] = {keyName, comboID}
```

### 3.2 动作分发缺失点

```
service.handleTrigger(action) (service.go:69-99):
  case screenshot / send / delete / toggle / clickthrough / move_up...move_right / scroll_up / scroll_down
  ✗ 无 "solve" —— 传入 "solve" 时 switch 无匹配，静默丢弃（F02）
```

---

## 4. 模型 / Provider 流

```
前端设置-模型 tab
  | 保存 → UpdateSettings
  └> llm.Service.subscribe → updateProviderLocked → NewOpenAIAdapter(&cfg)

拉模型列表：
  api.getModels(apiKey) → app.GetModels(apiKey)
    → llm.Service.GetModels(ctx, apiKey)
        ├─ if apiKey 与当前不同 → 临时 adapter（NewOpenAIAdapter 用临时 cfg）
        └─ else 复用当前 provider
连通性测试：
  api.testConnection(apiKey, model) → app.TestConnection(...)
    → llm.Service.TestConnection(ctx, apiKey, model)
        ├─ 临时 config 副本 + NewOpenAIAdapter
        └─ testChat (MaxTok 17, 15s 超时)
```

---

## 5. 隐身 / 窗口控制链

```
初始化：
  stateManager.Startup → 轮询 GetWindowHandle(hwnd) → 找到后 ApplyGhostMode(hwnd)
    → 立即设 WDA_EXCLUDEFROMCAPTURE, 窗口样式 (WS_EX_TOOLWINDOW|NOACTIVATE|TOPMOST)
    → 通知 state.StatusReady

切换可见性 (F9 / 按钮)
  → shortcut → service.handleTrigger("toggle") → delegate.ToggleVisibility()
    → stateManager.ToggleVisibility()
       → SetDisplayAffinity(hwnd, false/true)
    → EmitEvent("toggle-visibility", visibleToCapture)   [前端更新 isStealthMode + toast]

鼠标穿透 (F10)
  → shortcut → handleTrigger("clickthrough") → delegate.ToggleClickThrough()
    → stateManager.ToggleClickThrough()
       → setWindowClickThrough(hwnd, enabled)
        → EmitEvent("click-through-state", enabled)

移动 / 滚动
  move_* → stateManager.MoveWindow(dx, dy) → runtime.WindowGetPosition + WindowSetPosition
  scroll_* → stateManager.ScrollContent(dir)

置顶 (toolbar 开关)
  → app.SetWindowAlwaysOnTop(bool)
    → platform.SetWindowLevel(hwnd, level)
       ├─ Windows: 空实现 (F08, 无效)
       └─ macOS:  SetWindowLevelC 生效
```

---

## 6. KeepContext 历史链（失效点）

### 形态设计（若启用了 KeepContext）

```
第1轮: [system, user1] → LLM
第2轮: [system(+更新), user1, assistant1, user2]  ← 需 chatHistory 累积
第N轮: trimChatHistory 保留最近 MaxConversationRounds*2+1 条
```

### 实际行为（当前源码）

```
第1轮：Solve() 内 build messagesToSend=[system, user1]
        → GenerateContentStream
        → 完成后 chatHistory = []   (solution.go:137)
第2轮：再次 Solve → chatHistory 始终为空 → 又是 [system, user2]
  → 后端永远不携带历史；MaxConversationRounds/ensureSystemPrompt/trimChatHistory 均为死代码
  → 但前端 solution store 保留 rounds (多轮显示)；App.vue keepContext+rounds>1 分支弹出下一轮
  → 前端展示为多轮 → 实际请求是独立的单轮
```

**用户视角**：勾选 keepContext 后，界面新增 round 卡片，但模型每次只看到新截图，不记得上一轮问答内容。属于**功能级设计失效**，需列入 findings。

---

## 7. 小结（调用链中的失效点速查）

| 链 | 期望 | 实际 | 关联 Finding |
|---|---|---|---|
| KeepContext 多轮 | 携带历史多轮 | 后端每次清空 | F01 |
| solve 快捷键 | Cmd/solve 触发解题 | handleTrigger 无 solve case | F02 |
| PDF 输入 | 支持 PDF 内容 | 当图片发 | F03 |
| Windows 置顶开关 | 可切换 | SetWindowLevel 空实现 | F08 |
| 配置保存 prompt | 保留/可编辑 | 前端无 UI + 保存清空 | F07 |
| 冻结截图压缩配置 | 前端 `||` 覆盖 | config 值为 0/空被前端默认覆盖 | F06 |

---
*（本节完成）*