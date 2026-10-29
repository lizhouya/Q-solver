# Change 027 实施计划

日期：2026-10-25
状态：计划中

**Goal:** 在不破坏既有 KeepContext 与分段续接语义的前提下，确定并修复
“模型返回与截图不相关问题”的 Q-Solver 侧成因；若成因在 Q-Solver 之外，
则增加可观测的失败提示，使问题不再表现为静默编造。

**Architecture:** 先以 loopback 测试**测量**真实请求中图片的数量与位置，
据此决定修复层；不在测量前预设实现。所有结论必须来自可复现的自动测试。

**Spec:** `_dev/changes/027-vision-failure-detection/00-request.md`、`01-current-state.md`

## Global Constraints

- 工作树叠加 Change 024/025/026 未提交内容；只做增量修改，不清理、不回退。
- 固定 `GOTOOLCHAIN=go1.25.13` 与 Wails CLI v2.11.0。
- 不新增依赖，不修改 `go.mod`、`go.sum`、前端依赖版本。
- 自动测试只用 loopback `httptest`、fake provider、合成图片。
- 不读取真实配置/密钥；不调用真实 CCR 或模型；不消费额度。
- 不输出 API Key、Token、完整私人 URL、prompt 或响应内容。
- 不根据模型名、品牌、域名推断协议或视觉能力。
- 不破坏 KeepContext 与 `ContinueQuestionBankSession` 既有语义。
- 不删除、覆盖或迁移题库 `_记录`、截图、旧导出、正式 EXE、旧验证程序。
- 未授权暂存、commit、push、Tag、Release。
- race 若仍为 `0xc0000139`，记 `BLOCKED`；GUI 与真实 CCR 记 `NEEDS_USER`。

---

## Task 1：建立事前治理与基线

**Files:**
- Create: `_dev/changes/027-vision-failure-detection/00-request.md`
- Create: `_dev/changes/027-vision-failure-detection/01-current-state.md`
- Create: `_dev/changes/027-vision-failure-detection/02-plan.md`
- Create: `_dev/changes/027-vision-failure-detection/03-implementation.md`
- Create: `_dev/changes/027-vision-failure-detection/04-validation.md`
- Create: `_dev/changes/027-vision-failure-detection/05-rollback.md`
- Create: `_dev/changes/027-vision-failure-detection/06-notes.md`
- Create: `_dev/audits/027-vision-failure-detection.md`
- Modify: `_dev/changes/INDEX.md`

- [x] `00-request.md` 完成（含已排除项与待验证项）。
- [x] `01-current-state.md` 完成，并**校正**初稿中基于摘要推测的错误断言。
- [x] `02-plan.md`（本文件）。
- [ ] 记录基线：`go.mod`、`go.sum`、`frontend/package.json`、
      `frontend/package-lock.json` 的 SHA-256。
- [ ] 记录 `build/bin` 现有 EXE 名称/大小/SHA-256，作为受保护基线。
- [ ] 初始化 `03`～`06` 与审计文件为“未实施/未验证”；INDEX 登记为进行中。
- [ ] 不预写 PASS。

**Gate:** 上述记录完成且未改动业务源码后，进入 Task 2。

---

## Task 2：测量真实请求中的图片构成（诊断，非修复）

目的：用可复现的自动测试回答“Q-Solver 到底发了什么”，
而不是继续推测。**本任务不修改生产代码。**

**Files:**
- Create: `pkg/solution/conversation_images_test.go`
- 复用: `pkg/solution/conversation_test.go` 既有 fake provider 模式

- [ ] **Step 1：写测量测试（无断言，先打印结构）**

用 fake provider 捕获 `GenerateContentStream` 收到的 `[]llm.Message`，
统计每个消息的 image part 数量与角色，覆盖三种场景：

1. KeepContext=false，单轮，2 张图。
2. KeepContext=true，连续 3 轮，每轮 2 张图。
3. KeepContext=true，超过 `MaxConversationRounds` 轮。

- [ ] **Step 2：运行并记录实际数字**

```bash
GOTOOLCHAIN=go1.25.13 go test ./pkg/solution -run TestConversationImageComposition -v -count=1
```

记录：第 N 轮请求中各消息的 image part 数量。

- [ ] **Step 3：据结果确定断言**

- 若 KeepContext=true 时旧图随历史重发 → 图片总量随轮次线性增长，
  确认 §3.3 机制触发。→ 进入 Task 3。
- 若旧图未重发 → §3.3 机制不成立，转向 Task 4（prompt 层）与外部边界结论。

- [x] **Step 4：将测量结果写入 `01-current-state.md` 与 `03-implementation.md`。**

### Task 2 实测结果（2026-10-25）

`pkg/solution/conversation_images_test.go` 三个测量用例全部通过，输出：

