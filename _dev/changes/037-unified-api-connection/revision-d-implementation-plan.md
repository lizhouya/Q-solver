# Change 037 Revision D 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 在现有加密主配置中实现最多 20 条可命名、切换、修改和删除的完整 Connection Profiles，使请求地址和 API Key 输入一次后可重复选择，同时定向撤销 Revision C 模型历史并保留快捷键正确修复。

**架构：** `config.Config` 内嵌 Profiles 并以 `ActiveConnectionProfileID` 标识活动记录，根级连接字段继续镜像活动 Profile，既有 Solver、LLM Service 和题库调用链无需 Profile-aware。Wails 仅批量返回安全摘要，完整连接草稿按 ID 单条读取；Pinia 区分“已持久化活动连接”和“当前选中草稿”，只有连接专用保存或保存全部设置成功后才激活选中 Profile。

**技术栈：** Go 1.24.1+ 模块（当前环境 go1.27.1）、Wails v2.11.0、Vue 3、Pinia、Vite 4.5.14、Node.js 22+；不新增依赖。

**Spec:** `_dev/changes/037-unified-api-connection/design.md`（当前权威 Revision D，第 24～34 节）

## 执行状态（2026-11-10）

- [x] Task 1：Connection Profile 领域模型。
- [x] Task 2：ConfigManager 原子事务与后端模型历史撤销。
- [x] Task 3：Wails 安全 Profile API 与 GetSettings 最小化。
- [x] Task 4：Pinia Profile 状态机与前端模型历史撤销。
- [x] Task 5：Profile 管理 UI 与单一模型目录。
- [x] Task 6：Wails v2.11.0 bindings 与双语说明。
- [x] Task 7：普通自动门禁与安全审计；race 单列为环境 `BLOCKED`。
- [x] Task 8：最终保护候选与空隔离 Profile 烟测。
- [x] Task 9：用户确认固定隔离 Profile 双启动 GUI 验收通过；治理记录完成，稳定版已按哈希晋升，双提交与推送已获授权。

## 全局约束

- 遵守 `AGENTS.md > CLAUDE.md > Change 037`；仅在 `dev/qsolver-custom` 工作，禁止修改 `main`。
- 每个实现任务必须先写失败测试、确认失败原因正确、再写最小实现并运行聚焦测试。
- 不新增依赖，不升级 Go、Wails、Vue、Node、SDK 或 runtime；bindings 和候选均固定 Wails v2.11.0。
- 不读取、复制、打印或修改正式用户配置；Key、Token、Header、模板和地址实值不得进入日志、治理文档或提交信息，状态只记录 `SET / NOT SET`。
- Profiles 只能保存在现有加密主配置中，不新增明文连接文件，不使用浏览器存储保存连接。
- `GetSettings` 不返回 Profile 列表或根级连接编辑字段；完整草稿只能经 `GetConnectionProfile(id)` 单条定向读取。
- 选择、新建和删除记录不得发起模型目录、probe 或推理请求；选择不得切换活动 Provider。
- 根级连接字段始终镜像活动 Profile；应用正式 Profile 时 `ConnectionMode` 固定为 `direct`。
- 题库产品代码和文件格式不修改；固定隔离 Profile 只用于两次启动验收。
- 保留 720px 无滑动标签栏、“显示”文案、`comboID` 修复、动态快捷键提示、“隐身模式 / 隐藏/显示窗口”语义和手机端同步。
- 禁止使用 `git reset`、`git restore` 或批量恢复撤销 Revision C；必须逐字段、逐文件定向修改。
- 用户尚未授权 commit、push、稳定版晋升、Tag 或 Release；所有任务停在未暂存检查点。
- 每组 Git 检查前先执行 `git status`、`git branch`、`git remote -v`、`git diff`；禁止 `git add -A`、`git add .` 和 `git add -f`。
- `build/bin/**` 保持 ignored；Revision C 候选只能归档为 transient，根稳定版与 `build/bin/releases/**` 不得改变。

---

## 文件结构锁定

### 新建

- `pkg/config/connection_profiles.go`：Profile 常量、错误类型、规范化、自动命名、ID、查找、upsert、删除和活动镜像。
- `pkg/config/connection_profiles_test.go`：Profile 领域规则、容量、命名、深拷贝、活动镜像和旧单连接兼容测试。
- `app/connection_profiles.go`：Wails 安全 DTO、列表、单条读取、保存/激活和删除接口。
- `app/connection_profiles_test.go`：摘要脱敏、定向读取、无副作用选择、原子保存和删除测试。
- `frontend/src/components/ConnectionProfileManager.vue`：连接下拉、记录名称、新建和内联删除确认。
- `frontend/connection_profiles_regression_test.mjs`：Pinia Profile 初始化、选择、新建、保存、删除、竞态和全量保存回归。
- `_dev/changes/037-unified-api-connection/revision-d-implementation-plan.md`：本计划。

### 修改

- `pkg/config/config.go`：增加 `ConnectionProfile`、活动 ID 和 Profiles；移除 Revision C `ModelHistoryEntry` / `ModelHistory`。
- `pkg/config/connection.go`：把单连接规范化提取为不递归 helper，并接入 Profile 规范化；移除模型历史清洗。
- `pkg/config/manager.go`：深拷贝 Profiles；全量保存保留后端记录并按元数据 upsert/激活选中 Profile；移除模型历史写入。
- `pkg/config/manager_test.go`：替换 Revision C 模型历史事务测试为 Profile 全量保存、旧配置不写盘和失败原子性测试。
- `app/llm.go`：保留草稿目录/probe 接口；移除 `SavedModels` 和旧单连接保存实现，把 Profile 保存移至新文件。
- `app/llm_test.go`：保持 ConnectionDraft、目录和 probe 测试，增加生成 DTO 与草稿映射一致性断言。
- `app/settings.go`：`GetSettings` 返回脱敏副本；`UpdateSettings` 继续委托 ConfigManager。
- `app/settings_test.go`：改写 GetSettings、局部保存和安全结果测试，删除模型历史断言。
- `frontend/src/services/api.js`：增加 Profile 列表、读取、保存和删除 bridge wrapper。
- `frontend/src/stores/settings.js`：增加 Profile 状态机，区分活动连接与选中草稿；移除模型历史状态/helpers。
- `frontend/src/components/SettingsModal.vue`：接入 Profile 管理组件；按钮按草稿加载状态禁用；模型恢复单目录候选。
- `frontend/src/components/ModelSelect.vue`：移除“已保存模型”分组，恢复单一 `models` 目录候选 prop。
- `frontend/unified_api_settings_regression_test.mjs`：fixture 改为安全全局快照 + Profile 接口，继续覆盖两协议和旧协议草稿迁移。
- `frontend/ccr_ui_regression_test.mjs`、`frontend/ccr_settings_lifecycle_regression_test.mjs`、`frontend/model_connectivity_regression_test.mjs`、`frontend/settings_regression_test.mjs`、`frontend/model_layout_regression_test.mjs`：更新静态名称和 Profile 初始化夹具，保留原行为覆盖。
- `frontend/wailsjs/go/app/App.js`、`frontend/wailsjs/go/app/App.d.ts`、`frontend/wailsjs/go/models.ts`：由 Wails v2.11.0 生成 Profile 方法/类型，并移除模型历史类型。
- `README.md`、`README_EN.md`：用多连接记录说明替换模型历史说明，保留快捷键语义修正。
- `_dev/audits/037-unified-api-connection.md`、Change 037 的 `02`～`06`、`_dev/changes/INDEX.md`：记录 Revision D 实现、验证、回滚和提交边界。

### 删除

- `pkg/config/model_history.go`
- `pkg/config/model_history_test.go`
- `frontend/model_history_regression_test.mjs`

### 明确不修改

- `app/questionbank.go`、`frontend/src/stores/questionbank.js`：不改变题库产品持久化。
- `pkg/shortcut/**`：Revision C 已确认现有录制服务运行态行为正确。
- `frontend/src/App.vue`、`frontend/src/components/WelcomeView.vue`、`app/mobile.go`：除非保留测试暴露回归，否则不再改动已正确的快捷键实现。
- `pkg/mobile/web/**`：手机端通过 Snapshot 动态显示，无需硬编码 Profile。
- `go.mod`、`go.sum`、`frontend/package.json`、`frontend/package-lock.json`、`frontend/wailsjs/runtime/**`、`build/bin/releases/**`、正式用户配置。

---

### Task 1：建立 Connection Profile 领域模型

