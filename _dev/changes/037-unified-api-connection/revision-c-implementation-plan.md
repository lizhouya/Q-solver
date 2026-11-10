# Change 037 Revision C 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 为每个规范化模型连接保存有界模型历史，修复快捷键组合键持久化及全显示端提示同步，并用固定隔离 Profile 验证题库目录、模型历史和快捷键跨重启保留。

**架构：** 后端 Config 持有不含凭据的 `ModelHistoryEntry` 列表，并在 API 局部保存与全量保存事务成功前将当前模型写入候选快照；前端按当前草稿的规范化连接身份筛选历史，与目录候选分组展示。快捷键保留现有“录制成功立即更新运行态、保存全部设置后持久化”的行为，修复 `comboID` 字段并统一 GhostMode/WindowVisibility 文案；题库产品代码不变，只复用固定隔离 Profile 验证既有持久化。

**技术栈：** Go 1.24.1+ 模块（本机 go1.27.1）、Wails v2.11.0、Vue 3、Pinia、Vite 4.5.14、Node.js 22+；不新增依赖。

**Spec:** `_dev/changes/037-unified-api-connection/design.md`（Revision C，第 15～23 节）

## 全局约束

- 遵守 `AGENTS.md > CLAUDE.md > Change 037`；固定分支 `dev/qsolver-custom`，禁止修改 `main`。
- 不新增依赖，不升级 Go、Wails、Vue、Node 或 SDK；Wails CLI/runtime 固定 v2.11.0。
- 不读取、复制、打印或修改正式用户配置；配置实值只能报告 `SET` / `NOT SET`。
- 模型历史不得包含 API Key、Headers、请求模板、Pointer、响应、Token 或题库内容。
- GhostMode、WindowVisibility、ClickThrough、OutputMode 保持独立语义。
- 题库持久化产品代码不修改；固定隔离 Profile 仅用于候选人工验证。
- 每个任务先写失败测试，再实现最小代码，再运行聚焦测试。
- 用户尚未授权 commit、push、稳定版晋升、Tag 或 Release；所有“检查点”保持未暂存，不执行 commit。
- `build/bin/**` 继续 ignored，禁止 `git add -f`；根稳定版和 `build/bin/releases/**` 不得改变。

---

## 文件结构锁定

### 新建

- `pkg/config/model_history.go`：模型历史常量、规范化、身份匹配、记录与读取。
- `pkg/config/model_history_test.go`：历史旧配置兼容、身份隔离、去重/LRU、容量和安全边界。
- `frontend/model_history_regression_test.mjs`：Store 历史筛选/保存事务与 ModelSelect 分组回归。
- `frontend/shortcut_hint_sync_regression_test.mjs`：`comboID`、桌面提示响应式绑定和统一文案回归。

### 修改

- `pkg/config/config.go`：增加 `ModelHistoryEntry` 与 `Config.ModelHistory`。
- `pkg/config/connection.go`：在连接规范化时清洗历史，但不因旧配置自动记录当前模型。
- `pkg/config/manager.go`：全量保存成功前记录当前模型；深拷贝历史。
- `pkg/config/manager_test.go`：全量保存记录与失败原子性。
- `app/llm.go`：API 局部保存候选记录模型；安全 DTO 返回当前连接 `SavedModels`。
- `app/llm_test.go`、`app/settings_test.go`：局部保存、安全返回和失败原子性。
- `frontend/src/stores/settings.js`：历史基线、连接筛选、保存成功更新、目录/历史去重。
- `frontend/src/components/ModelSelect.vue`：分组显示“已保存模型”和“目录候选”。
- `frontend/src/components/SettingsModal.vue`：传递两组模型；模板模式继续禁用目录刷新。
- `frontend/src/App.vue`：录制事件写 `comboID`；成功后同步桌面运行态提示。
- `frontend/src/components/WelcomeView.vue`：F9 文案改为“隐身模式”。
- `app/mobile.go`、`app/mobile_test.go`：手机端统一 GhostMode/WindowVisibility 文案并验证自定义键名。
- `README.md`、`README_EN.md`：快捷键语义与模型历史说明。
- `frontend/wailsjs/go/models.ts`：由 Wails v2.11.0 生成新 Config/DTO 类型。
- `_dev/changes/037-unified-api-connection/00-request.md`、`02-plan.md`、`03-implementation.md`、`04-validation.md`、`05-rollback.md`、`06-notes.md`、`design.md`、`revision-c-implementation-plan.md`、`_dev/changes/INDEX.md`：如实记录实现、验证、人工项与提交边界。

### 明确不修改

- `app/questionbank.go`、`frontend/src/stores/questionBank.js`：审计已证明目录缺失来自不同隔离 Profile，不改产品逻辑。
- `pkg/shortcut/service.go`：保留录制成功立即更新运行态的现有行为。
- `pkg/mobile/web/index.html`：手机端从 Snapshot 动态渲染标签/键名，无需硬编码变更。
- `go.mod`、`go.sum`、`frontend/wailsjs/runtime/**`、`build/bin/releases/**`、正式用户配置。

---

### Task 1：建立后端模型历史领域模型

**Files:**
- Create: `pkg/config/model_history.go`
- Create: `pkg/config/model_history_test.go`
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/connection.go`

**Interfaces:**
- Produces: `ModelHistoryEntry`、`NormalizeModelHistory([]ModelHistoryEntry) []ModelHistoryEntry`、`(*Config).RememberCurrentModel()`、`(Config).SavedModelsForCurrentConnection() []string`。
- Consumes: `protocol.Normalize`、`NormalizeBaseURL`、`NormalizeCustomRequestURL`、`NormalizeEndpointPath` 和已规范化 `Config` 连接字段。
- Invariant: 每连接最多 20 个模型、最多 12 个连接、模型名称最多 256 UTF-8 字节；无凭据字段。

- [x] **Step 1：写类型缺失和旧配置兼容红灯**

在 `pkg/config/model_history_test.go` 写：

```go
func TestModelHistoryMissingFieldLoadsAsEmpty(t *testing.T) {
    cfg := NewDefaultConfig()
    if err := json.Unmarshal([]byte(`{"baseURL":"https://example.test/v1","apiProtocol":"openai_chat","model":"m1"}`), &cfg); err != nil {
        t.Fatal(err)
    }
    normalized, err := cfg.NormalizedConnectionSettings()
    if err != nil { t.Fatal(err) }
    if len(normalized.ModelHistory) != 0 {
        t.Fatalf("missing field must not auto-migrate: %#v", normalized.ModelHistory)
    }
}

