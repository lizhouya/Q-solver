# Change 029：回滚方案

## 当前回滚资产

旧根目录稳定版以两种形式保留：

1. 原始根目录快照：`build/bin/archive/original-root-pre-change029/Q-Solver.exe`；
2. 规范 superseded 归档：`build/bin/archive/superseded/change-029/Q-Solver-before-change029.exe`。

两者 SHA-256 均为：

```text
5DCABB17933BFA9AB4DC15F289B4B8E50AF99879C316B58831B09BBB4E39AA11
```

当前最终版 SHA-256：

```text
82ABBC4B5C88CA480A7CE65E89BC78FC0ACD3202D597266F259113742BD87F46
```

## 最终版回滚步骤

1. 确认没有运行中的 Q-Solver 进程。
2. 在 `archive/superseded/change-029/` 运行 `sha256sum -c SHA256SUMS`。
3. 把当前根目录最终版移动到新的失败/回滚归档，不直接删除。
4. 将 `Q-Solver-before-change029.exe` 复制为根目录临时文件。
5. 验证临时文件 SHA-256 为旧稳定版哈希。
6. 将临时文件重命名为 `build/bin/Q-Solver.exe`。
7. 启动验证旧稳定版，并在本 Change 记录回滚原因与结果。

清单缺失、哈希不一致或文件来源不明确时停止，不执行猜测性恢复。

## 布局回滚

`archive/original-root-pre-change029/` 保存实施前根目录所有 18 个文件的原件；`archive/original-validation-flat-pre-change029/` 保存原 validation 平铺文件原件；仓库外清单记录全部 44 个实施前文件。需要恢复旧布局时按清单逐项复制，不覆盖同名不同哈希文件。

## Git 回滚

本轮不重写历史，不使用 `git reset --hard`、force push 或自动 amend。提交后如发现问题，保留证据并根据用户授权使用新的 revert/fix 提交；不得影响 `main`、`upstream`、Tag 或 Release。
