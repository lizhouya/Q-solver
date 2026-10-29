# Change 026 CCR 网关接入与连接诊断修复审计

日期：2026-10-25
阶段：最终审计；实现与普通自动验证完成，人工验收待用户执行。

## 1. 最终结论

- 成功 probe 永久“测试中”的已确认根因已修复：typed-nil DTO 映射不再 panic，前端另有 20 秒等待上限作为界面恢复保障。
- 模型页已恢复单一纵向滚动，角色/场景不再依赖剩余 flex 高度，样式覆盖命中实际 `.domain-list`。
- 已增加显式 direct/CCR 模式。CCR 固定使用 OpenAI Chat Completions 入站，由 CCR 负责上游协议转换和路由。
- CCR 自动行为只读取模型目录；UI、Store、Service 三层禁止 probe，后端测试证明 text/vision probe 零网络请求。
- 目录 identity 与 probe identity 已分离，选择模型不会清空目录；同 revision 自动同步单发且竞态结果受 revision/version 保护。
- 普通自动验证全部通过；race 因 Windows `0xc0000139` 标记 `BLOCKED`；Windows GUI 和真实 CCR 标记 `NEEDS_USER`。

## 2. 配置与兼容审计

- 空 `connectionMode` 仅在内存解释为 `direct`；旧配置加载不主动写盘。
- 未知非空模式拒绝，不静默降级。
- Direct 保留用户 BaseURL 与 Change 025 三种显式协议。
- CCR BaseURL 仅接受 http/https root 或 `/v1`，拒绝 userinfo、query、fragment、完整 endpoint 与额外 path。
- CCR 保存和运行时协议均固定为 `openai_chat`；模型名、品牌、域名和残留协议不能改变 adapter。
- ConfigManager 继续执行“规范化候选 → 校验 → 持久化 → 替换内存 → 锁外通知”；失败测试确认不污染内存、磁盘或 subscriber。

## 3. 网络与费用审计

- 自动测试只使用 loopback `httptest`、fake provider 和合成 PNG。
- CCR 目录测试只收到一次 `GET /v1/models`，没有 POST。
- CCR text/vision probe 测试的服务器总请求数为 0。
- 未发现新增 CCR adapter；生产 adapter 仍只有 OpenAI Chat、OpenAI Responses、Anthropic Messages 三个。
- 搜索未发现生产代码根据 `claude-*`、`gpt-*`、`deepseek-*`、品牌或域名推断协议。
- 未调用真实 CCR、OpenAI、Anthropic、DeepSeek 或其他 provider，未消费真实 API 额度。

## 4. 前端语义审计

- `frontend_wait_timeout` 文案明确“界面已停止等待，后台请求不保证已取消”。
- timeout 不设置 `requestSent=false`，不声称后端取消或零费用。
- CCR 模式隐藏 provider/capability 推测和付费 probe，只显示中立目录说明。
- 目录同步成功不显示为文字、图片、文件或额度验证成功。
- 保存、切模型、API 页 CTA 本身不触发推理；CTA 只切换模型 tab，唯一 watcher 负责目录同步。
- 刷新按钮与 Store 均要求 API Key 和 BaseURL 完整。
- 已选模型与临时搜索词已分离；重新打开下拉框不会用保存值过滤目录，主动输入仍保留大小写不敏感 substring 过滤。

## 5. bindings、依赖与产物审计

- 固定 CLI 实测版本：Wails v2.11.0；未使用 PATH 中的 v2.16.0。
- `ConnectionDraft` 与 `config.Config` bindings 均包含 `connectionMode`。
- `App.js` 与 `App.d.ts` 均导出 79 个同名方法；Wails runtime 无内容漂移。
- `go.mod`、`go.sum`、`frontend/package.json`、`frontend/package-lock.json` 的 SHA-256 与 Change 026 事前基线完全一致；无新增依赖或版本升级。
- 11 个受保护 EXE 前后 SHA-256 完全一致。
- 新增程序：

```text
build/bin/Q-Solver-change026-ccr-gateway-validation.exe
44,032,512 bytes
SHA-256: 5584d5448e6808a230a4d01e35e098938a92690beb3b1f3d149129b0d9771fe1
```

## 6. 验证审计

- `go test ./... -count=1`：`PASS`。
- `go vet ./...`：`PASS`。
- `go build ./...`：`PASS`。
- `go mod verify`：`PASS`。
- 13 个 `frontend/*_regression_test.mjs`：全部 `PASS`。
- Vite build：`PASS`。
- Wails v2.11.0 build：`PASS`。
- `git diff --check`：`PASS`。
- race：`BLOCKED (0xc0000139)`，未伪报 PASS。
- GUI 与真实 CCR：`NEEDS_USER`，自动阶段未越权执行。

## 7. 数据与授权审计

- 未修改题库 `_记录`、截图、旧导出、正式 `Q-Solver.exe` 或旧验证程序。
- 未读取、修改或输出真实 API 配置、密钥或 Mobile Token。
- Change 024/025 的未提交工作树未被清理、回退或覆盖。
- 未执行 `git add`、commit、push、Tag、Release 或发布。
- 当前授权只完成实现、自动验证和独立本地验证程序；人工验收和任何 Git/发布动作需另行授权。