func TestRememberCurrentModelUsesNormalizedConnectionIdentity(t *testing.T) {
    cfg := NewDefaultConfig()
    cfg.BaseURL = "https://example.test/v1/"
    cfg.Model = "model-a"
    normalized, err := cfg.NormalizedConnectionSettings()
    if err != nil { t.Fatal(err) }
    normalized.RememberCurrentModel()
    if len(normalized.ModelHistory) != 1 || normalized.ModelHistory[0].BaseURL != "https://example.test/v1" {
        t.Fatalf("history=%#v", normalized.ModelHistory)
    }
    if got := normalized.SavedModelsForCurrentConnection(); !reflect.DeepEqual(got, []string{"model-a"}) {
        t.Fatalf("saved=%#v", got)
    }
}
```

- [x] **Step 2：运行红灯**

Run:

```bash
go test ./pkg/config -run 'Test(ModelHistory|RememberCurrentModel)' -count=1
```

Expected: FAIL，`ModelHistory`、`RememberCurrentModel` 或相关类型未定义。

- [x] **Step 3：增加持久类型与默认值**

在 `pkg/config/config.go` 增加：

```go
type ModelHistoryEntry struct {
    APIProtocol  string   `json:"apiProtocol"`
    BaseURL      string   `json:"baseURL"`
    EndpointPath string   `json:"endpointPath,omitempty"`
    Models       []string `json:"models"`
}
```

在 `Config` 增加：

```go
ModelHistory []ModelHistoryEntry `json:"modelHistory,omitempty"`
```

`NewDefaultConfig()` 显式初始化为空切片：

```go
ModelHistory: []ModelHistoryEntry{},
```

- [x] **Step 4：实现历史规范化和身份匹配**

新建 `pkg/config/model_history.go`，定义：

```go
const (
    MaxModelHistoryConnections = 12
    MaxModelsPerConnection      = 20
    MaxModelHistoryNameBytes    = 256
)

func NormalizeModelHistory(entries []ModelHistoryEntry) []ModelHistoryEntry
func (c Config) SavedModelsForCurrentConnection() []string
func (c *Config) RememberCurrentModel()
```

实现要求：

```go
func validHistoryModelName(value string) (string, bool) {
    value = strings.TrimSpace(value)
    return value, value != "" && len([]byte(value)) <= MaxModelHistoryNameBytes
}