| 场景 | 请求画像 | 图片总数 |
|---|---|---|
| KeepContext=false，单轮 2 图 | `[system:0 user:2]` | 2 |
| KeepContext=true，第 1 轮 | `[system:0 user:2]` | 2 |
| KeepContext=true，第 2 轮 | `[system:0 user:2 assistant:0 user:2]` | 4 |
| KeepContext=true，第 3 轮 | `… user:2 assistant:0 user:2` | 6 |
| KeepContext=true，第 13 轮（Max=10） | 10 轮历史 + 当前轮 | **22** |

结论：

1. **图片总数随轮次线性增长**，斜率 = 每轮截图数（本次为 2）。
   §3.2 机制**已确认成立**。
2. **上限是 10 轮 × 每轮图数 + 本轮图数**。`trimChatHistoryLocked`
   （`solution.go:264-278`）在**发送前**把 history 裁到 `MaxConversationRounds*2`
   条消息，但当前轮随后才追加（`solution.go:163`），因此请求实际携带
   **11 轮**的图片，而非 10 轮。
3. `MaxScreenshots=3` 时，理论上限为 `10*3 + 3 = 33` 张，**超过 Anthropic
   文档所述的 20 张阈值**，会触发更严格的单图尺寸限制且超限图被拒绝而非缩放。
4. 当前轮截图始终完整存在（`TestConversationImageComposition_KeepTrueBeyondMaxRounds`
   断言通过），**不存在当前轮丢图**。
5. KeepContext=false 时仅 1 轮图片，行为正确。

**该机制仅在 `KeepContext=true` 时生效。用户是否开启仍未确认**（见 Task 4 的前置问题）。
即使机制成立，也不能单独解释“Claude 返回完全不相关题目而 DeepSeek 正确”——
同一请求会同时发给两个模型。因此本 Change 的定位是：**消除一个真实的放大因子，
并让视觉失败变得可见**，而非宣称已找到唯一根因。

**Deliverable:** 关于“每轮请求实际包含多少张图”的可复现结论。已完成。

---

## Task 3：按测量结果限制历史图片规模（已由 Task 2 确认执行）

Task 2 已证明 KeepContext 会重发旧截图，且请求实际携带 11 轮图片。

**Files:**
- Modify: `pkg/solution/solution.go`
- Modify: `pkg/solution/solution_test.go`

- [ ] **Step 1：写 RED 测试**

```go
func TestKeepContextRequestDoesNotExceedHistoryRoundBudget(t *testing.T)
```

断言：KeepContext=true 且每轮 2 图、已完成 `MaxConversationRounds+3` 轮后，
最后一轮请求中的**历史图片总数不超过 `MaxConversationRounds` 轮对应的张数**，
且**当前轮截图完整存在**。

- [ ] **Step 2：运行确认 RED**（当前应为 22 > 20 而失败）

```bash
GOTOOLCHAIN=go1.25.13 go test ./pkg/solution -run TestKeepContextRequestDoesNotExceedHistoryRoundBudget -count=1
```

- [ ] **Step 3：最小实现**

在 `solution.go` 组装 `messagesToSend` 时，把 history 的裁剪预算从
「`MaxConversationRounds` 轮」收紧为「预留当前轮后的可用轮数」，
即 `maxHistoryMessages := (MaxConversationRounds - 1) * 2`，使请求总轮数
（历史 + 当前）恰好等于 `MaxConversationRounds`。

约束：
- **不改** `trimChatHistoryLocked` 的既有语义与 `MaxConversationRounds` 常量值。
- **不改** `SnapshotConversation` / `ReplaceConversation` 契约。
- **不改** KeepContext 的对外开关语义。
- 当前轮截图**不得**受影响。
- 保持 user/assistant 配对不变式。

- [ ] **Step 4：确认 GREEN 并跑全包回归**

```bash
GOTOOLCHAIN=go1.25.13 go test ./pkg/solution -count=1
```

注意 `TestTKC04TrimKeepsRecentRounds` 直接断言 `len(s.chatHistory) == MaxConversationRounds*2`。
若实现只改发送侧的裁剪而不改内部 history 存储，该测试应继续通过；
若它失败，说明改动误伤了存储层，须回到 Step 3 修正，**不得**为迁就实现而改测试。

- [ ] **Step 5：确认分段续接不回归**

```bash
GOTOOLCHAIN=go1.25.13 go test ./app -run QuestionBank -count=1
```

**Deliverable:** 请求图片数不超过 `MaxConversationRounds` 轮预算，当前轮图片完整。

---

## Task 4：增加视觉失败显式指令

独立于 Task 3，可单独生效。

**Files:**
- Modify: `pkg/domain/prompts.go`
- Modify: `pkg/domain/prompts_test.go`

- [ ] **Step 1：写 RED 测试**

