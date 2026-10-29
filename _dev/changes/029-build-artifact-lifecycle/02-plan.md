# Q-Solver 构建产物生命周期与整合提交实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `build/bin/` 整理为根目录唯一稳定版、测试中双产物和按 Change 归档的可审计结构，完成 Change 024～031 累计实现验证、两个实施提交及指定远端推送。

**Architecture:** 二进制继续由 Git 忽略，以迁移前清单、目录级 `SHA256SUMS` 和 Change 文档提供可审计性。根目录稳定版在候选完整验证前保持不变；候选只在 `testing/current/` 中构建和验证，通过后原子晋升，并把旧稳定版与验证候选分别归档。

**Tech Stack:** Go 1.24.1 module、Wails v2.11.0、Vue 3、Pinia、Vite 4、Node.js、Git Bash、Windows 11/WebView2、Git。

**Spec:** `_dev/changes/029-build-artifact-lifecycle/design.md`

## Global Constraints

- 根目录最终只保留 `build/bin/Q-Solver.exe` 一个 EXE。
- 测试期间 `build/bin/testing/current/` 只放测试前稳定版、一个最新候选和 `SHA256SUMS`。
- 二进制不进入 Git；所有移动、去重、晋升和回滚前后均核对 SHA-256。
- `build/bin/releases/` 内容和哈希保持不变。
- Wails 候选使用项目固定 v2.11.0 与 `-s -skipbindings -skipembedcreate -nosyncgomod -m -nocolour -v 1 -ldflags "-s -w" -tags prod`。
- 既有 `0xc0000139` race 阻塞和 `NEEDS_USER` 项必须如实保留，不冒充 PASS。
- 不读取、提交或输出用户配置、CCR 配置/数据库/日志、仓库外探针、API Key、Token、Cookie 或完整认证 URL。
- Git 仅允许三个已批准提交；最终只普通推送 `origin/dev/qsolver-custom`，不推 upstream、不修改 main、不 force push、不创建 Tag、Release 或 RC。
- 禁止 `git add .` 和 `git add -A`；每个提交仅按审计后的显式文件清单暂存。

---

## 文件结构与职责

| 文件或目录 | 操作 | 单一职责 |
|---|---|---|
| `AGENTS.md` | 修改 | 所有 Agent 必须遵守的构建产物生命周期底线 |
| `CLAUDE.md` | 修改 | Claude Code 构建、验证、晋升、归档和 Git 操作步骤 |
| `_dev/build-artifact-policy.md` | 新建 | 完整目录结构、状态机、SHA-256、失败处理和回滚政策 |
| `_dev/README.md` | 修改 | 增加构建产物政策入口 |
| `_dev/changes/029-build-artifact-lifecycle/02-plan.md`～`06-notes.md` | 新建/修改 | Change 029 计划、实施、验证、回滚和总结 |
| `_dev/changes/INDEX.md` | 修改 | Change 024～031 和 Change 029 的最终真实状态 |
| `_dev/audits/024-031-commit-boundary.md` | 新建 | 两个实施提交的文件白名单、排除项、验证证据和消息 |
| `build/bin/testing/current/` | 新建/清空 | 当前轮稳定版副本、唯一候选和测试期哈希 |
| `build/bin/archive/validation/change-NNN/` | 新建/整理 | 各 Change 验证产物与目录级哈希 |
| `build/bin/archive/superseded/change-029/` | 新建 | 本轮晋升后被替代的旧稳定版与哈希 |
| `build/bin/archive/transient/` | 新建/整理 | 历史开发版和 `.exe~` 临时备份 |
| `build/bin/Q-Solver.exe` | 最后替换 | 唯一用户可直接运行的当前稳定版 |
| `build/bin/releases/` | 只读保护 | 既有 RC 发布资产，不迁移、不覆盖、不删除 |

---

### Task 1：冻结 Git、二进制和 Release 基线

**Files:**
- Create: 仓库外临时清单 `%TEMP%/qsolver-change029-*/`
- Read-only: `build/bin/**`
- Read-only: Git 工作树与暂存区

