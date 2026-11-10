# 总结与注意事项 06

## 当前摘要

Change 037 已完成 Revision B 业务实现、普通自动验证、Wails v2.11.0 保护构建和候选独立启动。Q-Solver 现在把 API 接入收敛为 **OpenAI 兼容 Chat Completions** 与 **自定义请求模板** 两种模式：默认只需请求地址、API Key 和模型，高级连接与 JSON 模板折叠；模型与场景分页显示；模板模式手填模型并以受控 JSON 占位符做非流式请求。

当前不是正式 Close：Windows race 环境仍为 `BLOCKED`，真实 GUI 布局与真实 provider 验收为 `NEEDS_USER`，Git 尚未提交或推送，候选也未晋升稳定版。

## 已实现能力

- 两种面向用户的模式：OpenAI 兼容 Chat Completions、自定义请求模板；后端保留 `openai_responses`、`anthropic_messages` 作为未保存的 legacy 协议。
- 自定义模板：类型化 JSON 占位符、RFC 6901 Pointer 提取、非流式响应、模板模式手填模型并禁用目录刷新。
- 可配置鉴权 Header/前缀；Key 只注入 Header，可空以支持本地无鉴权服务。
- 旧 Responses/Anthropic 打开设置页只生成内存迁移草稿，未保存仍走旧 adapter。
- 模型与场景分页显示，窄弹窗单列，消除按钮与内容裁切。
- Keyless 截图解题链路（Solver + App 入口）。
- Revision A 历史能力（下列保留）：
- CCR 降为 OpenAI Chat 快捷预设，不再是第四种协议或禁 probe 特殊模式。
- 统一八字段连接草稿与持久配置。
- 受约束的 Base URL、Endpoint Path、Extra Headers 和 response format。
- API Key 可空；空 Key 不发送 SDK 默认鉴权头。
- 模型目录、文字 probe、图片 probe 和生产解题共享连接规范化与 transport。
- API 专用局部保存事务，不夹带其他标签草稿。
- 保存结果和错误 DTO 不回显 Key/Header 值。
- 连接、目录、probe、保存四层迟到结果隔离。
- Wails bindings 与 README 中英文说明同步完成。

## Revision B 关键实现结论（历史）

| 现象 / 风险 | 实际原因 | 最终处理 |
|---|---|---|
| CCR 看似独立协议 | UI/策略有独立分支，底层仍是 OpenAI Chat adapter | 迁移为 `direct + openai_chat`，UI 只保留 preset |
| SDK 固定 endpoint 与自定义路径冲突 | SDK 自己拼接固定推理路径 | transport 仅重写已知推理请求，不误改模型目录 |
| 旧 JSON 缺少 Endpoint 字段 | 从默认 Config 反序列化会错误继承 Chat 默认值 | 检测字段是否存在；缺失时按协议补默认，显式值不改写 |
| Keyless SDK 仍可能发空鉴权 | SDK 生成默认 Header | transport 在 Key 为空时删除默认 `Authorization` / `X-Api-Key` |
| Header 可能携带凭据或破坏 HTTP | 通用网关需要自定义 Header | 加密保存、限制 token/CRLF/数量/大小、拒绝传输控制头和鉴权冲突 |
| API/Profile 保存可能夹带 UI 草稿 | 前端全量 Config 提交不具备事务边界 | Revision B 从已保存 Config 克隆连接字段；Revision D 从后端快照深拷贝 Profiles，仅 upsert 当前草稿并在事务成功后激活 |
| JSON 模式 probe 不会原样返回纯文本 | `json_object` 强制结构化输出 | 允许顶层对象内恰好一次 token；拒绝数组、多次、非法或尾随文本 |
| 请求完成顺序可能错乱 | 刷新、probe、保存均是异步调用 | 使用 revision/version 丢弃迟到结果 |
| Wails 生成器产生空白噪音 | v2.11.0 的格式输出差异 | 定向删除空白 Tab，不升级 CLI/runtime/依赖 |

## Revision B 验证与产物（历史）

- 普通 Go 全量测试、`go vet`、`go build`、`go mod verify`：PASS。
- 16 份前端回归和 Vite production build：PASS。
- `git diff --check`：PASS，仅 LF→CRLF 提示。
- VS Code 当前 diagnostics：无问题。
- race：启动阶段 `0xc0000139`，`BLOCKED`。
- Change 037 候选：`build/bin/testing/current/Q-Solver-candidate-change037.exe`。
- 候选 SHA-256：`023762359C7FAEF5C00D392ADC2CA6E45DBF00179AF07BC3DFBC92777B48F9B4`。
- 候选内嵌 Wails runtime：v2.11.0。
- 根稳定版和 `build/bin/releases/**` 均未改变。
- 候选独立启动、WebView2、无 Node/Vite、正常退出和无残留进程：PASS。

详细命令和证据见 [04-validation.md](04-validation.md)。

## 使用与安全注意事项

1. `Base URL` 表示 API root；`Endpoint Path` 只表示推理相对路径。自定义 Endpoint 不会改变标准模型目录。
2. API Key 为空并不自动表示无鉴权；如本地网关要求自定义鉴权，可在 Extra Headers 中显式配置。
3. API Key 非空时不得在 Headers 中重复设置 `Authorization` 或 `x-api-key`。
4. 文字“测试 API 连接”会发起一次最小真实请求，可能产生费用；“刷新候选模型”只读取模型目录。
5. 模型目录成功不等于模型可完成文字或图片请求；三个能力必须分开判断。
6. 选择快捷预设只修改未保存草稿，不联网、不自动保存。
7. 错误提示和保存返回值只报告 `SET` / `NOT SET` 类状态，不应输出 Key 或 Header 值。
8. 当前不支持任意请求模板、脚本化 body、自定义 SSE 解析、协议自动探测或 fallback。

