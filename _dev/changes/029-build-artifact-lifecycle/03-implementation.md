# Change 029：实施记录

## 执行概况

- 2026-10-29：用户复核书面设计并要求继续完成全部提交。
- 使用 `writing-plans` 形成 `02-plan.md`，采用当前会话内联执行。
- Git 实施前预检确认分支为 `dev/qsolver-custom`、设计提交 `8047488` 为 HEAD、暂存区为空、远端仍为实施前基线。
- 仓库外审计目录记录 44 个迁移前构建产物、7 个 Release 文件、5 个关键保护文件和完整 Git index；不进入 Git。

## 规则落地

- 新建 `_dev/build-artifact-policy.md`。
- `AGENTS.md` 新增所有 Agent 必须遵守的根目录唯一 EXE、通过前不晋升、SHA-256、失败处理和 Release 独立授权底线。
- `CLAUDE.md` 新增测试区、Wails runtime、保护构建、启动验证、晋升、回滚和 Git 边界步骤。
- `_dev/README.md` 增加政策入口。

## 历史产物整理

原批量移动因自动权限策略将“移动并去重”识别为本地删除而被拒绝。实施改为零删除、完全可逆的两阶段方案：

1. 按 32 个 dry-run 映射将规范归档目标全部复制并校验。
2. 根目录同名同哈希文件不删除，统一移动到 `archive/original-root-pre-change029/` 保存原始根目录快照。
3. 既有平铺 validation 文件移动到 `archive/original-validation-flat-pre-change029/`；规范副本按 `validation/change-NNN/` 或 `validation/legacy/` 保存。
4. Change 017 同名不同哈希产物均保留；根目录版本使用 `-root-pre-change029` 后缀。
5. `Q-Solver-dev.exe` 和 `.exe~` 规范副本进入 `archive/transient/`。
6. 原平铺 `SHA256SUMS` 字节级原件保存为 `archive/manifests/validation-flat-pre-change029-SHA256SUMS`。
7. 迁移前 44 个文件均能映射到保留原件且路径、大小、SHA-256 一致。

最终结构：

- `build/bin/` 根目录只有 `Q-Solver.exe`。
- `build/bin/testing/current/` 已清空。
- 历史验证产物按 Change 目录归档。
- 被替代稳定版位于 `archive/superseded/change-029/`。
- 原始根目录和原 validation 平铺结构各有完整可回滚快照。
- `build/bin/releases/` 未修改。

## Wails 生成噪音处理

- `frontend/wailsjs/go/app/App.js`、`App.d.ts`、`models.ts` 中新增方法和 DTO 属于业务实现并保留。
- `models.ts` 的 85 处行尾制表符已清除。
- 本机 Wails CLI v2.16.0 注入的 Notification runtime API 没有项目调用，已精确排除。
- 候选构建后 CLI 再次生成该噪音；通过构建前后哈希定位并清除，最终 `frontend/wailsjs` 恢复到候选构建前哈希。
- 实际候选通过 `go version -m` 确认链接 `github.com/wailsapp/wails/v2 v2.11.0`。

## 最终候选与晋升

- 测试区稳定版哈希：`5DCABB17933BFA9AB4DC15F289B4B8E50AF99879C316B58831B09BBB4E39AA11`。
- 候选路径：`build/bin/testing/current/Q-Solver-candidate-change029.exe`。
- 候选大小：42,331,136 字节。
- 候选 SHA-256：`82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46`。
- 候选自动验证通过后，原根目录稳定版移动到原始快照并另存 superseded 规范归档；候选复制为根目录 `Q-Solver.exe`。
- 晋升后根目录哈希与候选一致，并再次完成独立启动和正常退出验证。
