# 验证记录 04

## 当前状态

**Revision D 实现、普通自动门禁、Wails v2.11.0 保护构建、候选隔离启动和用户固定 Profile 双启动 GUI 验收均已完成；候选已晋升稳定版。Windows race 仍因环境 `0xc0000139` 为 `BLOCKED`，真实外部 provider 保持 `NOT_RUN`。**

Revision B 的验证证据见下方“Revision B 验证矩阵”；其后的 Revision A 段落仅为历史记录，不能作为 Revision B PASS，也不能用于稳定版晋升。当前未调用真实收费 API，未读取或修改用户真实配置。

## Revision A 自动验证结果（历史）

| 类别 | 命令 / 检查 | 结果 | 证据摘要 |
|---|---|---|---|
| Config 聚焦 | `go test ./pkg/config -count=1` | PASS | 统一字段、旧 CCR 迁移、协议默认 Endpoint、Header/response format 校验、`PatchValidated` 事务通过 |
| LLM 聚焦 | `go test ./pkg/llm -count=1` | PASS | 三协议 Keyless、Endpoint、Headers、模型目录、probe、Chat JSON 与 CCR 兼容通过 |
| App 聚焦 | `go test ./app -count=1` | PASS | 八字段草稿、局部保存、失败原子性和安全 DTO 通过 |
| 聚焦组合 | `go test ./pkg/config ./pkg/llm ./app -count=1` | PASS | 三个核心包共同通过 |
| Go 全量测试 | `go test ./... -count=1` | PASS | 全部包通过；无失败包 |
| Go 静态检查 | `go vet ./...` | PASS | 退出码 0，无输出 |
| Go 全量编译 | `go build ./...` | PASS | 退出码 0，无输出 |
| 模块校验 | `go mod verify` | PASS | `all modules verified` |
| 前端回归 | `for test_file in frontend/*_test.mjs; do node "$test_file" || exit 1; done` | PASS | 16 份回归脚本全部通过，包含 Unified API、CCR 生命周期/UI、设置、模型连接与布局 |
| Vite | `npm --prefix frontend run build` | PASS | Vite v4.5.14，131 modules transformed；仅有既有的主 chunk 超过 500 kB 提示 |
| 差异空白 | `git diff --check` | PASS | 无空白错误；仅显示工作区 LF→CRLF 提示 |
| VS Code diagnostics | Problems 面板诊断 | PASS | 当前已分析文件均无 diagnostics |

## Race 结果

命令：

```bash
go test -race ./pkg/config ./pkg/llm ./app -count=1
```

结果：**BLOCKED**。

三个包均在测试进程启动阶段返回：

```text
exit status 0xc0000139
```

测试逻辑未开始执行。这与项目已知 Windows race runtime 环境阻塞一致；未把该结果冒充 PASS，也未用禁用检查后的结果替代标准 race。

## Wails bindings 与保护构建

### Bindings

- 使用 Wails CLI v2.11.0 生成 `SaveConnectionSettings` 及相关 DTO bindings。
- `frontend/wailsjs/runtime/**` 未改变。
- `go.mod`、`go.sum` 未改变。
- v2.11.0 生成的空白 Tab 噪音已定向清除，只保留语义差异。

### 构建命令

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build \
  -s \
  -o testing/current/Q-Solver-candidate-change037.exe \
  -skipbindings \
  -skipembedcreate \
  -nosyncgomod \
  -m \
  -nocolour \
  -v 1 \
  -ldflags "-s -w" \
  -tags prod
