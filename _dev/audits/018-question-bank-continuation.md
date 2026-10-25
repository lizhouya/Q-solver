# Change 018 题库历史分段真正续接审计

日期：2026-10-23
状态：实现后复核完成；普通自动验证 PASS；GUI NEEDS_USER；race BLOCKED。

## 审计结论

### 会话一致性：PASS

题库 current、Solver 非 system turn、App retained refs 和前端 writable item 由 conversation epoch 关联。Continue 在同一 `conversationMu` 临界区内预解析目标、Replace Solver、激活 Store；Store 拒绝时恢复旧 Solver，refs 与 epoch 不推进。模型成功但最终题库落盘失败时同样回滚 Solver 与 refs。

### 旧数据可信度：PASS

旧 Record 仅以 `completed && Answer != ""` 进入重建，截图为 user image、Answer 为 assistant，并显示固定警告。failed/cancelled/interrupted/empty 仅展示。新 snapshot 只存 Session/Record 引用及 KeepContext 边界，不宣称保存原 system。

### 并发与迟到事件：PASS（普通测试）

TaskCoordinator 独立记录 in-flight；Cancel 后到 goroutine Complete 前仍拒绝切换。桌面事件必须同时匹配 epoch、sessionId、recordId；A→B 后 A 的迟到 chunk/error/solution 被拒绝。图片请求有独立失效代际。Mobile 仍由原 display epoch 隔离，未复用 conversation epoch。

### Store 事务与故障恢复：PASS

source pause、target activate、source rollback 三阶段均有确定性失败测试。单次失败可补偿；双重失败后内存对齐磁盘 paused 状态并全局阻断写操作，只读访问保留。完成态写失败使用独立 pending completion，Retry 原子提交 Record 与 ConversationSnapshot，避免 running 记录永久无恢复路径。

### 导出与数据保留：PASS

exported 续接不删除或重写旧导出元数据；追加、再次结束和导出创建 `_02`，测试确认旧原始稿字节不变。第 21 段及更早数据只隐藏，不删除、不自动整理或导出。

### 隐私与指令边界：PASS

ConversationSnapshot 不包含 API Key、Mobile Token、完整 Config、Prompt、ResumeContent 或 thinking。历史 Answer 不进入 system。测试使用本地 fake provider 与生成图片，没有真实模型调用、上传或额度消耗。

### 分段操作区布局：PASS（结构与构建） / NEEDS_USER（视觉）

桌面端操作列固定为 `7.5rem`，继续与重新导出按钮纵向等宽，卡片内容上下排列；窄屏改为全宽单列。静态回归锁定结构且 Vite/Wails 构建通过；WebView2 中的最终几何、长文本和键盘焦点仍需人工验收。业务状态矩阵、事件和可访问性属性未改。

### 生成与发布边界：PASS

bindings 由固定 Wails v2.11.0 正式生成；验证 EXE 独立归档并通过 SHA-256。未覆盖正式入口，未执行 commit、push、Tag 或 Release。

## 未关闭项

- Windows race 四个包均在启动阶段返回 `0xc0000139`，状态 BLOCKED，普通测试不能替代。
- Windows GUI 仍需按 Change 018 `04-validation.md` 完成 11 项人工验收，包括最新分段操作区布局。
- 在上述人工证据完成前，Change 018 不得关闭或发布。
