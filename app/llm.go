package app

import (
	"Q-Solver/pkg/llm"
	"context"
)

func (a *App) TestConnection(apiKey, model string) string {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.llmService.TestConnection(ctx, apiKey, model)
}

func (a *App) GetModels(apiKey string) ([]string, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.llmService.GetModels(ctx, apiKey)
}

// 单次连接 API 不依赖已保存设置或后台服务初始化完成。
func (a *App) GetModelsForConnection(apiKey, baseURL string) ([]string, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return (&llm.Service{}).GetModelsForConnection(ctx, apiKey, baseURL)
}

func (a *App) TestConnectionForConnection(apiKey, baseURL, model string) string {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return (&llm.Service{}).TestConnectionForConnection(ctx, apiKey, baseURL, model)
}