func sameHistoryConnection(entry ModelHistoryEntry, cfg Config) bool {
    return entry.APIProtocol == cfg.APIProtocol &&
        entry.BaseURL == cfg.BaseURL &&
        entry.EndpointPath == cfg.EndpointPath
}
```

`NormalizeModelHistory` 必须逐条：

1. 使用与连接配置相同的协议、URL、Endpoint 规范化函数。
2. 跳过非法协议、URL、Endpoint、空模型名和超长模型名。
3. 同一连接和模型精确字符串去重，保留首次出现顺序。
4. 每连接截断 20 项，连接截断 12 项。
5. 返回全新的 entry 和 model slice，不复用输入底层数组。

`RememberCurrentModel` 先清洗既有历史；当前模型无效时只保留清洗结果，不新增；有效时把当前模型移动到当前连接模型列表首位，并把当前连接移动到历史首位。

- [x] **Step 5：接入连接规范化但禁止自动迁移**

在 `Config.NormalizedConnectionSettings()` 的 `custom_json` 和非 custom 返回前仅执行：

```go
c.ModelHistory = NormalizeModelHistory(c.ModelHistory)
```

禁止在加载/规范化路径调用 `RememberCurrentModel()`，保证旧配置不会因读取而写入历史。

- [x] **Step 6：补充边界红灯/绿灯**

在 `model_history_test.go` 增加：

```go
func TestModelHistoryIsolatedByConnectionAndKeyRotationDoesNotSplit(t *testing.T)
func TestRememberCurrentModelMovesExactDuplicateToFront(t *testing.T)
func TestNormalizeModelHistoryLimitsConnectionsAndModels(t *testing.T)
func TestNormalizeModelHistorySkipsInvalidAndOverlongNames(t *testing.T)
func TestSavedModelsReturnsCopy(t *testing.T)
func TestCustomJSONHistoryUsesFullRequestURLAndEmptyEndpoint(t *testing.T)
```

关键断言：

```go
copy := cfg.SavedModelsForCurrentConnection()
copy[0] = "mutated"
if cfg.ModelHistory[0].Models[0] == "mutated" { t.Fatal("history alias leaked") }
```

- [x] **Step 7：运行 Task 1 聚焦测试**

Run:

```bash
gofmt -w pkg/config/config.go pkg/config/connection.go pkg/config/model_history.go pkg/config/model_history_test.go
go test ./pkg/config -run 'Test(ModelHistory|RememberCurrentModel|NormalizeModelHistory|SavedModels|CustomJSONHistory)' -count=1
```

Expected: PASS。

- [x] **Step 8：检查点（不提交）**

Run:

```bash
git diff --check -- pkg/config/config.go pkg/config/connection.go pkg/config/model_history.go pkg/config/model_history_test.go
git status --short
```

Expected: 仅出现 Change 037 预期未提交文件；暂存区保持空。

---

### Task 2：把模型历史纳入 ConfigManager 事务

**Files:**
- Modify: `pkg/config/manager.go`
- Modify: `pkg/config/manager_test.go`

**Interfaces:**
- Consumes: Task 1 的 `RememberCurrentModel()` 和 `NormalizeModelHistory()`。
- Produces: 全量保存成功前记录当前模型；所有 Config 快照深拷贝历史。
- Invariant: 保存/验证失败不得改变磁盘哈希、内存 Config、`oldConfig`、subscriber 或历史。

- [x] **Step 1：写全量保存红灯**

在 `manager_test.go` 增加：

```go
func TestUpdateFromJSONRemembersCurrentModelAfterSuccessfulSave(t *testing.T) {
    manager := isolatedManager(t)
    if err := manager.Load(); err != nil { t.Fatal(err) }
    cfg := NewDefaultConfig()
    cfg.BaseURL = "https://example.test/v1"
    cfg.APIProtocol = "openai_chat"
    cfg.EndpointPath = "/chat/completions"
    cfg.Model = "model-a"
    payload, err := json.Marshal(cfg)
    if err != nil { t.Fatal(err) }
    if err := manager.UpdateFromJSON(string(payload)); err != nil { t.Fatal(err) }
    got := manager.Get().SavedModelsForCurrentConnection()
    if !reflect.DeepEqual(got, []string{"model-a"}) { t.Fatalf("saved=%#v", got) }
}
```

再增加：

```go
func TestUpdateFromJSONFailedSaveLeavesModelHistoryUnchanged(t *testing.T)
func TestCloneConfigSnapshotDeepCopiesModelHistory(t *testing.T)
func TestLoadOldConfigDoesNotRememberCurrentModelOrRewriteFile(t *testing.T)
```

- [x] **Step 2：运行红灯**

Run:

```bash
go test ./pkg/config -run 'Test(UpdateFromJSON.*Model|CloneConfigSnapshot.*Model|LoadOldConfig.*Model)' -count=1
```

Expected: FAIL，成功保存后历史仍为空或快照共享 slice。

- [x] **Step 3：全量保存候选记录模型**

在 `UpdateFromJSON` 完成连接规范化后、`Validate()` 和 `saveSnapshot()` 前执行：

```go
newConfig = normalized
newConfig.RememberCurrentModel()
if err := newConfig.Validate(); err != nil { return err }
```

这样历史只存在于候选快照；后续验证/写盘失败不会交换内存状态。

- [x] **Step 4：深拷贝模型历史**

扩展 `cloneConfigSnapshot`：

```go
if source.ModelHistory != nil {
    result.ModelHistory = make([]ModelHistoryEntry, len(source.ModelHistory))
    for i, entry := range source.ModelHistory {
        result.ModelHistory[i] = entry
        result.ModelHistory[i].Models = append([]string(nil), entry.Models...)
    }
}
```

同时把 `UpdateFromJSON` 的旧配置快照改为 `oldConfigCopy := cloneConfigSnapshot(cm.config)`，并在交换后用 `configCopy := cloneConfigSnapshot(cm.config)` 通知 subscriber，避免后续修改共享历史底层数组。

- [x] **Step 5：验证通知和磁盘原子性**

测试必须记录保存前加密文件 SHA-256、subscriber 调用次数和 `manager.Get().ModelHistory`；向不可写路径或无效配置保存后断言三者均不变。

- [x] **Step 6：运行 Config 全包**

Run:

```bash
gofmt -w pkg/config/manager.go pkg/config/manager_test.go
go test ./pkg/config -count=1
```

Expected: PASS。

- [x] **Step 7：检查点（不提交）**

```bash
git diff --check -- pkg/config
git diff --cached
```

Expected: 无空白错误，暂存区为空。

---

### Task 3：API 局部保存记录模型并返回安全历史

**Files:**
- Modify: `app/llm.go`
- Modify: `app/llm_test.go`
- Modify: `app/settings_test.go`

**Interfaces:**
- Consumes: `Config.RememberCurrentModel()`、`Config.SavedModelsForCurrentConnection()`、`ConfigManager.PatchValidated`。
- Produces: `ConnectionSettingsView.SavedModels []string`。
- Invariant: DTO 不返回模型历史中的其他连接、Key、Header、模板或 Pointer。

- [x] **Step 1：写局部保存红灯**

在 `app/llm_test.go` 增加：

```go
func TestSaveConnectionSettingsRemembersModelForNormalizedConnection(t *testing.T) {
    app := newConnectionSettingsTestApp(t)
    result := app.SaveConnectionSettings(ConnectionDraft{
        ConnectionMode: "direct", BaseURL: "https://example.test/v1/",
        APIProtocol: "openai_chat", EndpointPath: "/chat/completions", Model: "model-a",
        ResponseFormat: "text", ExtraHeaders: map[string]string{},
    })
    if !result.Success || result.Connection == nil { t.Fatalf("result=%+v", result) }
    if !reflect.DeepEqual(result.Connection.SavedModels, []string{"model-a"}) {
        t.Fatalf("saved=%#v", result.Connection.SavedModels)
    }
}
```

增加：

```go
func TestSaveConnectionSettingsHistoryIsAtomicOnValidationFailure(t *testing.T)
func TestConnectionSettingsViewReturnsOnlyCurrentConnectionModels(t *testing.T)
func TestConnectionSettingsViewSavedModelsIsDetachedCopy(t *testing.T)
```

- [x] **Step 2：运行红灯**

Run:

```bash
go test ./app -run 'Test(SaveConnectionSettings.*Model|ConnectionSettingsView.*Model)' -count=1
```

Expected: FAIL，`SavedModels` 未定义或历史未记录。

- [x] **Step 3：扩展安全 DTO**

在 `ConnectionSettingsView` 增加：

```go
SavedModels []string `json:"savedModels"`
```

在 `connectionSettingsView` 中只读取当前连接：

```go
SavedModels: cfg.SavedModelsForCurrentConnection(),
```

禁止返回完整 `ModelHistory`，避免 API 专用保存响应暴露其他内部连接地址。

- [x] **Step 4：局部保存前规范化草稿并在候选中记录**

`SaveConnectionSettings` 先执行：

```go
connection, err := draft.config().NormalizedConnectionSettings()
if err != nil { return failedConnectionSave(err) }
```

`PatchValidated` 的 patch 回调按现有白名单复制规范化连接字段，然后：

```go
cfg.RememberCurrentModel()
```

因为 `cfg` 只存在于事务候选中，Provider 校验或 `saveSnapshot` 失败不会更新历史。

- [x] **Step 5：扩展安全泄露测试**

在 `app/settings_test.go` 的 JSON 序列化结果中继续禁止出现 Key、Header 值、模板 marker、Pointer、AuthPrefix；同时构造第二连接历史，断言局部保存返回 JSON 不含第二连接 URL 或模型名。

- [x] **Step 6：运行 App/Config 聚焦与全包**

Run:

```bash
gofmt -w app/llm.go app/llm_test.go app/settings_test.go
go test ./pkg/config ./app -count=1
```

Expected: PASS。

- [x] **Step 7：检查点（不提交）**

```bash
git diff --check -- app/llm.go app/llm_test.go app/settings_test.go
git diff --cached
```

Expected: 暂存区为空。

---

### Task 4：前端 Store 按连接管理历史

**Files:**
- Create: `frontend/model_history_regression_test.mjs`
- Modify: `frontend/src/stores/settings.js`
- Modify: `frontend/unified_api_settings_regression_test.mjs`

**Interfaces:**
- Consumes: `config.modelHistory`、`SaveConnectionResult.connection.savedModels`。
- Produces: `savedModelsForCurrentConnection`、`upsertModelHistory(entries, identity, models)`。
- Invariant: 保存失败、probe、目录刷新和草稿修改不得更新历史。

- [x] **Step 1：写 Store 红灯**

在新文件中复制 `frontend/unified_api_settings_regression_test.mjs:1-60` 的 `vm`/Pinia `setup(overrides)` harness（保持同名 `setup` 返回 `{ store, calls, ui }`），只把 fixture 扩展为：

```js
modelHistory: [
  { apiProtocol: 'openai_chat', baseURL: 'https://a.test/v1', endpointPath: '/chat/completions', models: ['a-new', 'a-old'] },
  { apiProtocol: 'custom_json', baseURL: 'https://b.test/generate', endpointPath: '', models: ['b-only'] },
]
```

断言：

```js
assert.deepEqual([...store.savedModelsForCurrentConnection], ['a-new', 'a-old'])
store.tempSettings.baseURL = 'https://b.test/generate'
store.tempSettings.apiProtocol = 'custom_json'
store.tempSettings.endpointPath = ''
assert.deepEqual([...store.savedModelsForCurrentConnection], ['b-only'])
```

增加失败用例：

```js
const failed = setup({
  saveConnectionSettings: async () => ({ success: false, error: { message: 'failed' } }),
})
await failed.store.loadSettings()
await failed.store.openSettings()
const beforeHistory = structuredClone(failed.store.settings.modelHistory)
assert.equal(await failed.store.saveConnectionSettings(), false)
assert.deepEqual(failed.store.settings.modelHistory, beforeHistory)
```

目录候选保持在 `ui.availableModels`；与历史的去重由 Task 5 的 `ModelSelect` 分组 computed 负责，Store 不维护第三份合并数组。

- [x] **Step 2：运行红灯**

Run:

```bash
node frontend/model_history_regression_test.mjs
```

Expected: FAIL，历史 computed/helper 不存在。

- [x] **Step 3：把历史纳入 Settings 基线**

在 `settings` 增加：

```js
modelHistory: [],
```

`applyConfig` 深拷贝：

```js
settings.modelHistory = JSON.parse(JSON.stringify(config.modelHistory || []))
```

`buildConfigToSave` 显式携带有界历史，不依赖对象展开偶然保留：

```js
modelHistory: JSON.parse(JSON.stringify(sourceSettings.modelHistory || [])),
```

- [x] **Step 4：实现前端规范化身份与历史筛选**

在 Store 中增加：

```js
function normalizedHistoryIdentity(source) {
  const apiProtocol = source?.apiProtocol || 'openai_chat'
  try {
    const url = new URL(String(source?.baseURL || '').trim())
    if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || url.search || url.hash) return null
    if (apiProtocol === 'custom_json') {
      return { apiProtocol, baseURL: url.toString(), endpointPath: '' }
    }
    let pathname = url.pathname.replace(/\/+$/, '')
    if (apiProtocol === 'openai_chat' && pathname.endsWith('/chat/completions')) {
      pathname = pathname.slice(0, -'/chat/completions'.length)
    }
    url.pathname = pathname || '/'
    const baseURL = url.toString().replace(/\/$/, '')
    const endpointPath = apiProtocol === 'openai_chat'
      ? '/chat/completions'
      : (source?.endpointPath || defaultEndpointPath(apiProtocol))
    return { apiProtocol, baseURL, endpointPath }
  } catch (_) {
    return null
  }
}

