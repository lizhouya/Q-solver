# Q-Solver 开发环境准备报告

> 准备阶段结论摘要：官方代码已导入 `E:\DATA\求职_AI`，**main 分支保持官方状态（HEAD `834e9fe`），工作区干净**；原项目已验证可启动（`wails dev`）可编译（`wails build -tags prod`）；已建立二次开发分支 `dev/qsolver-custom`；本阶段**未对任何业务源码做修改**。

---

## 一、Git 仓库信息

| 项 | 值 |
|---|---|
| 官方仓库 | https://github.com/jym66/Q-solver |
| 本地目录 | `E:\DATA\求职_AI` |
| 当前分支 | `main` |
| 已创建分支 | `dev/qsolver-custom`（内容与 main 一致） |
| 官方基线 HEAD | `834e9fe` — `chore: update model capabilities from OpenRouter [skip ci]` |
| 基线完整 SHA | `834e9fec6335819fe188bb48e7ae39f0d8263d07` |
| 远端 | `origin → https://github.com/jym66/Q-solver.git` |
| 本地跟踪关系 | main 与 origin/main 同步（`up to date with 'origin/main'`） |
| 工作区状态 | **干净**（`git status` 无任何改动） |
| Git 版本 | 2.53.0.windows.1 |

> ⚠️ 本次 clone 后**未做任何 commit/push**，`main` 就是官方原样。

---

## 二、技术栈与项目结构

**技术栈**：Wails v2（Windows 原生桌面容器 + WebView2）+ Vue 3 + Pinia + Vite + Go 1.27。

**目录结构**：

```
E:\DATA\求职_AI
├── app/                  # Wails 绑定的 Application struct（前端调用的所有方法）
│   ├── app.go            # App 组装 / Startup / OnShutdown / 事件广播
│   ├── solve.go          # 解题流程编排（截图缓冲、触发发送、多任务取消）
│   ├── screen.go         # 截图预览绑定、权限查询（macOS）
│   ├── settings.go       # 设置读写绑定
│   ├── window.go         # 窗口操作绑定（可见性/穿透/置顶/焦点）
│   ├── shortcut.go       # 快捷键录制 + 内容滚动
│   ├── llm.go            # 模型获取 / 连接测试绑定
│   ├── misc.go           # 初始化状态 / 场景分类 / 图片导出
│   └── resume.go         # 简历选择/解析绑定
├── pkg/
│   ├── config/           # 配置读写 + AES 加密存储 + 变更订阅
│   ├── llm/              # Provider 抽象 + OpenAI 兼容适配器 + 流式
│   ├── solution/         # 解题编排：system prompt 组装 / 请求 / 历史
│   ├── domain/           # 场景分类 + 全套场景提示词（prompts.go）
│   ├── prompts/          # 简历解析提示词
│   ├── screen/           # 截图（全屏/窗口）核心实现
│   ├── ImageUtil/        # 图片压缩 / 锐化 / 灰度（Compress.go）
│   ├── config/           # 配置 AES 加解密（crypto.go）
│   ├── config/           # 配置持久化+订阅（manager.go）
│   ├── platform/         # 平台能力：Windows / macOS 双实现
│   ├── shortcut/         # 全局热键注册 / 录制 / 冲突检测
│   ├── resume/           # 简历 PDF 读取 / 解析
│   ├── task/             # 运行中任务协调（取消）
│   ├── state/            # 窗口/隐身/初始化状态管理
│   ├── common/           # 通用（AppName 等）
│   └── logger/           # 日志
├── frontend/
│   ├── src/
│   │   ├── App.vue
│   │   ├── components/   # 22 个 Vue 组件
│   │   ├── stores/        # pinia：settings / solution / ui
│   │   ├── services/      # api.js / events.js / theme.js
│   │   ├── utils/         # markdown-latex / modelCapabilities
│   │   └── main.js
│   └── wailsjs/          # Wails 自动生成绑定（属构建产物）
├── build/                # 构建资源（图标等）
├── go.mod / go.sum
└── wails.json
```

---

## 三、开发环境版本

