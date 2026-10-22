# 实施方案 — 02

## 1. 已批准方案

采用“自动记录 + 当前分段 + 一键整理导出”。配置题库目录后，首次解题懒创建当前段；答案区“导出题库”锁定当前段 ID，按设置决定是否调用模型整理，随后只生成一次用户可见导出。导出成功封存该段，下一题自动创建新段。手动“新建分段”同样封存当前段；“清空当前段”采用需确认的软丢弃，不物理删除原始截图。

## 2. 数据和兼容策略

### 2.1 Session/Question 演进

- 保留现有 JSON 字段，`Question` 新增 `category`，`type` 继续表示更细题型。
- Session 状态扩展为可写、封存、已导出、已丢弃；旧 `active`/`paused`/`ended` 均可读取并映射到新操作规则。
- 增加导出时间、导出目录和丢弃时间等可选元数据；缺失字段按旧数据处理。
- 当前段是 Store 唯一可追加对象；所有整理、导出、分段操作使用稳定 Session ID，不能依赖可变化的 `current` 指针。

### 2.2 v2 目录

```text
<ArchiveDir>/
├─ YYYY-MM-DD_HHMMSS_<单位或笔试题库>/
│  ├─ 题库.md
│  ├─ 原始记录.md
│  └─ assets/
└─ _记录/<分段ID>/
   ├─ session.json
   └─ assets/
```

- `ArchiveDir` 是用户选择的最终题库目录，不再追加“求职笔试/题库/按单位/未命名/短 ID”。
- 未填写单位时使用“笔试题库”；同名目录依次追加 `_02`、`_03`。
- 新 Store 默认写入 `ArchiveDir/_记录`；旧记录目录只读兼容并允许原位补写，不移动、不删除。
- v1 `Root` 若已以 `求职笔试/题库` 结尾则直接转为 `ArchiveDir`，否则转为 `Root/求职笔试/题库`；同时保留 `Root/求职笔试/会话记录` 为 legacy 搜索位置。
- 根目录本身丢失时不自动重建；内部 `_记录` 只在已验证根目录下按需建立。

## 3. 实施步骤

### 步骤 A：题库存储层（先写失败测试）

1. 扩展类型和状态，保留旧 JSON 兼容。
2. Store 分离 `archiveDir`、新记录根和 legacy 记录根，并记录当前 Session 的实际目录。
3. `Begin` 在没有可写当前段时自动创建；继续使用 record ID 隔离迟到 callback。
4. 增加按 ID 的 `Seal`、`Split`、`Discard`、`SnapshotByID`、`Organize` 和 `Export` 能力。
5. 软丢弃段不参与“当前段”导出，但仍可在历史记录中查看；封存/导出后不可继续追加。
6. 导出使用临时目录完成后原子改名，固定文件名并实现顺序冲突后缀。
7. 本地 Markdown 按 `Category → Type` 分组；无分类字段使用明确兜底值，不丢失原记录。
8. 模型整理只处理目标段内尚无 Questions 的成功记录；单条失败保留错误并继续，取消/部分失败仍可导出原稿和成功结果。

### 步骤 B：App 服务和偏好

1. 将独立偏好升级到 v2：`archiveDir`、可选单位名、`autoOrganize`、新/旧记录位置；不写入主 Config。
2. 新增偏好读取和更新、当前段分段、软清空、一键导出接口；旧 API 暂保留为兼容入口，不再由新 UI 主流程调用。
3. 一键导出开始时锁定目标段 ID；整理期间新题可进入新段，但不能修改目标段。
4. 未显式开启 `autoOrganize` 时只做本地导出，不发送截图；启用后使用当前 provider/model，每条记录独立整理。
5. 只在最终阶段生成一次用户可见目录；成功后标记目标段已导出。失败时保留目标段供重试。
6. 保留“捕获先于 outputMode 路由”和关闭前 Flush/中断恢复行为。

### 步骤 C：前端交互

