# Q-Solver 构建产物生命周期政策

## 目的

本政策管理被 Git 忽略的 `build/bin/` 构建产物。目标是让用户能唯一识别当前稳定版，让每次候选测试具备可回滚基线，并以 SHA-256 和 Change 档案保留可审计证据。

本政策服从 [AGENTS.md](../AGENTS.md) 和 [CLAUDE.md](../CLAUDE.md)。二进制归档不替代源码测试、Change 验证或 Git 提交边界审计。

## 稳定目录结构

```text
build/bin/
├── Q-Solver.exe
├── testing/
│   └── current/
│       ├── Q-Solver-stable-before-test.exe
│       ├── Q-Solver-candidate-changeNNN.exe
│       └── SHA256SUMS
├── archive/
│   ├── validation/change-NNN/
│   ├── superseded/change-NNN/
│   ├── failed/change-NNN/
│   ├── transient/
│   └── manifests/
└── releases/
```

- 根目录唯一 EXE `Q-Solver.exe` 是最近一次通过既定门禁并完成晋升的稳定版。
- `testing/current/` 只保存当前轮测试前稳定版、一个最新候选和哈希清单；空闲时不保留上一轮文件。
- `archive/validation/change-NNN/` 保存通过验证的候选或历史验证构建。
- `archive/superseded/change-NNN/` 保存晋升时被替代的旧稳定版。
- `archive/failed/change-NNN/` 仅保存有诊断价值的失败候选。
- `archive/transient/` 保存开发版、`.exe~` 和其他不能被识别为稳定版的临时产物。
- `archive/manifests/` 保存历史迁移清单。
- `releases/` 是独立发布资产；没有 Release 授权不得迁移、覆盖、删除或新增其内容。

## 命名规则

| 用途 | 名称 |
|---|---|
| 当前稳定版 | `Q-Solver.exe` |
| 测试前稳定版 | `Q-Solver-stable-before-test.exe` |
| 当前候选 | `Q-Solver-candidate-changeNNN.exe` |
| 历史验证归档 | 保留原验证文件名，放入 `archive/validation/change-NNN/` |
| 被替代稳定版 | 使用能表明对应 Change 的名称，放入 `archive/superseded/change-NNN/` |
| 目录哈希 | `SHA256SUMS` |

同一目录不得存在两个“当前候选”。文件名含义与真实状态不一致时，以验证记录为准并立即纠正归档位置，不能仅重命名伪造通过状态。

## SHA-256 清单

每个含 EXE 的当前测试或归档目录维护 `SHA256SUMS`，格式为：

```text
<64 位大写 SHA-256> *<相对文件名>
```

规则：

1. 移动、去重、晋升、回滚前先计算源和目标哈希。
2. 同名同哈希可以去除重复副本；同名不同哈希必须停止自动处理并人工审计。
3. 清单只记录相对文件名和哈希，不记录 Token、API Key、环境变量、认证 URL 或用户配置。
4. Change 的验证记录同步写明最终 EXE 的路径、大小和 SHA-256；二进制继续由 Git 忽略。

## 生命周期

### 准备

1. 确认当前分支、Change、验收标准和 Git 授权范围。
2. 确认根目录只有稳定版 `Q-Solver.exe`。
3. 清空上一轮已归档完毕的 `testing/current/`。
4. 复制根目录稳定版为 `Q-Solver-stable-before-test.exe` 并记录哈希。
5. 使用项目固定 Wails v2.11.0 和保护参数构建唯一候选。
6. 生成并验证 `testing/current/SHA256SUMS`。

### 验证

- 自动与人工验证均使用候选的完整路径。
- 根目录稳定版在候选通过前保持不变。
- 结论使用 `PASS`、`FAIL`、`BLOCKED`、`NEEDS_USER` 或 `NOT_RUN`，不得把历史结果、阻塞或未执行写成 PASS。
- 重新构建前先归档或移出旧候选，`current/` 始终只有一个最新候选。

### 通过与晋升

