package llm

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"
)

func NewProvider(cfg config.Config) (Provider, error) {
	normalized, err := cfg.NormalizedConnectionSettings()
	if err != nil {
		return nil, NewProviderError(ErrorInvalidConfig, StageValidateConfig, "", err)
	}
	p, err := protocol.Normalize(normalized.APIProtocol)
	if err != nil {
		return nil, NewProviderError(ErrorInvalidConfig, StageValidateConfig, "", err)
	}
	switch p {
	case protocol.OpenAIChat:
		return NewOpenAIAdapter(normalized), nil
	case protocol.OpenAIResponses:
		return NewOpenAIResponsesAdapter(normalized), nil
	case protocol.AnthropicMessages:
		return NewAnthropicAdapter(normalized), nil
	default:
		return nil, NewProviderError(ErrorInvalidConfig, StageValidateConfig, p, nil)
	}
}
