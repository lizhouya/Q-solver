package llm

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/logger"
)

// Service manages the active OpenAI-compatible provider.
type Service struct {
	mu       sync.RWMutex
	config   config.Config
	provider Provider
}

func NewService(cfg config.Config, cm *config.ConfigManager) *Service {
	s := &Service{
		config: cfg,
	}
	s.updateProviderLocked()

	cm.Subscribe(func(newConfig config.Config, oldConfig config.Config) {
		s.mu.Lock()
		s.config = newConfig
		s.updateProviderLocked()
		s.mu.Unlock()
		logger.Println("LLM Provider 已更新")
	})

	return s
}

func (s *Service) updateProviderLocked() {
	s.provider = NewOpenAIAdapter(&s.config)
}

func (s *Service) GetProvider() Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.provider
}

func (s *Service) TestConnection(ctx context.Context, apiKey, model string) string {
	if apiKey == "" {
		return "API Key 不能为空"
	}
	if model == "" {
		return "请选择模型"
	}

	s.mu.RLock()
	tempConfig := s.config
	s.mu.RUnlock()

	tempConfig.APIKey = apiKey
	tempConfig.Model = model

	tempProvider := NewOpenAIAdapter(&tempConfig)

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := tempProvider.TestChat(timeoutCtx); err != nil {
		return err.Error()
	}

	return ""
}

func (s *Service) GetModels(ctx context.Context, apiKey string) ([]string, error) {
	s.mu.RLock()
	currentAPIKey := s.config.APIKey
	currentProvider := s.provider
	cfg := s.config
	s.mu.RUnlock()

	if apiKey == "" {
		apiKey = currentAPIKey
	}

	if apiKey != currentAPIKey {
		tempConfig := cfg
		tempConfig.APIKey = apiKey
		tempProvider := NewOpenAIAdapter(&tempConfig)
		return tempProvider.GetModels(ctx)
	}

	if currentProvider == nil {
		return nil, fmt.Errorf("provider not initialized")
	}
	return currentProvider.GetModels(ctx)
}

// connectionProvider 使用本次草稿创建临时客户端，不修改运行中 provider 或持久配置。
func connectionProvider(apiKey, baseURL, model string) (*OpenAIAdapter, error) {
	apiKey, baseURL = strings.TrimSpace(apiKey), strings.TrimSpace(baseURL)
	if apiKey == "" {
		return nil, fmt.Errorf("请先填写 API Key")
	}
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return nil, fmt.Errorf("请填写有效的 HTTP/HTTPS 接口地址，且不要在地址中嵌入账号密码")
	}
	return NewOpenAIAdapter(&config.Config{APIKey: apiKey, BaseURL: baseURL, Model: model}), nil
}

func (s *Service) GetModelsForConnection(ctx context.Context, apiKey, baseURL string) ([]string, error) {
	provider, err := connectionProvider(apiKey, baseURL, "")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// 不使用旧 GetModels 的错误日志，避免把请求地址或服务商响应写入日志。
	response, err := provider.client.Models.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取模型列表失败，请检查本次接口地址、API Key 和网络连接")
	}
	models := make([]string, 0, len(response.Data))
	for _, item := range response.Data {
		models = append(models, item.ID)
	}
	return models, nil
}

func (s *Service) TestConnectionForConnection(ctx context.Context, apiKey, baseURL, model string) string {
	if strings.TrimSpace(model) == "" {
		return "请先选择模型"
	}
	provider, err := connectionProvider(apiKey, baseURL, model)
	if err != nil {
		return err.Error()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := provider.TestChat(ctx); err != nil {
		return "连接测试失败，请检查本次接口地址、API Key、模型和网络连接"
	}
	return ""
}