1. 确认所有必要门禁已满足，候选哈希与验证记录一致。
2. 将测试前稳定版归档到 `archive/superseded/change-NNN/` 并验证清单。
3. 将候选复制到根目录临时文件，验证哈希后以同卷替换方式更新 `Q-Solver.exe`。
4. 再次确认根目录哈希等于候选哈希。
5. 将候选副本归档到 `archive/validation/change-NNN/`。
6. 验证归档清单后清空 `testing/current/`。
7. 确认根目录只有 `Q-Solver.exe`，并复核 `releases/` 未改变。

### 失败、阻塞与中止

- `FAIL`：不改根目录稳定版；有诊断价值的候选归档到 `archive/failed/change-NNN/`，否则在记录哈希后移除。
- `BLOCKED`：不晋升；保留 current 供解除阻塞后继续，或由用户明确中止。
- `NEEDS_USER`：是否阻止晋升由该 Change 的事前验收标准决定，执行者不得自行降级。
- 中途异常：优先保持或恢复测试前稳定版，不以清理目录为由删除唯一可回滚副本。

## Wails 独立候选构建

项目 Go 模块固定 Wails v2.11.0。候选构建首选同版本 CLI：

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build \
  -s \
  -o Q-Solver-candidate-changeNNN.exe \
  -skipbindings \
  -skipembedcreate \
  -nosyncgomod \
  -m \
  -nocolour \
  -v 1 \
  -ldflags "-s -w" \
  -tags prod
```

构建前后核对 `go.mod`、`go.sum`、`frontend/wailsjs/**`、Git index 和受保护文件哈希。不得把本机 Wails v2.16.0 自动生成或依赖同步噪音混入提交。

若同版本 CLI 因权限或环境不可执行，可以使用已安装的较新 CLI 作为编排器，但必须同时满足：启用全部保护参数；`go version -m` 证明候选链接的 Wails runtime 仍为 v2.11.0；构建后依赖、Git index 与 `frontend/dist` 不变；bindings/runtime 恢复到构建前哈希；再次运行相关回归与独立启动验证。任一条件不满足都不得晋升。

## 回滚

1. 停止占用目标 EXE 的 Q-Solver 进程。
2. 选择对应 `archive/superseded/change-NNN/` 中的旧稳定版。
3. 使用该目录 `SHA256SUMS` 验证旧稳定版。
4. 复制到根目录临时文件，验证后替换 `Q-Solver.exe`。
5. 将被回滚的新稳定版移入对应归档，不直接删除。
6. 在 Change 记录中写明原因、前后哈希、命令结果和遗留状态。

清单缺失、哈希不一致或来源不明确时停止回滚，不猜测文件身份。

## 首次整理与去重

首次从平铺目录迁移时，先在仓库外生成包含相对路径、大小和 SHA-256 的完整清单。历史 `Q-Solver-changeNNN-*.exe` 按编号迁移；无编号验证产物进入 `validation/legacy/`；开发版和临时备份进入 `transient/`。只有字节级同哈希副本可以去重，迁移后必须证明 Release 哈希不变且除已确认重复项外文件集合无丢失。

## Git 与 Release 边界

- `build/bin/**` 保持 Git ignored，不使用 `git add -f` 纳入二进制或哈希清单。
- Git 中仅提交规则、Change 文档、验证哈希摘要和提交边界审计。
- 提交前使用显式文件清单，禁止 `git add .`、`git add -A`。
- commit、push、Tag 和 Release 分别需要明确授权。
- 普通开发提交不得修改 `main`、推送 `upstream`、force push 或触发 Tag Release 工作流。
- 用户配置、CCR 配置/数据库/日志、仓库外探针和任何 secret 不得进入归档政策文件或 Git。

## 每轮检查清单

- [ ] Change 事前档案和验收标准完整。
- [ ] 根目录稳定版哈希已记录。
- [ ] `testing/current/` 只有稳定版副本、一个候选和清单。
- [ ] 相关测试、全量测试、Vite 和独立 Wails 构建结果已记录。
- [ ] 独立启动、WebView2、退出和残留进程已检查。
- [ ] 构建前后依赖、bindings 和 Git index 无非预期变化。
- [ ] 晋升前候选哈希再次核对。
- [ ] 旧稳定版和验证候选均已归档且清单通过。
- [ ] 根目录最终只有 `Q-Solver.exe`。
- [ ] `releases/` 哈希不变，除非另有明确发布授权。
- [ ] Git 暂存区不含二进制、本机配置、日志或 secret。