**Interfaces:**
- Consumes: 设计提交 `8047488` 与现有 `build/bin/`。
- Produces: 迁移前路径/大小/SHA-256 清单、Release 哈希清单、保护文件哈希和 Git 预检证据。

- [ ] **Step 1: 执行 Git 只读预检**

```bash
export GIT_OPTIONAL_LOCKS=0
git status --short --branch
git branch --show-current
git remote -v
git diff --stat
git diff --cached --name-status
git rev-parse HEAD
git rev-parse origin/dev/qsolver-custom
```

Expected: 当前分支为 `dev/qsolver-custom`；HEAD 为设计提交；暂存区为空；远端仍落后本地一个提交。

- [ ] **Step 2: 生成迁移前二进制清单**

```bash
find build/bin -type f -print0 | sort -z | xargs -0 sha256sum
find build/bin -type f -printf '%P|%s\n' | sort
```

Expected: 每个现有文件都有唯一相对路径、大小和 SHA-256；清单写到仓库外临时目录。

- [ ] **Step 3: 单独冻结 Release 与受保护源码哈希**

```bash
find build/bin/releases -type f -print0 | sort -z | xargs -0 sha256sum
sha256sum go.mod go.sum frontend/wailsjs/go/models.ts frontend/wailsjs/runtime/runtime.js frontend/wailsjs/runtime/runtime.d.ts
```

Expected: 后续可逐项比较；命令不修改文件。

- [ ] **Step 4: 检查运行中 Q-Solver 进程**

```bash
powershell.exe -NoProfile -Command "Get-Process | Where-Object { \$_.ProcessName -like 'Q-Solver*' } | Select-Object Id,ProcessName,Path"
```

Expected: 若候选或根目录 EXE 正在运行，在移动或晋升前停止对应测试进程；不得停止无关用户服务。

### Task 2：写入长期产物规则和 Change 029 生命周期文档

**Files:**
- Modify: `AGENTS.md`
- Modify: `CLAUDE.md`
- Create: `_dev/build-artifact-policy.md`
- Modify: `_dev/README.md`
- Create: `_dev/changes/029-build-artifact-lifecycle/03-implementation.md`
- Create: `_dev/changes/029-build-artifact-lifecycle/04-validation.md`
- Create: `_dev/changes/029-build-artifact-lifecycle/05-rollback.md`
- Create: `_dev/changes/029-build-artifact-lifecycle/06-notes.md`

**Interfaces:**
- Consumes: `design.md` 的目录、状态、哈希和回滚约束。
- Produces: 所有后续 Agent 可直接执行的统一政策；Change 029 完整七文件骨架。

- [ ] **Step 1: 在 `AGENTS.md` 增加不可放宽的生命周期底线**

新增“构建产物生命周期”章节，逐项明确：根目录唯一 EXE、测试前稳定版与候选共处路径、通过前不晋升、失败不改稳定版、SHA-256、按 Change 归档、Release 独立授权和二进制继续忽略。

- [ ] **Step 2: 在 `CLAUDE.md` 增加可执行步骤**

写入固定命名、Wails v2.11.0 保护命令、构建前后哈希、测试状态、原子晋升、回滚、显式 Git 暂存和禁止自动发布步骤。

- [ ] **Step 3: 创建完整政策文档**

`_dev/build-artifact-policy.md` 必须包含目录树、命名规则、准备/验证/通过/失败/阻塞状态机、首次整理、`SHA256SUMS` 格式、回滚、Release 边界和检查清单。

- [ ] **Step 4: 在 `_dev/README.md` 增加政策入口**

在目录结构与配套文档表中链接 `_dev/build-artifact-policy.md`，说明被忽略二进制仍受该政策管理。

- [ ] **Step 5: 创建 `03`～`06` 文档骨架并写入已知事实**

文档只记录已完成事实；尚未执行的验证保留为“待执行”，不得预填 PASS。

- [ ] **Step 6: 运行文档自检**

```bash
grep -RInE 'TODO|TBD|待定|\?\?\?' AGENTS.md CLAUDE.md _dev/build-artifact-policy.md _dev/README.md _dev/changes/029-build-artifact-lifecycle || true
git diff --check -- AGENTS.md CLAUDE.md _dev/build-artifact-policy.md _dev/README.md _dev/changes/029-build-artifact-lifecycle
```

