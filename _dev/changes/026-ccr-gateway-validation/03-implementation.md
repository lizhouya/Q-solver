# Change 026 实现过程记录

日期：2026-10-25
状态：实现与普通自动验证完成；Windows GUI、真实 CCR 待用户验收；race 被环境阻塞；未提交。

## 1. 实现日志

| 时间 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-10-25 | 事前治理 | 建立请求、现状、方案、验证、回滚、备注与审计骨架 | `_dev/changes/026-*`、`_dev/audits/026-*`、INDEX | 未修改业务源码；未运行真实 CCR/模型 |
| 2026-10-25 | typed-nil TDD | 先用测试复现成功 probe DTO 解引用 panic，再在 mapper 与调用点增加双层 nil 防护 | `app/llm.go`、`app/llm_test.go` | RED 准确定位 `providerErrorDTO`；GREEN 后成功 DTO 为 `Error=nil` |
| 2026-10-25 | 连接配置 TDD | 新增 direct/CCR 模式、CCR URL canonical、旧配置只读内存迁移和保存事务规范化 | `pkg/config/connection*.go`、`config.go`、`manager.go`、相关测试、`app/llm.go` | direct 保留原 URL/协议；CCR 强制 Chat 和 `/v1` |
| 2026-10-25 | CCR 后端边界 | CCR 复用 OpenAI Chat adapter；目录只执行一次 GET；text/vision probe 在 provider 构造前返回 `probe_disabled` | `pkg/llm/errors*.go`、`factory*.go`、`service*.go` | UI、Store、Service 三层禁止 probe，以 Service 为权威零网络边界 |
| 2026-10-25 | 前端 bridge deadline | 只为目录和 probe Wails Promise 增加 20 秒界面等待上限 | `frontend/src/services/modelBridgeTimeout.js`、`api.js`、回归测试 | timeout 只停止 UI 等待，不宣称后端取消或请求未发送 |
| 2026-10-25 | Store 生命周期 | 拆分 catalog/probe revision；CCR 同 revision 单发、在途复用、成功缓存，切模型不清目录 | `frontend/src/stores/settings.js`、`ui.js`、相关回归 | 保存、direct、API 页本身不隐式同步；CCR probe 零 bridge 调用 |
| 2026-10-25 | direct/CCR UI | 增加连接模式、CCR 地址/Key、固定 Chat 说明、目录状态、本地过滤；CCR 隐藏能力推测和付费实测 | `ProviderSelect.vue`、`ModelSelect.vue`、`SettingsModal.vue`、相关回归 | 模型目录不冒充连接或多模态能力验证；刷新同时要求 Key 与 BaseURL |
| 2026-10-25 | 布局修复 | 模型页改为单一纵向滚动容器，移除场景区剩余高度依赖并修正 `.domain-list` 目标类名 | `SettingsModal.vue`、`model_layout_regression_test.mjs` | 小窗口可滚动访问角色/场景，不扩大弹窗重构范围 |
| 2026-10-25 | bindings 与产物 | 使用已核验 Wails CLI v2.11.0 重生 bindings，构建独立 Change 026 验证程序 | `frontend/wailsjs/go/*`、`build/bin/Q-Solver-change026-ccr-gateway-validation.exe` | runtime 无内容漂移；仅清理生成文件尾随空白；未覆盖正式或旧程序 |
| 2026-10-25 | 全量验证 | 完成 Go、13 个前端回归、Vite、Wails、差异、依赖、网络与受保护产物审计 | 见 `04-validation.md` | 普通自动验证 PASS；race 为环境 `BLOCKED`；GUI/真实 CCR 为 `NEEDS_USER` |
| 2026-10-25 | 模型目录回归修复 | 将已选模型与临时搜索词拆分；重新打开或点击箭头显示完整目录，仅主动输入时过滤 | `ModelSelect.vue`、`ccr_ui_regression_test.mjs` | 修复第二次打开设置时已保存模型把 8 个目录项过滤为 1 个的问题；重建独立验证程序 |

## 2. 关键实现结果

### 2.1 成功 probe 能正常结束 loading

`providerErrorDTO` 对 typed nil 进行防护，`ProbeModelConnection` 只在实际存在错误时映射 DTO。App 级 loopback 成功测试确认 Wails 返回 `Outcome=passed`、`Error=nil`、`RequestSent=true`。

### 2.2 direct 与 CCR 显式分离

- 旧配置缺少 `connectionMode` 时只在内存解释为 `direct`，加载不重写文件。
- direct 保留 `openai_chat | openai_responses | anthropic_messages` 三协议。
- CCR 强制使用 `openai_chat`，BaseURL 规范为 `/v1`，模型名、品牌、域名和残留协议不参与路由选择。
- CCR 列表只读取 `/v1/models`；切模型只修改模型 ID。

### 2.3 CCR 不执行付费 probe

- UI 隐藏文字/图片实测。
- Store 在 bridge 调用前返回固定 `probe_disabled`。
- Service 在请求级校验、provider 构造和网络路径之前拒绝，测试证明 text/vision 总网络请求数为 0。

### 2.4 目录与 probe 状态解耦

目录 identity 不包含 model；模型变化只清 probe。模式、Key、URL 或 direct 协议变化才清目录。CCR 同 session/revision 的成功结果不重复请求，在途 Promise 复用；失败、空目录和 UI timeout 可手动刷新。

### 2.5 模型页恢复可访问性

模型页使用普通文档流和唯一纵向滚动容器；角色/场景区不再被上方模型或 probe 内容挤压裁剪，父样式改为命中实际 `.domain-list`。

## 3. 与计划的偏差

无功能性偏差。最终 UI 审计补充了刷新按钮同时要求 API Key 与 BaseURL，并将 CCR UI 回归的通过日志移动到全部断言之后；两项均属于计划内安全收尾。

## 4. 遗留未完成项

- Windows GUI：`NEEDS_USER`。
- 真实 CCR 目录与主截图解题：`NEEDS_USER`，只允许用户在独立验证程序中主动执行。
- Windows race：`BLOCKED (0xc0000139)`，无法据此得出竞态通过或失败结论。
- 未暂存、未提交、未推送、未创建 Tag、未发布。