```go
func TestSystemPolicyRequiresExplicitVisionFailure(t *testing.T)
```

断言 `GetSystemBehaviorPrompt()` 包含“无法读取/看到图片时须明确说明”的规则，
且**不得**与既有 `不要编造内容`（`prompts.go:156`）冲突或重复。

- [ ] **Step 2：运行确认 RED**

```bash
GOTOOLCHAIN=go1.25.13 go test ./pkg/domain -run TestSystemPolicyRequiresExplicitVisionFailure -count=1
```

- [ ] **Step 3：在 `<QuestionAnsweringRules>` 内新增一条 Rule**

要求：当无法读取当前用户消息中的任何图片时，明确说明无法识别图片，
不得依据文本提示生成或猜测题目。措辞需与既有规则风格一致。

- [ ] **Step 4：确认 GREEN**

```bash
GOTOOLCHAIN=go1.25.13 go test ./pkg/domain -count=1
```

- [ ] **Step 5：回归 prompt 相关测试**

```bash
GOTOOLCHAIN=go1.25.13 go test ./pkg/solution ./pkg/domain -count=1
```

**Deliverable:** 上游丢图时模型明确报错，而非静默编造。

---

## Task 5：全量 Go 验证

- [ ] `GOTOOLCHAIN=go1.25.13 go test ./... -count=1`
- [ ] `GOTOOLCHAIN=go1.25.13 go vet ./...`
- [ ] `GOTOOLCHAIN=go1.25.13 go build ./...`
- [ ] `GOTOOLCHAIN=go1.25.13 go mod verify`

---

## Task 6：前端回归与构建

- [ ] 逐个运行 `frontend/*_regression_test.mjs`（当前 13 个）。
- [ ] `npm --prefix frontend run build`
- [ ] `git diff --check`

本 Change 预期不改前端逻辑；若前端回归失败，须先确认是否为
Change 024/025/026 既有状态，不得归因于本 Change。

---

## Task 7：Wails bindings 与独立验证程序

- [ ] 用固定 Wails v2.11.0 CLI 重生 bindings（**不得**用 PATH 中 v2.16.0）。
- [ ] 审计方法数量与 runtime 无漂移。
- [ ] 仅对生成文件清理尾随空白。
- [ ] 构建 `build/bin/Q-Solver-change027-vision-failure-detection.exe`。
- [ ] 记录大小与 SHA-256；比对受保护 EXE 前后哈希一致。

本 Change 不新增 App 方法，bindings 应无实质变化；若出现差异须查明原因。

---

## Task 8：race 与审计

- [ ] `GOTOOLCHAIN=go1.25.13 go test -race ./pkg/solution ./pkg/domain ./app -count=1`
      若 `0xc0000139` → `BLOCKED`。
- [ ] 依赖文件哈希与基线一致。
- [ ] 网络审计：全部 loopback；零真实调用。
- [ ] 隐私审计：无 Key/Token/URL/prompt/响应体输出。
- [ ] 数据审计：未触碰题库、截图、导出、用户配置。

---

## Task 9：治理收尾

- [ ] 完成 `03`～`06` 与审计文件，逐项标 `PASS/FAIL/BLOCKED/NEEDS_USER`。
- [ ] 更新 `INDEX.md`。
- [ ] 保持工作树未暂存、未提交、未推送、未 Tag、未发布。

---

## 人工验收（NEEDS_USER）

1. Direct 三协议多图解题正常。
2. CCR + 已知支持视觉的模型（DeepSeek）多图解题正常。
3. KeepContext 开启下连续多轮解题，答案不串题、不重复旧题。
4. `StartNextExam` 后新分段答案不含旧分段内容。
5. 若上游丢图，模型明确说明无法识别图片。
6. 用户在 CCR 侧核对 alias 上游与视觉透传。

---

## 回滚

1. 只按 Change 027 白名单定向回滚。
2. 禁止 destructive Git 命令；不触碰 Change 024/025/026 与用户数据。
3. 还原 `pkg/solution/solution.go` 与 `pkg/domain/prompts.go` 的改动。
4. Wails v2.11.0 重生 bindings，确认无漂移。
5. 不删除 Change 027 文档与验证程序；标注为已回滚。
6. 未获授权不提交回滚。

---

## 明确排除范围

- 不新增 CCR adapter，不改 CCR 入站协议。
- 不按模型名推断视觉能力或切换格式。
- 不修改用户 CCR 配置或 APMIX 订阅。
- 不实现“答案是否匹配”的启发式后处理。
- 不重构 KeepContext 架构，不改变其对外语义。
- 不修改题库导出、编号或 `_记录` 结构。
- 不调用真实服务做自动验收。
- 不新增依赖，不覆盖正式 EXE。
- 未授权不做任何 Git 写操作。
