package app

import "Q-Solver/pkg/config"

// GetSettings 返回不含连接凭据的全局设置快照。
func (a *App) GetSettings() (config.Config, error) {
	if err := a.configManager.Load(); err != nil {
		return config.Config{}, err
	}
	safe := a.configManager.Get()
	safe.ConnectionProfiles = nil
	safe.APIKey = ""
	safe.BaseURL = ""
	safe.ConnectionMode = ""
	safe.APIProtocol = ""
	safe.EndpointPath = ""
	safe.ExtraHeaders = nil
	safe.ResponseFormat = ""
	safe.RequestTemplate = ""
	safe.ResponseJSONPointer = ""
	safe.AuthHeader = ""
	safe.AuthPrefix = ""
	safe.Model = ""
	return safe, nil
}

// UpdateSettings 更新配置（从前端 JSON）
func (a *App) UpdateSettings(configJson string) string {
	if err := a.configManager.UpdateFromJSON(configJson); err != nil {
		return "设置保存失败，原配置未修改"
	}
	return ""
}