function sameHistoryIdentity(entry, identity) {
  return Boolean(identity) && entry?.apiProtocol === identity.apiProtocol &&
    entry?.baseURL === identity.baseURL && entry?.endpointPath === identity.endpointPath
}

function upsertModelHistory(entries, identity, models) {
  const cleanModels = [...new Set((models || [])
    .map(model => String(model).trim())
    .filter(Boolean))].slice(0, 20)
  const rest = (entries || [])
    .filter(entry => !sameHistoryIdentity(entry, identity))
    .map(entry => ({ ...entry, models: [...(entry.models || [])].slice(0, 20) }))
  if (!identity || cleanModels.length === 0) return rest.slice(0, 12)
  return [{ ...identity, models: cleanModels }, ...rest].slice(0, 12)
}
```

`normalizedHistoryIdentity` 必须使用 `new URL()`：拒绝 credential/query/fragment；Chat 去除完整 `/chat/completions` 后再得到 root，并固定 endpoint；custom 保留完整 URL、endpoint 为空。无法解析时返回 `null`，历史 computed 返回空。后端仍负责 UTF-8 256 字节和全部安全边界，前端 helper 只用于展示与构造候选。

增加：

```js
const savedModelsForCurrentConnection = computed(() => {
  const identity = normalizedHistoryIdentity(tempSettings)
  const entry = (settings.modelHistory || []).find(item => sameHistoryIdentity(item, identity))
  return entry ? [...entry.models] : []
})
```

- [x] **Step 5：API 专用保存成功后更新历史**

读取 `canonical.savedModels`，用 canonical 连接身份更新：

```js
const nextHistory = upsertModelHistory(
  settings.modelHistory,
  normalizedHistoryIdentity(savedConnection),
  canonical.savedModels || [savedConnection.model],
)
savedConnection.modelHistory = nextHistory
```

随后同步 `settings`、`tempSettings`、`backendSnapshot`。失败和迟到响应路径禁止执行 upsert。

- [x] **Step 6：全量保存 payload 预先记录当前模型**

`buildConfigToSave` 使用同一 helper，把当前非空模型加入候选 `modelHistory`。后端再次执行同一事务规则，结果应幂等。保存成功后 `settings.modelHistory` 与 `backendSnapshot.modelHistory` 采用 payload；失败时保持旧基线。

- [x] **Step 7：模板模式和连接切换回归**

测试：

- custom 模式可显示历史，但 `refreshModels()` 返回 false 且 bridge 调用数为 0。
- A→B→A 同 tick 切换时显示各自历史，迟到目录结果不能复活。
- 更换 API Key 不改变历史身份。
- Base URL 尾斜线与完整 Chat endpoint 能匹配同一历史。

- [x] **Step 8：运行前端 Store 回归**

Run:

```bash
node frontend/model_history_regression_test.mjs
node frontend/unified_api_settings_regression_test.mjs
node frontend/model_connectivity_regression_test.mjs
```

Expected: PASS。

- [x] **Step 9：检查点（不提交）**

```bash
git diff --check -- frontend/src/stores/settings.js frontend/model_history_regression_test.mjs frontend/unified_api_settings_regression_test.mjs
git diff --cached
```

---

### Task 5：模型下拉分组显示历史与目录候选

**Files:**
- Modify: `frontend/src/components/ModelSelect.vue`
- Modify: `frontend/src/components/SettingsModal.vue`
- Modify: `frontend/model_history_regression_test.mjs`
- Modify: `frontend/model_layout_regression_test.mjs`

**Interfaces:**
- Consumes: `savedModelsForCurrentConnection` 和 `ui.availableModels`。
- Produces: props `savedModels: string[]`、`catalogModels: string[]`；仍 emit `update:modelValue`、`open-change`。
- Invariant: 已保存项优先且目录组不重复；自由输入和 Teleport 定位行为不变。

- [x] **Step 1：写 UI 红灯**

静态断言 `SettingsModal.vue`：

```js
assert.ok(modal.includes(':saved-models="settingsStore.savedModelsForCurrentConnection"'))
assert.ok(modal.includes(':catalog-models="ui.availableModels"'))
```

断言 `ModelSelect.vue` 含：

```js
assert.ok(modelSelect.includes('已保存模型'))
assert.ok(modelSelect.includes('目录候选'))
assert.ok(modelSelect.includes('savedModels'))
assert.ok(modelSelect.includes('catalogModels'))
```

- [x] **Step 2：运行红灯**

```bash
node frontend/model_history_regression_test.mjs
```

Expected: FAIL，组件仍只有 `models` prop。

- [x] **Step 3：拆分 props 与过滤结果**

`ModelSelect.vue` props：

```js
savedModels: { type: Array, default: () => [] },
catalogModels: { type: Array, default: () => [] },
```

computed：

```js
function filterModels(models) {
  const query = searchQuery.value.trim().toLowerCase()
  const values = (models || []).map(String)
  return query ? values.filter(model => model.toLowerCase().includes(query)) : values
}

