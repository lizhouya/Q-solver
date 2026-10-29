package app

import (
	"context"
	"errors"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
)

type ConnectionDraft struct {
	ConnectionMode string `json:"connectionMode"`
	APIKey         string `json:"apiKey"`
	BaseURL        string `json:"baseURL"`
	Model          string `json:"model"`
	APIProtocol    string `json:"apiProtocol"`
}

type ProviderErrorDTO struct {
	Code       string `json:"code"`
	Stage      string `json:"stage"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable"`
	HTTPStatus int    `json:"httpStatus"`
	Protocol   string `json:"protocol"`
}

type ModelListResult struct {
	Models []string          `json:"models"`
	Error  *ProviderErrorDTO `json:"error,omitempty"`
}

type ModelProbeRequest struct {
	Connection ConnectionDraft `json:"connection"`
	Kind       string          `json:"kind"`
}

type ModelProbeResult struct {
	Kind        string            `json:"kind"`
	Outcome     string            `json:"outcome"`
	Protocol    string            `json:"protocol"`
	Message     string            `json:"message"`
	Error       *ProviderErrorDTO `json:"error,omitempty"`
	RequestSent bool              `json:"requestSent"`
}

func (draft ConnectionDraft) config() config.Config {
	return config.Config{
		ConnectionMode: draft.ConnectionMode, APIKey: draft.APIKey, BaseURL: draft.BaseURL,
		Model: draft.Model, APIProtocol: draft.APIProtocol,
	}
}

func appContext(a *App) context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func providerErrorDTO(err error) *ProviderErrorDTO {
	if err == nil {
		return nil
	}
	var providerErr *llm.ProviderError
	if !errors.As(err, &providerErr) {
		return &ProviderErrorDTO{Code: string(llm.ErrorNetwork), Stage: string(llm.StageSendRequest), Message: "无法连接模型服务，请检查网络和接口地址"}
	}
	if providerErr == nil {
		return nil
	}
	return &ProviderErrorDTO{
		Code: string(providerErr.Code), Stage: string(providerErr.Stage), Message: providerErr.Message,
		Retryable: providerErr.Retryable, HTTPStatus: providerErr.HTTPStatus, Protocol: string(providerErr.Protocol),
	}
}

func (a *App) ListModelsForDraft(draft ConnectionDraft) ModelListResult {
	models, err := (&llm.Service{}).ListModelsForConnection(appContext(a), draft.config())
	if err != nil {
		return ModelListResult{Models: []string{}, Error: providerErrorDTO(err)}
	}
	return ModelListResult{Models: models}
}

func (a *App) ProbeModelConnection(request ModelProbeRequest) ModelProbeResult {
	result := (&llm.Service{}).ProbeConnection(appContext(a), request.Connection.config(), llm.ProbeRequest{Kind: request.Kind})
	var errorDTO *ProviderErrorDTO
	if result.Error != nil {
		errorDTO = providerErrorDTO(result.Error)
	}
	return ModelProbeResult{
		Kind: result.Kind, Outcome: string(result.Outcome), Protocol: string(result.Protocol),
		Message: result.Message, Error: errorDTO, RequestSent: result.RequestSent,
	}
}

func (a *App) TestConnection(apiKey, model string) string {
	return a.llmService.TestConnection(appContext(a), apiKey, model)
}

func (a *App) GetModels(apiKey string) ([]string, error) {
	return a.llmService.GetModels(appContext(a), apiKey)
}

// 单次连接 API 不依赖已保存设置或后台服务初始化完成。
func (a *App) GetModelsForConnection(apiKey, baseURL string) ([]string, error) {
	return (&llm.Service{}).GetModelsForConnection(appContext(a), apiKey, baseURL)
}

func (a *App) TestConnectionForConnection(apiKey, baseURL, model string) string {
	return (&llm.Service{}).TestConnectionForConnection(appContext(a), apiKey, baseURL, model)
}
