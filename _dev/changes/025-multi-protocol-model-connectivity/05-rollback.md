# Change 025 回滚预案

日期：2026-10-25
状态：预案已建立，未执行回滚。

## 回滚原则

- 只定向撤销 Change 025 白名单文件，不使用 `git reset --hard`、`git clean` 或批量覆盖。
- 不删除或改写题库 Session、Record、Questions、截图、原答案、旧导出、用户配置或旧验证程序。
- 先移除前端新 UI/DTO 并重生固定 Wails v2.11.0 bindings，再恢复生产入口 wiring，最后移除 adapter、工厂与依赖。
- 仅在所有 Anthropic imports 已移除后，移除 SDK v2.3.0 requirement 并恢复实施前 `go` directive；全量验证依赖图。
- 回滚后保留本 Change 治理和审计，记录原因与实际边界；未经授权不提交。