const filteredSavedModels = computed(() => filterModels(props.savedModels))
const filteredCatalogModels = computed(() => {
  const saved = new Set(props.savedModels.map(String))
  return filterModels(props.catalogModels).filter(model => !saved.has(String(model)))
})
const hasSuggestions = computed(() => filteredSavedModels.value.length > 0 || filteredCatalogModels.value.length > 0)
```

`filterModels` 对 `searchQuery` 做现有大小写不敏感过滤，不修改原数组。

- [x] **Step 4：渲染两组**

Dropdown 中按顺序渲染：

```vue
<template v-if="hasSuggestions">
  <section v-if="filteredSavedModels.length" class="model-group">
    <p class="model-group-label">已保存模型</p>
    <button
      v-for="model in filteredSavedModels"
      :key="`saved:${model}`"
      type="button"
      class="dropdown-item"
      :class="{ selected: modelValue === model }"
      @click="selectModel(model)"
    >
      <span class="model-name">{{ model }}</span>
      <span class="provider-name">{{ getProviderName(model) }}</span>
      <span v-if="modelValue === model" class="check-icon">✓</span>
    </button>
  </section>
  <section v-if="filteredCatalogModels.length" class="model-group">
    <p class="model-group-label">目录候选</p>
    <button
      v-for="model in filteredCatalogModels"
      :key="`catalog:${model}`"
      type="button"
      class="dropdown-item"
      :class="{ selected: modelValue === model }"
      @click="selectModel(model)"
    >
      <span class="model-name">{{ model }}</span>
      <span class="provider-name">{{ getProviderName(model) }}</span>
      <span v-if="modelValue === model" class="check-icon">✓</span>
    </button>
  </section>
