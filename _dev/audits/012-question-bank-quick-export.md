# Change 012 架构审计：自动分段题库与远端控制

日期：2026-10-19

## 结论

采用“自动记录 + 当前分段 + 一键整理导出”。Change 010 的 Store、原子写入、截图校验、逐条模型整理与路由前捕获可复用，但必须演进 Session 生命周期、路径布局和 Wails API，不能只替换前端按钮。

## 数据与生命周期边界

- 当前段是唯一可自动追加的分段；封存、导出或清空后不可继续写入。
- 解题开始时没有可写当前段则自动创建。
- 手动分段封存旧段；成功导出同样封存旧段，下一题懒创建新段。
- 清空使用软丢弃状态，不在首版永久删除截图。
- 整理和导出锁定目标分段 ID；异步期间新题不得串入目标段。
- 原始记录、整理结果和导出结果分层保存，模型输出不覆盖原答案。

## 路径语义

- v2 偏好的 ArchiveDir 是最终题库目录；新导出直接写入其下。
- 内部记录目录独立建模，默认位于 ArchiveDir 下的 `_记录`。
- v1 Root 迁移只计算新布局，不删除、移动或覆盖旧记录；Store 同时枚举 v2 和 legacy 记录位置。
- v1 Root 若已以 `求职笔试/题库` 结尾，则直接作为 ArchiveDir；否则使用 `Root/求职笔试/题库`。
- 导出目录只含时间和安全名称；同名使用顺序后缀，不在用户路径显示随机 ID。

## 模型整理边界

- 设置中的持久化授权只控制点击导出时是否调用当前模型，默认关闭。
- 每条记录独立整理，新增 Category 与 Type；Markdown 由本地代码按 Category → Type 分组。
- 取消或部分失败仍导出原稿及成功部分，并保留历史分段供重新整理。
- 不发送简历、API Key、BaseURL 查询参数、KeepContext 历史或其他分段内容。

## Mobile 协议边界

- 快捷键提示是可重放 UI 元数据，可放入全量 Snapshot。
- 滚动是一次性控制，必须有独立递增 ControlSeq。
- SSE `snapshot` 首帧只同步最近 ControlSeq，不执行命令；实时 `message` 才执行新控制。
- ScrollContent 按 outputMode 选择桌面、手机或双端，不把任意 Toast 转发手机。
- Mobile payload 只含动作标签和快捷键显示名，不含 VK 值、Token、路径或秘密。

## Windows hook 静态检查

- lParam 在 hook callback 期间有效，但直接 `uintptr -> unsafe.Pointer` 违反 go vet 检查模式。
- 平台层负责把原生结构复制为 Go 值，shortcut 层只读取返回值。
- 不使用忽略注释、关闭 vet 或改测试命令掩盖问题。

## 风险控制

- 先以 Store 测试锁定自动分段、兼容读取和路径规范，再接入 App 与 UI。
- Mobile 增加初始快照不重放和实时命令序号测试。
- Wails build 前保存生成文件状态，构建后逐项审查 CLI 版本噪音。
- 不修改旧输出，不自动永久删除记录，不执行 Git 提交或发布。

## 实施后审计（2026-10-20）

### 已确认的数据可靠性

- `Begin` 在首次懒建段落盘失败时返回稳定记录 ID 与错误，并保留当前 Session、Record、截图 pending 与 dirty 状态；App 继续关联流式分片和最终答案，目录恢复后可 `Flush`。
- Ended/Exported/Discarded 等非可写状态只要仍为 dirty，就不得被下一次 Begin 替换，避免未落盘终态段被覆盖。
- 当前段普通导出仍通过 `SealCurrent` 防止重复处理；显式历史 API 只接受 Ended/Exported，并按稳定 ID 重试尚未结构化的记录或生成新版本。
- 导出文件已生成但 `session.json` 状态保存失败时，`LastExport` 仍保留实际目录，用户提示明确区分“部分成功”和“完全未生成”。
- 截图入口与整理/导出读取端统一执行解码后 20 MiB 限制，避免产生已保存但无法导出的记录。

### 已确认的协议与安全边界

- Mobile 普通全量快照可在慢消费者队列满时丢弃；一次性控制帧会非阻塞清理积压快照后写入，避免滚动命令被业务帧静默挤掉。
- 初始 SSE `snapshot` 只建立 `ControlSeq` 基线，只有实时 `message` 执行新控制。
- 模型整理继续默认关闭；偏好文件与主密钥配置分离。没有自动调用用户真实模型，也没有发送简历、完整 Config、BaseURL 查询参数或 KeepContext 历史。
- 模型生成的结构化 Markdown 字段会中和 HTML 与外部 URL；原答案逐字保留在原始记录中。
- Windows hook 的原生 callback 参数读取已集中到 platform 层，shortcut 层不再直接执行 `uintptr -> unsafe.Pointer`；没有使用忽略规则或关闭 vet。

