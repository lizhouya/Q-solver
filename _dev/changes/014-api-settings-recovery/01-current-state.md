# 现状审计

## 已确认事实
- 只读检查：本机加密配置存在、能解密，API Key/Base URL/模型均为 SET。未输出实值，未验证服务商是否接受密钥。
- settings.loadSettings 将 GetSettings 和 GetDomainCategories 放入 Promise.all；场景失败会丢弃成功的配置，已用内存 mock 复现。
- SettingsModal.saveAllSettings 无条件先保存题库，失败直接返回，普通设置保存未执行，已用内存 mock 复现。
- GetModels/TestConnection 只有 API Key/模型参数；Service 从已保存 config 取 Base URL；前端修改本地 settings.baseURL 无法影响 Go。
- ConfigManager.Load 在读/解密失败时仍返回 nil，Startup 在题库恢复后才读取主配置；Save 缺少成功加载前置条件。
- 当前截图的具体运行时触发原因尚未确认，不把可复现缺陷冒充现场日志结论。

## 安全性
只读检查不修改私人配置；所有后续测试使用合成凭据。防止错误回退到默认值后保存覆盖原文件。
