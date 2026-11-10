# 回滚预案 05

## 回滚策略

- [ ] **A. 全量还原**：仅在经审计的提交完成后使用 `git revert` 回退 Change 037 双提交；禁止 `git reset --hard`。
- [x] **B. 定向还原**：优先按 Config、transport、Wails、前端页面分层回滚。
- [x] **C. 配置级回滚**：新字段有兼容默认值；必要时恢复变更前加密配置保护副本。
- [x] **D. 产物回滚**：仅使用 `archive/superseded/change-037/` 中哈希通过的稳定版。

## 回滚触发条件

- 旧配置无法加载或首次保存丢失 Base URL、Key、Model，或旧 Responses/Anthropic 配置在未保存时被改写。
- API 专用保存夹带其他标签草稿。
- 保存失败后活动 Provider 或磁盘配置发生变化。
- Endpoint 重写影响模型目录或把请求发送到错误路径。
- Key 为空仍发送空鉴权头，或 Key/模板/Header 值进入日志或错误。
- 模板渲染注入未知/嵌入式占位符、执行非 JSON 逻辑，或在模板未承载图片时仍发起请求。
- 模板模式目录刷新发出网络请求，或 legacy 协议在未保存时被静默迁移。
- OpenAI Chat 兼容模式现有生产解题或流式解析出现回归。
- 必要 GUI 验收 FAIL 且无法在确认范围内修复。

## 分层回滚

1. **前端**：恢复 API/模型双标签和原按钮；保留后端兼容读取。
2. **CCR 策略**：恢复 `connectionMode=ccr` 的固定 Chat、自动目录和禁 probe。
3. **两模式**：把模式下拉恢复为三种结构化协议；保留后端 `custom_json` 读取以兼容已保存配置。
4. **模板**：停止写入 `requestTemplate`/`responseJSONPointer`/`authHeader`/`authPrefix`；读取时忽略这四个兼容字段。
5. **transport 与 adapter**：移除 endpoint/header/auth 包装与 `CustomJSONAdapter` 注册，恢复 SDK 默认 endpoint。
6. **Solver/App**：恢复空 API Key 前置拒绝（会重新禁用无鉴权本地服务解题）。
7. **Wails**：恢复变更前 bindings 和旧接口调用。
8. **产物**：失败候选归档，不覆盖根稳定版。

## 配置保护

- 实施前只记录配置文件存在状态、大小和 SHA-256，不读取或打印明文。
- 如需真实 GUI 验收，先在仓库外创建加密配置保护副本并记录哈希。
- 回滚只使用身份和哈希明确的保护副本；不猜测配置来源。

## 实际回滚记录

Revision B 业务实现、普通自动门禁、保护构建与候选隔离启动已完成，当前未触发回滚。根稳定版保持原哈希（`29F0100E...DB380FE`），Revision B 候选（`5ACDE382...479FC82`）未晋升，Revision A 候选已归档到 `build/bin/archive/transient/Q-Solver-candidate-change037-revision-a.exe`。

如人工验收失败，优先保留当前稳定版并按上述分层策略定向修复或回滚；根稳定版仅在用户明确授权晋升后才改变。

## Revision C 定向回滚

- 停止写入 `modelHistory` 时保留字段供旧/新版本安全忽略，不自动删除用户历史。
- 前端可定向恢复单一目录候选，但不得恢复 `vkCode` 错误字段。
- 快捷键文案回滚不得重新混淆 GhostMode 与 WindowVisibility。
- 题库产品代码本轮未改，无题库代码回滚项；固定测试 Profile 失败时保留根稳定版并检查隔离环境。
- Revision C 候选失败时使用根稳定版，不把 Revision B transient 候选冒充稳定版晋升。

## Revision D 回滚补充

- Connection Profiles 字段由旧版本自然忽略；根级活动连接镜像继续供旧版本和既有运行链路读取。
- 回滚 Profiles 写入时不主动删除加密配置中的 `connectionProfiles`，避免破坏已保存的连接记录；只停止新写入并保留根级镜像。
- 若 Profile 规范化、活动 ID 或全量保存出现故障，优先定向回滚 `connection_profiles.go`、Wails Profile 接口和前端 Profile 状态机，不回滚 Revision B 两模式连接能力。
- Revision C 模型历史代码已在未发布状态定向删除，不迁移为 Profile；未知 `modelHistory` JSON 字段可被 Go 解码安全忽略。
- 快捷键 `comboID`、动态提示、“隐身模式 / 隐藏/显示窗口”和 720px 无滑动布局不随 Profiles 回滚。
- Revision D 最终候选已在用户确认固定 Profile 双启动 GUI 验收通过后晋升为根稳定版；当前根哈希为 `8B19B242...B4994`。
- 回滚源固定为 `build/bin/archive/superseded/change-037/Q-Solver-stable-before-change037.exe`，SHA-256 为 `29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE`；通过候选归档于 `build/bin/archive/validation/change-037/`。
- 晋升已验证根目录唯一 EXE、`testing/current/` 清空以及 `releases/**` 7 个文件哈希不变；回滚时仍须先验证归档 `SHA256SUMS`，不得猜测文件身份。
- Revision C 候选和 legacy-name-fix 前候选均已按哈希归入 `archive/transient`，不得作为回滚稳定版。
