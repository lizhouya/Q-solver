# Change 029：构建产物生命周期与整合提交设计

## 1. 目标与非目标

### 目标

建立一个单一、可重复的构建产物生命周期：用户只需在 `build/bin/` 根目录识别 `Q-Solver.exe`；测试期间可同时保留测试前稳定版和最新候选；验证结束后按结果晋升或归档；所有关键移动均由 SHA-256 和记录支撑。

同时，在不改写历史、不伪造独立 Change 边界的前提下，将当前 Change 024～031 的累计实现整理为经批准的两个整合实施提交并推送到指定分支。

### 非目标

- 不把 EXE 纳入 Git 追踪。
- 不创建安装器、自动更新器、Tag、GitHub Release 或新 RC。
- 不清理 `build/bin/releases/` 中既有发布资产。
- 不修改模型、Provider、CCR 端口、协议或用户配置。
- 不解决既有 Windows race 环境错误，也不把 `NEEDS_USER` 项改写为已通过。
- 不将本机 CCR 配置、数据库、日志、Q-Solver 用户配置或仓库外探针复制到仓库。

## 2. 目录职责

实施后的稳定结构为：

```text
build/bin/
├── Q-Solver.exe
├── testing/
│   └── current/
│       ├── Q-Solver-stable-before-test.exe
│       ├── Q-Solver-candidate-changeNNN.exe
│       └── SHA256SUMS
├── archive/
│   ├── validation/
│   │   └── change-NNN/
│   │       ├── <验证产物>.exe
│   │       └── SHA256SUMS
│   ├── superseded/
│   │   └── change-NNN/
│   │       ├── <旧稳定产物>.exe
│   │       └── SHA256SUMS
│   ├── transient/
│   │   └── <开发版或临时备份>
│   └── failed/
│       └── change-NNN/
│           ├── <失败候选>.exe
│           └── SHA256SUMS
└── releases/
    └── <既有发布资产，保持不变>
```

职责约束：

- 根目录唯一 EXE `Q-Solver.exe` 是最近一次完整验证通过并正式晋升的稳定版。
- `testing/current/` 只服务当前一轮测试。测试开始前放入稳定版副本，候选构建放入同一路径，二者通过固定命名区分。
- `archive/validation/change-NNN/` 保存通过验证的 Change 候选或历史验证构建。
- `archive/superseded/change-NNN/` 保存被本轮候选替代的旧根目录稳定版。
- `archive/failed/change-NNN/` 仅保存有诊断价值的失败候选；失败不触碰根目录稳定版。
- `archive/transient/` 保存 `Q-Solver-dev.exe`、`.exe~` 等不应被识别为稳定版的历史开发或临时产物。
- `releases/` 不参与本 Change 的迁移、覆盖或删除。

## 3. 首次整理算法

首次整理按“盘点—哈希—迁移—核对”执行：

1. 生成迁移前清单，记录每个现有文件的相对路径、大小和 SHA-256。
2. 对根目录历史 Change 验证 EXE，按文件名中的 Change 编号迁移到对应 `archive/validation/change-NNN/`。
3. 若目标目录已有同哈希文件，仅在确认字节级一致后去除根目录重复副本；若同名不同哈希，则两者均保留并使用不会覆盖的名称。
4. 将旧根目录 `Q-Solver.exe` 作为本轮测试前稳定版复制到 `testing/current/Q-Solver-stable-before-test.exe`；在候选通过晋升时，再将该副本归档到 `archive/superseded/change-029/`。
5. 将 `Q-Solver-dev.exe` 和 `.exe~` 迁移到 `archive/transient/`，不得把它们晋升为稳定版。
6. 保持 `releases/` 内容和哈希不变。
7. 为每个新归档目录生成 `SHA256SUMS`，并用迁移前清单复核：除已确认的字节级重复副本外，文件总集合无丢失。

任何覆盖、删除重复副本或晋升操作都必须先完成源与目标哈希比较。出现冲突时停止该文件的迁移，不通过自动重命名掩盖异常。

## 4. 后续候选生命周期

### 4.1 准备

1. 确认根目录仅有稳定版 `Q-Solver.exe`。
2. 清空上一轮已经归档完毕的 `testing/current/` 内容。
3. 将根目录稳定版复制为 `Q-Solver-stable-before-test.exe`，记录哈希。
4. 使用项目固定的 Wails v2.11.0 和保护参数构建 `Q-Solver-candidate-changeNNN.exe`，避免依赖同步、bindings 重生成和本机 CLI v2.16.0 噪音。
5. 生成 `testing/current/SHA256SUMS`，验证两个 EXE 均可读取且哈希稳定。

### 4.2 验证中

- 所有自动和人工验证只针对候选的明确路径。
- 根目录 `Q-Solver.exe` 保持不变，因此候选失败不会影响可用稳定版。
- 若验证需要重新构建，旧候选先按失败/被替代状态归档或移出 `current/`，`current/` 始终只有一个最新候选。
- 自动测试结论必须区分 `PASS`、`FAIL`、`BLOCKED` 和 `NEEDS_USER`；不得用未执行或环境阻塞冒充通过。

### 4.3 通过与晋升

候选满足既定自动验证，并且本轮要求的人工门禁已明确处理后：

1. 再次核对候选哈希与测试记录一致。
2. 把测试前稳定版归档到 `archive/superseded/change-NNN/`。
3. 把候选复制到根目录临时文件，核对哈希后以同卷原子重命名替换 `Q-Solver.exe`。
4. 将通过验证的候选副本归档到 `archive/validation/change-NNN/`。
5. 更新相应 `SHA256SUMS` 和 Change 验证记录。
6. 删除或迁移 `testing/current/` 内本轮文件，使其回到空闲状态。
7. 最终确认根目录只有 `Q-Solver.exe`，且其哈希等于已通过候选。