**Files:**
- Create: `pkg/config/connection_profiles.go`
- Create: `pkg/config/connection_profiles_test.go`
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/connection.go`

**Interfaces:**
- Produces: `ConnectionProfile`、`NormalizeConnectionProfiles(Config) (Config, error)`、`FindConnectionProfile(Config, string) (ConnectionProfile, bool)`、`UpsertConnectionProfile(*Config, ConnectionProfile) (ConnectionProfile, error)`、`DeleteConnectionProfile(*Config, string) error`、`ApplyProfileToActiveConnection(*Config, ConnectionProfile)`。
- Produces: `LegacyConnectionProfileID`、`MaxConnectionProfiles`、`ProfileError` 及固定 `profile_*` 错误码。
- Consumes: 当前连接规范化、`protocol.Normalize`、`crypto/rand`、`net/url` 和标准库；不新增依赖。
- Invariant: 正式 ID 为 32 位小写十六进制；名称最多 64 UTF-8 字节；最多 20 条；每个 map/slice 返回独立副本。

- [ ] **Step 1：写 Profile 类型和旧单连接红灯**

在 `pkg/config/connection_profiles_test.go` 增加：

```go
func TestNormalizeConnectionProfilesLeavesLegacyRootConnectionUnpersisted(t *testing.T) {
    cfg := NewDefaultConfig()
    cfg.APIKey = "test-only-key"
    cfg.BaseURL = "https://legacy.example.test/v1"
    cfg.Model = "legacy-model"

    got, err := NormalizeConnectionProfiles(cfg)
    if err != nil { t.Fatal(err) }
    if got.ActiveConnectionProfileID != "" || len(got.ConnectionProfiles) != 0 {
        t.Fatalf("legacy load must remain in-memory only: %#v", got.ConnectionProfiles)
    }
    if got.APIKey != cfg.APIKey || got.BaseURL != cfg.BaseURL || got.Model != cfg.Model {
        t.Fatal("legacy root connection changed")
    }
}

func TestUpsertConnectionProfileCreatesRandomStableIDAndAutomaticName(t *testing.T) {
    cfg := NewDefaultConfig()
    draft := ConnectionProfile{
        BaseURL: "https://api.example.test/v1", APIProtocol: "openai_chat",
        EndpointPath: "/chat/completions", ResponseFormat: "text", Model: "model-a",
    }
    saved, err := UpsertConnectionProfile(&cfg, draft)
    if err != nil { t.Fatal(err) }
    if matched, _ := regexp.MatchString(`^[0-9a-f]{32}$`, saved.ID); !matched {
        t.Fatalf("id=%q", saved.ID)
    }
    if saved.Name != "api.example.test · OpenAI Chat" {
        t.Fatalf("name=%q", saved.Name)
    }
    again, err := UpsertConnectionProfile(&cfg, saved)
    if err != nil || again.ID != saved.ID || len(cfg.ConnectionProfiles) != 1 {
        t.Fatalf("update changed identity: %#v err=%v", again, err)
    }
}
```

- [ ] **Step 2：运行红灯**

Run:

```bash
go test ./pkg/config -run 'Test(NormalizeConnectionProfiles|UpsertConnectionProfile)' -count=1
```

Expected: FAIL，`ConnectionProfile` 和 helper 尚未定义。

- [ ] **Step 3：增加持久类型和固定错误类型**

在 `pkg/config/config.go` 增加：

```go
type ConnectionProfile struct {
    ID                  string            `json:"id"`
    Name                string            `json:"name"`
    APIKey              string            `json:"apiKey,omitempty"`
    BaseURL             string            `json:"baseURL"`
    APIProtocol         string            `json:"apiProtocol"`
    EndpointPath        string            `json:"endpointPath,omitempty"`
    ExtraHeaders        map[string]string `json:"extraHeaders,omitempty"`
    ResponseFormat      string            `json:"responseFormat,omitempty"`
    RequestTemplate     string            `json:"requestTemplate,omitempty"`
    ResponseJSONPointer string            `json:"responseJSONPointer,omitempty"`
    AuthHeader          string            `json:"authHeader,omitempty"`
    AuthPrefix          string            `json:"authPrefix,omitempty"`
    Model               string            `json:"model,omitempty"`
}
```

在 `Config` 增加：

```go
ActiveConnectionProfileID string              `json:"activeConnectionProfileId,omitempty"`
ConnectionProfiles        []ConnectionProfile `json:"connectionProfiles,omitempty"`
```

`NewDefaultConfig()` 显式初始化 `ConnectionProfiles: []ConnectionProfile{}`。

在新文件定义：

```go
const (
    LegacyConnectionProfileID = "legacy-current"
    MaxConnectionProfiles     = 20
    MaxConnectionProfileNameBytes = 64
)

type ProfileError struct {
    Code string
}

func (e *ProfileError) Error() string { return e.Code }
```

固定错误码必须与设计一致：`profile_load_failed`、`profile_not_found`、`profile_invalid`、`profile_limit_reached`、`profile_name_duplicate`、`profile_active_delete_forbidden`、`profile_legacy_delete_forbidden`、`profile_save_failed`、`profile_delete_failed`。

- [ ] **Step 4：提取不递归的单连接规范化 helper**

将 `pkg/config/connection.go` 现有 `NormalizedConnectionSettings` 主体提取为：

```go
func normalizeConnectionFields(c Config) (Config, error) {
    // 保留现有 CCR → direct、协议、URL、Endpoint、Headers、response format、
    // custom_json 模板/Pointer/Auth 的全部规范化；不读取或修改 Profiles。
}

func (c Config) NormalizedConnectionSettings() (Config, error) {
    normalized, err := normalizeConnectionFields(c)
    if err != nil { return Config{}, err }
    return NormalizeConnectionProfiles(normalized)
}
```

从两个协议返回分支删除 `NormalizeModelHistory` 调用。`NormalizeConnectionProfiles` 规范化每条 Profile 时只能调用 `normalizeConnectionFields`，禁止递归调用 `NormalizedConnectionSettings`。

- [ ] **Step 5：实现 Profile 转换、深拷贝和活动镜像**

在 `connection_profiles.go` 定义以下私有 helper，并在所有公开 helper 中复用：

```go
func cloneStringMap(source map[string]string) map[string]string
func cloneConnectionProfile(source ConnectionProfile) ConnectionProfile
func profileFromConnectionConfig(cfg Config, id, name string) ConnectionProfile
func connectionConfigFromProfile(profile ConnectionProfile) Config
func normalizeConnectionProfile(profile ConnectionProfile) (ConnectionProfile, error)
```

活动镜像必须覆盖全部连接字段并脱离 map：

```go
func ApplyProfileToActiveConnection(cfg *Config, profile ConnectionProfile) {
    cfg.ActiveConnectionProfileID = profile.ID
    cfg.ConnectionMode = ConnectionModeDirect
    cfg.APIKey = profile.APIKey
    cfg.BaseURL = profile.BaseURL
    cfg.APIProtocol = profile.APIProtocol
    cfg.EndpointPath = profile.EndpointPath
    cfg.ExtraHeaders = cloneStringMap(profile.ExtraHeaders)
    cfg.ResponseFormat = profile.ResponseFormat
    cfg.RequestTemplate = profile.RequestTemplate
    cfg.ResponseJSONPointer = profile.ResponseJSONPointer
    cfg.AuthHeader = profile.AuthHeader
    cfg.AuthPrefix = profile.AuthPrefix
    cfg.Model = profile.Model
}
```

- [ ] **Step 6：实现随机 ID、自动命名和 upsert**

随机 ID 只使用标准库：

```go
func newConnectionProfileID() (string, error) {
    raw := make([]byte, 16)
    if _, err := rand.Read(raw); err != nil { return "", err }
    return hex.EncodeToString(raw), nil
}
```

自动名称规则：解析规范化 URL 的 `Hostname()`；协议后缀只允许 `OpenAI Chat` 或 `自定义模板`；无法解析时使用“未命名连接”；冲突按 ` 2`、` 3` 递增。实现 `truncateUTF8Bytes`，只截 hostname 部分并为 ` · <协议><编号>` 预留字节，确保最终名称 `len([]byte(name)) <= 64`。

`UpsertConnectionProfile` 必须：

1. `ID==""` 或 `legacy-current` 时视为新建，检查 20 条上限并生成新 ID。
2. 新建名称为空时自动命名；更新名称为空时返回 `profile_invalid`。
3. 手动名称 Trim 后按 `strings.EqualFold` 检查重复，重复返回 `profile_name_duplicate`，不自动改名。
4. 规范化连接字段并深拷贝 Headers。
5. 更新保持原 slice 位置；新建追加尾部；同 URL 不按地址覆盖。
6. 只 upsert，不自动激活；调用者显式执行 `ApplyProfileToActiveConnection`。

- [ ] **Step 7：实现完整规范化、查找和删除**

`NormalizeConnectionProfiles` 在空列表时保留旧根连接且要求活动 ID 为空；非空时验证：

- 不超过 20 条。
- ID 为 32 位小写十六进制且不等于 `legacy-current`。
- 名称非空、不超 64 字节、大小写不敏感唯一。
- 每条连接字段可规范化。
- `ActiveConnectionProfileID` 指向现有记录；否则返回 `profile_invalid`。
- 最后把活动 Profile 镜像到根级连接字段。

`FindConnectionProfile` 返回 detached copy。`DeleteConnectionProfile` 对活动 ID、`legacy-current`、不存在 ID 分别返回固定错误；成功删除非活动项并保持其余顺序。

- [ ] **Step 8：补齐领域边界测试**

增加并实现以下测试：

```go
func TestAutomaticProfileNamesResolveCollisionsWithin64Bytes(t *testing.T)
func TestManualProfileNameRejectsCaseInsensitiveDuplicateAndOverlongValue(t *testing.T)
func TestConnectionProfileLimitIsTwenty(t *testing.T)
func TestSameURLProfilesCoexistByID(t *testing.T)
func TestNormalizeConnectionProfilesRejectsMissingActiveID(t *testing.T)
func TestApplyProfileMirrorsAllConnectionFieldsAndForcesDirectMode(t *testing.T)
func TestFindAndNormalizeConnectionProfilesReturnDetachedHeaderMaps(t *testing.T)
func TestDeleteConnectionProfileRejectsActiveLegacyAndMissingIDs(t *testing.T)
func TestDeleteConnectionProfileKeepsOrder(t *testing.T)
func TestCustomJSONProfileNormalizesFullRequestURL(t *testing.T)
```

关键别名断言：修改 `FindConnectionProfile` 返回值的 `ExtraHeaders` 后，原 Config 中对应 Header 必须不变。

- [ ] **Step 9：运行 Task 1 聚焦测试**

Run:

```bash
gofmt -w pkg/config/config.go pkg/config/connection.go pkg/config/connection_profiles.go pkg/config/connection_profiles_test.go
go test ./pkg/config -run 'Test(.*ConnectionProfile|AutomaticProfile|ManualProfile|SameURLProfile|CustomJSONProfile)' -count=1
```

Expected: PASS。

- [ ] **Step 10：未暂存检查点**

先执行完整 Git 预检，再执行：

```bash
git diff --check -- pkg/config/config.go pkg/config/connection.go pkg/config/connection_profiles.go pkg/config/connection_profiles_test.go
git diff --cached
```

Expected: 仅有 Task 1 预期差异，暂存区为空；不提交。

---

### Task 2：把 Profiles 纳入 ConfigManager 原子事务并撤销后端模型历史

**Files:**
- Modify: `pkg/config/manager.go`
- Modify: `pkg/config/manager_test.go`
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/connection.go`
- Delete: `pkg/config/model_history.go`
- Delete: `pkg/config/model_history_test.go`