### 验证结论

- PASS：聚焦 Go 测试、`go test ./... -count=1`、`go vet ./...`、`go build ./...`、Frontend build、Mobile 内联脚本语法、Wails Windows/amd64 生产构建、`git diff --check`。
- BLOCKED：`go test -race` 在测试进程启动阶段统一返回 `exit status 0xc0000139`，属于既有 Windows race 工具链/运行环境问题，不能记为通过。
- NEEDS_USER：Windows GUI 全流程、手机真机三种输出模式滚动、退出保护弹窗，以及用户主动启用后的非敏感真实模型整理。

### 工具链与提交边界

- 本机 Wails CLI 为 v2.16.0，项目模块为 v2.11.0。bindings 采用精确增量更新，构建使用 `-skipbindings -nosyncgomod`；CLI 改写的三个 runtime shim 经差异和哈希确认后精确恢复。
- `go.mod`、`go.sum` 未变化，没有新增依赖。
- 暂存区为空。Change 010/011 的既有未提交资产仍保留；未执行 commit、push、Tag 或 Release。
- Change 012 在人工验收前保持未关闭。

## 仓库外旧输出整改（2026-10-20）

用户明确指出旧目录仍重复嵌套后，已实际执行一次受控扁平化：3 个 v1 会话迁入 `ArchiveDir/_记录`，5 份旧导出迁入 `ArchiveDir` 根并使用固定 `题库.md`。迁移逐目录比较全部文件 SHA-256，不合并同内容版本，不覆盖现有目标；只在数据验证完成后删除空目录。应用 Store 对真实迁移目录的只读探针成功列出全部 3 个会话，Markdown 图片相对链接全部存在。旧空 `自动网申` 同时移除，保留 `求职网申`。

## Mobile 人工反馈回修审计（2026-10-20）

### 根因与修复边界

- 连接入口只在设置组件首次挂载时读取，读取失败又与服务不可用合并，导致页面无法反映最新状态；临时 `outputMode` 选择也容易被误认为已经生效。
- Server 失败后缺少安全重试入口；直接重跑旧初始化会重复创建 Bridge 或配置订阅，因此改为 Bridge/订阅单例、Server 可重试的生命周期。
- 手机浏览器对每个控制帧无条件应用完整 Snapshot，造成答案与历史 DOM 重建、平滑滚动受干扰以及历史选择跳回最新。
- 顶部箭头已有统一滚动链路，本次只在该控制组内增加 wheel；未接管整个窗口、设置弹窗、答案阅读区或系统全局滚轮，也未修改 Windows 原生 Hook。

### 已确认契约

- `MobileInfo` 使用固定状态 `ready`、`not_started`、`initialization_failed`、`port_unavailable`、`stopped`，并仅在 `ready` 时返回连接入口。
- 端口释放后可通过设置页重试；运行中重试幂等且不轮换连接地址，重试方法不调用窗口隐藏策略。
- 首帧只同步 `controlSeq`；实时帧只有在业务 `seq` 增加时才重建业务 DOM，新 `controlSeq` 只触发一次 70% 视口滚动。
- 查看历史答案时，控制帧不修改 `selectedSeq`；重复控制序号和断线重连均不重放旧命令。
- bindings 已由 Wails CLI 重新生成并完成备用输出名应用编译；项目 runtime 仍保持 v2.11.0，CLI 2.16.0 的 runtime shim 噪音定向恢复。

### 仍需人工确认

- Windows WebView2 中箭头组 wheel/触控板阈值与节流手感。
- `desktop`、`mobile`、`dual` 三种已保存模式下，按钮、键盘与 wheel 的真实路由。
- 手机查看历史答案时滚动不跳回最新，以及 SSE 断线重连不重放。
- 端口占用后释放并从设置页重试，连接入口恢复且桌面窗口不突然隐藏。

## 2026-10-22 实时答案序号审计

已用失败测试确认：Bridge 内部计数增加但实际 Snapshot.Seq 未更新，浏览器去重过滤了正文及完成帧。现广播前统一同步序号，控制/hints 保持不推进业务序号。done/error 满队列清理过时业务帧并保留有界控制，实现最终状态可达。

新增真实 HTTP/SSE 帧驱动页面脚本的两题连续回归，避免手造 seq 掩盖生产者缺陷。全量测试、vet/build、前端与 Wails 构建通过，真实跨电脑体验仍待用户复验。不将测试通过冒充 GUI 通过。
