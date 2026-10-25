# 回滚预案 05

## 原则

- 只撤销 Change 017 的 API、DTO、前端浏览、实时 history reset 和元数据确认 UI。
- 不删除、移动、覆盖或批量重写 `_记录`、截图、旧导出目录和用户配置。
- 不恢复已成功软清空的段为可写状态。
- 不回滚或覆盖 Change 016 的未提交文件内容。

## 数据兼容

新字段使用 `omitempty`，旧代码可忽略。若回滚代码，已经写入的 `Name`、`Issuer`、`NameCustomized` 和 `MetadataSuggestion` 留在 JSON 中，不主动清除。

## 操作限制

禁止使用 `git reset --hard`、`git clean` 或批量恢复叠加文件。任何破坏性回滚、commit、push、Tag 或 Release 必须另获授权。

状态：未执行回滚。
