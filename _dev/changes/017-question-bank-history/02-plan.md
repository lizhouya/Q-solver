# 实施方案 02

状态：用户已批准，允许进入实现；不构成 Git 提交、推送或发布授权。

## 架构方案

1. 在兼容 v1/v2 的 JSON 中增加可选 Session 权威元数据和 Record 模型建议；不提升版本、不批量迁移。
2. 保持 `Store.List()` 全量，在 App/Wails 层提供固定最近 20 列表、只读详情和按需图片接口。详情使用 `SnapshotByID`，接口重新校验目标仍位于最近 20。
3. 历史图片只按 sessionID、recordID 和 imageIndex 定位；前端不能传路径。
4. 清空题库段后，只有后端成功才调用 `solution.resetLiveHistory()`；不清 Solver 上下文、不取消任务、不清 Mobile。
5. 模型整理响应增加可选 `metadataSuggestion`，复用既有逐 Record 调用。启用整理时先整理并返回候选，人工核对后再纯本地导出。
6. 前端题库 store 管理独立只读详情、图片缓存和 latest-wins 请求版本；归档文本只使用文本插值或 `<pre>`，不用 `v-html`。

## 固定接口

```go
ListRecentQuestionBankSessions() (QuestionBankSessionList, error)
GetQuestionBankSessionDetail(id string) (QuestionBankSessionDetail, error)
GetQuestionBankRecordImage(sessionID, recordID string, imageIndex int) (string, error)
```

模型整理与最终导出采用职责分开的 App 方法；最终签名先由 Go 测试锁定。RawOnly 保持单阶段、零模型调用。

## 数据模型

```go
type SourceMetadataSuggestion struct {
    Company    string `json:"company,omitempty"`
    Issuer     string `json:"issuer,omitempty"`
    Type       string `json:"type,omitempty"`
    ReviewNote string `json:"reviewNote,omitempty"`
}
```

- `Record.MetadataSuggestion *SourceMetadataSuggestion`
- `Session.Name string`
- `Session.Issuer string`
- `Session.NameCustomized bool`

全部新字段使用 `omitempty`，不提升 Session 版本。

## 命名、列表和详情规则

- 自动名称优先组合 `招聘单位_出题方_题型`，忽略空值和大小写相同的重复项。
- 单位与出题方均为空时使用 `笔试_YYYYMMDD_NN_题型`。
- 最终目录继续使用 `YYYY-MM-DD_HHMMSS_<名称>` 和 `_02/_03` 防覆盖后缀，不新增物理分层。
- 列表返回 `sessions/total/hidden/limit`，limit 固定 20；第 21 段及更早段不提供详情和图片。
- `discarded` 可只读，但不可整理、导出、恢复或继续追加。
- 单段详情 JSON 上限 8 MiB，超限直接提示去文件夹查看，不截断原答案。
- DTO 不返回恢复目录或图片路径；时间统一 RFC3339Nano。

## 图片安全

复用既有安全函数并校验声明 MIME、扩展名和实际格式一致。拒绝非法 ID、越界索引、绝对路径、`..`、反斜杠、冒号、符号链接、非普通文件、超过 20 MiB 或 80M 像素的文件。

## 实施顺序

1. 以测试锁定兼容模型与严格整理协议。
2. 实现安全历史读取和元数据原子写回。
3. 实现 App/Wails 最近 20 与详情 API。
4. 实现两阶段整理、候选聚合和最终本地导出。
5. 实现前端实时历史重置、只读详情和确认 UI。
6. 使用 Wails v2.11.0 生成 bindings，合并并核对 Change 016 方法。
7. 执行聚焦、全量、前端、Wails、diff 和 race 验证；构建独立 Change 017 验证 EXE，不覆盖旧文件。

## 文件白名单

- `pkg/questionbank/types.go`、`organize.go`、`history.go`、`export.go` 及对应测试。
- `app/questionbank.go`、`questionbank_history.go` 及对应测试。
- `frontend/src/stores/solution.js`、`questionbank.js`。
- `frontend/src/components/QuestionBankPanel.vue`、`QuestionBankExportModal.vue`。
- `frontend/src/services/api.js`、`frontend/questionbank_history_regression_test.mjs`。
- Wails bindings三文件、README 中英文、Change 016/017 文档、审计和 INDEX。

## 回滚与数据安全

回滚只撤销 Change 017 的代码、API、DTO 与 UI，不删除或重写题库文件。新 JSON 字段均为 `omitempty`；已写入字段即使代码回滚也保留。已 discarded 的段不恢复为可写。