Expected: 无占位符；无本轮新增空白错误。

### Task 3：整理历史产物并建立当前测试区

**Files:**
- Modify: `build/bin/**`（全部被 Git 忽略）
- Protect: `build/bin/releases/**`

**Interfaces:**
- Consumes: Task 1 的迁移前清单与当前根目录稳定版。
- Produces: 按 Change 归档的历史验证产物、transient 归档、当前稳定版副本和目录级 `SHA256SUMS`。

- [ ] **Step 1: 创建目标目录但不覆盖任何文件**

```bash
mkdir -p build/bin/testing/current \
  build/bin/archive/validation \
  build/bin/archive/superseded/change-029 \
  build/bin/archive/transient \
  build/bin/archive/failed
```

Expected: 目录可创建；`releases/` 未改变。

- [ ] **Step 2: 复制测试前稳定版并核对哈希**

```bash
cp -p build/bin/Q-Solver.exe build/bin/testing/current/Q-Solver-stable-before-test.exe
sha256sum build/bin/Q-Solver.exe build/bin/testing/current/Q-Solver-stable-before-test.exe
```

Expected: 两个哈希均为迁移前根目录稳定版哈希。

- [ ] **Step 3: 按文件名中的 Change 编号迁移根目录验证 EXE**

对 `Q-Solver-changeNNN-*.exe` 创建 `archive/validation/change-NNN/`。若既有 `archive/validation/` 有同哈希副本，保留规范目标中的一个副本；同名不同哈希时分别保留并停止自动去重。Change 030、031 也只作为历史验证版迁移，不能直接充当本轮最终候选。

- [ ] **Step 4: 整理既有平铺 validation 目录**

将既有 `archive/validation/Q-Solver-changeNNN-*.exe` 移入相应 `change-NNN/`。没有 Change 编号的 `Q-Solver-api-fix.exe` 与 `Q-Solver-sync-fix.exe` 放入 `archive/validation/legacy/`。旧平铺 `SHA256SUMS` 在新目录清单全部验证后移入 `archive/manifests/` 保存为历史证据。

- [ ] **Step 5: 归档开发与临时产物**

```text
Q-Solver-dev.exe                              -> archive/transient/Q-Solver-dev.exe
Q-Solver-change026-ccr-gateway-validation.exe~ -> archive/transient/Q-Solver-change026-ccr-gateway-validation.exe~
```

如目标存在，先比较哈希；不允许静默覆盖。

- [ ] **Step 6: 为所有新增归档目录生成 `SHA256SUMS`**

每个清单格式为：

```text
<64 位大写 SHA-256> *<相对文件名>
```

Expected: 每个含 EXE 的 `change-NNN/`、`legacy/`、`transient/` 和 `testing/current/` 都有可重新计算验证的清单。

- [ ] **Step 7: 对照迁移前清单核对文件集合**

Expected: 除已证明同哈希的重复根目录副本外，所有迁移前文件都能映射到新位置；`releases/` 哈希逐项不变；根目录暂时只保留旧 `Q-Solver.exe`。

### Task 4：审计累计实现并执行完整自动验证

**Files:**
- Read-only/verify: Change 024～031 所有源码、测试、bindings、依赖和文档
- Create: `_dev/audits/024-031-commit-boundary.md`（先记录候选边界，验证后补结果）

**Interfaces:**
- Consumes: 当前累计工作树与历史 Change 档案。
- Produces: 功能/测试白名单、治理白名单、排除项、测试结果和敏感信息扫描证据。

- [ ] **Step 1: 将工作树差异按职责分类**

使用 `git diff --name-status`、`git status --short` 和各 Change 00～06 文件，把文件分为：功能源码、对应测试、必要 bindings/依赖、治理/README、排除项。任何无法归属的文件停止暂存。

- [ ] **Step 2: 运行聚焦 Go 测试**

```bash
go test ./pkg/questionbank ./pkg/config ./pkg/llm ./pkg/solution ./pkg/mobile ./pkg/resume ./app -count=1
```