### 4.4 失败、阻塞与中止

- `FAIL`：根目录稳定版不变；候选按诊断价值归档到 `archive/failed/change-NNN/` 或在记录哈希后移除；测试前稳定版副本仅作为核对材料，结束后移除。
- `BLOCKED`：不晋升。保留 `testing/current/`，直到阻塞解除或用户明确中止；记录阻塞原因。
- `NEEDS_USER`：是否构成晋升门禁由该 Change 事前验收标准决定，不能由执行者自行降级。未获得必要确认时不晋升。
- 中途异常：根目录稳定版始终不应被候选覆盖；若异常发生在原子替换窗口，使用已归档的测试前稳定版和哈希恢复。

## 5. 哈希、清单与回滚

`SHA256SUMS` 使用相对文件名和大写 SHA-256。清单覆盖目录内所有 EXE，但不记录密钥、URL 查询参数、环境变量或用户配置。

回滚顺序：

1. 停止正在运行的 Q-Solver 实例，避免 Windows 文件占用。
2. 从对应 `archive/superseded/change-NNN/` 选择旧稳定版。
3. 校验旧稳定版与其 `SHA256SUMS` 一致。
4. 复制到根目录临时文件，校验后原子替换 `Q-Solver.exe`。
5. 将被回滚的新稳定版移入对应归档目录，不直接删除。
6. 在 Change 记录中写明回滚原因、前后哈希和结果。

如果清单缺失或哈希不一致，停止回滚并人工审计，不猜测哪个文件正确。

## 6. 规则与文档落点

实施阶段修改：

- `AGENTS.md`：加入必须遵守的产物生命周期底线、根目录唯一 EXE、验证通过前不得晋升、哈希及 Release 边界。
- `CLAUDE.md`：加入 Claude Code 的具体执行步骤、固定 Wails 版本、显式构建路径、验证/晋升/失败处理和 Git 安全要求。
- `_dev/build-artifact-policy.md`：保存完整操作规范、目录结构、命名、状态机、回滚和故障处理。
- `_dev/README.md`：增加政策入口。
- Change 029 的 `02-plan.md`～`06-notes.md` 与 `_dev/changes/INDEX.md`：记录计划、实施、验证、回滚和最终状态。

被忽略的 `build/bin/` 二进制和哈希清单保存在本地文件系统；Git 中的规则与 Change 文档记录已验证最终 EXE 的路径、大小和 SHA-256，以补足可审计性。

## 7. 验证设计

晋升前至少执行并记录：

1. 聚焦于 Change 024～031 涉及功能的 Go 和前端回归测试；
2. `go test ./... -count=1`；
3. `go vet ./...`；
4. `go build ./...`；
5. 全部 14 份前端回归脚本；
6. Vite production build；
7. 使用 Wails v2.11.0 的独立候选构建；
8. 独立启动、WebView2 初始化、无 Node/Vite 子进程、正常退出和无残留进程验证；
9. 构建前后保护文件哈希、依赖和 bindings 差异检查；
10. 范围内 `git diff --check`，并把既有基线问题与新增问题分开；
11. staged/未跟踪文件的敏感信息和越界文件扫描；
12. 迁移前后文件集合、根目录唯一 EXE、`releases/` 不变及所有 `SHA256SUMS` 复核。

既有 Windows race `0xc0000139` 若仍存在，记录为环境 `BLOCKED`，不篡改结果。Change 031 的 GUI 真实导出若没有新的人工结果，继续保持 `NEEDS_USER`；最终晋升只依据事前确认的发布门禁，不把它静默改成 PASS。

## 8. Git 提交与推送边界

本轮采用三个提交：

1. 设计提交：仅包含 Change 029 事前设计文件和 INDEX 中 Change 029 的独立登记。
2. 功能/测试整合提交：Change 024～031 的业务源码、测试和必要 bindings/依赖改动，提交消息为 `feat: integrate question bank recovery and provider diagnostics`。
3. 治理/规则整合提交：Change 档案、审计、README、产物政策与规则，提交消息为 `chore: archive changes 024-031 and define artifact lifecycle`。

每个提交消息末尾必须是：

```text
Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

提交前使用显式文件清单和 `git diff --cached --name-status`、`--stat`、内容审计，不使用无边界的 `git add .`。二进制、用户配置、CCR 文件、日志、仓库外辅助程序和任何 secret 一律排除。

完成三个提交并确认工作树符合预期后，只执行普通 push 到 `origin/dev/qsolver-custom`。禁止推送 `upstream`、修改 `main`、force push、创建或推送 Tag、创建 GitHub Release 或发布新 RC。

## 9. 实施顺序与停止条件

1. 先提交本设计并由用户复核。
2. 用户确认书面设计后，使用 `writing-plans` 生成详细实施计划。
3. 按计划整理二进制和规则，执行完整验证；任何关键验证失败都停止晋升。
4. 候选通过后才替换根目录最终版。
5. 完成跨 Change 提交边界审计后创建两个实施提交。
6. 最终核对本地提交和远端目标，再执行授权 push。

以下任一情况必须停止而不是猜测：源/目标哈希冲突、根目录稳定版缺失、候选无法独立运行、测试出现新增失败、发现敏感文件进入暂存区、提交边界无法可靠区分，或远端分支在 push 前发生未预期变化。