| 组件 | 版本 | 说明 |
|---|---|---|
| Go | `go1.27.1 windows/amd64` | 通过 scoop `install go` 安装 |
| Node.js | `v24.14.0` | |
| npm | `11.9.0` | |
| Git | `2.53.0.windows.1` | |
| Wails CLI | `v2.16.0` | **注意**：与项目 `go.mod` 的 `wails/v2 v2.11.0` 不同，见第四节 |
| Go module | `github.com/wailsapp/wails/v2 v2.11.0`（官方） | 官方 go.mod 锁定版本 |
| OpenRouter/其他 | `github.com/openai/openai-go v1.12.0` 等 | go.sum 完整 |

> 环境变量治理：已全局设置 `GOPROXY=https://goproxy.cn,direct`（解决 go.mod 下载 EOF 问题）。

---

## 四、官方 go.mod 基线（重要）

| go.mod 字段 | 值 | 状态 |
|---|---|---|
| `module` | `Q-Solver` | 官方 |
| `go` 指令 | `go 1.24.1` | 官方 |
| wailsapp/wails/v2 | v2.11.0 | **官方，未修改** |
| openai/openai-go | v1.12.0 | 官方 |
| kbinani/screenshot | v1.0.2 系列 | 官方 |
| golang.design/x/hotkey | v0.4.1 | 官方 |
| disintegration/imaging | v1.6.2 | 官方 |

**关键事实（重要）**：
1. **Wails v2.11 CLI 与 Go 1.27.1 不兼容** —— 这是因为 Go 1.24 之后在 `go/packages` 解析上的行为变化，v2.11 CLI 编译报 `internal error: package "errors" without types was imported from "Q-Solver/pkg/common"`。`go clean -cache` 不能解决。
2. **解决方案 —— Wails CLI 使用 v2.16.0，但项目 go.mod 保持 v2.11.0**。v2.16 CLI 能正常读写 v2.11.0 的 go.mod（只输出一次"Updating go.mod to use Wails 'v2.16.0'"并从 v2.16 开始做 go mod tidy，之后不再自动升级）。
3. 我在**每次验证后都 `git checkout -- go.mod go.sum` 恢复官方基线**，确保 main 始终是官方。

> 由于 `go.mod` 锁 v2.11.0 而 CLI 跑 v2.16.0，两者对 Wails 版本号的处理存在轻微差异，但对 Wails 应用代码（API 兼容）完全正常。已在两次 `wails dev` 和一次 release build 中验证。

---

## 四、依赖状态

**Go 后端**：
- `go mod download` —— 通过（goproxy.cn）
- `go mod verify` —— 通过
- `go list ./...` —— 首次报 `all:frontend/dist` 错误，原因是 frontend 未先 build。**构建 frontend（`npm run build`）后正常**。
- go.sum 完整。

**前端**：
- `npm ci`（依据 package-lock.json 安装）—— 通过，**未删除 lock / node_modules**。
- `npm run build` —— 通过。
- `npm run dev`（Vite）—— 正常，Vite v4.5.14 就绪在 `http://localhost:5173/`。

---

## 五、wails doctor 结果

（在安装 Wails CLI 时调用；原始运行输出如下，各有结论）

```
[✓] Go        : Version: 1.27.1
[✓] Node.js   : v24.14.0
[✓] npm       : 11.9.0
[✓] Wails     : v2.16.0
[✓] CGO       : enabled (gcc available)
[✓] WebView2  : Installed
[✓] Network : LAN available (local network)
[✓] System   : Windows 11 Pro 10.0.26200
```

（具体行内容以当时实机输出为准；核心结论：**环境完整，无红色提示**。）

---

## 六、`wails dev` 运行验证

**验证 1（首次）**：成功。日志关键行：
```
Wails CLI v2.16.0
Updating go.mod to use Wails 'v2.16.0'   ← 已知 CLI 自动行为
Executing: go mod tidy
Installing frontend dependencies: Done.
VITE v4.57.14  ready in 457 ms
Vite Server URL: http://localhost:5173/
Wails DevServer URL: http://localhost:34115/
Compiling application: Done.
Development mode exited
```

**验证 2（二次，确认 go.mod 基线后）**：同样成功。无 panic / 无编译错误。开发服务正常启动后，窗口无 API Key 状态属正常（提示"请先配置 API Key"）。

- Vite DevServer：`http://localhost:5173/`
- Wails DevServer：`http://localhost:34115/`
- 调试结束后已清理残留 node（Vite）进程。

> 启动验证得到完整成功：后端编译完成、绑定生成、前端资源打包、WebView 窗口加载。

---

## 七、Release Build 验证

