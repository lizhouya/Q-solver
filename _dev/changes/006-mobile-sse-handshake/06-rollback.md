# 回滚方案 — 06 · 006-mobile-sse-handshake

> **状态**：实施前预案

## 回退范围

- `pkg/mobile/bridge.go`
- `pkg/mobile/server.go`
- `pkg/mobile/bridge_server_test.go`

## 回退步骤

1. 恢复 `Subscribe()` 原单返回值 API。
2. 恢复 Server 的 `GetState()`、Flush、Subscribe 顺序。
3. 移除本 Change 新增的时序测试与测试辅助类型。
4. 重新执行 Mobile 测试、全量 Go 测试/构建、前端构建和 diff-check。

## 数据与兼容性

无配置、持久化、数据库或协议变化；回滚无需迁移数据，也不影响已保存设置。

## 安全限制

只做文件级精确回退，不使用 `git reset --hard` 等破坏性操作，不覆盖其他未提交工作。