```

构建结果：PASS，Wails CLI 报告 v2.11.0，候选输出成功。

`go version -m` 证明候选链接：

```text
github.com/wailsapp/wails/v2 v2.11.0
```

### 产物清单

| 产物 | 路径 | 大小 | SHA-256 |
|---|---|---:|---|
| 根稳定版（未变） | `build/bin/Q-Solver.exe` | 44,029,440 | `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE` |
| 测试前稳定副本 | `build/bin/testing/current/Q-Solver-stable-before-test.exe` | 44,029,440 | `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE` |
| Change 037 候选 | `build/bin/testing/current/Q-Solver-candidate-change037.exe` | 42,386,432 | `023762359C7FAEF5C00D392ADC2CA6E45DBF00179AF07BC3DFBC92777B48F9B4` |

`testing/current/SHA256SUMS` 已验证，恰好包含稳定副本和一个 Change 037 候选。

上一轮 Change 036 候选已按其已提交验证哈希归档到 `build/bin/archive/validation/change-036/`，未将其冒充稳定版晋升。

### 构建前后保护

| 保护项 | 结果 |
|---|---|
| `go.mod` / `go.sum` | UNCHANGED |
| `frontend/wailsjs/go/**` | UNCHANGED（相对于本轮候选构建前的已实现状态） |
| `frontend/wailsjs/runtime/**` | UNCHANGED |
| `frontend/dist/**` | UNCHANGED，823 个文件逐一比较 SHA-256 |
| `build/bin/releases/**` | UNCHANGED，7 个文件逐一比较 SHA-256 |
| Git 暂存区 | EMPTY |
| 根稳定版 | UNCHANGED |

## 候选独立启动验证

验证使用隔离的临时 `APPDATA` 与 `LOCALAPPDATA`，未读取或修改用户真实配置。

| 检查项 | 结果 |
|---|---|
| 启动后持续运行至少 8 秒 | PASS |
| WebView2 direct child | PASS，1 个 |
| Node/Vite direct child | PASS，0 个 |
| `CloseMainWindow` | PASS，返回 `True` |
| ExitCode | PASS，`0` |
| ForcedCleanup | PASS，`False` |
| 残留 direct child | PASS，`0` |
| 隔离临时目录 | 已清理 |

## 验收标准对照

| 验收项 | 自动证据 | 当前结论 |
|---|---|---|
| “模型与 API”单页与三个按钮 | UI 静态回归、Vite build | PASS（自动） |
| CCR 仅为预设，无 adapter/禁 probe/自动联网 | Config/Service/前端回归 | PASS（自动） |
| 旧 CCR 无损加载并首次保存规范化 | Config manager/factory/Service 测试 | PASS（自动） |
| 三协议默认与自定义 Endpoint | Config + adapter + transport 测试 | PASS（自动） |
| Keyless 无默认鉴权，目录/probe/解题链路可用 | 三协议本地 `httptest` | PASS（自动） |
| Headers 校验并作用于目录/probe/正式调用 | Config + transport + adapter 测试 | PASS（自动） |
| 刷新不保存、不覆盖手填模型 | 前端 Store 回归 | PASS（自动） |
| 测试连接不自动保存 | Service + 前端 Store 回归 | PASS（自动） |
| API 专用保存不夹带其他草稿 | App + ConfigManager + 前端回归 | PASS（自动） |
| 全局保存仍工作 | Settings 回归 | PASS（自动） |
| 连接变化和迟到响应隔离 | Unified API Store 回归 | PASS（自动） |
| 普通全量门禁与保护构建 | 本文上述命令 | PASS |
| race | Windows 启动错误 `0xc0000139` | BLOCKED |
| Windows GUI 与真实 provider | 未调用真实接口 | NEEDS_USER |

## Windows GUI / 真实接口人工清单

以下项目必须使用当前 Change 037 候选验收，当前均为 `NEEDS_USER`：

1. 旧 CCR 配置在统一页面无损展示，且表现为 CCR 快捷预设。
2. 切换 CCR 预设不自动联网、刷新、测试或保存。
3. 无 Key 本地兼容服务可刷新、测试、保存和实际解题。
4. 至少一种经用户授权的真实直连接口可刷新、测试、保存和实际解题。
5. API 专用保存不提交其他标签尚未保存的草稿。
6. “保存全部设置”仍能保存全部标签。
7. 非法 Header/Endpoint 明确报错且不泄露值。
8. 刷新失败保留手填模型，文字测试不自动保存。
9. 保存失败后旧 Provider 继续生效。

真实文字 probe 和解题可能产生费用，必须由用户主动提供配置并触发；本轮没有代为执行。

## 结论

Change 037 的代码实现、普通自动验证、保护构建和候选独立启动均已完成。由于 race 为 `BLOCKED`、必要 GUI/真实 provider 验收为 `NEEDS_USER`，当前状态不能标记正式 Close，候选不得晋升根稳定版。Git 仍未暂存、未提交、未推送。

---

# Revision B 验证矩阵（2026-11-09）

## 1. 当前状态

| 项目 | 结果 |
|---|---|
| Revision B 代码实现 | 完成 |
| 普通 Go 自动门禁 | PASS |
| 前端回归与 Vite 构建 | PASS |
| Wails v2.11.0 保护构建 | PASS |
| 候选隔离启动 | PASS |
| race | `BLOCKED`（Windows `0xc0000139`） |
| 真实 WebView2 GUI 布局与真实 provider | `NEEDS_USER` |
| 提交 / 推送 / 晋升 | 未执行 |

## 2. 后端聚焦验证

| 类别 | 命令 | 结果 | 证据摘要 |
|---|---|---|---|
| 协议 | `go test ./pkg/llm/protocol -count=1` | PASS | `custom_json` 纳入后端协议枚举；旧协议仍可解析 |
| 配置 | `go test ./pkg/config -count=1` | PASS | 模板/auth 四字段、URL/Header/Pointer/模板校验、原子 Patch、旧协议不自动迁移 |
| LLM | `go test ./pkg/llm -count=1` | PASS | 模板渲染与类型化占位符、Pointer 提取、CustomJSONAdapter、probe 本地拒绝不计已发送、目录零网络 |
| 解题 | `go test ./pkg/solution -count=1` | PASS | Keyless 成功与错误路径、nil provider 安全事件 |
| App | `go test ./app -count=1` | PASS | 十二字段草稿、安全 DTO（不返回模板/Key/Header 值）、Keyless 端到端 `httptest`、保存失败原子性 |

## 3. 本轮修复的边界红灯

| 红灯 | 处理 |
|---|---|
| 自定义 URL 丢失路径编码与末尾 `/` | 改用 `ContainsRune(...,13/10)` + `url.Parse`，保留 EscapedPath |
| 模板嵌套无限 | 增加 64 层深度上限 |
| 重复大小写 / 控制字节 Header | 规范化后重复即拒绝，拒绝 <32（除 TAB）与 127 |
| 图片未被模板承载仍发起请求 | 渲染前解析模板树，缺少图片占位符时在 HTTP 前以 `ErrorInvalidConfig` 拒绝 |
| JSON Pointer 接受 `+0` / `-0` / `00` | 数组索引逐字符 0-9，拒绝前导 0 与符号 |
| Probe 把本地构建失败报为“已发送” | `RequestSent` 依据规范化 `ProviderError.Stage` 判定，`StageBuildRequest`/`StageValidateConfig` 为 false |
| App 三处空 Key 前置拒绝 | 改为校验请求地址非空，并在捕获前做无网络配置校验 |

## 4. 全量自动门禁

| 命令 | 结果 | 证据摘要 |
|---|---|---|
| `go test ./... -count=1` | PASS | 全部包通过，无失败包 |
| `go vet ./...` | PASS | 退出码 0，无输出 |
| `go build ./...` | PASS | 退出码 0，无输出 |
| `go mod verify` | PASS | `all modules verified` |
| `for f in frontend/*_test.mjs; do node "$f"; done` | PASS | 16 份回归脚本全部通过 |
| `npm --prefix frontend run build` | PASS | Vite v4.5.14，131 modules；仅既有大 chunk 提示 |
| `go test -race ./pkg/config ./pkg/llm ./pkg/solution ./app -count=1` | BLOCKED | 四个包均在启动阶段 `exit status 0xc0000139`，测试逻辑未执行 |

## 5. 候选构建与产物

构建命令（Wails CLI v2.11.0，跳过 bindings/embed/gomod 同步）：

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build   -s -o testing/current/Q-Solver-candidate-change037.exe   -skipbindings -skipembedcreate -nosyncgomod -m -nocolour -v 1   -ldflags "-s -w" -tags prod
```

`go version -m` 证明候选链接 `github.com/wailsapp/wails/v2 v2.11.0`，Go 工具链 `go1.27.1`。

| 产物 | 路径 | 大小 | SHA-256 |
|---|---|---:|---|
| 根稳定版（未变） | `build/bin/Q-Solver.exe` | 44,029,440 | `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE` |
| 测试前稳定副本 | `build/bin/testing/current/Q-Solver-stable-before-test.exe` | 44,029,440 | `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE` |
| Revision A 候选（已归档） | `build/bin/archive/transient/Q-Solver-candidate-change037-revision-a.exe` | 42,386,432 | `023762359C7FAEF5C00D392ADC2CA6E45DBF00179AF07BC3DFBC92777B48F9B4` |
| Revision B 候选 | `build/bin/testing/current/Q-Solver-candidate-change037.exe` | 42,439,680 | `5ACDE382BA7954239A5306D0AB490D078AE093E53FA476BB61CBD46AA479FC82` |

构建前后保护清单（`build/bin/archive/manifests/change-037-revb-prebuild/`）逐项复核：

| 保护项 | 结果 |
|---|---|
| `go.mod` / `go.sum` | UNCHANGED |
| `frontend/wailsjs/go/**` | UNCHANGED |
| `frontend/wailsjs/runtime/**` | UNCHANGED |
| `frontend/dist/**` | UNCHANGED（聚合哈希一致） |
| `build/bin/releases/**` | UNCHANGED（聚合哈希一致） |
| Git 暂存区 | EMPTY |
| 根稳定版 | UNCHANGED |

`build/bin/archive/transient/SHA256SUMS` 已补登 `change036-top-card` 并校验全部条目通过；Revision A 候选以不同文件名归档，未覆盖任何既有文件。

## 6. 候选隔离启动

使用隔离的临时 `APPDATA` 与 `LOCALAPPDATA`，未读取或修改用户真实配置。

| 检查项 | 结果 |
|---|---|
| 持续运行至少 9 秒 | PASS |
| WebView2 direct child | PASS，1 个（`msedgewebview2.exe`） |
| Node/Vite direct child | PASS，0 个 |
| `CloseMainWindow` | PASS，返回 `True` |
| ExitCode | PASS，`0` |
| ForcedCleanup | PASS，`False` |
| 残留进程 | PASS，`0` |
| 隔离临时目录 | 已清理 |

本会话遗留的 Vite 开发服务（监听 `127.0.0.1:5179`）已停止，端口复核无监听。

## 7. 验收标准对照（Revision B）

| 验收项 | 自动证据 | 结论 |
|---|---|---|
| 两种模式：OpenAI 兼容 + 自定义模板 | 前端 UI/Store 回归、Config/协议测试 | PASS（自动） |
| 默认只填地址、Key、模型；高级折叠 | 前端布局与 UI 回归 | PASS（自动） |
| 选定模型后提供保存 / 刷新 / 测试按钮 | 前端 UI 回归、App 保存测试 | PASS（自动） |
| 模型与场景分页，角色/场景不再被遮挡 | 前端布局回归（grid 跨列、container 查询、min-width 0） | PASS（自动） |
| 模板模式手填模型、禁用目录刷新 | Store/UI 回归，目录零 bridge | PASS（自动） |
| 模板为类型化 JSON 占位符 + Pointer 提取 | 模板渲染与 Pointer 单元测试 | PASS（自动） |
| 模板响应仅 JSON 非流式 | CustomJSONAdapter 单 chunk 测试 | PASS（自动） |
| 图片经安全占位符承载，未承载则发送前拒绝 | 模板图片 guard 测试（零网络） | PASS（自动） |
| Key 只注入鉴权 Header，可配置 Header/前缀 | Adapter/Config 测试 | PASS（自动） |
| 旧 Responses/Anthropic 仅生成迁移草稿，未保存仍走旧 adapter | Config/Store 回归 | PASS（自动） |
| Keyless 截图解题可达本地模板服务 | App `httptest` 端到端测试 | PASS（自动） |
| 保存失败原子性、错误不泄露模板/Key/Header | App 安全 DTO 测试 | PASS（自动） |
| 全量门禁与保护构建 | 本文第 4、5 节 | PASS |
| race | Windows 启动错误 `0xc0000139` | BLOCKED |
| 真实 WebView2 GUI 布局与真实 provider | 未在真实窗口验收 | NEEDS_USER |

## 8. 已知限制（须如实披露）

1. 旧 Responses/Anthropic 迁移使用固定 Pointer `/output/0/content/0/text` 与 `/content/0/text`，只提取指定位置的单个文本块；reasoning/tool block 在首位或存在多文本块时可能提取失败，不等价于旧 SDK 的聚合与终态解析。迁移草稿必须按目标服务响应人工核对。
2. 自定义模板为受控 JSON 注入，不执行脚本、循环、表达式、环境变量或额外网络请求；不支持通用 SSE、协议自动探测或自动 fallback。
3. Keyless 支持范围限于模型测试与截图解题链路；题库的模型整理入口仍保留既有 Key 前置要求，本次未修改。
4. 真实浏览器自动化布局验证因 ego-browser bootstrap 不可用而 `BLOCKED`，未绕过权限；完整 WebView2 视觉验收保留为人工项。
5. 未调用任何真实收费 API，未读取或修改用户真实配置。

## 9. 结论

Revision B 的代码实现、普通自动验证、保护构建与候选隔离启动均已完成。由于 race 为 `BLOCKED`、真实 GUI/provider 验收为 `NEEDS_USER`，当前状态不能标记正式 Close，候选不得晋升根稳定版。Git 仍未暂存、未提交、未推送。

---

# GUI 标签布局修订验证（2026-11-09）

| 检查 | 结果 | 证据 |
|---|---|---|
| TDD 红灯 | PASS | `model_layout_regression_test.mjs` 首次失败于 `width: 520px`，证明新约束有效 |
| 布局聚焦 | PASS | 要求 720px、无横向滚动、窄屏换行、“显示”文案、无滚动专用 JS |
| 全部前端回归 | PASS | 16 份 `frontend/*_test.mjs` 全部通过 |
| Vite build | PASS | 131 modules；仅既有大 chunk 提示 |
| Go 全量 | PASS | `go test ./... -count=1` 全部包通过 |
| Go build | PASS | `go build ./...` 退出码 0 |
| Wails 保护构建 | PASS | CLI/runtime v2.11.0；依赖、bindings、runtime、dist、releases、Git index、根稳定版保护哈希一致 |
| 候选隔离启动 | PASS | 运行 9 秒；WebView2=1；Node/Vite=0；ExitCode=0；无强制清理及残留进程 |

## 新候选产物

| 产物 | 大小 | SHA-256 |
|---|---:|---|
| 布局修订前 Revision B 候选（已归档） | 42,439,680 | `5ACDE382BA7954239A5306D0AB490D078AE093E53FA476BB61CBD46AA479FC82` |
| 当前 Change 037 候选 | 42,439,680 | `831ADCE51614ACDEB90B95624F85231A50BD018548056F62A6CE55FB787C7028` |
| 根稳定版（未变） | 44,029,440 | `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE` |

布局修订前候选归档路径：`build/bin/archive/transient/Q-Solver-candidate-change037-revision-b-before-tab-layout.exe`。新保护清单位于 `build/bin/archive/manifests/change-037-revb-ui-prebuild/`。

自动验证已完成；“标签无滚动条且一行完整显示”的真实窗口结论仍为 `NEEDS_USER`，等待用户使用当前新候选确认。未提交、未推送、未晋升。

---

# Revision C 验证矩阵（2026-11-09）

| 检查 | 结果 | 证据摘要 |
|---|---|---|
| Config 模型历史聚焦与全包 | PASS | 旧配置不迁移、身份隔离、Key 轮换、去重/LRU、12×20、256 字节、深拷贝、事务原子性 |
| App 局部保存与安全 DTO | PASS | 成功记录；失败不变；只返回当前连接 `savedModels`，不泄露其他连接或敏感值 |
| 前端模型历史 | PASS | 按连接筛选、模板目录零 bridge、成功/失败保存、全量 payload、历史/目录分组 |
| 快捷键桌面/手机端 | PASS | `comboID`、桌面运行态同步、GhostMode/WindowVisibility 文案、Snapshot 不含 ComboID |
| `go test ./... -count=1` | PASS | 全部包通过 |
| `go vet ./...` | PASS | 退出码 0 |
| `go build ./...` | PASS | 退出码 0 |
| `go mod verify` | PASS | `all modules verified` |
| 18 份 `frontend/*_test.mjs` | PASS | 全部通过 |
| Vite production build | PASS | 131 modules；仅既有大 chunk warning |
| race | BLOCKED | config/llm/solution/app 启动阶段均返回 `0xc0000139` |
| Wails v2.11.0 保护构建 | PASS | 依赖、bindings、runtime、dist、Git index、releases、根稳定版保护哈希一致 |
| 候选隔离启动 | PASS | 运行 9 秒；WebView2=1；Node/Vite=0；ExitCode=0；无强制清理或残留 |
| 固定 Profile 双启动 | NEEDS_USER | 题库目录、连接历史和已保存快捷键跨重启待人工确认 |

## Revision C 产物

| 产物 | 大小 | SHA-256 |
|---|---:|---|
| Revision B 布局候选（已归档） | 42,439,680 | `831ADCE51614ACDEB90B95624F85231A50BD018548056F62A6CE55FB787C7028` |
| Revision C 候选（后续已废弃） | 42,444,800 | `281CEA02436EAE1D9AC071C412B3529AD54256786D0E534730CC85FD224926F1` |
| 根稳定版（未变） | 44,029,440 | `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE` |

Revision B 布局候选归档到 `build/bin/archive/transient/Q-Solver-candidate-change037-revision-b-before-model-history.exe`；Revision C 保护清单位于 `build/bin/archive/manifests/change-037-revc-prebuild/`。候选 `go version -m` 证明链接 `github.com/wailsapp/wails/v2 v2.11.0`。

当前未读取或修改正式用户配置，未调用真实收费 API，未提交、未推送、未晋升。固定 Profile 验收未完成前不得正式 Close。

## Revision D 对 Revision C 候选的处置

Revision C 候选 `281CEA02436EAE1D9AC071C412B3529AD54256786D0E534730CC85FD224926F1` 的自动验证结果仍是真实历史证据，但其模型历史功能不符合用户澄清后的需求，因此：

- 状态改为 **SUPERSEDED / NOT FOR PROMOTION**。
- 停止固定 Profile 人工验收，不把未完成项补写为通过。
- 候选不得晋升根稳定版，不得发布。
- Revision D 随后已重新执行全部自动门禁与保护构建；固定 Profile 双启动仍为 `NEEDS_USER`。
- race `0xc0000139` 仍保持历史 `BLOCKED`，不因方案变更改写。

## Revision D 自动验证（2026-11-10）

| 验证项 | 命令 / 方法 | 结果 | 说明 |
|---|---|---|---|
| Profile Config/Manager 聚焦 | `go test ./pkg/config -count=1` | PASS | 旧单连接只读、随机 ID、命名、20 条上限、根镜像、全量保存合并、加密落盘和失败原子性通过 |
| Wails Profile 聚焦 | `go test ./app -count=1` | PASS | 安全摘要、单条读取、创建/更新/激活、非活动删除、脱敏错误和 GetSettings 最小化通过 |
| Go 全量 | `go test ./... -count=1` | PASS | 全部包通过 |
| Go vet | `go vet ./...` | PASS | 无输出，退出码 0 |
| Go build | `go build ./...` | PASS | 无输出，退出码 0 |
| 模块校验 | `go mod verify` | PASS | `all modules verified` |
| 前端回归 | `for test_file in frontend/*_test.mjs; do node "$test_file"; done` | PASS | 18 份现存回归全部通过；Profile 新回归纳入，模型历史回归已删除 |
| Vite build | `npm --prefix frontend run build` | PASS | 133 modules；仅既有大 chunk warning |
| VS Code diagnostics | Warning 及以上 | PASS | 空列表；不替代测试/构建证据 |
| 差异检查 | `git diff --check` | PASS | 仅换行转换提示，无空白错误 |
| Windows race | `go test -race ./pkg/config ./pkg/llm ./pkg/solution ./app -count=1` | BLOCKED | 四包均在测试逻辑启动前返回 `exit status 0xc0000139`；普通测试结果单列 |
| 真实外部 API | 未执行 | NOT_RUN | 未读取真实凭据，未触发收费调用 |

### Revision D 安全边界

- Config 原始磁盘字节不含测试 Key/Header/模板 sentinel，解密后 Profile 可完整往返。
- `GetSettings` 清空 Profiles 及全部根级连接编辑字段，只保留活动 ID 和普通全局设置。
- 列表/保存/删除 DTO 不返回 Key、Header、模板、Pointer 或 AuthPrefix；完整草稿只经单条 `GetConnectionProfile(id)` 返回。
- API Key 输入继续使用 `type="password"`，未增加明文显示按钮。
- full-save payload 不包含 `connectionProfiles` 或模型历史；后端从旧快照保留记录后 upsert 选中项。
- Revision C 的 `ModelHistoryEntry`、后端/前端 helper、分组 UI、bindings 类型和专用回归均已定向删除；负向测试断言允许保留关键词。
- 未读取或修改正式用户配置；Wails bindings 生成仅打印了配置路径，不读取或记录配置内容，治理记录不保存该路径。

## Revision D 保护构建与启动

| 项目 | 结果 |
|---|---|
| Wails CLI/runtime | PASS：CLI v2.11.0；`go version -m` 链接 `github.com/wailsapp/wails/v2 v2.11.0` |
| 保护哈希 | PASS：`go.mod`、`go.sum`、bindings/runtime、dist、Git index、根稳定版和 releases 构建前后无非预期变化 |
| Revision C 候选处置 | PASS：按原哈希归入 `archive/transient`，未冒充验证通过版本 |
| 修复前 Revision D 候选处置 | PASS：发现 legacy 显示名边界后归入 `archive/transient`，未继续使用 |
| 最终候选（晋升前） | `build/bin/testing/current/Q-Solver-candidate-change037.exe`；晋升后归档至 `build/bin/archive/validation/change-037/` |
| 大小 | 42,498,048 字节 |
| SHA-256 | `8B19B242F7917D4BC0267E45BB6248077E2BA5BD2D3CE428AC9E1B836C8B4994` |
| 空隔离 Profile 启动 | PASS：启动成功，WebView2=1，Node/Vite=0，正常关闭，ExitCode=0，无强制终止或残留 |
| 根稳定版 | PROMOTED：42,498,048 字节；SHA-256 `8B19B242F7917D4BC0267E45BB6248077E2BA5BD2D3CE428AC9E1B836C8B4994` |
| 旧稳定版归档 | PASS：`archive/superseded/change-037/Q-Solver-stable-before-change037.exe`；SHA-256 `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE` |
| 验证候选归档 | PASS：`archive/validation/change-037/Q-Solver-candidate-change037.exe`；哈希与根稳定版一致 |
| releases | UNCHANGED：7 个文件逐项哈希一致 |
| 测试区 | PASS：`testing/current/` 已清空；根目录唯一 EXE 为 `Q-Solver.exe` |
| 候选晋升 | PASS：用户确认 GUI 验收通过并明确授权；同卷原子替换后根哈希复核通过 |

## Revision D 固定 Profile 双启动

状态：**PASS（用户确认）**。

用户于 2026-11-10 明确回复“验证通过”，确认固定隔离 Profile 双启动 GUI 验收完成。该结论覆盖事前清单中的旧单连接首次正式保存、多条 Profile 创建/切换/修改/删除约束、Key `SET / NOT SET` 恢复、活动记录保留、题库目录保留、快捷键显示与实际绑定保留及第二次启动复核。未读取或记录任何地址、Key、Header、模板或题库路径实值。

Windows race 仍按原始结果记为环境 `BLOCKED`；真实外部 provider 未执行并保持 `NOT_RUN`。二者未被改写为 PASS。Change 037 的必要自动验证、用户 GUI 验收、产物归档与稳定版晋升现已完成。
