package llm

import (
	"errors"
	"testing"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"
)

func TestNewProviderRoutesOnlyByExplicitProtocol(t *testing.T) {
	chatCases := []config.Config{
		{APIProtocol: "", Model: "claude-opus-5", BaseURL: "https://api.anthropic.com"},
		{APIProtocol: string(protocol.OpenAIChat), Model: "gpt-test", BaseURL: "https://example.invalid/v1"},
	}
	for _, cfg := range chatCases {
		provider, err := NewProvider(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := provider.(*OpenAIAdapter); !ok {
			t.Fatalf("expected Chat adapter for protocol %q, got %T", cfg.APIProtocol, provider)
		}
	}

	provider, err := NewProvider(config.Config{APIProtocol: string(protocol.OpenAIResponses), Model: "claude-opus-5", BaseURL: "https://api.openai.com/v1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := provider.(*OpenAIResponsesAdapter); !ok {
		t.Fatalf("expected Responses adapter, got %T", provider)
	}

	provider, err = NewProvider(config.Config{APIProtocol: string(protocol.AnthropicMessages), Model: "claude-opus-5", BaseURL: "https://api.openai.com/v1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := provider.(*AnthropicAdapter); !ok {
		t.Fatalf("expected Anthropic adapter, got %T", provider)
	}
}

func TestNewProviderRejectsUnknownProtocol(t *testing.T) {
	_, err := NewProvider(config.Config{APIProtocol: "claude-opus-5"})
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != ErrorInvalidConfig || providerErr.Stage != StageValidateConfig {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIChatAdapterKeepsConfigSnapshot(t *testing.T) {
	cfg := config.Config{APIProtocol: string(protocol.OpenAIChat), APIKey: "first", BaseURL: "https://first.invalid/v1", Model: "first-model"}
	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.APIKey = "second"
	cfg.BaseURL = "https://second.invalid/v1"
	cfg.Model = "second-model"

	adapter := provider.(*OpenAIAdapter)
	if adapter.config.APIKey != "first" || adapter.config.BaseURL != "https://first.invalid/v1" || adapter.config.Model != "first-model" {
		t.Fatalf("adapter observed mutable config: %+v", adapter.config)
	}
}

func TestNewProviderAllowsUnconfiguredStartup(t *testing.T) {
	provider, err := NewProvider(config.Config{})
	if err != nil || provider == nil {
		t.Fatalf("empty config must construct safely: provider=%T err=%v", provider, err)
	}
}

func TestNewProviderCCRAlwaysUsesOpenAIChatAdapter(t *testing.T) {
	for _, cfg := range []config.Config{
		{ConnectionMode: config.ConnectionModeCCR, APIProtocol: "anthropic_messages", BaseURL: "https://gateway.example.com", Model: "claude-opus-5"},
		{ConnectionMode: config.ConnectionModeCCR, APIProtocol: "openai_responses", BaseURL: "https://router.example.com/v1/", Model: "gpt-test"},
		{ConnectionMode: config.ConnectionModeCCR, APIProtocol: "not-a-protocol", BaseURL: "http://127.0.0.1:3456", Model: "deepseek-test"},
	} {
		provider, err := NewProvider(cfg)
		if err != nil {
			t.Fatal(err)
		}
		adapter, ok := provider.(*OpenAIAdapter)
		if !ok {
			t.Fatalf("CCR constructed %T instead of OpenAI Chat adapter", provider)
		}
		if adapter.config.ConnectionMode != config.ConnectionModeCCR || adapter.config.APIProtocol != "openai_chat" || adapter.config.BaseURL[len(adapter.config.BaseURL)-3:] != "/v1" {
			t.Fatalf("CCR adapter did not receive normalized settings: %+v", adapter.config)
		}
	}
}

func TestNewProviderRejectsUnknownConnectionMode(t *testing.T) {
	_, err := NewProvider(config.Config{ConnectionMode: "claude", APIProtocol: "openai_chat"})
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != ErrorInvalidConfig || providerErr.Stage != StageValidateConfig {
		t.Fatalf("unexpected error: %v", err)
	}
}