**Interfaces:**
- Consumes: Task 1 Profile helper。
- Produces: `UpdateFromJSON` 保留全部后端 Profiles，按 `connectionProfileId` / `connectionProfileName` 创建或更新选中项并原子激活。
- Produces: `ConfigManager.Get()`、事务候选、subscriber 参数均对 Profile slice 和 Header map 深拷贝。
- Invariant: 失败不得改变磁盘、内存 Config、`oldConfig`、Provider 或 subscriber；旧单连接 Load 不自动写盘。

- [ ] **Step 1：写全量保存和旧配置红灯**

在 `manager_test.go` 增加：

```go
func TestUpdateFromJSONCreatesProfileFromLegacyConnection(t *testing.T) {
    manager := isolatedManager(t)
    if err := manager.Load(); err != nil { t.Fatal(err) }
    payload := `{
      "connectionProfileId":"legacy-current","connectionProfileName":"",
      "baseURL":"https://api.example.test/v1","apiProtocol":"openai_chat",
      "endpointPath":"/chat/completions","responseFormat":"text","model":"model-a"
    }`
    if err := manager.UpdateFromJSON(payload); err != nil { t.Fatal(err) }
    got := manager.Get()
    if len(got.ConnectionProfiles) != 1 || got.ActiveConnectionProfileID == "" ||
        got.ActiveConnectionProfileID == LegacyConnectionProfileID {
        t.Fatalf("profile not created: %#v", got.ConnectionProfiles)
    }
    if got.ConnectionProfiles[0].ID != got.ActiveConnectionProfileID ||
        got.BaseURL != got.ConnectionProfiles[0].BaseURL {
        t.Fatal("root mirror and active profile diverged")
    }
}

func TestUpdateFromJSONPreservesUnselectedProfilesAndUpdatesSelected(t *testing.T)
func TestUpdateFromJSONMissingProfileArrayCannotEraseStoredProfiles(t *testing.T)
func TestUpdateFromJSONFailureKeepsProfilesRootAndSubscribersUnchanged(t *testing.T)
func TestLoadLegacyConnectionDoesNotWriteProfileOrRewriteEncryptedFile(t *testing.T)
func TestConnectionProfilesPersistEncryptedWithoutPlaintextSentinels(t *testing.T)
```

最后一个测试保存包含测试 Key/Header/模板 sentinel 的 Profile，读取磁盘原始字节并断言不含任一 sentinel；随后使用现有 `decrypt` 测试 helper 解密并确认 Profile 字段可往返。测试输出不得打印 sentinel 实值。

- [ ] **Step 2：运行红灯**

Run:

```bash
go test ./pkg/config -run 'Test(UpdateFromJSON.*Profile|LoadLegacyConnectionDoesNotWriteProfile)' -count=1
```

Expected: FAIL；当前 `UpdateFromJSON` 会从前端 payload 重建 Config，且没有 Profile 元数据事务。

- [ ] **Step 3：深拷贝 Get、快照和 Profiles**

修改：

```go
func (cm *ConfigManager) Get() Config {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    return cloneConfigSnapshot(cm.config)
}
```

`cloneConfigSnapshot` 对每条 Profile 及其 `ExtraHeaders` 建立新 slice/map；保留现有根级 `ExtraHeaders` 和 `Shortcuts` 深拷贝。增加测试：

```go
func TestGetAndCloneConfigSnapshotDeepCopyConnectionProfiles(t *testing.T)
```

测试修改 `Get()` 返回值中的名称和 Header，manager 内部 Config 必须不变。

- [ ] **Step 4：解析非持久化 Profile 元数据**

在 `manager.go` 定义私有 payload 元数据：

```go
type connectionProfileUpdateMeta struct {
    ID   *string `json:"connectionProfileId"`
    Name *string `json:"connectionProfileName"`
}
```

选择规则：

```go
selectedID := oldConfig.ActiveConnectionProfileID
if meta.ID != nil { selectedID = strings.TrimSpace(*meta.ID) }
if meta.ID == nil && selectedID == "" { selectedID = LegacyConnectionProfileID }
```

名称字段缺失且更新正式 Profile 时保留原名称；字段存在且为空只允许新建/legacy 自动命名。前端显式发送空 ID 表示新建，不能回退到活动 ID。

- [ ] **Step 5：重写 UpdateFromJSON 候选顺序**

实现顺序必须是：

```go
cm.mu.RLock()
oldConfig := cloneConfigSnapshot(cm.config)
cm.mu.RUnlock()

newConfig := NewDefaultConfig()
json.Unmarshal(payload, &newConfig)

// 先只规范化 payload 中根级连接草稿，避免旧活动 Profile 提前覆盖新草稿。
newConfig, err = normalizeConnectionFields(newConfig)

// Profiles 永远取自后端旧快照，绝不信任或依赖前端 connectionProfiles 字段。
newConfig.ConnectionProfiles = cloneConfigSnapshot(oldConfig).ConnectionProfiles
newConfig.ActiveConnectionProfileID = oldConfig.ActiveConnectionProfileID

draft := profileFromConnectionConfig(newConfig, selectedID, selectedName)
saved, err := UpsertConnectionProfile(&newConfig, draft)
ApplyProfileToActiveConnection(&newConfig, saved)
newConfig, err = newConfig.NormalizedConnectionSettings()
err = newConfig.Validate()
```

然后保持现有顺序：先 `saveSnapshot`，成功后才交换 `oldConfig/config` 并通知 subscribers。任何失败均在交换前返回。

- [ ] **Step 6：验证 Provider 构造边界**

ConfigManager 不导入 `pkg/llm`，继续只做 Config 校验。连接专用保存由 App 的 `PatchValidated` 调用 `llm.NewProvider`；全量保存后的现有 LLM Service subscriber 根据已验证 Config 重建 Provider，不执行网络请求。

增加测试确保 Profile-only 字段不会跳过根级连接验证：非法 Endpoint、模板或活动 ID均使全量保存失败。

- [ ] **Step 7：定向删除 Revision C 后端模型历史**

按以下顺序定向移除：

1. 删除 `Config.ModelHistory`、`ModelHistoryEntry` 和默认值。
2. 删除 `NormalizedConnectionSettings` 中剩余历史调用。
3. 删除 `UpdateFromJSON` 的 `RememberCurrentModel()`。
4. 删除 `cloneConfigSnapshot` 的模型历史分支。
5. 删除 `pkg/config/model_history.go` 和 `pkg/config/model_history_test.go`。
6. 将 `manager_test.go` 中四个模型历史事务测试替换为本 Task 的 Profile 测试；不得删除无关 ConfigManager 覆盖。

- [ ] **Step 8：运行 Task 2 聚焦和 Config 全包**

Run:

```bash
gofmt -w pkg/config/config.go pkg/config/connection.go pkg/config/connection_profiles.go pkg/config/connection_profiles_test.go pkg/config/manager.go pkg/config/manager_test.go
go test ./pkg/config -run 'Test(UpdateFromJSON.*Profile|LoadLegacyConnection|GetAndCloneConfigSnapshot|FailedLoad|PatchValidated)' -count=1
go test ./pkg/config -count=1
```