</template>
```

复用 `selectModel`；不增加删除按钮。

- [x] **Step 5：SettingsModal 传入分组数据**

替换原 `:models="ui.availableModels"`：

```vue
:saved-models="settingsStore.savedModelsForCurrentConnection"
:catalog-models="ui.availableModels"
```

模板模式仍只禁用刷新按钮，不能禁用 ModelSelect。

- [x] **Step 6：样式不引入裁切**

组标题使用小字号和 sticky 非必要；只增加：

```css
.model-group + .model-group { border-top: 1px solid var(--border-subtle); }
.model-group-label { margin: 0; padding: 8px 12px 4px; color: var(--text-tertiary); font-size: 11px; }
```

保持 dropdown 的 `overscroll-behavior: contain`、`touch-action: pan-y`、Teleport 和 `min-width: 0`。

- [x] **Step 7：运行布局与历史测试**

```bash
node frontend/model_history_regression_test.mjs
node frontend/model_layout_regression_test.mjs
```

Expected: PASS。

- [x] **Step 8：检查点（不提交）**

```bash
git diff --check -- frontend/src/components/ModelSelect.vue frontend/src/components/SettingsModal.vue frontend/model_history_regression_test.mjs frontend/model_layout_regression_test.mjs
```

---

### Task 6：修复快捷键字段并同步桌面提示

**Files:**
- Create: `frontend/shortcut_hint_sync_regression_test.mjs`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/stores/settings.js`
- Modify: `frontend/src/components/WelcomeView.vue`
- Modify: `frontend/settings_regression_test.mjs`
- Verify only: `frontend/src/components/TopBar.vue`
- Verify only: `frontend/src/components/ScreenshotDock.vue`
- Verify only: `frontend/src/components/EmptyState.vue`
- Verify only: `frontend/src/components/ErrorView.vue`

**Interfaces:**
- Consumes: backend `key-recorded {action,keyName,comboID}` 和 `shortcut-saved action`。
- Produces: `tempShortcuts[action].comboID` 与运行态 `shortcuts[action]` 同步。
- Invariant: 不创建 `vkCode`；保存失败/未保存重启语义按设计保留。

- [x] **Step 1：写字段与同步红灯**

在新回归脚本断言：

```js
assert.ok(app.includes('.comboID = data.comboID'))
assert.equal(app.includes('.vkCode = data.comboID'), false)
assert.ok(app.includes('settingsStore.shortcuts[action]'))
assert.ok(app.includes('settingsStore.tempShortcuts[action]'))
```

Settings harness 模拟：

```js
store.tempShortcuts.screenshot = { keyName: 'Ctrl+1', comboID: '49+162' }
// shortcut-saved handler 后
assert.equal(store.shortcuts.screenshot.keyName, 'Ctrl+1')
assert.equal(store.shortcuts.screenshot.comboID, '49+162')
assert.equal(store.solveShortcut, 'Ctrl+1')
```

- [x] **Step 2：运行红灯**

```bash
node frontend/shortcut_hint_sync_regression_test.mjs
```

Expected: FAIL，源码仍写 `vkCode` 或未同步运行态 Store。

- [x] **Step 3：修复录制字段**

`key-recorded` 更新既有/新 binding 时统一写：

```js
settingsStore.tempShortcuts[data.action] = {
  ...(settingsStore.tempShortcuts[data.action] || {}),
  keyName: data.keyName,
  comboID: data.comboID,
}
```

不得保留新增的 `vkCode` 字段；已有后端配置中的未知字段由 JSON 序列化自然忽略，不做破坏性迁移。

- [x] **Step 4：录制成功后同步桌面运行态提示**

`shortcut-saved` 中在结束录制状态前执行：

```js
const draft = settingsStore.tempShortcuts[action]
if (draft) settingsStore.shortcuts[action] = JSON.parse(JSON.stringify(draft))
```

Toast 改为：“快捷键已应用；保存全部设置后将在重启后保留”。这不触发全量保存，不夹带其他设置草稿。

- [x] **Step 5：统一 F9 文案**

`settings.js`：

```js
{ action: 'toggle', label: '隐身模式', default: 'F9', macDefault: 'Cmd+2' },
{ action: 'toggle_window', label: '隐藏/显示窗口', default: 'Ctrl+Shift+H', macDefault: 'Cmd+4' },
```

`WelcomeView.vue` 将 F9 对应文字从“隐藏窗口”改为“隐身模式”。不要把 Welcome 的 binding 换成 `toggle_window`，保持截图中的第三个紧凑提示为 F9 GhostMode。

- [x] **Step 6：验证其他桌面显示端使用响应式值**

静态断言：

- `TopBar.vue` 使用 `settingsStore.solveShortcut/sendShortcut/toggleShortcut`。
- `ScreenshotDock.vue` 通过 `storeToRefs` 使用 solve/send/delete。
- `EmptyState.vue` 与 `ErrorView.vue` 由 `SolveView.vue` 传入实际 computed shortcut。
- 禁止这些组件硬编码 `F8`/`Ctrl+J` 作为运行时展示（prop default 仅作为组件降级允许）。

- [x] **Step 7：运行快捷键前端回归**

```bash
node frontend/shortcut_hint_sync_regression_test.mjs
node frontend/settings_regression_test.mjs
node frontend/screenshot_dock_regression_test.mjs
```

Expected: PASS。

- [x] **Step 8：检查点（不提交）**

```bash
git diff --check -- frontend/src/App.vue frontend/src/stores/settings.js frontend/src/components/WelcomeView.vue frontend/shortcut_hint_sync_regression_test.mjs frontend/settings_regression_test.mjs
```

---

### Task 7：同步手机端提示和双语文档

**Files:**
- Modify: `app/mobile.go`
- Modify: `app/mobile_test.go`
- Modify: `README.md`
- Modify: `README_EN.md`
- Verify only: `pkg/mobile/web/index.html`

**Interfaces:**
- Consumes: Config subscriber 推送的 `map[string]shortcut.KeyBinding`。
- Produces: Snapshot hints 中统一的 action label + 当前 `KeyName`。
- Invariant: Snapshot 不包含 ComboID、VK、配置路径、Token 或其他配置字段。

- [x] **Step 1：写手机端文案红灯**

扩展 `TestMobileShortcutHintsOnlyExposeLabelsAndDisplayNames`，并在测试文件中增加：