Expected: PASS；失败则记录原始包和测试名，停止候选晋升。

- [ ] **Step 3: 运行全量 Go 验证**

```bash
go test ./... -count=1
go vet ./...
go build ./...
go mod verify
```

Expected: 四项分别记录退出码；新增失败阻止晋升。已知 race `0xc0000139` 不属于这些标准命令。

- [ ] **Step 4: 运行全部 14 份前端回归脚本**

```bash
for test_file in frontend/*_test.mjs; do node "$test_file"; done
```

Expected: 14 个脚本全部退出 0；记录实际文件数和名称。

- [ ] **Step 5: 运行 Vite production build**

```bash
npm --prefix frontend run build
```

Expected: PASS，生成可由 `main.go` 嵌入的 `frontend/dist`。

- [ ] **Step 6: 运行差异和敏感信息检查**

```bash
git diff --check -- app pkg frontend/src frontend/*.mjs README.md README_EN.md go.mod go.sum
git status --short
git diff --cached --name-status
```

敏感扫描覆盖 staged 候选和未跟踪文本文件，匹配 `serviceToken`、`ccr_web_token`、Authorization bearer、常见 API key 赋值；只报告命中路径和规则，不输出 secret 内容。

Expected: 暂存区仍为空；既有 `models.ts` 空白问题单列，不归因于本 Change；无敏感文件进入候选提交。

### Task 5：构建并验证独立候选

**Files:**
- Create: `build/bin/testing/current/Q-Solver-candidate-change029.exe`
- Modify: `build/bin/testing/current/SHA256SUMS`
- Protect: `go.mod`, `go.sum`, `frontend/wailsjs/**`, `frontend/dist/**` 的预期边界

**Interfaces:**
- Consumes: Task 4 全部通过的源码和 Vite `dist`。
- Produces: 由 Wails v2.11.0 构建的唯一候选、候选哈希、独立启动与退出证据。

- [ ] **Step 1: 构建前记录保护哈希与 Git 状态**

记录 `go.mod`、`go.sum`、Wails bindings/runtime、`frontend/dist` 和 Git index tree；构建后逐项比较。

- [ ] **Step 2: 使用固定 Wails v2.11.0 构建候选**

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build \
  -s \
  -o Q-Solver-candidate-change029.exe \
  -skipbindings \
  -skipembedcreate \
  -nosyncgomod \
  -m \
  -nocolour \
  -v 1 \
  -ldflags "-s -w" \
  -tags prod
