# 总结与注意事项 06

## 变更摘要

Change 035 将快捷键 action 从 Windows 低级 Hook 的同步业务回调迁移到 `shortcut.Service` 单 worker FIFO，解除截图捕获和压缩对 Hook 线程的阻塞，同时保持连续截图、删除和发送动作顺序。ScreenshotDock 恢复继续截图、已满、删除和发送提示，并响应实际快捷键配置。

用户已确认候选 GUI 验收通过；候选已晋升为根稳定版，晋升后独立启动、WebView2、无 Node/Vite、正常退出和无残留进程验证通过。

## 经验与踩坑

| 现象/坑 | 原因 | 结论/建议 |
|---|---|---|
| 首张截图后快捷键偶发失效 | 低级 Hook 回调同步执行截图与压缩，存在超时移除风险 | Hook 回调只做轻量识别和内存入队 |
| 简单改成 goroutine 可能修复阻塞却引入乱序 | 截图和发送存在严格先后关系 | 使用单 worker FIFO，而不是每次触发一个 goroutine |
| 队列锁若覆盖 handler 会继续阻塞入队 | 截图是耗时动作 | handler 必须在队列锁外执行 |
| Dock 快捷键提示可能停留默认值 | setup 时一次性读取 Pinia computed | 使用 `storeToRefs` 保持响应性 |
| race 测试未进入测试逻辑 | Windows 环境启动错误 `0xc0000139` | 保持 BLOCKED，不能用普通测试替代 race PASS |

## 最终交付

- 稳定版：`build/bin/Q-Solver.exe`
- 大小：44,029,440 字节
- SHA-256：`29F0100EA970A84DE3C53FF8E51626C6E4E1E85C1DC6FA4FF9AB12B18DB380FE`
- 通过候选归档：`build/bin/archive/validation/change-035/`
- 旧稳定版归档：`build/bin/archive/superseded/change-035/`
- `testing/current/`：已清空
- `releases/`：未修改

## 对本项目后续的影响

- 快捷键业务动作统一脱离底层 Manager/Hook 回调线程执行。
- action 保持 FIFO；后续新增快捷键无需自行创建 goroutine。
- `Service.Stop` 先停止 Manager，再停止调度器；停止后不接受新动作。
- 截图上限、模型请求、题库和配置格式保持不变。
- 截图 Dock 展示截图、删除、发送三类实际快捷键；达到 3 张时明确显示已满。

## 提交结果

- 功能与测试提交：`e23f86a fix: restore reliable multi-screenshot shortcuts`
- 治理提交：本总结、审计、INDEX 与其余 Change 035 档案随 `chore: add change 035 governance records` 落地；按规则不在文档中记录自身提交哈希。
- 推送目标：仅 `origin/dev/qsolver-custom`；不涉及 `main` 或 `upstream`。

## 遗留项与 TODO

- race 环境错误 `0xc0000139` 继续作为已知限制保留。
- Tag、Release 未授权且未执行。

## 参考资源

- `_dev/audits/035-screenshot-capture-recovery.md`
- `pkg/shortcut/dispatcher.go`
- `pkg/shortcut/service.go`
- `frontend/src/components/ScreenshotDock.vue`
- `_dev/changes/034-question-bank-hard-delete/04-validation.md`