Expected: PASS；`pkg/config/model_history*.go` 已不存在。

- [ ] **Step 9：未暂存检查点**

完整 Git 预检后运行：

```bash
git diff --check -- pkg/config
git diff --cached
```

Expected: Profile 实现与模型历史定向删除均清晰可见，暂存区为空。

---

### Task 3：实现 Wails 安全 Profile API 与 GetSettings 数据最小化

**Files:**
- Create: `app/connection_profiles.go`
- Create: `app/connection_profiles_test.go`
- Modify: `app/llm.go`
- Modify: `app/llm_test.go`
- Modify: `app/settings.go`
- Modify: `app/settings_test.go`

**Interfaces:**
- Produces: `ListConnectionProfiles() ConnectionProfileListResult`。
- Produces: `GetConnectionProfile(id string) GetConnectionProfileResult`。
- Produces: `SaveConnectionSettings(draft ConnectionProfileDraft) SaveConnectionResult`。
- Produces: `DeleteConnectionProfile(id string) DeleteConnectionProfileResult`。
- Invariant: 列表/保存结果无凭据；完整 Key 只出现在单条 `GetConnectionProfile` 结果；读取和删除不发外部网络请求。

- [ ] **Step 1：写安全 DTO 和读取行为红灯**

在 `app/connection_profiles_test.go` 增加：

```go
func seedTwoProfiles(t *testing.T, manager *config.ConfigManager) []string {
    t.Helper()
    var ids []string
    var seedErr error
    err := manager.Patch(func(cfg *config.Config) {
        first, err := config.UpsertConnectionProfile(cfg, config.ConnectionProfile{
            Name: "A 连接", APIKey: "test-private-key", BaseURL: "https://a.example.test/v1",
            APIProtocol: "openai_chat", EndpointPath: "/chat/completions",
            ExtraHeaders: map[string]string{"X-Test": "test-private-header"},
            ResponseFormat: "text", Model: "model-a",
        })
        if err != nil { seedErr = err; return }
        second, err := config.UpsertConnectionProfile(cfg, config.ConnectionProfile{
            Name: "B 连接", BaseURL: "https://b.example.test/generate", APIProtocol: "custom_json",
            RequestTemplate: `{"messages":"{{openai_messages}}","marker":"test-private-template"}`,
            ResponseJSONPointer: "/answer", AuthHeader: "Authorization", AuthPrefix: "Bearer ", Model: "model-b",
        })
        if err != nil { seedErr = err; return }
        config.ApplyProfileToActiveConnection(cfg, first)
        ids = []string{first.ID, second.ID}
    })
    if err != nil { t.Fatal(err) }
    if seedErr != nil { t.Fatal(seedErr) }
    return ids
}

func TestListConnectionProfilesReturnsSafeSummariesOnly(t *testing.T) {
    a := newConnectionSettingsTestApp(t)
    seedTwoProfiles(t, a.configManager)
    result := a.ListConnectionProfiles()
    if result.Error != nil || len(result.Profiles) != 2 { t.Fatalf("result=%+v", result) }
    encoded, _ := json.Marshal(result)
    for _, sentinel := range []string{"test-private-key", "test-private-header", "test-private-template", "https://"} {
        if strings.Contains(string(encoded), sentinel) { t.Fatalf("summary leaked %q", sentinel) }
    }
}

func TestGetConnectionProfileReturnsOnlyRequestedDraftWithoutChangingRuntime(t *testing.T) {
    a := newConnectionSettingsTestApp(t)
    ids := seedTwoProfiles(t, a.configManager)
    beforeConfig := a.configManager.Get()
    beforeProvider := a.llmService.GetProvider()
    result := a.GetConnectionProfile(ids[1])
    if !result.Success || result.Profile == nil || result.Profile.ID != ids[1] { t.Fatalf("result=%+v", result) }
    if !reflect.DeepEqual(a.configManager.Get(), beforeConfig) || a.llmService.GetProvider() != beforeProvider {
        t.Fatal("targeted read changed active runtime")
    }
}

func TestGetSettingsRedactsProfilesAndAllRootConnectionFields(t *testing.T)
func TestLegacyConnectionAppearsAsTemporaryActiveSummaryAndTargetedDraft(t *testing.T)
```

- [ ] **Step 2：运行红灯**

Run:

```bash
go test ./app -run 'Test(ListConnectionProfiles|GetConnectionProfile|GetSettingsRedacts|LegacyConnectionAppears)' -count=1
```

Expected: FAIL；新 DTO/方法不存在，现有 `GetSettings` 仍返回根级 Key。

- [ ] **Step 3：定义 DTO 和固定安全错误映射**

在新文件完整定义：

```go
type ConnectionProfileSummary struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    AddressLabel string `json:"addressLabel"`
    APIProtocol  string `json:"apiProtocol"`
    Model        string `json:"model"`
    Active       bool   `json:"active"`
    APIKeySet    bool   `json:"apiKeySet"`
    TemplateSet  bool   `json:"templateSet"`
}

type ConnectionProfileDraft struct {
    ID         string          `json:"id"`
    Name       string          `json:"name"`
    Connection ConnectionDraft `json:"connection"`
}

type ConnectionProfileErrorDTO struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type ConnectionProfileListResult struct {
    Profiles []ConnectionProfileSummary `json:"profiles"`
    Error    *ConnectionProfileErrorDTO  `json:"error,omitempty"`
}

type GetConnectionProfileResult struct {
    Success bool                       `json:"success"`
    Profile *ConnectionProfileDraft    `json:"profile,omitempty"`
    Error   *ConnectionProfileErrorDTO `json:"error,omitempty"`
}

type DeleteConnectionProfileResult struct {
    Success   bool                       `json:"success"`
    DeletedID string                     `json:"deletedId,omitempty"`
    Error     *ConnectionProfileErrorDTO `json:"error,omitempty"`
}

type SaveConnectionResult struct {
    Success    bool                       `json:"success"`
    Profile    *ConnectionProfileSummary `json:"profile,omitempty"`
    Connection *ConnectionSettingsView   `json:"connection,omitempty"`
    Error      *ProviderErrorDTO          `json:"error,omitempty"`
}
```

`ConnectionSettingsView` 删除 `SavedModels`，只保留安全连接状态。新增：

```go
func profileErrorDTO(code string) *ConnectionProfileErrorDTO
func profileSaveError(code string) SaveConnectionResult
func profileErrorCode(err error, fallback string) string
```

固定文案按错误码映射，禁止拼接 `err.Error()`、Profile 内容或路径。保存接口沿用 `ProviderErrorDTO` 容器时只填稳定 `Code`、`Stage=validate_config` 和固定 `Message`。

- [ ] **Step 4：实现安全地址摘要和列表**

实现：

```go
func connectionAddressLabel(rawURL string) string
func connectionProfileSummary(profile config.ConnectionProfile, activeID string) ConnectionProfileSummary
```

地址摘要仅包含 `Hostname()`；显式非默认端口保留 `host:port`，IPv6 使用方括号；不得包含 scheme、userinfo、path、query 或 fragment。

`ListConnectionProfiles`：

- 先 `configManager.Load()`。
- 无正式 Profiles 时从根级连接生成 `legacy-current` 安全摘要，Name 为“当前连接”、Active 为 true。
- 有正式 Profiles 时保持保存顺序，标记唯一活动项。
- `APIKeySet` / `TemplateSet` 只返回布尔值。

- [ ] **Step 5：实现单条定向读取**

`GetConnectionProfile`：

- 正式 ID 使用 `FindConnectionProfile` detached copy。
- `legacy-current` 仅在正式列表为空时允许，返回根级连接完整草稿但 `Name=""`。
- 不调用 `Patch`、`Save`、LLM 目录或 probe。
- 不存在/非法 ID 返回 `profile_not_found` 固定结果。

增加 `ConnectionDraft` 与 Profile 的双向映射测试，确保 ExtraHeaders detached。

- [ ] **Step 6：迁移连接专用保存为 Profile upsert/激活**

把 `app/llm.go` 中旧 `SaveConnectionSettings(ConnectionDraft)` 移到新文件并改为 `ConnectionProfileDraft`：

```go
connection, err := draft.Connection.config().NormalizedConnectionSettings()
profile := config.ConnectionProfile{
    ID: draft.ID, Name: draft.Name,
    APIKey: connection.APIKey, BaseURL: connection.BaseURL,
    APIProtocol: connection.APIProtocol, EndpointPath: connection.EndpointPath,
    ExtraHeaders: connection.ExtraHeaders, ResponseFormat: connection.ResponseFormat,
    RequestTemplate: connection.RequestTemplate,
    ResponseJSONPointer: connection.ResponseJSONPointer,
    AuthHeader: connection.AuthHeader, AuthPrefix: connection.AuthPrefix,
    Model: connection.Model,
}
```