```go
func assertHint(t *testing.T, hints []mobile.ShortcutHint, label, keyName string) {
    t.Helper()
    for _, hint := range hints {
        if hint.Label == label && hint.KeyName == keyName { return }
    }
    t.Fatalf("missing hint %q=%q in %#v", label, keyName, hints)
}
```

测试主体使用：

```go
bindings := map[string]shortcut.KeyBinding{
    "toggle":        {ComboID: "75+162", KeyName: "Ctrl+K"},
    "toggle_window": {ComboID: "72+160+162", KeyName: "Ctrl+Shift+H"},
}
hints := mobileShortcutHints(bindings)
// 按定义顺序查找
assertHint(t, hints, "隐身模式", "Ctrl+K")
assertHint(t, hints, "隐藏/显示窗口", "Ctrl+Shift+H")
```

序列化 Snapshot 后断言不含 `ComboID`、`75+162` 或其他内部字段。

- [x] **Step 2：运行红灯**

```bash
go test ./app -run 'TestMobileShortcutHints' -count=1
```

Expected: FAIL，当前 label 仍为“隐藏/显示”和“隐藏窗口”。

- [x] **Step 3：修改手机端定义**

`mobileShortcutHintDefinitions`：

```go
{label: "隐身模式", actions: []string{"toggle"}},
{label: "隐藏/显示窗口", actions: []string{"toggle_window"}},
```

其他动作顺序和 label 不变。

- [x] **Step 4：验证 Snapshot 订阅链路**

保留并扩展现有测试，证明 `ConfigManager` subscriber 更新后 `Bridge.SetHints` 使用新 `KeyName`；不得修改 `pkg/mobile/web/index.html` 的动态渲染逻辑。

- [x] **Step 5：更新 README 双语快捷键表**

中文：

```text
隐身模式（防录屏） | F9 | Cmd + 2
隐藏 / 显示窗口     | Ctrl + Shift + H | Cmd + 4
```

英文：

```text
Ghost mode (screen-capture protection) | F9 | Cmd + 2
Hide / show window                     | Ctrl + Shift + H | Cmd + 4
```

在模型配置说明增加一段：成功保存连接后，模型名称按接口地址/协议保存在加密配置中；不保存多份 Key；模板模式也可复用历史。

- [x] **Step 6：运行 App/Mobile 与文档静态回归**

```bash
gofmt -w app/mobile.go app/mobile_test.go
go test ./app ./pkg/mobile -count=1
node frontend/shortcut_hint_sync_regression_test.mjs
```

Expected: PASS。

- [x] **Step 7：检查点（不提交）**

```bash
git diff --check -- app/mobile.go app/mobile_test.go README.md README_EN.md
git diff --cached
```

---

### Task 8：生成 Wails bindings 并完成聚焦集成验证

**Files:**
- Modify generated: `frontend/wailsjs/go/models.ts`
- Verify unchanged: `frontend/wailsjs/go/app/App.js`
- Verify unchanged: `frontend/wailsjs/go/app/App.d.ts`
- Verify unchanged: `frontend/wailsjs/runtime/**`
- Verify unchanged: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: Go `config.ModelHistoryEntry`、`Config.ModelHistory`、`ConnectionSettingsView.SavedModels`。
- Produces: TypeScript bindings 与 Go DTO 一致。

- [x] **Step 1：运行生成前聚焦测试**

```bash
go test ./pkg/config ./app -count=1
node frontend/model_history_regression_test.mjs
node frontend/shortcut_hint_sync_regression_test.mjs
```

Expected: PASS。

- [x] **Step 2：冻结 bindings/依赖保护哈希**

在仓库外临时目录记录 SHA-256：

```text
go.mod
go.sum
frontend/wailsjs/go/app/App.js
frontend/wailsjs/go/app/App.d.ts
frontend/wailsjs/go/models.ts
frontend/wailsjs/runtime/**
.git/index
```

不读取或记录任何配置文件。

