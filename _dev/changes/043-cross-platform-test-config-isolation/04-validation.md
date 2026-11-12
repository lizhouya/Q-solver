# 验证记录 04

## 当前状态

Change 043 helper、所有 APPDATA-only 调用点迁移和本地普通全量门禁已完成。race 仍为环境 `BLOCKED`；rc.5 远程三平台验证尚未执行。

## 远程红灯基线

| 检查项 | 结果 | 证据 |
|---|---|---|
| Build Windows | PASS | run `43212159818` |
| Build macOS (Apple Silicon) / Build App | PASS | 应用构建成功 |
| Build macOS (Apple Silicon) / Regression Tests | FAIL | 两个 app 测试失败 |
| Build macOS (Intel) | CANCELLED | 无通过证据 |
| Create Release | SKIPPED | rc.4 Release 不存在 |

## TDD 与本地验证

| 命令 | 结果 | 备注 |
|---|---|---|
| 配置目录隔离合同红灯 | EXPECTED FAIL | `undefined: isolateUserConfigDirs`，失败只指向缺失 helper |
| 配置目录隔离合同转绿 | PASS | `os.UserConfigDir()` 位于 helper 临时根内 |
| 两个 rc.4 失败测试聚焦 | PASS | keyless local template 与原图 retry 均通过；不调用真实 API |
| `go test ./app -count=1` | PASS | app 全包通过 |
| `go test ./... -count=1` | PASS | 全仓 Go 测试通过 |
| `go vet ./...` | PASS | 无输出 |
| `go build ./...` | PASS | 退出码 0 |
| `go mod verify` | PASS | `all modules verified` |
| 19 个前端回归 + Mobile DOM | PASS | 全部通过 |
| `npm --prefix frontend run build` | PASS | Vite 4.5.14，135 modules；仅既有大 chunk warning |
| `go test -race ./app -count=1` | BLOCKED | 启动阶段 `exit status 0xc0000139` |
| VS Code diagnostics | PASS | Warning 及以上为空；此前未改动文件的瞬时 SA4000 未再出现，`go vet` 始终 PASS |
| `git diff --check` | PASS | 无空白错误；无 frontend/dist 或生产源码差异 |
| rc.5 文档、链接与 Tag 占用 | PASS | body/相对目标存在；README 入口为 rc.5；本地/远程 Tag 未占用 |
| 敏感信息扫描 | PASS | 未发现凭据、Token 实值或本机路径 |

## rc.5 远程验证

| 检查项 | run `43232814169` | run `43246684317` | 结论 |
|---|---|---|---|
| Build Windows | PASS | PASS | Windows 构建、回归和 artifact 上传稳定通过 |
| Build macOS (Apple Silicon) | CANCELLED | CANCELLED | 均未分配 runner，`runner_id=0`、steps 为空 |
| Build macOS (Intel) | CANCELLED | CANCELLED | 均未分配 runner，`runner_id=0`、steps 为空 |
| Create Release | SKIPPED | SKIPPED | 依赖 macOS job 未成功 |
| rc.5 Release | NOT CREATED | NOT CREATED | 无附件或 `SHA256SUMS` |

跨平台测试修复尚未获得 macOS runner 执行机会；不能将取消状态记为 PASS 或测试 FAIL。

## 验收标准对照

- [x] 跨平台配置目录 helper 合同通过。
- [x] 所有 APPDATA-only 测试完成迁移。
- [x] 两个 rc.4 失败测试和 app 全包通过。
- [x] 普通全量本地门禁通过；race BLOCKED 已披露。
- [ ] 后续 rc.5 三平台和 Create Release 通过：**未满足**，macOS runner 未分配，两次 Create Release 均跳过。

## 结论

状态：**实现完成 / 本地普通门禁 PASS / Windows CI PASS / macOS CI BLOCKED（无 runner）/ race BLOCKED / rc.5 Release 未创建 / 不关闭**。需要用户决定继续等待三平台或另行批准 Windows-only 预发布。