在一个 `PatchValidated` 中执行 `UpsertConnectionProfile`、`ApplyProfileToActiveConnection`，validateFn 调用 `llm.NewProvider(candidate)`。成功后从 manager 重新按活动 ID读取正式 Profile，返回安全 summary 和 `ConnectionSettingsView`。失败不得增加 Profile、改变根镜像或切换 Provider。

- [ ] **Step 7：实现非活动删除**

`DeleteConnectionProfile` 在一个 `PatchValidated` 中调用 config helper；不设置新活动项、不调用模型服务、不联网。活动、legacy、缺失记录返回对应固定错误码。成功结果只返回 `DeletedID`。

- [ ] **Step 8：实现 GetSettings 脱敏副本**

`app/settings.go`：

```go
func (a *App) GetSettings() (config.Config, error) {
    if err := a.configManager.Load(); err != nil { return config.Config{}, err }
    safe := a.configManager.Get()
    safe.ConnectionProfiles = nil
    safe.APIKey = ""
    safe.BaseURL = ""
    safe.ConnectionMode = ""
    safe.APIProtocol = ""
    safe.EndpointPath = ""
    safe.ExtraHeaders = nil
    safe.ResponseFormat = ""
    safe.RequestTemplate = ""
    safe.ResponseJSONPointer = ""
    safe.AuthHeader = ""
    safe.AuthPrefix = ""
    safe.Model = ""
    return safe, nil
}
```

保留 `ActiveConnectionProfileID` 和所有普通全局设置。更新 `TestGetSettingsLoadsBeforeStartup`：先确认 Load 成功和普通设置存在，再断言所有连接字段为空且 Profiles 未返回。

同文件将 `UpdateSettings` 的失败返回改为固定安全文案，例如“设置保存失败，原配置未修改”；禁止把 `err.Error()`、配置路径或 Profile 内容直接返回前端。增加 `TestUpdateSettingsReturnsSafeErrorWithoutPathOrSecret`，用不可写目标触发失败并断言结果不包含测试路径或 sentinel。

- [ ] **Step 9：补齐原子性、删除和 secret 测试**

增加：

```go
func TestSaveConnectionSettingsCreatesUpdatesAndActivatesProfileAtomically(t *testing.T)
func TestSaveConnectionSettingsFailureLeavesProfilesProviderAndSubscribersUnchanged(t *testing.T)
func TestSaveConnectionResultContainsGeneratedIdentityButNoSecret(t *testing.T)
func TestDeleteConnectionProfileAllowsOnlyInactiveAndDoesNotChangeRootConnection(t *testing.T)
func TestProfileErrorsNeverContainSecretOrFilesystemPath(t *testing.T)
func TestConnectionAddressLabelDropsSchemePathQueryAndUserinfo(t *testing.T)
```

定向删除 `settings_test.go` 中所有 `SavedModels` / `ModelHistory` 测试和断言，保留无关连接原子性覆盖并改为 Profile 语义。

- [ ] **Step 10：运行 Task 3 聚焦和 App 全包**

Run:

```bash
gofmt -w app/connection_profiles.go app/connection_profiles_test.go app/llm.go app/llm_test.go app/settings.go app/settings_test.go
go test ./app -run 'Test(.*ConnectionProfile|ListConnectionProfiles|GetConnectionProfile|GetSettings|SaveConnection|ProfileErrors|ConnectionAddressLabel)' -count=1
go test ./app -count=1
```

Expected: PASS；测试仅使用本地 Config 和 `httptest`，不调用真实 API。

- [ ] **Step 11：未暂存检查点**

完整 Git 预检后运行：

```bash
git diff --check -- app pkg/config
git diff --cached
```

Expected: 暂存区为空；无 secret 实值或无关 app 文件差异。

---

### Task 4：实现 Pinia Profile 状态机并定向撤销前端模型历史

**Files:**
- Create: `frontend/connection_profiles_regression_test.mjs`
- Modify: `frontend/src/services/api.js`
- Modify: `frontend/src/stores/settings.js`
- Modify: `frontend/unified_api_settings_regression_test.mjs`
- Modify: `frontend/ccr_settings_lifecycle_regression_test.mjs`
- Modify: `frontend/model_connectivity_regression_test.mjs`
- Modify: `frontend/settings_regression_test.mjs`
- Delete: `frontend/model_history_regression_test.mjs`

**Interfaces:**
- Produces Store state: `connectionProfiles`、`activeConnectionProfileID`、`selectedConnectionProfileID`、`connectionProfileName`、`connectionProfileLoading`、`connectionProfileLoaded`、`pendingDeleteProfileID`。
- Produces actions: `selectConnectionProfile`、`beginNewConnectionProfile`、`requestDeleteConnectionProfile`、`cancelDeleteConnectionProfile`、`confirmDeleteConnectionProfile`。
- Invariant: `settings` 始终表示已持久化活动连接；`tempSettings` 表示当前选中草稿；选择/新建只改 temp，不改 settings。

- [ ] **Step 1：建立 Profile Store harness 红灯**

新测试复制 `unified_api_settings_regression_test.mjs` 的 Pinia/vm harness，但 fixture 拆成：

```js
const globalFixture = {
  activeConnectionProfileId: 'profile-a',
  outputMode: 'dual', resumeContent: 'keep-resume', theme: 'light',
}
const profileA = {
  id: 'profile-a', name: 'A 连接',
  connection: {
    connectionMode: 'direct', apiKey: 'test-a', baseURL: 'https://a.example.test/v1',
    apiProtocol: 'openai_chat', endpointPath: '/chat/completions', extraHeaders: {},
    responseFormat: 'text', requestTemplate: '', responseJSONPointer: '',
    authHeader: '', authPrefix: '', model: 'model-a',
  },
}
const profileById = {
  'profile-a': profileA,
  'profile-b': {
    id: 'profile-b', name: 'B 连接',
    connection: {
      ...profileA.connection, apiKey: '', baseURL: 'https://b.example.test/v1', model: 'model-b',
    },
  },
}
const summaries = Object.values(profileById).map(profile => ({
  id: profile.id, name: profile.name,
  addressLabel: new URL(profile.connection.baseURL).hostname,
  apiProtocol: profile.connection.apiProtocol, model: profile.connection.model,
  active: profile.id === 'profile-a', apiKeySet: Boolean(profile.connection.apiKey), templateSet: false,
}))
const clone = value => JSON.parse(JSON.stringify(value))
```

API mock 必须记录：

```js
listConnectionProfiles: async () => ({ profiles: summaries, error: null }),
getConnectionProfile: async id => {
  calls.push(['get-profile', id])
  return { success: true, profile: clone(profileById[id]) }
},
saveConnectionSettings: async draft => {
  calls.push(['save-profile', clone(draft)])
  const id = draft.id || 'generated-profile-id'
  const name = draft.name || 'api.example.test · OpenAI Chat'
  return {
    success: true,
    profile: {
      id, name, addressLabel: 'api.example.test', apiProtocol: draft.connection.apiProtocol,
      model: draft.connection.model, active: true,
      apiKeySet: Boolean(draft.connection.apiKey), templateSet: Boolean(draft.connection.requestTemplate),
    },
    connection: {
      connectionMode: 'direct', baseURL: draft.connection.baseURL,
      model: draft.connection.model, apiProtocol: draft.connection.apiProtocol,
      endpointPath: draft.connection.endpointPath, responseFormat: draft.connection.responseFormat,
      apiKeySet: Boolean(draft.connection.apiKey),
      extraHeadersSet: Object.keys(draft.connection.extraHeaders || {}).length > 0,
      templateSet: Boolean(draft.connection.requestTemplate),
    },
  }
},
deleteConnectionProfile: async id => { calls.push(['delete-profile', id]); return { success: true, deletedId: id } },
```

首批断言：

```js
await store.loadSettings()
assert.equal(store.activeConnectionProfileID, 'profile-a')
assert.equal(store.selectedConnectionProfileID, 'profile-a')
assert.equal(store.settings.apiKey, 'test-a')
assert.equal(store.tempSettings.apiKey, 'test-a')
assert.equal(calls.filter(x => x[0] === 'get-profile').length, 1)
```

- [ ] **Step 2：运行红灯**

Run:

```bash
node frontend/connection_profiles_regression_test.mjs
```

Expected: FAIL；Store 和 API wrapper 尚无 Profile 接口。

- [ ] **Step 3：增加 API wrapper**

在 `api.js` 增加：

```js
listConnectionProfiles: () => callApp('ListConnectionProfiles'),
getConnectionProfile: id => callApp('GetConnectionProfile', id),
saveConnectionSettings: draft => callApp('SaveConnectionSettings', draft),
deleteConnectionProfile: id => callApp('DeleteConnectionProfile', id),
```

保留目录/probe 的 bridge timeout；Profile 本地读写不套模型网络 timeout。

- [ ] **Step 4：重构初始化为全局快照 + 单条连接草稿**

`applyConfig` 只应用普通全局字段，不从 `GetSettings` 读取连接字段。新增：