```

构建输出必须位于 `build/bin/testing/current/`；如果 Wails 只接受文件名输出到 `build/bin/`，先输出唯一临时名称，再在哈希确认后移入 `testing/current/`。

Expected: 退出 0；不得自动同步 `go.mod` 或生成 bindings。

- [ ] **Step 3: 更新测试区哈希清单**

Expected: 稳定版副本与候选均存在；`SHA256SUMS` 可验证；候选哈希不等于旧稳定版时记录差异，不以哈希不同判失败。

- [ ] **Step 4: 执行独立启动验证**

使用 PowerShell `Start-Process -PassThru` 启动候选，观察至少 8 秒，核对：进程存活、WebView2 子进程出现、没有 Node/Vite 子进程、没有意外 CCR 探针请求。通过窗口关闭或正常终止路径退出，记录退出码并确认无残留 Q-Solver 候选进程。

- [ ] **Step 5: 比较构建前后保护状态**

Expected: `go.mod`、`go.sum`、bindings/runtime 和 Git index 语义未因构建变化；如存在变动，停止晋升并人工区分已有差异与生成噪音。

### Task 6：晋升候选并完成最终产物核对

**Files:**
- Replace after verification: `build/bin/Q-Solver.exe`
- Create: `build/bin/archive/superseded/change-029/Q-Solver-before-change029.exe`
- Create: `build/bin/archive/validation/change-029/Q-Solver-candidate-change029.exe`
- Remove after archive: `build/bin/testing/current/*`

**Interfaces:**
- Consumes: Task 5 已通过候选及测试前稳定版。
- Produces: 根目录唯一最新稳定版、可回滚旧稳定版、Change 029 验证归档和哈希证据。

- [ ] **Step 1: 再次确认无运行中 Q-Solver 进程**

Expected: 没有占用根目录稳定版、候选或测试前稳定版的进程。

- [ ] **Step 2: 归档测试前稳定版**

复制 `Q-Solver-stable-before-test.exe` 为 `archive/superseded/change-029/Q-Solver-before-change029.exe`，生成并验证 `SHA256SUMS`；哈希必须与 Task 1 根目录旧稳定版一致。

- [ ] **Step 3: 使用临时文件晋升候选**

```text
candidate -> build/bin/Q-Solver.promote.tmp
校验 tmp hash == candidate hash
替换 build/bin/Q-Solver.exe
校验 root hash == candidate hash
```

Expected: 替换前旧稳定版已有可验证归档；任一步哈希不一致立即恢复旧稳定版。

- [ ] **Step 4: 归档通过候选并清空 current**

把候选副本归档为 `archive/validation/change-029/Q-Solver-candidate-change029.exe`，生成清单。确认根目录和归档候选哈希相同后，移除 `testing/current/` 的本轮三个文件。

- [ ] **Step 5: 验证最终目录不变量**

```bash
find build/bin -maxdepth 1 -type f -printf '%f\n'
find build/bin/releases -type f -print0 | sort -z | xargs -0 sha256sum
```

Expected: 根目录唯一文件为 `Q-Solver.exe`；Release 哈希与 Task 1 完全一致；所有新 `SHA256SUMS` 验证通过。

### Task 7：完成治理记录与跨 Change 提交边界审计

**Files:**
- Modify: `_dev/changes/029-build-artifact-lifecycle/03-implementation.md`
- Modify: `_dev/changes/029-build-artifact-lifecycle/04-validation.md`
- Modify: `_dev/changes/029-build-artifact-lifecycle/05-rollback.md`
- Modify: `_dev/changes/029-build-artifact-lifecycle/06-notes.md`
- Modify: `_dev/changes/INDEX.md`
- Create: `_dev/audits/024-031-commit-boundary.md`
- Modify: Change 024～031 已有治理记录，仅修正本轮真实提交/验证状态

**Interfaces:**
- Consumes: Tasks 1～6 的真实命令、哈希、验证和目录结果。
- Produces: 两个实施提交的精确白名单、最终验证记录、已知限制和 push 边界。

- [ ] **Step 1: 写入实施和验证事实**

记录迁移数量、去重判定、最终候选路径/大小/SHA-256、每项测试命令与结果、启动验证、Release 保护比较、既有 BLOCKED/NEEDS_USER 项。

- [ ] **Step 2: 写入回滚和总结**

`05-rollback.md` 写明旧稳定版归档位置、哈希验证和恢复步骤；`06-notes.md` 说明本轮三个提交、未发布边界和遗留人工验收。

- [ ] **Step 3: 更新 INDEX 真实状态**

Change 024～031 只按实际验证和提交进度更新；没有新人工证据的条目保持 `NEEDS_USER`/`BLOCKED`。Change 029 标记产物整理和自动验证完成，但不把未完成人工项写成 Close。

- [ ] **Step 4: 完成提交边界审计**

审计文件必须列出：

```text
Commit 2: feat: integrate question bank recovery and provider diagnostics
- 仅业务源码、对应测试、必要 bindings/依赖

Commit 3: chore: archive changes 024-031 and define artifact lifecycle
- README、AGENTS/CLAUDE、_dev 政策、Change、审计和 INDEX
```

同时列出所有排除项：`build/bin/**`、用户配置、CCR 文件、日志、仓库外 helper/probe、secret、Tag/Release 操作。

- [ ] **Step 5: 最终文档和敏感扫描**

Expected: 无 TODO/TBD；文档内部哈希/路径一致；不包含认证实值；本轮新文档 `git diff --check` 通过。

### Task 8：创建累计功能/测试提交

**Files:**
- Stage: `_dev/audits/024-031-commit-boundary.md` 中 Commit 2 白名单列出的业务源码、测试和必要生成/依赖文件
- Exclude: 所有治理文档、二进制、本机文件和仓库外文件

**Interfaces:**
- Consumes: Task 7 已验证功能白名单。
- Produces: 第二个本轮提交（设计提交之后的功能/测试整合提交）。

- [ ] **Step 1: 再做 Git 预检和远端头核对**

```bash
export GIT_OPTIONAL_LOCKS=0
git status --short --branch
git branch --show-current
git remote -v
git diff --stat
git diff --cached --name-status
git ls-remote origin refs/heads/dev/qsolver-custom
```

Expected: 分支正确；暂存区为空；远端头仍是设计提交前基线或预期值。

- [ ] **Step 2: 使用显式白名单暂存功能/测试文件**

不得使用目录级无边界暂存。暂存后检查：

```bash
git diff --cached --name-status
git diff --cached --stat
git diff --cached --check
```

Expected: 只有功能、测试和必要 bindings/依赖；没有 `_dev/`、README、AGENTS/CLAUDE、`build/bin/` 或本机文件。

- [ ] **Step 3: 对暂存内容运行敏感信息扫描和最终测试证据关联**

Expected: 无 secret 模式；暂存内容与 Task 4 已验证内容一致，暂存后未发生业务文件变化。

- [ ] **Step 4: 创建功能/测试提交**

```bash
git commit -m "feat: integrate question bank recovery and provider diagnostics" \
  -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

Expected: 提交成功；治理文件仍未暂存。

### Task 9：创建治理/规则提交

**Files:**
- Stage: `README.md`, `README_EN.md`, `AGENTS.md`, `CLAUDE.md`, `_dev/build-artifact-policy.md`, `_dev/README.md`, `_dev/audits/024-031-commit-boundary.md`, `_dev/changes/INDEX.md`, Change 024～031 档案和 Change 029 全部档案

**Interfaces:**
- Consumes: Task 8 提交结果与完整治理白名单。
- Produces: 第三个本轮提交；本地提交序列完整。

- [ ] **Step 1: 确认功能提交后暂存区为空**

Expected: `git diff --cached --name-status` 无输出；剩余工作树只有治理/README 预期改动。

- [ ] **Step 2: 显式暂存治理白名单并审计**

```bash
git diff --cached --name-status
git diff --cached --stat
git diff --cached --check
```

Expected: 无业务源码、二进制、本机配置或敏感文件；Change 029 七份文档齐全。

- [ ] **Step 3: 创建治理/规则提交**

```bash
git commit -m "chore: archive changes 024-031 and define artifact lifecycle" \
  -m "Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

Expected: 提交成功；总计形成设计、功能/测试、治理/规则三个提交。

### Task 10：验证提交序列并推送指定远端

**Files:**
- No file modifications expected
- Remote write: `origin/dev/qsolver-custom`

**Interfaces:**
- Consumes: Tasks 8～9 的三个本地提交和用户已授予的普通 push 权限。
- Produces: `origin/dev/qsolver-custom` 指向治理提交；不产生 Tag 或 Release。

- [ ] **Step 1: 验证三个提交与工作树**

```bash
git log -3 --oneline --decorate
git show --name-status --format=fuller HEAD~2
git show --name-status --format=fuller HEAD~1
git show --name-status --format=fuller HEAD
git status --short --branch
git diff --cached --name-status
```

Expected: 三个消息和文件边界正确；署名存在；暂存区为空；若仍有未提交文件必须明确归属并阻止 push。

- [ ] **Step 2: 最后核对远端与禁止项**

Expected: push remote 仅为 `origin`，分支仅为 `dev/qsolver-custom`；命令不含 `--force`、Tag、`main`、`upstream` 或 Release。

- [ ] **Step 3: 执行普通 push**

```bash
git push origin dev/qsolver-custom
```

Expected: 远端分支快进到治理提交；不触发 Tag Release 工作流。

- [ ] **Step 4: 读取远端头并报告最终状态**

```bash
git ls-remote origin refs/heads/dev/qsolver-custom
git status --short --branch
```

Expected: 远端头等于本地 HEAD；报告最终 EXE 路径、大小、SHA-256、测试结果、三个提交、遗留 `BLOCKED`/`NEEDS_USER` 和未执行发布动作。
