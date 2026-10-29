# Change 029：当前状态

## 仓库与授权状态

- 分支：`dev/qsolver-custom`。
- 设计前 HEAD：`43022e10cd8fb1b38b72b56d95bc39b8bb4d201d`。
- `origin/dev/qsolver-custom` 在设计前与 HEAD 一致。
- 当前工作树包含 Change 024～031 的累计未提交源码、测试、bindings、文档和审计记录；设计前暂存区为空。
- `build/bin/` 与 `*.exe` 均被 `.gitignore` 忽略，因此二进制移动不会直接进入 Git 历史。
- 本轮 Git 授权仅覆盖本 Change 设计提交、批准后的两个整合实施提交，以及向 `origin/dev/qsolver-custom` 的普通 push。

## 构建产物现状

设计前 `build/bin/` 根目录存在 18 个 EXE 或备份产物，包括：

- Change 017～028、030、031 的验证构建；
- `Q-Solver-dev.exe`；
- 旧 `Q-Solver.exe`；
- 一个 `.exe~` 临时备份。

此外：

- `archive/` 已有 19 个文件，约 374 MB；
- `releases/` 已有 7 个文件，约 70 MB；
- 根目录至少 7 个验证 EXE 与既有 `archive/validation` 内容字节级重复；
- 根目录旧 `Q-Solver.exe` SHA-256 为 `5DCABB17933BFA9AB4DC15F289B4B8E50AF99879C316B58831B09BBB4E39AA11`；
- 当前最新候选 Change 031 SHA-256 为 `82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46`。

根目录同时承担“最终稳定版、历史验证版、开发版和临时备份”四种职责，无法仅凭目录位置判断哪个文件应由用户双击运行。

## 已知验证基础

- Change 030 已证明当前工作树可以生成不依赖 Vite/Wails DevServer 的独立 EXE。
- Change 031 已完成聚焦及全量 Go 测试、`go build`、`go vet`、14 份前端回归、Vite 构建和独立 Wails 构建；Windows GUI 真实导出仍标记为 `NEEDS_USER`。
- Windows race 测试仍受既有 `0xc0000139` 环境错误阻塞，不能伪报通过。
- 全树 `git diff --check` 存在既有 `frontend/wailsjs/go/models.ts` 尾随空白基线问题；最终审计必须区分既有基线和本轮新增问题。

## 风险

1. 直接覆盖根目录 `Q-Solver.exe` 会丢失可回滚的测试前稳定版。
2. 在哈希确认前删除重复文件，可能误删同名但内容不同的产物。
3. 使用本机 Wails CLI v2.16.0 默认流程可能同步依赖或重新生成 bindings，扩大 Change 024～031 的未提交差异。
4. 将整个 `_dev/changes/INDEX.md` 暂存会把其他 Change 的未提交登记混入设计提交。
5. 批量 `git add .` 可能误纳本机配置、日志或与本轮无关的文件；必须使用显式清单和提交前审计。
6. 二进制被 Git 忽略，Git 提交本身不能证明根目录 EXE 的内容；必须依赖 SHA-256 清单和验证记录。