```js
function applyConnectionDraft(target, draft) {
  Object.assign(target, {
    connectionMode: 'direct', apiKey: draft.apiKey || '', baseURL: draft.baseURL || '',
    apiProtocol: draft.apiProtocol || 'openai_chat',
    endpointPath: draft.endpointPath ?? defaultEndpointPath(draft.apiProtocol),
    extraHeaders: { ...(draft.extraHeaders || {}) }, responseFormat: draft.responseFormat || 'text',
    requestTemplate: draft.requestTemplate || '', responseJSONPointer: draft.responseJSONPointer || '',
    authHeader: draft.authHeader || '', authPrefix: draft.authPrefix ?? '', model: draft.model || '',
  })
}
```

`loadSettings` 必须顺序执行：

1. `getSettings()` 取得脱敏普通设置和活动 ID。
2. `listConnectionProfiles()` 取得安全摘要。
3. 选择 `activeConnectionProfileId`；旧配置则选择列表中的 `legacy-current`。
4. `getConnectionProfile(targetID)` 单条读取。
5. 先把后端原始草稿应用到 `settings`，使其继续表示真实持久化活动连接；再通过现有 `migrateLegacyConnectionDraft(settings)` 生成 `tempSettings`。`openai_responses` / `anthropic_messages` 因此只在 temp 中成为 `custom_json` 迁移草稿，未保存前不得改写 settings 或磁盘。
6. 记录活动/选中 ID 与名称，格式化 temp Headers 文本。
7. 全部成功后才把 `settingsLoaded` / `connectionProfileLoaded` 设为 true。

任一步失败时设置安全错误、禁止保存，绝不能以默认空连接覆盖后端。

- [ ] **Step 5：实现无副作用选择与新建**

`selectConnectionProfile(id)`：

- 递增 `sessionRevision`、`connectionRevision`、`catalogRevision`、`probeRevision` 和专用 `profileLoadRequestVersion`。
- 清空目录/probe 状态。
- 单条读取并只写 `tempSettings`、选中 ID/name、Headers 文本。
- 不写 `settings` 或 active ID，不调用目录/probe/save。
- 迟到结果通过 request version + session revision 丢弃。

`beginNewConnectionProfile()` 建立：

```js
{
  connectionMode: 'direct', apiKey: '', baseURL: '', apiProtocol: 'openai_chat',
  endpointPath: '/chat/completions', extraHeaders: {}, responseFormat: 'text',
  requestTemplate: '', responseJSONPointer: '', authHeader: '', authPrefix: '', model: '',
}
```

选中 ID/name 均为空，不复制当前 Key/Header/模板；本地空草稿创建完成后 `connectionProfileLoaded=true`，允许用户填写并保存，但活动 ID 与 `settings` 保持不变。

`resetTempSettings` / 关闭设置必须把选中 ID和名称恢复为活动摘要，并从 `settings` 重建 temp（旧协议仍通过迁移 helper 生成草稿）；不得保留未保存的非活动选择或新建草稿。

自动名称预览 computed 只显示 `<hostname> · OpenAI Chat/自定义模板` 或“未命名连接”；冲突编号由后端最终确认。

- [ ] **Step 6：改造连接专用保存**

发送结构必须精确为：

```js
{
  id: selectedConnectionProfileID.value,
  name: connectionProfileName.value,
  connection: connectionDraft(),
}
```

成功后：

- 使用返回的安全 summary 更新/追加列表并把其他项 `active=false`。
- 把保存草稿应用到 `settings` 和 `tempSettings`。
- 更新 active/selected ID 和正式名称。
- 更新 `backendSnapshot.activeConnectionProfileId`，但不得把 Key/Profiles写回 backendSnapshot。
- 失败保留 temp 草稿，settings、活动 ID和列表不变。

保留现有 save request version，确保修改草稿、切换 Profile 或关闭弹窗后的迟到结果被忽略。

- [ ] **Step 7：改造保存全部设置和静默保存**

`buildConfigToSave` 新增参数 `profileID/profileName` 并发送：

```js
connectionProfileId: profileID,
connectionProfileName: profileName,
```

删除 `modelHistory`。普通“保存全部设置”使用当前 `tempSettings + selected ID/name`；成功后选中记录成为活动记录，并重新调用安全列表与单条读取，取得后端生成的 ID、正式名称和规范化草稿。

`saveSettingsSilent` 必须使用 `settings + activeConnectionProfileID`，禁止使用当前正在查看的非活动 `tempSettings`，避免主题自动保存意外激活另一条连接。若活动草稿尚未加载，直接返回 false，不写空连接。

- [ ] **Step 8：实现删除状态机**

- `requestDeleteConnectionProfile(id)` 仅对非活动、非 legacy ID设置 `pendingDeleteProfileID`。
- `cancelDeleteConnectionProfile()` 清空确认状态。
- `confirmDeleteConnectionProfile()` 调用本地删除接口；成功后移除摘要并加载下一条可用记录到 temp。
- 删除失败保留列表和草稿并显示固定错误。
- 删除调用不得触发目录/probe。

- [ ] **Step 9：定向删除前端模型历史**

从 Store 删除：

- `settings.modelHistory`
- `normalizedHistoryIdentity`
- `sameHistoryIdentity`
- `upsertModelHistory`
- `savedModelsForCurrentConnection`
- 保存成功与 payload 中全部历史更新
- return object 中的历史 computed

删除 `frontend/model_history_regression_test.mjs`。在新回归中断言 full-save payload 不含 `modelHistory` 或 `connectionProfiles`。

- [ ] **Step 10：补齐竞态与安全回归**

`connection_profiles_regression_test.mjs` 至少覆盖：

```js
// 选择 B 只改 temp，不改活动 settings，也没有 models/probe/save 调用。
// 新建为空，不复制 A 的 key/header/template。
// 迟到的 B 读取结果不能覆盖后来选择的 A。
// 保存失败不增加摘要、不改变 active ID。
// 保存新建后取得后端生成 ID/name 并激活。
// 全量保存携带 selected ID/name 且不携带 Profiles/modelHistory。
// 静默主题保存使用 active ID，不激活当前查看的非活动项。
// active 和 legacy 删除被前端阻止；非活动删除只调用 delete bridge。
// 初始化 Profile 读取失败时 settingsLoaded=false，所有保存动作被阻止。
```

同步更新 `unified_api_settings_regression_test.mjs` fixture，使所有原有两模式、Keyless、旧 Responses/Anthropic 草稿迁移、目录/probe 和迟到保存测试在 Profile 初始化后继续执行。

- [ ] **Step 11：运行 Task 4 前端回归**

Run:

```bash
node frontend/connection_profiles_regression_test.mjs
node frontend/unified_api_settings_regression_test.mjs
node frontend/ccr_settings_lifecycle_regression_test.mjs
node frontend/model_connectivity_regression_test.mjs
node frontend/settings_regression_test.mjs
```

Expected: PASS；删除的 `model_history_regression_test.mjs` 不再运行。

- [ ] **Step 12：未暂存检查点**

完整 Git 预检后运行：

```bash
git diff --check -- frontend/src/services/api.js frontend/src/stores/settings.js frontend/connection_profiles_regression_test.mjs frontend/unified_api_settings_regression_test.mjs frontend/ccr_settings_lifecycle_regression_test.mjs frontend/model_connectivity_regression_test.mjs frontend/settings_regression_test.mjs
git diff --cached
```

Expected: 暂存区为空；Store 中无 `modelHistory` / `savedModelsForCurrentConnection`。

---

### Task 5：实现 Profile 管理 UI 并恢复单一模型目录

**Files:**
- Create: `frontend/src/components/ConnectionProfileManager.vue`
- Modify: `frontend/src/components/SettingsModal.vue`
- Modify: `frontend/src/components/ModelSelect.vue`
- Modify: `frontend/connection_profiles_regression_test.mjs`
- Modify: `frontend/ccr_ui_regression_test.mjs`
- Modify: `frontend/model_layout_regression_test.mjs`

**Interfaces:**
- Consumes: Task 4 Profile refs/actions。
- Produces: 页面顶部“已配置连接”、新建、记录名称和内联删除确认。
- Invariant: Key 输入继续由 `ProviderSelect` 以 `type="password"` 呈现；无明文显示按钮；弹窗保持 720px 和无横向标签滚动。

- [ ] **Step 1：写组件静态红灯**

在 `connection_profiles_regression_test.mjs` 读取三个 Vue 文件并断言：

```js
assert.ok(modalSource.includes('<ConnectionProfileManager'))
assert.ok(profileSource.includes('已配置连接'))
assert.ok(profileSource.includes('新建连接'))
assert.ok(profileSource.includes('记录名称'))
assert.ok(profileSource.includes('确认删除'))
assert.equal(profileSource.includes('confirm('), false)
assert.equal(modelSource.includes('savedModels'), false)
assert.equal(modelSource.includes('已保存模型'), false)
```

- [ ] **Step 2：运行红灯**

Run:

```bash
node frontend/connection_profiles_regression_test.mjs
```

Expected: FAIL；Profile 组件不存在，ModelSelect 仍含模型历史分组。

- [ ] **Step 3：实现 ConnectionProfileManager 组件**

组件 props：