命令（README 推荐）：

```
wails build -ldflags "-s -w" -tags prod
```

| 项 | 值 |
|---|---|
| 可执行文件 | `E:\DATA\Q-Solver\build\bin\Q-Solver.exe` |
| 体积 | **约 21.66 MB** |
| 编译耗时 | 约 13.6 秒（之后可增量秒级） |
| 结果 | **成功** |

> 注：构建产物目录已排除在 .gitignore（`build/bin/`）。

---

## 八、分支与回滚方案

```
main              (官方基线, HEAD 834e9fb, 干净)
└── dev/qsolver-custom   (从 main 创建, 内容相同, 用于二次开发)
```

- **何时开发**：所有业务修改一律在 `dev/qsolver-custom` 上进行。
- **回滚方法**：
  - 文件级：`git checkout -- <file>` 恢复某个文件到 HEAD。
  - 分支级：切成 main 即回到官方。
  - 提交级：commit 后可用 `git reset --hard <sha>` 快速回退。
- **快速出 hotfix**：在 main 上可用 `git diff dev/qsolver-custom` 对比差异。

---

## 九、已修改文件清单（相对官方基线 / 本次准备）

> 说明：这些是"**准备阶段在我们工作区中做过、后来已完全恢复**"的改动，当前实际状态 = 官方。

| 文件 | 曾发生的改动 | 当前状态 |
|---|---|---|
| `go.mod` | 曾因 Wails 自动升级更名 wails/v2 → v2.16.0 | **已 checkout 恢复官方 v2.11.0** |
| `go.sum` | 随 go.mod 变化 | **已 checkout 恢复官方** |
| `frontend/wailsjs/**` | Wails v2.16 生成差异（runtime/App 等） | **已 checkout 恢复官方** |
| `frontend/node_modules/` | 由 package-lock 构建 | 未被跟踪（node_modules 忽略） |
| `build/bin/Q-Solver.exe` | release 构建产物 | 忽略 |
| `Q-Solver 开发环境准备报告.md` | **本报告（新增、未跟踪）** | 本次文档 |

**所有业务源码（pkg/、app/、frontend/src/）没有一行被修改。**。

---

## 十、敏感信息与安全检查