## Revision B 产物

| 产物 | 路径 | 大小 | SHA-256 |
|---|---|---:|---|
| 根稳定版（未变） | `build/bin/Q-Solver.exe` | 44,029,440 | `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE` |
| Revision A 候选（已归档） | `build/bin/archive/transient/Q-Solver-candidate-change037-revision-a.exe` | 42,386,432 | `023762359C7FAEF5C00D392ADC2CA6E45DBF00179AF07BC3DFBC92777B48F9B4` |
| Revision B 候选（布局修订前，已归档） | `build/bin/archive/transient/Q-Solver-candidate-change037-revision-b-before-tab-layout.exe` | 42,439,680 | `5ACDE382BA7954239A5306D0AB490D078AE093E53FA476BB61CBD46AA479FC82` |
| 当前 Revision B 候选 | `build/bin/testing/current/Q-Solver-candidate-change037.exe` | 42,439,680 | `831ADCE51614ACDEB90B95624F85231A50BD018548056F62A6CE55FB787C7028` |

候选内嵌 Wails runtime 为 v2.11.0（`go version -m` 证明）。构建前后保护清单见 `build/bin/archive/manifests/change-037-revb-prebuild/`，`go.mod`、`go.sum`、bindings、runtime、`frontend/dist`、`build/bin/releases` 全部 UNCHANGED。

## GUI 验收反馈修订

设置弹窗已由 520px 加宽到 720px；标签“输出”改为“显示”；标签栏不再横向滚动或自动滚动，窄窗口改为自然换行。自动门禁和新候选启动已通过，等待用户在新候选中确认真实显示效果。

## 已知限制（须向用户如实披露）

1. 旧 Responses/Anthropic 迁移默认 Pointer 只提取单个文本块，不等价于旧 SDK 的聚合与终态解析，须按目标服务响应人工核对。
2. 模板为受控 JSON 注入，不支持通用 SSE、协议自动探测或自动 fallback。
3. Keyless 仅覆盖模型测试与截图解题；题库模型整理入口仍保留 Key 前置要求。

## 提交与发布状态

- Commit Boundary Audit 已完成，规划为一个功能提交和一个治理提交。
- 当前未暂存、未 commit、未 push。
- 未创建 Tag、Release，未修改 `main` 或 `upstream`。
- `build/bin/**` 继续 ignored，不允许 `git add -f`。
- 只有用户明确授权后才能执行两个提交；push、稳定版晋升和发布仍需各自单独授权。

## Revision B 当时遗留项（历史）

1. 用户使用 Revision B 候选完成 `04-validation.md` 的 GUI/真实接口验收。
2. 若人工验收通过，更新验证结论并重新复核提交边界。
3. 获得明确授权后才按双提交计划暂存和提交。
4. 稳定版晋升、push、Tag 和 Release 不属于当前授权范围。

## Revision C 摘要

- 增加按规范化连接保存的有界模型历史；模板模式可复用历史，目录仍零请求。
- 模型历史不保存多份 API Key、Headers、模板或响应，局部 DTO 不暴露其他连接。
- 修复快捷键 `comboID` 持久字段和桌面运行态提示；统一“隐身模式”与“隐藏/显示窗口”语义并同步手机端。
- 题库目录缺失确认来自候选隔离 Profile 不同，未修改题库产品代码。
- Revision C 候选（后续已废弃）SHA-256：`281CEA02436EAE1D9AC071C412B3529AD54256786D0E534730CC85FD224926F1`，大小 42,444,800；根稳定版不变。
- 自动门禁与候选启动通过，race `BLOCKED`，固定 Profile 跨重启人工验收为 `NEEDS_USER`。

## Revision D 当前结论

- 用户需要保存多条完整连接 Profiles，使请求地址和 API Key 输入一次后可再次选择；不是模型历史。
- Revision C 模型历史代码和候选已被 Revision D 取代并定向撤销；其自动验证仅保留为历史证据。
- Profiles 保存在现有加密主配置中，根级连接字段镜像活动记录；安全列表不返回 Key，显式选择单条时才加载完整草稿。
- Revision C 的快捷键 `comboID`、动态提示与语义文案修复继续保留。
- Revision D 书面规格和逐文件 TDD 计划均已获用户批准并完成实施。
- 普通自动门禁、Wails v2.11.0 保护构建和空隔离 Profile 启动均通过；race 仍因 `0xc0000139` 为环境 `BLOCKED`，真实外部 provider 保持 `NOT_RUN`。
- 用户于 2026-11-10 确认固定隔离 Profile 双启动 GUI 验收通过，并明确授权版本晋升、双提交和推送。
- 候选 SHA-256 `8B19B242F7917D4BC0267E45BB6248077E2BA5BD2D3CE428AC9E1B836C8B4994` 已晋升为根稳定版；旧稳定版归档到 `archive/superseded/change-037/`，验证候选归档到 `archive/validation/change-037/`。
- 根目录唯一 EXE，`testing/current/` 已清空，`releases/**` 7 个文件哈希未变；Change 037 满足关闭条件。
- 功能提交已完成：`0728d8c feat: add unified API connection profiles`；治理记录按项目双提交制度独立提交，不要求文档写入自身提交哈希。
