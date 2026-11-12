# 实现过程记录 03

## 当前状态

rc.4 CI 根因已定位，Change 043 已完成 helper、调用点迁移、本地普通全量验证和功能/测试提交 `532c828`。生产代码零改动；race 仍受 Windows `0xc0000139` 阻塞。rc.5 发布文档已准备，待治理提交、分支与 Tag 推送和三平台验证。

## 实现日志

| 时间 | 步骤 | 改动内容 | 涉及文件 | 说明 / 决策原因 |
|---|---|---|---|---|
| 2026-11-12 | CI 取证 | 读取 run/job/失败日志并确认 Release 未创建 | GitHub Actions | Windows PASS；macOS arm64 app tests FAIL |
| 2026-11-12 | 根因审计 | 对比 `os.UserConfigDir()` 平台行为与 app 测试环境变量 | app tests / config manager | APPDATA-only 导致 macOS 真实 HOME 污染 |
| 2026-11-12 | TDD 红灯 | 新增配置目录隔离合同，调用尚不存在 helper | `app/test_config_env_test.go` | 按预期仅因 `undefined: isolateUserConfigDirs` 失败 |
| 2026-11-12 | 最小 helper | 五个跨平台目录环境变量指向同一 `t.TempDir()` | `app/test_config_env_test.go` | 合同测试转绿 |
| 2026-11-12 | 调用点迁移 | 解题、Mobile、题库和 Settings 测试统一使用 helper | 5 个既有 app 测试文件 | APPDATA-only 调用点清零 |
| 2026-11-12 | 聚焦回归 | 合同测试和两个 rc.4 失败测试通过 | app tests | 不调用真实 API |
| 2026-11-12 | 本地全量门禁 | Go 全量/vet/build/mod、19 个前端回归、Mobile DOM、Vite 通过 | 全仓相关范围 | 仅既有 Vite chunk warning |
| 2026-11-12 | 已知阻塞 | app race 启动返回 `0xc0000139`；diagnostics 报既有 SA4000 | Windows / `solve_attempt_test.go` | 均非 Change 043 差异，不冒充 PASS |
| 2026-11-12 | 功能/测试提交 | 创建 `532c828 fix: isolate test config directories across platforms` | 6 个 app 测试文件 | 治理文件未混入 |
| 2026-11-12 | rc.5 准备 | 新增 rc.5 正文，README/README_EN/CHANGELOG 指向前进修复 | 发布文档 | rc.4 Tag 与失败 run 保持不可变 |
| 2026-11-12 | rc.5 文档验证 | 核对相对链接、敏感模式、diagnostics、diff 和 Tag 占用 | 发布/治理文档 | 全部通过；本地/远程 rc.5 Tag 未占用 |

## 与计划的偏差

暂无。

## 关键决策纪要

- 修复所有同类 APPDATA-only 测试，不只修两个表面失败点。
- 直接验证 `os.UserConfigDir()` 位于隔离根，不依赖平台名猜测。
- rc.4 Tag 保持不可变，使用 rc.5。

## 遗留未完成项

- 治理/rc.5 发布准备提交。
- 推送开发分支并创建不可变 rc.5 Tag。
- rc.5 三平台构建、回归、Create Release、附件与校验和核对。
- macOS GUI 仍无人工验收；发布说明继续披露。