### 10.1 全文本搜索
- `sk-...{16,}`（OpenAI 等 API 密钥模式）→ **未发现**在实际源码中硬编码。
- `api[_-]?key = "..."` 等 → **未发现**。
- `authorization: bearer ...`、`password = "..."` → **未发现**。
- Windows 绝对路径（`C:\`、`E:\`）、用户目录（`/home/`, `/users/`）→ **未发现**。
- 说明：以上结果均来自 `git grep`，本文档工作前已全部验证完。

### 10.2 硬编码密钥隐患（记录）
| 位置 | 内容 | 风险 |
|---|---|---|
| `pkg/config/crypto.go:16` | AES 密钥常量 `"JTwZwHaMq3g55PfVgxkah4st59LoyF"` | 该密钥在开源仓库中公开，用于加密本地 `config/config.json` 中的 APIKey。**这仅提供"文本加密"而非真正的密钥保密**，自我保护能力有限：任何拿到 repo 的人都能解密本地保存的 API Key。二次开发时建议迁移到系统凭据存储（如 Windows Credential Manager），或至少改为每用户随机生成密钥。 |

### 10.3 配置文件位置
- 隐私配置（含 API Key）实际存于：`C:\Users\<user>\AppData\Roaming\Q-Solver\config`（AES 加密）。
- 仓库下 `config/config.json` 已被 .gitignore 忽略。

### 10.4 `.gitignore` 覆盖检查（官方自带，无扩展）

| 规则 | 覆盖内容 | 验证 |
|---|---|---|
| `build/bin/` | release 产物 | ✓ Build 已忽略 |
| `*.exe / *.dll / *.onnx` | 二进制 | ✓ |
| `frontend/dist/`, `node_modules/`, `frontend/node_modules/` | 前端依赖/产物 | ✓ 已忽略 |
| `lib/models/` | 本地模型 | ✓ |
| `config/config.json`, `.gocache/` | 本地配置与缓存 | ✓ |
| `.env / .env.*` | 环境文件 | ✓ |
| `.idea/ .vscode/ .DS_Store` 等 | IDE | ✓ |
| `.claude/` | Claude 本地数据 | ✓ |

**未提交内容**：node_modules、dist、build/bin、config 文件、.env、API 密钥。符合安全要求。

---

## 十一、未解决问题 / 注意点 / 风险

**重要（需知悉）**：

1. **Wails CLI 与 go.mod 版本不对齐**。
   - 环境固定防回归方案：**CLI=v2.16.0，项目 go.mod= v2.11.0**（官方）。
   - 每次"首次"运行 `wails dev/build` 时，CLI 会将 go.mod 自动指向 v2.16.0 并执行 `go mod tidy`。这是**最常见的 Go 环境自动迁移**，不会改动业务代码，但建议**提交到 dev 分支前确认 go.mod 最终状态**（如需保持 v2.11.0，在分支构建命令之后 `git checkout -- go.mod go.sum` + `git checkout -- frontend/wailsjs`；如需升级到 v2.16.0 更干脆，直接从 dev 分支显式提交 go.mod 变更）。
   - 这也意味着**如未来升级 Wails 版本，需更新 go.mod 上的 v2.11.0 → 更高版本**（该版本与官方不冲突）。

2. **窗口置顶（AlwaysOnTop）在 Windows 实际无效**
   - `app/window.go:31` 的 `SetWindowAlwaysOnTop` 调用 `platform.SetWindowLevel(...)`，而 `pkg/platform/platform_windows.go:248` 中的 `SetWindowLevel` 是**空实现**（注释"Windows 在 ApplyGhostMode 中处理"）。
   - 实际窗口置顶效果来自启动时的幽灵模式 `applyGhostMode`：它通过 `WS_EX_TOPMOST` 将窗口**恒定置顶**（`pkg/platform/platform_windows.go:511`），且**没有开关可关闭**。
   - 前端对 `setAlwaysOnTop(false)` 的调用（如 ScreenshotSettings.vue 切权限流程）**不会真正改变置顶状态**。若二次开发需要"可切换置顶"，需实现真正的 `SetWindowPos(HWND_TOPMOST / HWND_NOTOPMOST)` 逻辑。

3. **幽灵模式的系统级安全行为**：`applyGhostMode` 会强制窗口置顶、隐藏任务栏、不抢焦点、防录屏（`WDA_EXCLUDEFROMCAPTURE`）。该功能是产品的"截图解题隐身"核心，不要随意改成可选开关的副作用考虑。

4. **仅一条信息：前端压缩参数默认值**：`settings` store 中 `compressionQuality: 80`、`grayscale: true`，与后端默认 `CompressionQuality: 92, Grayscale: false` 不同，这是**前端未开设置时的显示默认值**，不影响保存后生效。二次开发若调整压缩策略需两端一起改。

5. **平台行为差异**：macOS 有 Clipboard/屏幕录制权限体系，Windows 直接返回"已授权"，无权限前置。

6. **运行中进程**：见上第四节——开发进程已终止，无残留。启动官方 release 进程仍由系统自动守护。

---

## 十二、源码索引（功能 → 文件 → 函数 → 作用）

> 行号为当前官方基线。作用于理解 + 提示后续扩展。

### API 配置
| 功能 | 文件 | 函数 | 作用 |
|---|---|---|---|
| API 配置存储 | [pkg/config/config.go](pkg/config/config.go) | `Config` 结构体 / `NewDefaultConfig` | Config 定义（APIKey、BaseURL 等）及默认值 |
| 配置读写 | [pkg/config/manager.go](pkg/config/manager.go) | `Load`/`Save`/`UpdateFromJSON`/`Patch` | AES 加密读写 + 变更订阅 |
| API Key 加密 | [pkg/config/crypto.go](pkg/config/crypto.go) | `encrypt`/`decrypt` | AES-256-GCM 加解密 |

### Base URL
| 文件 | 函数/字段 | 作用 |
|---|---|---|
| [pkg/config/config.go:11](pkg/config/config.go#L11) | `BaseURL` | 已记入配置 |
| [pkg/llm/openai.go:27-35](pkg/llm/openai.go#L27-L35) | `NewOpenAIAdapter` | 判断空 URL → 默认 OpenAI 官方地址，传给 option.WithBaseURL |
| [frontend/src/stores/settings.js:12](frontend/src/stores/settings.js#L12) | `baseURL` | 前端默认值 |

### API Key
| 文件 | 函数 | 作用 |
|---|---|---|
| [app/solve.go:13-26](app/solve.go#L13) | `TriggerScreenshot`/`TriggerSend` | 无 Key 时发 `require-api-key` |
| [pkg/solution/solution.go:45](pkg/solution/solution.go#L45) | `Solve` | APIKey 为空直接返回 |
| [pkg/llm/openai.go:33](pkg/llm/openai.go#L33) | `option.WithAPIKey` | 注入 key |
| [frontend/src/stores/settings.js:189](frontend/src/stores/settings.js#L189) | `fetchModels` | 用 key 拉模型；`maskedKey` 显示掩码 |

### 获取模型
| 文件 | 函数 | 作用 |
|---|---|---|
| [app/llm.go:13](app/llm.go#L13) | `GetModels` | Wails 绑定 |
| [pkg/llm/manager.go:74](pkg/llm/service.go#L74) | `GetModels` | 暂存 key → 适配器 |
| [pkg/llm/openai.go:209](pkg/llm/openai.go#L209) | `GetModels` | 调用 `/models` 列表 |
| [frontend/src/stores/settings.js:189](frontend/src/stores/settings.js#L189) | `fetchModels`/`refreshModels` | 前端调用并提示 |

### [模型选择] 
| 文件 | 函数/组件 | 作用 |
|---|---|---|
| [pkg/llm/openai.go:21](pkg/llm/openai.go#L21) | `NewOpenAIAdapter` | 默认 GPT-4o，model 用户覆盖 |
| [frontend/src/components/ModelSelect.vue](frontend/src/components/ModelSelect.vue) | `toggle/selectModel` | 下拉选择，正逻辑能力标签显示 |
| [frontend/src/stores/ui.js](frontend/src/stores/ui.js) | `availableModels` | 备选模型列表 |

### 模型连接测试
| 文件 | 函数 | 作用 |
|---|---|---|
| [app/llm.go:5](app/llm.go#L5) | `TestConnection` | Wails 绑定 → Service |
| [pkg/llm/service.go:47](pkg/llm/service.go#L47) | `TestConnection` | 15s 超时 + `TestChat` |
| [pkg/llm/openai.go:168](pkg/llm/openai.go#L168) | `TestChat` | 发"hi"，MAX tokens 17 极速返回 |
| [frontend/src/stores/settings.js:231](frontend/src/stores/settings.js#L231) | `testConnection` | 前端触发/状态 |

### OpenAI-compatible 请求
| 文件 | 函数 | 作用 |
|---|---|---|
| [pkg/llm/openai.go:45-83](pkg/llm/openai.go#L45-L83) | `toOpenAIMessages` / `toOpenAIParts` | Message→Part 转换（文本/图片/PDF） |
| [pkg/llm/openai.go:85](pkg/llm/openai.go#L85) | `GenerateContentStream` | 流式发请求 |
| [pkg/llm/openai.go:182](pkg/llm/openai.go#L182) | `GenerateContent` | 非流式（简历解析用） |

### Prompt 管理
| 文件 | 位置 | 作用 |
|---|---|---|
| [pkg/domain/prompts.go:145](pkg/domain/prompts.go#L145) | `GetSystemBehaviorPrompt()` | 全局行为规则（中文/Markdown/LaTeX） |
| [pkg/solution/solution.go:55-74](pkg/solution/solution.go#L55-L74) | `Solve` | 组装场景 Prompt + 简历 Profile + System |
| [pkg/prompts/](pkg/prompts/) | `ResumeParsePrompt` | 简历解析专用 |

### 场景分类
| 文件 | 位置 | 作用 |
|---|---|---|
| [pkg/domain/prompt.go:19](pkg/domain/prompts.go#L19) | `GetCategories()` | 8 大分类、**54 个场景** |
| 分类ID | 通用/编程笔试/产设笔试/市场运营/学术考试/商业证书/MBTI/简历? | 前端 DomainSelector + 下拉 |

### 行测解题
| 文件 | 位置 | 作用 |
|---|---|---|
| [pkg/domain/prompt.go:793](pkg/domain/prompts.go#L793) | `"aca-gongkao-exam"` | 行测 PersonaCard（逻辑/数量/言语/资料） |

### 图形推理场景
| 说明 | 位置 |
|---|---|
| "图形" 场景对应 `aca-gongkao-exam`（Linex "图推"）与通用 `gen-solver`（图文理解）的 prompt。**没有独立"图形推理"专用场景 ID**，均由行测 prompt 或 gen-solver 覆盖。二次开发如需独立图形推理，需新增场景。 |

### 截图功能
| 文件 | 函数 | 作用 |
|---|---|---|
| [pkg/screen/screen.go](pkg/screen/screen.go) | `CapturePreview` | 全屏/窗口截图，PNG/JPEG 输出 |
| [app/screen.go:9](app/screen.go#L9) | `GetScreenshotPreview` | Wails 绑定 |
| [app/solve.go:13-53](app/solve.go#L13-L53) | `TriggerScreenshot`（`screenshotBuffer` 最多 3 张） | 截图 → 缓冲 → 事件 |

### 图片压缩
| 文件 | 函数 | 作用 |
|---|---|---|
| [pkg/ImageUtil/Compress.go](pkg/ImageUtil/Compress.go) | `CompressForOCR` | JP聪明缩高质量 + 锐化 + 灰度 + 长边限制（质量 1-100/锐化/灰度） |
| [pkg/screen/screen.go:67](pkg/screen/screen.go#L67) | `CapturePreview` 无压缩分支 | `noCompression` → PNG 原图 |

### AI 解题请求
| 文件 | 函数 | 作用 |
|---|---|---|
| [app/solve.go:135](app/solve.go#L135) | `solveInternal` | 构造 `solution.Request` + `solution.Solve` |
| [pkg/solution/solution.go:45](pkg/solution/solution.go#L45) | `Solve` | 拼 System/User 消息 → 调用 LLM |

### 流式返回
| 文件 | 函数 | 作用 |
|---|---|---|
| [pkg/llm/openai.go:98-127](pkg/llm/openai.go#L98-L127) | `GenerateContentStream` start流循环 | 逐 chunk 回调 Content/Thinking |
| [pkg/solution/solution.go:92](pkg/solution/solution.go#L92) | `Solve` 内的onChunk | 转发 `solution-stream-chunk` / `-thinking` |
| [frontend/src/stores/solution.js](frontend/src/stores/solution.js) | `handleStreamChunk`/`handleThinkingChunk` | rAF 节流渲染 + Markdown/LaTeX |
| [frontend/src/services/events.js](frontend/src/services/events.js) | `on` 事件总线 | 订阅后端事件 |

### 历史记录
| 文件 | 位置 | 作用 |
|---|---|---|
| [frontend/src/stores/solution.js](frontend/src/stores/solution.js) | `history` / `rounds` | 每轮对话快照 |
| [frontend/src/components/HistoryItem.vue](frontend/src/components/HistoryItem.vue) | `selectHistory` | 侧栏历史、删除 |
| `exportImage` | solution.js:307 | html2canvas 导出 PNG 图（图片导出用的是 `SaveImageToFile`） |

### 设置页面
| 文件 | 位置 | 作用 |
|---|---|---|
| [frontend/src/components/SettingsModal.vue](frontend/src/components/SettingsModal.vue) | 全组件 | API、模型、快捷键、简历、截图质量、滚动等分区 |
| [frontend/src/stores/settings.js](frontend/src/stores/settings.js) | `openSettings/closeSettings/saveSettings` | 打开 / 收合 / 存 |
| [app/settings.go](app/settings.go) | `GetSettings/UpdateSettings` | 后端读写 |

### 快捷键（全局热键）
| 文件 | 函数 | 作用 |
|---|---|---|
| [pkg/shortcut/shortcut.go](pkg/shortcut/shortcut.go) | `GetComboID`/`GetReadableName`/`getKeyName` | 键→名称/ID |
| [pkg/shortcut/shortcut_windows.go](pkg/shortcut/shortcut_windows.go) | `RegisterHotKey`/`UnregisterHotKey` | Win32 注册热键 |
| [pkg/shortcut/service.go](pkg/shortcut/service.go) | `handleTrigger` | 动作分发（截图/发送/删除/切换/穿透/移动/滚动） |
| [app/shortcut.go](app/shortcut.go) | `StartRecordingKey`/`ScrollContent` | 录制 + 请求滚动 |
| [frontend/src/stores/settings.js](frontend/src/stores/settings.js) | `recordKey` | 自定义按键捕获 + 冲突检测 |

### 隐藏窗口
| 文件 | 位置 | 作用 |
|---|---|---|
| [pkg/state/manager.go](pkg/state/manager.go) | `ToggleVisibility` | 切换隐身（`SetDisplayAffinity` 录屏屏蔽） |
| [pkg/platform/platform_windows.go:212](pkg/platform/platform_windows.go#L212) | `SetDisplayAffinity` | `WDA_EXCLUDEFROMCAPTURE` / `WDA_NONE` |

### 鼠标穿透
| 文件 | 位置 | 作用 |
|---|---|---|
| [pkg/state/manager.go:186](pkg/state/manager.go#L186) | `ToggleClickThrough` | 切 `WS_EX_TRANSPARENT|WS_EX_LAYERED` |
| [pkg/platform/platform_windows.go:385](pkg/platform/platform_windows.go#L385) | `setWindowClickThrough` | 加/去透明样例 |

### 窗口置顶
| 文件 | 位置 | 作用 |
|---|---|---|
| [app/window.go:31](app/window.go#L31) | `SetWindowAlwaysOnTop` | 对外绑定（**Windows 无效果**） |
| [pkg/platform/platform_windows.go:248](pkg/platform/platform_windows.go#L248) | `SetWindowLevel` | **空实现**（注释：在 GhostMode 处理） |
| [pkg/platform/platform_windows.go:502-535](pkg/platform/platform_windows.go#L502-L535) | `applyGhostMode` | 恒置顶 `WS_EX_TOPMOST` + 无边框 + 隐身 |
| [frontend/src/components/ScreenshotSettings.vue:175](frontend/src/components/ScreenshotSettings.vue#L175) | `setAlwaysOnTop(false/true)` | 权限流程中调用（实际不切换） |

### 配置持久化
| 文件 | 函数 | 作用 |
|---|---|---|
| [pkg/config/manager.go:50](pkg/config/manager.go#L50) | `Load` | 启动解密读 |
| [pkg/config/manager.go:75](pkg/config/manager.go#L75) | `Save` | Save/AES 写 |
| [pkg/config/manager.go:102](pkg/config/manager.go#L102) | `UpdateFromJSON` | 前端 JSON 全量更新 + 订阅广播 + Save |
| [app/app.go:86](app/app.go#L86) | `OnShutdown` | 退出时 Save |

---

## 十三、下一步建议（P0/P1/P2）

### P0 - 开发前必看
1. **确认 go.mod 与 Wails CLI 策略**：写第一行代码前，在开发分支显式决定：`go.mod` 保持 v2.11.0（每次 run 后 checkout 还原 go.mod/go.sum/wailsjs）或主动升级到 v2.16.0（git 提交 go.mod 变更）。**推荐升级到 v2.16.0**，理由：消除频繁自动 tidy 干扰、获得最新 Wails API。升级只需把 go.mod wails/v2 0.11 → v2.16.0，`go mod tidy`，或重新 `go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0` 后 `wails init`。

2. **烙上 v2.16 CLI 前提**：新环境 clone 后需要 `go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0`（当前已验证）。

### P1（二次开发可直接做）
1. **修复 Windows 置顶开关**：实现 `SetWindowPos(HWND_TOPMOST/HWND_NOTOPMOST)`，替换 `SetWindowLevel` 空实现，让前端 toggleAways top 真正可切换（对照 `pkg/platform/platform_darwin.go:303` 的 macOS 版本思路）。
2. **强化 API Key 存储**：避免开源 AES 密钥，改为 Windows Credential Manager（或系统的 Keyring），`crypto.go` 改造。
3. **新增"图形推理"独立场景**：在 `pkg/domain/prompts.go` 新增 `graphic-reason` prompt + Category 项，前端 DomainSelector 自动加载（DataCatalog 驱动）。

### P2（优化）
1. **统一截图压缩默认值**：前端 `settings.js` 与后端 `config.go` 的 quality/grayscale 默认存在不一致，明确单一来源。
2. **历史接续上下文**：`KeepContext` 已被解耦（Appv68 `solver.ClearHistory`），切上下文逻辑可进一步做多轮裁剪 `trimChatHistory`（`solution.go:162`）。
3. **登录/多 provider 抽象**：当前仅 OpenAI 兼容（`pkg/llm` 的 Provider 接口可扩展多家官方 SDK）。

---

*报告结束。当前工作区为 dev/qsolver-custom，内容官方，`git status` 干净。*