- [x] **Step 3：固定 Wails v2.11.0 生成 bindings**

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 generate module -nocolour
```

Expected: `models.ts` 出现 `ModelHistoryEntry`、`modelHistory`、`savedModels`；无新方法，因此 App.js/App.d.ts 语义不变。

- [x] **Step 4：逐文件比较并清除纯空白噪音**

- `go.mod`、`go.sum`、runtime 必须逐文件哈希不变。
- App.js/App.d.ts 若仅空白变化，定向清除；不得批量恢复用户改动。
- `models.ts` 只保留新类型/字段语义差异。

- [x] **Step 5：运行核心集成测试**

```bash
go test ./pkg/config ./pkg/llm ./pkg/solution ./app -count=1
for f in frontend/*_test.mjs; do node "$f" || exit 1; done
npm --prefix frontend run build
```

Expected: Go 聚焦、全部前端回归和 Vite build PASS；只允许记录既有大 chunk warning。

- [x] **Step 6：检查点（不提交）**

```bash
git diff --check
git status --short
git diff --cached
```

Expected: 仅有 Change 037 预期修改；暂存区为空。

---

### Task 9：完整门禁、候选构建与固定 Profile 验收

**Files:**
- Modify governance: `_dev/changes/037-unified-api-connection/03-implementation.md`
- Modify governance: `_dev/changes/037-unified-api-connection/04-validation.md`
- Modify governance: `_dev/changes/037-unified-api-connection/05-rollback.md`
- Modify governance: `_dev/changes/037-unified-api-connection/06-notes.md`
- Modify governance: `_dev/changes/INDEX.md`
- Ignored output: `build/bin/testing/current/Q-Solver-candidate-change037.exe`
- External/temporary state: `%TEMP%\qsolver-change037-manual-profile\`

**Interfaces:**
- Consumes: Tasks 1～8 的完整实现和测试证据。
- Produces: 当前候选哈希、保护清单、隔离启动结果和人工验收清单。
- Invariant: 根稳定版与 releases 不变；固定 Profile 不含正式配置副本。

- [x] **Step 1：执行 Git 只读预检**

严格运行：

```bash
git status
git branch
git remote -v
git diff
git diff --cached
```

确认分支为 `dev/qsolver-custom`、暂存区为空、remote 目标无误；不暂存。

- [x] **Step 2：完整自动门禁**

```bash
go test ./... -count=1
go vet ./...
go build ./...
go mod verify
for f in frontend/*_test.mjs; do node "$f" || exit 1; done
npm --prefix frontend run build
git diff --check
```

逐项记录真实退出结果。不得把历史结果写作当前结果。

- [x] **Step 3：执行 race 并如实记录**

```bash
go test -race ./pkg/config ./pkg/llm ./pkg/solution ./app -count=1
```

若仍为 Windows 启动错误 `0xc0000139`，记录 `BLOCKED`；不得写成 PASS，也不得用禁用 race 替代。

- [x] **Step 4：冻结构建产物保护清单**

记录：

- `build/bin/Q-Solver.exe` 与测试前稳定副本路径、大小、SHA-256。
- 当前 Revision B 候选 SHA-256，并归档到 `build/bin/archive/transient/Q-Solver-candidate-change037-revision-b-before-model-history.exe`，禁止覆盖不同哈希文件。
- `go.mod`、`go.sum`、bindings、runtime、`frontend/dist/**`、`.git/index`、`build/bin/releases/**` 哈希。

更新 transient `SHA256SUMS` 并验证全部已有条目。

- [x] **Step 5：Wails v2.11.0 保护构建 Revision C 候选**

关闭正在运行的旧候选 PID，只允许 `CloseMainWindow` 正常退出；构建：

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

构建后逐项比对保护哈希；任何依赖/bindings/runtime/releases/根稳定版意外变化都停止后续验收。

- [x] **Step 6：证明候选运行时与更新清单**

```bash
go version -m build/bin/testing/current/Q-Solver-candidate-change037.exe
```

必须包含：

```text
github.com/wailsapp/wails/v2 v2.11.0
```

更新 `build/bin/testing/current/SHA256SUMS`，恰好包含稳定副本与一个当前候选并校验通过。

- [x] **Step 7：隔离启动烟测**

使用一次性空 Profile 启动至少 9 秒，记录：

- WebView2 direct child ≥ 1。
- Node/Vite direct child = 0。
- `CloseMainWindow=True`、ExitCode=0。
- ForcedCleanup=False、残留进程=0。

烟测 Profile 用完清理。

- [ ] **Step 8：固定人工 Profile 首次配置**

使用同一目录：

```text
APPDATA=%TEMP%\qsolver-change037-manual-profile\AppData\Roaming
LOCALAPPDATA=%TEMP%\qsolver-change037-manual-profile\AppData\Local
```

首次启动由用户执行：

1. 选择一个测试题库目录并保存（文档只记 `SET`）。
2. 在连接 A 保存模型 A1，再保存 A2。
3. 切换连接 B 保存模型 B1；切回 A，确认只显示 A2/A1。
4. 修改截图/发送/隐身任一快捷键，确认顶部、欢迎页和截图 Dock 当前运行态提示同步。
5. 点击“保存全部设置”，确认手机端提示同步。
6. 正常关闭。

不调用真实收费 API；可使用本地假服务或只保存配置。

- [ ] **Step 9：固定人工 Profile 重启验证**

用完全相同环境变量重启当前候选，确认：

- 题库目录 `SET` 且不需重选。
- 当前模型和连接 A 历史存在；连接 B 不串入 A。
- 快捷键提示名称与实际组合键一致。
- F9 文案为“隐身模式”；实体窗口动作显示“隐藏/显示窗口”。
- 手机端只显示已保存快捷键，不暴露 ComboID/路径/Token。

人工未完成时标 `NEEDS_USER`，不得关闭 Change。

- [x] **Step 10：更新治理与提交边界**

在 `03` 记录实际文件和偏差；在 `04` 记录每条命令/结果/候选哈希；在 `05` 补模型历史与快捷键定向回滚；在 `06` 记录已知限制；更新 INDEX 为“Revision C 已实现 / 人工验收状态 / 未提交未晋升”。

更新审计 Commit 1 文件白名单，新增：

```text
pkg/config/model_history.go
pkg/config/model_history_test.go
frontend/model_history_regression_test.mjs
frontend/shortcut_hint_sync_regression_test.mjs
app/mobile.go
app/mobile_test.go
frontend/src/App.vue
frontend/src/components/WelcomeView.vue
```

不得实际暂存或提交。

- [x] **Step 11：最终状态检查**

```bash
git status
git branch
git remote -v
git diff
git diff --cached
```

结论只能是：

- 仅有 Change 037 预期未提交修改。
- 暂存区为空。
- 根稳定版和 releases 未改变。
- 无 commit、push、Tag、Release 或晋升。

---

## 计划自审

### Spec 覆盖

- Revision C §17 模型历史数据模型：Task 1～3。
- §18 前端模型复用与分组：Task 4～5。
- §19 固定隔离 Profile 与题库目录：Task 9 Step 8～9；明确不改题库源码。
- §20 快捷键字段、运行态同步和语义：Task 6～7。
- §21 错误/事务边界：Task 1～4 的失败与原子性测试。
- §22 自动/人工验收：Task 8～9。
- §23 回滚：Task 9 Step 10。

### 类型一致性

- Go 持久字段统一使用 `ModelHistory []ModelHistoryEntry`。
- Go 局部保存 DTO 统一使用 `SavedModels []string` / JSON `savedModels`。
- 前端完整历史统一使用 `modelHistory`；当前连接 computed 统一使用 `savedModelsForCurrentConnection`。
- ModelSelect props 统一使用 `savedModels`、`catalogModels`。
- 快捷键组合字段统一使用 JSON `comboID`；禁止新增 `vkCode`。

### Placeholder 与范围扫描

- 占位符扫描无命中，所有接口和代码步骤均已具体定义。
- 不创建完整连接 Profile，不修改题库产品逻辑，不新增依赖。
- 每项业务修改都有先失败后通过的聚焦测试。
- 所有提交步骤均被用户“未授权 commit/push”约束替换为不暂存检查点。