```js
profiles, selectedId, activeId, profileName, loading, loaded,
pendingDeleteId, automaticNamePreview
```

组件 emits：

```js
'select', 'new', 'update:profileName', 'request-delete', 'cancel-delete', 'confirm-delete'
```

模板要求：

- 下拉 option 展示名称、协议、`addressLabel`、模型、活动标记和 Key `SET / NOT SET`。
- 名称 input `maxlength="64"` 仅作交互提示；后端仍按 UTF-8 字节校验。
- 新建时名称 placeholder 使用自动预览，实际 value 保持空。
- 新建按钮在正式 Profile 已达 20 条时禁用并显示上限提示；`legacy-current` 不计入正式数量，后端上限仍是最终权威。
- 活动记录和 `legacy-current` 的删除按钮禁用并显示原因。
- 删除确认使用组件内区域和两个普通按钮，不调用浏览器 `confirm()`。
- 所有异步加载期间禁用重复操作。

- [ ] **Step 4：接入 SettingsModal**

在 API 页 `ProviderSelect` 前插入组件并绑定 Store。连接保存、刷新、文字/图片 probe 和 footer“保存全部设置”额外要求 `connectionProfileLoaded && !connectionProfileLoading`；加载失败时显示安全提示和“重新读取连接记录”入口。

保持：

- `ProviderSelect` 两模式和高级设置。
- `ModelSelect` Teleport/滚动锁。
- 场景独立标签。
- 720px 弹窗、七标签完整显示和窄屏换行。

- [ ] **Step 5：恢复 ModelSelect 单一目录候选**

props 改为：

```js
modelValue, models, loading, disabled, placeholder, active
```

单一 computed：

```js
const filteredModels = computed(() => filterModels(props.models))
const hasSuggestions = computed(() => filteredModels.value.length > 0)
```

模板只循环 `filteredModels`，删除“已保存模型 / 目录候选”分组标签和相关去重逻辑；仍允许自由输入。SettingsModal 改为 `:models="ui.availableModels"`。

- [ ] **Step 6：补齐布局和可访问性回归**

更新 `ccr_ui_regression_test.mjs`：要求 `filteredModels`，禁止 `filteredSavedModels` / `savedModels`。更新布局测试断言：Profile 区域可换行、select/input `min-width: 0`、按钮不造成横向滚动，原标签栏无滚动断言继续通过。

- [ ] **Step 7：运行 Task 5 回归和 Vite build**

Run:

```bash
node frontend/connection_profiles_regression_test.mjs
node frontend/ccr_ui_regression_test.mjs
node frontend/model_layout_regression_test.mjs
node frontend/unified_api_settings_regression_test.mjs
npm --prefix frontend run build
```

Expected: PASS；无 Vue 编译错误或横向布局回归。

- [ ] **Step 8：未暂存检查点**

完整 Git 预检后运行：

```bash
git diff --check -- frontend/src/components frontend/src/stores/settings.js frontend/connection_profiles_regression_test.mjs frontend/ccr_ui_regression_test.mjs frontend/model_layout_regression_test.mjs
git diff --cached
```

Expected: 暂存区为空；不出现题库、手机 Web UI 或 shortcut service 差异。

---

### Task 6：固定 Wails v2.11.0 bindings 并同步双语说明

**Files:**
- Modify generated: `frontend/wailsjs/go/app/App.js`
- Modify generated: `frontend/wailsjs/go/app/App.d.ts`
- Modify generated: `frontend/wailsjs/go/models.ts`
- Modify: `README.md`
- Modify: `README_EN.md`
- Verify unchanged: `frontend/wailsjs/runtime/**`
- Verify unchanged: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: Tasks 1～5 的 Go DTO/方法。
- Produces: 前端生成绑定与 Profile API 一致；模型历史生成类型消失。

- [ ] **Step 1：运行生成前聚焦测试**

```bash
go test ./pkg/config ./app -count=1
node frontend/connection_profiles_regression_test.mjs
npm --prefix frontend run build
```

Expected: PASS。

- [ ] **Step 2：冻结生成边界哈希**

在仓库外临时清单记录 SHA-256：

```text
go.mod
go.sum
frontend/wailsjs/go/app/App.js
frontend/wailsjs/go/app/App.d.ts
frontend/wailsjs/go/models.ts
frontend/wailsjs/runtime/**
frontend/dist/**
.git/index
```

只记录仓库文件哈希，不读取任何用户配置。

