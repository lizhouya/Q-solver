package llm

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"
	"Q-Solver/pkg/logger"
)

// Service 管理当前活动模型 provider，并为未保存的连接草稿创建隔离 provider。
type Service struct {
	mu       sync.RWMutex
	config   config.Config
	provider Provider
}

func NewService(cfg config.Config, cm *config.ConfigManager) *Service {
	s := &Service{config: cfg}
	s.updateProviderLocked()
	if cm == nil {
		return s
	}

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
	provider, err := NewProvider(s.config)
	if err != nil {
		s.provider = nil
		return
	}
	s.provider = provider
}

func (s *Service) GetProvider() Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.provider
}

// TestConnection 保持旧 Wails 入口的 OpenAI Chat 语义。
func (s *Service) TestConnection(ctx context.Context, apiKey, model string) string {
	if strings.TrimSpace(apiKey) == "" {
		return "API Key 不能为空"
	}
	if strings.TrimSpace(model) == "" {
		return "请选择模型"
	}
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()
	cfg.APIKey = apiKey
	cfg.Model = model
	cfg.APIProtocol = string(protocol.OpenAIChat)
	provider, err := providerForConnection(cfg, true)
	if err != nil {
		return err.Error()
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := provider.TestChat(timeoutCtx); err != nil {
		return err.Error()
	}
	return ""
}

// GetModels 保持旧 Wails 入口的 OpenAI Chat 语义。
func (s *Service) GetModels(ctx context.Context, apiKey string) ([]string, error) {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()
	if strings.TrimSpace(apiKey) != "" {
		cfg.APIKey = apiKey
	}
	cfg.APIProtocol = string(protocol.OpenAIChat)
	return s.ListModelsForConnection(ctx, cfg)
}

func normalizedConnectionConfig(cfg config.Config, requireModel bool) (config.Config, *ProviderError) {
	normalized, err := cfg.NormalizedConnectionSettings()
	if err != nil {
		return config.Config{}, NewProviderError(ErrorInvalidConfig, StageValidateConfig, "", err)
	}
	cfg = normalized
	p, err := protocol.Normalize(cfg.APIProtocol)
	if err != nil {
		return config.Config{}, NewProviderError(ErrorInvalidConfig, StageValidateConfig, "", err)
	}
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.APIKey == "" || (requireModel && cfg.Model == "") {
		return config.Config{}, NewProviderError(ErrorInvalidConfig, StageValidateConfig, p, nil)
	}
	u, parseErr := url.Parse(cfg.BaseURL)
	if parseErr != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return config.Config{}, NewProviderError(ErrorInvalidConfig, StageValidateConfig, p, parseErr)
	}
	return cfg, nil
}

func providerForConnection(cfg config.Config, requireModel bool) (Provider, *ProviderError) {
	normalized, err := normalizedConnectionConfig(cfg, requireModel)
	if err != nil {
		return nil, err
	}
	provider, factoryErr := NewProvider(normalized)
	if factoryErr != nil {
		var providerErr *ProviderError
		if errors.As(factoryErr, &providerErr) {
			return nil, providerErr
		}
		p, _ := protocol.Normalize(normalized.APIProtocol)
		return nil, ClassifyProviderError(factoryErr, StageValidateConfig, p)
	}
	return provider, nil
}

func (s *Service) ListModelsForConnection(ctx context.Context, cfg config.Config) ([]string, error) {
	provider, err := providerForConnection(cfg, false)
	if err != nil {
		return nil, err
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	models, requestErr := provider.GetModels(timeoutCtx)
	if requestErr != nil {
		return nil, requestErr
	}
	return deduplicateModels(models), nil
}

func deduplicateModels(models []string) []string {
	result := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		if _, exists := seen[model]; exists {
			continue
		}
		seen[model] = struct{}{}
		result = append(result, model)
	}
	return result
}

func (s *Service) ProbeConnection(ctx context.Context, cfg config.Config, request ProbeRequest) ProbeResult {
	mode, modeErr := config.NormalizeConnectionMode(cfg.ConnectionMode)
	if modeErr != nil {
		err := NewProviderError(ErrorInvalidConfig, StageValidateConfig, "", modeErr)
		return failedProbe(request.Kind, "", err, false)
	}
	if mode == config.ConnectionModeCCR {
		err := NewProviderError(ErrorProbeDisabled, StageValidateProbe, protocol.OpenAIChat, nil)
		return failedProbe(request.Kind, protocol.OpenAIChat, err, false)
	}
	normalized, validationErr := normalizedConnectionConfig(cfg, true)
	if validationErr != nil {
		return failedProbe(request.Kind, validationErr.Protocol, validationErr, false)
	}
	provider, factoryErr := NewProvider(normalized)
	if factoryErr != nil {
		var providerErr *ProviderError
		if !errors.As(factoryErr, &providerErr) {
			p, _ := protocol.Normalize(normalized.APIProtocol)
			providerErr = ClassifyProviderError(factoryErr, StageValidateConfig, p)
		}
		return failedProbe(request.Kind, providerErr.Protocol, providerErr, false)
	}
	p, _ := protocol.Normalize(normalized.APIProtocol)
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return providerProber{provider: provider, protocol: p, model: normalized.Model}.Probe(timeoutCtx, request)
}

func (s *Service) GetModelsForConnection(ctx context.Context, apiKey, baseURL string) ([]string, error) {
	return s.ListModelsForConnection(ctx, config.Config{
		APIKey: apiKey, BaseURL: baseURL, APIProtocol: string(protocol.OpenAIChat),
	})
}

func (s *Service) TestConnectionForConnection(ctx context.Context, apiKey, baseURL, model string) string {
	provider, err := providerForConnection(config.Config{
		APIKey: apiKey, BaseURL: baseURL, Model: model, APIProtocol: string(protocol.OpenAIChat),
	}, true)
	if err != nil {
		return err.Error()
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if requestErr := provider.TestChat(timeoutCtx); requestErr != nil {
		return requestErr.Error()
	}
	return ""
}
