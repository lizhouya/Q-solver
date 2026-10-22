package app

import "Q-Solver/pkg/config"

// GetSettings 返回当前配置
func (a *App) GetSettings() (config.Config, error) {
	if err := a.configManager.Load(); err != nil {
		return config.Config{}, err
	}
	return a.configManager.Get(), nil
}

// UpdateSettings 更新配置（从前端 JSON）
func (a *App) UpdateSettings(configJson string) string {
	if err := a.configManager.UpdateFromJSON(configJson); err != nil {
		return err.Error()
	}
	return ""
}