- [ ] **Step 3：固定 v2.11.0 生成 bindings**

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 generate module -nocolour
```

Expected:

- `App.js/App.d.ts` 增加 List/Get/Delete Profile 方法，Save 方法参数变为 `ConnectionProfileDraft`。
- `models.ts` 增加 Profile 持久类型和安全 DTO。
- `ModelHistoryEntry`、`modelHistory`、`savedModels` 消失。
- `go.mod`、`go.sum`、runtime 和 `.git/index` 不变。

若生成器只产生纯空白行 Tab，定向清除纯空白噪音；不得手工改生成类型语义。

- [ ] **Step 4：同步 README 中英文**

说明必须包含：

- 支持保存最多 20 条加密连接记录。
- 每条记录保存完整连接字段，Key 输入仍为密码框。
- 选择记录只加载草稿，保存成功后才激活。
- 活动记录不能直接删除。
- 自定义模板仍不提供模型目录，模型可手填。
- 不再宣称“按连接保存模型历史”。
- 保留“隐身模式 / 隐藏/显示窗口”快捷键说明。
- 许可边界仍为 CC BY-NC 4.0。

示例不得包含真实 URL、Key、Header 或模板值。

- [ ] **Step 5：复核生成边界与回归**

```bash
go test ./pkg/config ./app -count=1
node frontend/connection_profiles_regression_test.mjs
node frontend/shortcut_hint_sync_regression_test.mjs
npm --prefix frontend run build
go mod verify
```

比较 Step 2 哈希，除预期三个 Go binding 文件和 Vite dist 外全部保持不变；dist 变化只作为构建产物，不加入 Git。

- [ ] **Step 6：未暂存检查点**

完整 Git 预检后运行：

```bash
git diff --check -- frontend/wailsjs/go README.md README_EN.md
git diff --cached
```

Expected: 暂存区为空；依赖/runtime 无变化。

---

### Task 7：执行自动验证矩阵和安全边界审计

**Files:**
- Modify governance after results: `_dev/changes/037-unified-api-connection/03-implementation.md`
- Modify governance after results: `_dev/changes/037-unified-api-connection/04-validation.md`
- Modify audit after results: `_dev/audits/037-unified-api-connection.md`

**Interfaces:**
- Consumes: 完整 Revision D 工作区。
- Produces: 可重复的自动验证证据；不构建或晋升稳定版。

- [ ] **Step 1：扫描 Revision C 错误方向残留**

使用专用内容搜索确认业务代码和生成代码中不存在：

```text
ModelHistoryEntry
modelHistory
RememberCurrentModel
SavedModelsForCurrentConnection
savedModelsForCurrentConnection
filteredSavedModels
已保存模型
```

历史治理文档中的 Revision C 事实记录允许保留；业务源码、tests、bindings 和 README 不允许残留产品实现/说明。

- [ ] **Step 2：运行 Go 聚焦测试**

```bash
go test ./pkg/config -run 'Test(.*ConnectionProfile|UpdateFromJSON.*Profile|LoadLegacyConnection)' -count=1 -v
go test ./app -run 'Test(.*ConnectionProfile|ListConnectionProfiles|GetConnectionProfile|GetSettings|SaveConnection)' -count=1 -v
go test ./pkg/llm ./pkg/solution -count=1
```

Expected: PASS。

- [ ] **Step 3：运行全部前端回归**

```bash
for test_file in frontend/*_test.mjs; do node "$test_file" || exit 1; done
```

Expected: 全部现存测试 PASS；新 Profile 回归存在，模型历史回归已删除，快捷键同步回归继续 PASS。

- [ ] **Step 4：运行普通全量门禁**

```bash
go test ./... -count=1
go vet ./...
go build ./...
go mod verify
npm --prefix frontend run build
```

逐项独立记录退出码和结果，任何 FAIL 不得汇总为 PASS。

- [ ] **Step 5：运行 Windows race 并如实分类**

```bash
go test -race ./pkg/config ./pkg/llm ./pkg/solution ./app -count=1
```

若仍在测试逻辑启动前返回 `exit status 0xc0000139`，记录 `BLOCKED`，保留原始结果；普通非 race PASS 不能替代 race。

- [ ] **Step 6：执行 secret 和数据最小化审计**

检查：

- 测试 sentinel 仅存在测试输入/断言，不出现在生产日志或固定错误文案。
- `GetSettings` 清空 Profiles 和全部根级连接编辑字段。
- List/Save/Delete DTO 不含完整 Key/Header/模板/AuthPrefix。
- `GetConnectionProfile` 只按一个 ID 返回一个草稿。
- 前端 full-save payload 不含 `connectionProfiles`。
- Key 输入仍是 `type="password"`，无显示明文按钮。
- 无正式配置路径、真实地址或凭据进入 diff。

- [ ] **Step 7：差异和诊断检查**

先执行完整 Git 预检，再运行：

```bash
git diff --check
git diff --cached
```

调用 VS Code diagnostics 检查所有当前可见问题；缺少诊断不冒充全项目 clean，以实际测试/构建为准。

- [ ] **Step 8：记录真实结果**

在 `03-implementation.md` 和 `04-validation.md` 逐项写命令、环境、退出结果与 PASS/FAIL/BLOCKED；不复制 secret、配置路径或临时 Profile 内容。任何必要自动门禁失败时停止候选构建。

---

### Task 8：构建 Wails v2.11.0 保护候选并完成隔离启动

**Files/Artifacts:**
- Modify ignored artifact: `build/bin/testing/current/Q-Solver-candidate-change037.exe`
- Modify ignored manifest: `build/bin/testing/current/SHA256SUMS`
- Archive superseded wrong-direction candidate: `build/bin/archive/transient/**`
- Verify unchanged: `build/bin/Q-Solver.exe`
- Verify unchanged: `build/bin/releases/**`
- Modify governance: `_dev/changes/037-unified-api-connection/04-validation.md`

**Interfaces:**
- Consumes: Task 7 普通门禁通过的源码和固定 bindings。
- Produces: 唯一 Revision D 候选及其 SHA-256、模块版本和独立启动证据。
- Invariant: 不晋升根稳定版，不创建 Release，不读取正式用户配置。

- [ ] **Step 1：盘点并冻结构建保护清单**

在仓库外记录 `build/bin/**` 相对路径、大小和 SHA-256；单独冻结：

```text
build/bin/Q-Solver.exe
build/bin/releases/**
go.mod
go.sum
frontend/wailsjs/**
frontend/dist/**
.git/index
```

确认根目录唯一 EXE 为 `Q-Solver.exe`。

- [ ] **Step 2：归档 Revision C 错误候选**

计算当前 `testing/current` 中 Revision C 候选哈希，将其移动到 `build/bin/archive/transient/` 的 Change 037 明确命名目录并生成/验证 `SHA256SUMS`。不得放入 validation、superseded 或 releases；不得删除唯一稳定版副本。

- [ ] **Step 3：准备 current 测试区**

复制根稳定版为：

```text
build/bin/testing/current/Q-Solver-stable-before-test.exe
```

记录哈希。确认 current 中没有第二个候选。

- [ ] **Step 4：固定 Wails v2.11.0 构建唯一候选**

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build \
  -s \
  -o Q-Solver-candidate-change037.exe \
  -skipbindings \
  -skipembedcreate \
  -nosyncgomod \
  -m \
  -nocolour \
  -v 1 \
  -ldflags "-s -w" \
  -tags prod
```

把候选放入 `testing/current/` 并生成 SHA256SUMS。构建失败则记录 FAIL，根稳定版不变。

- [ ] **Step 5：证明 runtime 与保护文件未漂移**

```bash
go version -m build/bin/testing/current/Q-Solver-candidate-change037.exe
```

Expected: 链接 `github.com/wailsapp/wails/v2 v2.11.0`。比较 Step 1 哈希：依赖、bindings/runtime、Git index、根稳定版和 releases 全部不变；任何新增生成噪音先停止后续验证并定向处理。

- [ ] **Step 6：空隔离 Profile 启动烟测**

使用全新 `%TEMP%` 子目录分别设置 `APPDATA` 和 `LOCALAPPDATA`，启动候选 8～10 秒；确认：

- WebView2 主进程启动。
- 无 Node/Vite 子进程。
- 可以正常退出，退出码符合正常关闭路径。
- 无候选残留进程。
- 未访问正式用户 Profile。

不在日志或治理文档记录实际临时路径。

- [ ] **Step 7：记录候选证据**

在 `04-validation.md` 记录候选相对路径、大小、SHA-256、Wails v2.11.0、启动/退出/残留结果，以及根稳定版和 releases `UNCHANGED`。不晋升候选。

---

### Task 9：固定隔离 Profile 双启动人工验收与治理收尾

**Files:**
- Modify: `_dev/changes/037-unified-api-connection/03-implementation.md`
- Modify: `_dev/changes/037-unified-api-connection/04-validation.md`
- Modify: `_dev/changes/037-unified-api-connection/05-rollback.md`
- Modify: `_dev/changes/037-unified-api-connection/06-notes.md`
- Modify: `_dev/audits/037-unified-api-connection.md`
- Modify: `_dev/changes/INDEX.md`

**Interfaces:**
- Consumes: Task 8 候选。
- Produces: Profile、题库目录、活动连接和快捷键跨重启验收证据；提交边界清单。
- Invariant: 人工验收未完成时状态为 `NEEDS_USER`，不得 Close、晋升或提交。

- [ ] **Step 1：准备固定隔离 Profile**

固定复用：

```text
%TEMP%\qsolver-change037-manual-profile\
```

候选重建时不删除。只设置候选进程的 `APPDATA/LOCALAPPDATA`，不读取、复制或修改正式用户配置。记录状态只使用 `SET / NOT SET`。

- [ ] **Step 2：第一次启动验收旧单连接和 Profile 管理**

在候选 GUI 验证：

1. 若隔离配置仍是 Revision C 单连接，列表显示临时“当前连接”，读取页面不自动改写配置。
2. 首次保存后生成正式自动名称和随机 ID，`legacy-current` 消失。
3. 新建第二条记录时地址、Key、Header、模板和模型为空/default，不复制第一条秘密字段。
4. 保存两条记录后可选择任一记录，地址和 Key 可恢复，Key 始终密码显示。
5. 选择记录不自动刷新模型、不测试连接、不切换当前解题 Provider。
6. 保存选中记录后才更新活动标记。
7. 活动记录删除按钮禁用；非活动记录显示应用内确认并可删除；删除不调用外部接口。
8. 同 URL 不同名称记录可共存；重复手动名称被安全拒绝。

不点击真实模型 probe，不产生收费 API 调用。

- [ ] **Step 3：第一次启动保存跨重启状态**

在同一隔离 Profile 中保存：

- 至少两条 Connection Profiles，其中 Key 状态分别记录 `SET / NOT SET`。
- 一条活动 Profile。
- 题库目录状态 `SET`。
- 一个非默认快捷键状态 `SET`。

核对桌面提示立即更新；保存全部设置后手机端提示使用已持久化快捷键。正常关闭候选并确认无残留进程。

- [ ] **Step 4：第二次启动复核**

使用完全相同隔离 Profile 重启，确认：

- Profile 数量、名称和顺序保留。
- 活动 Profile 保留，根级运行连接与其一致。
- 显式选择另一条记录后地址、Key 状态及完整高级字段恢复。
- 题库目录仍为 `SET`。
- 快捷键仍为 `SET`，显示名与实际绑定一致。
- “隐身模式”和“隐藏/显示窗口”语义仍正确。
- 没有模型历史分组，模型目录/手填模式仍正常。

再次正常关闭并检查无残留进程。

- [ ] **Step 5：如实记录人工结论**

每项写 `PASS`、`FAIL` 或 `NEEDS_USER`，不得记录地址、Key、Header、模板或题库路径。任何必要项未完成时保持 Change 037 待验收，候选不晋升。

- [ ] **Step 6：执行 Commit Boundary Audit**

完整 Git 预检后，按白名单审计：

**功能提交候选范围（仅规划，不执行）：**

- Config/Profile/Manager 代码与测试。
- App/Wails Profile 接口与测试。
- 前端 Store、API、组件、回归测试。
- Wails v2.11.0 生成文件。
- README / README_EN。

建议 message：

```text
feat: add encrypted connection profiles

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

**治理提交候选范围（仅规划，不执行）：**

- `_dev/audits/037-unified-api-connection.md`
- `_dev/changes/037-unified-api-connection/**`
- `_dev/changes/INDEX.md`

建议 message：

```text
chore: add change 037 governance records

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

检查暂存区为空，禁止夹带其他 Change、正式配置、构建产物、日志或 secret。

- [ ] **Step 7：最终状态**

在用户另行明确授权前：

- 不执行 `git add`、commit、push、稳定版晋升、Tag 或 Release。
- 候选保留在 `testing/current`，根稳定版保持不变。
- INDEX 状态准确写为“Revision D 已实现/验证到实际阶段，待人工验收或待提交”，不得提前 Close。
- 报告工作区“仅有已说明的预期未提交修改”，不得称为 clean。

---

## 计划执行顺序与停止条件

1. Task 1～5 严格 TDD，小步运行聚焦测试。
2. Task 6 固定生成边界后再生成 bindings。
3. Task 7 全部普通自动门禁通过后，才允许 Task 8 构建候选。
4. Task 8 保护哈希、runtime 或独立启动任一失败，停止 Task 9，不触碰根稳定版。
5. Task 9 必要人工项为 `FAIL`、`BLOCKED`、`NEEDS_USER` 时，不 Close、不晋升。
6. 任一阶段发现范围需要修改题库产品代码、依赖、runtime、正式配置或 releases，立即暂停并重新走用户确认。