1. `SettingsModal` 增加并列“题库”标签，配置最终目录、单位名称和“一键导出时调用模型整理”开关，并展示数据发送说明和当前模型。
2. 题库 Pinia store 改为自动分段状态，提供 `exportCurrent`、`splitCurrent`、`discardCurrent`、偏好加载/保存。
3. `ActionBar` 的“导出”改为“导出题库”，显示整理进度与忙碌状态；`SolveView` 接入题库 store。
4. `QuestionBankPanel` 缩减为当前段、手动分段、软清空、历史分段和重新导出管理，不再要求“开始场次”。
5. 历史项的原答案图片导出入口保持不变。
6. 更新 API wrapper 和 Wails bindings；只接受本 Change 对绑定造成的必要差异。

### 步骤 D：Mobile 提示与滚动控制

1. Snapshot 增加只含动作标签和快捷键显示名的 `hints`。
2. 增加独立递增 `controlSeq` 和一次性 `control`；Bridge 提供发送滚动控制的方法。
3. SSE 初始 `snapshot` 只同步最近序号；实时 `message` 仅在序号增加时执行命令。
4. `ScrollContent` 按 `desktop`、`mobile`、`dual` 路由；mobile 服务不可用时安全回退桌面。
5. 手机页显示实际快捷键卡片，收到新命令后滚动约 70% 可视高度；不暴露 VK 值、路径、Token 或配置。
6. 顶部栏增加滚动上/下按钮，并复用相同后端接口。

### 步骤 E：Windows hook 静态检查修复

1. 在 `pkg/platform/platform_windows.go` 增加 hook 原生结构复制函数，限定只在 callback 生命周期读取 lParam。
2. `pkg/shortcut/shortcut_windows.go` 仅使用复制后的普通 Go 值并移除 `unsafe` import。
3. 不使用 `//nolint`、构建标签规避或关闭 vet；以原有键盘/鼠标侧键测试和 `go vet ./...` 验证。

### 步骤 F：生成、验证和记录

1. 运行 gofmt 与聚焦测试：questionbank、mobile、shortcut、app。
2. 运行 `go test ./...`、`go vet ./...`、`go build ./...`。
3. 运行 `npm --prefix frontend run build` 和 `wails build`；构建前后审查 bindings、go.mod/go.sum 和其他生成噪音。
4. 运行 `git diff --check`，核对未暂存的 Change 010/011 内容仍在且暂存区为空。
5. 更新 03/04/06、审计和 INDEX；Windows GUI、手机真机、真实模型调用标记 `NEEDS_USER`，不以自动测试代替。

## 4. 测试矩阵

- 自动建段：无手动 Start 的首次 Begin 建立当前段并保存 1～3 张图片。
- 分段边界：Split 后新题使用新 ID；旧段不进入当前段导出。
- 软清空：Discard 后旧文件保留，但下一次导出不含旧段。
- 并发隔离：导出/整理锁定目标 ID，新题和迟到 callback 不串段。
- 路径：ArchiveDir 直出、默认“笔试题库”、非法名称净化、`_02` 冲突、可搬移相对图片、旧输出不覆盖。
- 兼容：v1 偏好迁移、legacy 会话枚举/打开/重新导出、旧 JSON 无 Category 可读。
- 整理：Category/Type schema、成功/无效 JSON/取消/部分失败/只重试失败项，原答案不变。
- Mobile：提示更新、实时控制、重连首帧不重放、outputMode 三路路由与不可用回退。
- Windows hook：快捷键触发、录制、鼠标 XButton 行为保持；vet 零对应告警。
- 前端：构建通过，ActionBar 不再调用答案图片导出，历史图片导出仍存在。

## 5. 文件白名单

预计修改或新增：

- `pkg/questionbank/*.go`、`pkg/questionbank/*_test.go`
- `app/questionbank*.go`、`app/questionbank*_test.go`、`app/shortcut.go`、`app/mobile.go`、必要的 App 测试
- `pkg/mobile/*.go`、`pkg/mobile/*_test.go`、`pkg/mobile/web/index.html`
- `pkg/platform/platform_windows.go`、`pkg/shortcut/shortcut_windows.go` 及相关测试
- `frontend/src/stores/questionbank.js`
- `frontend/src/components/SettingsModal.vue`、`QuestionBankPanel.vue`、`ActionBar.vue`、`SolveView.vue`、`TopBar.vue`
- `frontend/src/services/api.js`、`frontend/src/App.vue`
- `frontend/wailsjs/go/app/App.js`、`App.d.ts`、`models.ts`
- `README.md`、`README_EN.md`
- `_dev/changes/012-question-bank-quick-export/*`、`_dev/changes/INDEX.md`、`_dev/audits/012-question-bank-quick-export.md`

