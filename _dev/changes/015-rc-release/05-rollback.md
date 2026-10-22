# 回退说明

旧 EXE 在 build/bin/archive 中保留并核对 SHA-256；恢复时先正常退出当前实例，再将所需备份复制为 Q-Solver.exe。不得覆盖用户配置或题库数据。

代码回退使用明确的新提交，不 reset/clean、不强制推送；不自动删除或移动公开 Tag/Release。如需撤回公开版本，另行取得授权。