如必须修改白名单外文件，先记录其必要性和影响，不进行无关重构。

## 6. 风险与控制

| 风险 | 控制措施 |
|---|---|
| 导出时新题写入同一段 | 导出开始先按 ID 封存目标段，后续 Begin 懒创建新段 |
| 模型失败导致原稿不可用 | 整理逐条进行，最终导出始终包含原稿；失败项保留错误 |
| v1 路径迁移丢记录 | 只增加 legacy 搜索根，不移动、不覆盖、不删除 |
| 软清空误删 | 仅标记 discarded，首版不物理删除 |
| SSE 重连重复滚动 | snapshot 只同步 controlSeq，实时 message 才执行新 control |
| 手机泄露本地配置 | 快照只发送显示文本和答案状态，不发送内部值或路径 |
| Wails CLI 版本噪音 | 构建前记录差异，构建后逐文件审查，不批量恢复 |
| 既有未提交改动被覆盖 | 只编辑白名单文件，禁止 reset/clean/add-all |

## 7. 回滚边界

按 `05-rollback.md` 定向恢复 Change 012 代码和文档。新旧 `_记录` 与已导出的 Markdown 一律保留；任何永久数据删除、Git 提交或发布都不在本次授权内。


## 8. 2026-10-20 Mobile 人工反馈回修方案

1. Mobile 生命周期增加互斥与稳定状态码；Bridge 和配置订阅只初始化一次，Server 可幂等重试。`RetryMobileOutput` 只恢复监听，不触发窗口隐藏，不向前端返回原始监听错误。
2. `MobileInfo` 扩展 `status/retryable`；设置页每次打开或进入输出标签都刷新，并独立显示模式草稿、已保存模式和服务连接状态。读取失败提供重新读取，可恢复的服务失败提供显式重试。
3. 手机端分别维护业务 `seq`、`controlSeq` 和 hints 变化：控制帧不得执行业务快照渲染；布局稳定后再滚动约 70% 视口；首帧仅同步控制基线。
4. 顶部箭头组成局部滚轮控制组，只在悬停该组时接管纵向 wheel；累计触控板小位移并节流，继续复用 `api.scrollContent`。不修改全局 Windows mouse hook。
5. 使用 Go 测试和无依赖 Node DOM harness 覆盖状态、重试、控制去重、历史选择保留与新业务快照；前端构建和 Windows/手机人工验收覆盖局部 wheel。
6. 更新 bindings、README、审计、Change 012 实现/验证/回退/总结与 INDEX。真实 URL/token 不进入日志、快照、文档或测试输出。

## 2026-10-22 实时同步补修方案

1. 在 Publish 广播前统一写入 b.state.Seq=b.seq；控制和 hints 不增加业务序号。
2. 新增 Go 断言真实流事件逐帧递增，并用真实 SSE 连接连续发布两题。将该真实 SSE 流通过临时文件送入现有 Node DOM harness，避免手造序号掩盖协议缺陷。临时数据仅为合成答案，不包含地址/token。
3. done/error 满队列时丢弃陈旧业务全量快照，保留有界控制队列并写入最终快照；不能阻塞解题，不能重播旧控制。补充终态与控制保留测试。
4. 范围：pkg/mobile/bridge.go、bridge_server_test.go、web_dom_test.mjs 及 Change 012 文档/审计/INDEX。保持前端序号去重和 API 修复不变，不新增依赖或协议。
5. 执行 Go 聚焦/全量/vet/build、Node DOM/设置回归、前端构建和备用名 Wails 构建；恢复构建生成噪音，保留新 EXE 供用户验收，不覆盖旧程序。真实跨电脑验收仍为 NEEDS_USER。
