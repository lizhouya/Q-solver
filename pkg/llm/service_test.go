package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"Q-Solver/pkg/config"
)

func TestDraftConnectionUsesOwnEndpointWithoutMutatingSavedProvider(t *testing.T) {
	var oldCalls, draftCalls atomic.Int32
	old := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oldCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"saved-model","object":"model"}],"object":"list"}`))
	}))
	defer old.Close()
	draft := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		draftCalls.Add(1)
		if r.Header.Get("Authorization") != "Bearer draft-test-key" {
			t.Error("unexpected synthetic credential")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/models" {
			w.Write([]byte(`{"data":[{"id":"draft-model","object":"model"}],"object":"list"}`))
		} else if r.URL.Path == "/v1/chat/completions" {
			w.Write([]byte(`{"id":"test","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
		} else {
			t.Error("unexpected route")
			w.WriteHeader(404)
		}
	}))
	defer draft.Close()
	s := &Service{config: config.Config{APIKey: "saved-test-key", BaseURL: old.URL + "/v1", Model: "saved-model"}}
	s.updateProviderLocked()
	before := s.GetProvider()
	models, err := s.GetModelsForConnection(context.Background(), "draft-test-key", draft.URL+"/v1")
	if err != nil || len(models) != 1 || models[0] != "draft-model" {
		t.Fatal("draft model request failed")
	}
	if result := s.TestConnectionForConnection(context.Background(), "draft-test-key", draft.URL+"/v1", "draft-model"); result != "" {
		t.Fatal("draft connection test failed")
	}
	if draftCalls.Load() != 2 || oldCalls.Load() != 0 {
		t.Fatal("draft request used saved endpoint")
	}
	if s.GetProvider() != before || s.config.BaseURL != old.URL+"/v1" {
		t.Fatal("saved provider changed")
	}
	if _, err := s.GetModels(context.Background(), "saved-test-key"); err != nil || oldCalls.Load() != 1 {
		t.Fatal("legacy route changed")
	}
}

func TestDraftConnectionRejectsInvalidInput(t *testing.T) {
	s := &Service{}
	for _, u := range []string{"", "file:///tmp/models", "invalid", "https://user:password@example.invalid/v1"} {
		if _, err := s.GetModelsForConnection(context.Background(), "test-key", u); err == nil {
			t.Fatal("invalid endpoint accepted")
		}
	}
	if _, err := s.GetModelsForConnection(context.Background(), "", "http://localhost/v1"); err == nil {
		t.Fatal("empty key accepted")
	}
}

func TestDraftModelListRoutesByProtocolAndDeduplicatesInOrder(t *testing.T) {
	tests := []struct {
		name, protocolValue, baseSuffix, wantPath, response string
	}{
		{"chat", "openai_chat", "/v1", "/v1/models", `{"data":[{"id":"first","object":"model"},{"id":"second","object":"model"},{"id":"first","object":"model"}],"object":"list"}`},
		{"responses", "openai_responses", "/v1", "/v1/models", `{"data":[{"id":"first","object":"model"},{"id":"second","object":"model"},{"id":"first","object":"model"}],"object":"list"}`},
		{"anthropic", "anthropic_messages", "", "/v1/models", `{"data":[{"id":"first","type":"model","display_name":"First","created_at":"2026-01-01T00:00:00Z","lifecycle":"active","line":"opus","max_input_tokens":1,"max_tokens":1,"capabilities":{}},{"id":"second","type":"model","display_name":"Second","created_at":"2026-01-01T00:00:00Z","lifecycle":"active","line":"opus","max_input_tokens":1,"max_tokens":1,"capabilities":{}},{"id":"first","type":"model","display_name":"First","created_at":"2026-01-01T00:00:00Z","lifecycle":"active","line":"opus","max_input_tokens":1,"max_tokens":1,"capabilities":{}}],"has_more":false,"first_id":"first","last_id":"first"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != tt.wantPath {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()
			s := &Service{config: config.Config{APIKey: "saved", BaseURL: "https://saved.invalid/v1", Model: "hand-written"}}
			beforeConfig := s.config
			models, err := s.ListModelsForConnection(context.Background(), config.Config{
				APIKey: "draft-key", BaseURL: server.URL + tt.baseSuffix, Model: "draft-manual", APIProtocol: tt.protocolValue,
			})
			if err != nil || len(models) != 2 || models[0] != "first" || models[1] != "second" {
				t.Fatalf("models=%#v err=%v", models, err)
			}
			if calls.Load() != 1 {
				t.Fatalf("expected one request, got %d", calls.Load())
			}
			if s.config.APIKey != beforeConfig.APIKey || s.config.BaseURL != beforeConfig.BaseURL || s.config.Model != beforeConfig.Model || s.config.APIProtocol != beforeConfig.APIProtocol {
				t.Fatal("draft model list mutated saved service state")
			}
		})
	}
}

func TestDraftProbeLocalValidationDoesNotSendRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer server.Close()
	result := (&Service{}).ProbeConnection(context.Background(), config.Config{
		APIKey: "draft-key", BaseURL: server.URL + "/v1", APIProtocol: "openai_chat",
	}, ProbeRequest{Kind: string(ProbeText)})
	if result.RequestSent || result.Error == nil || result.Error.Code != ErrorInvalidConfig || calls.Load() != 0 {
		t.Fatalf("unexpected validation result: %+v calls=%d", result, calls.Load())
	}
}

func TestDraftTextProbeUsesExplicitProtocolSingleLowCostRequest(t *testing.T) {
	tests := []struct {
		name, protocolValue, baseSuffix, wantPath, maxField string
	}{
		{"chat", "openai_chat", "/v1", "/v1/chat/completions", "max_completion_tokens"},
		{"responses", "openai_responses", "/v1", "/v1/responses", "max_output_tokens"},
		{"anthropic", "anthropic_messages", "", "/v1/messages", "max_tokens"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != tt.wantPath {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body[tt.maxField] != float64(32) {
					t.Fatalf("missing low-cost limit: %#v", body)
				}
				token := findProbeToken(body)
				if token == "" {
					t.Fatalf("missing probe token without inspecting private data: %#v", body)
				}
				w.Header().Set("Content-Type", "application/json")
				switch tt.protocolValue {
				case "openai_chat":
					encoded, _ := json.Marshal(token)
					_, _ = w.Write([]byte(`{"id":"test","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":` + string(encoded) + `},"finish_reason":"stop"}]}`))
				case "openai_responses":
					writeResponsesResponse(w, "completed", completedResponsesOutput(token))
				case "anthropic_messages":
					writeAnthropicMessage(w, "end_turn", anthropicTextContent(token))
				}
			}))
			defer server.Close()
			result := (&Service{}).ProbeConnection(context.Background(), config.Config{
				APIKey: "draft-key", BaseURL: server.URL + tt.baseSuffix, Model: "test-model", APIProtocol: tt.protocolValue,
			}, ProbeRequest{Kind: string(ProbeText)})
			if result.Outcome != ProbePassed || !result.RequestSent || result.Error != nil || calls.Load() != 1 {
				t.Fatalf("result=%+v calls=%d", result, calls.Load())
			}
		})
	}
}

func findProbeToken(value any) string {
	switch typed := value.(type) {
	case string:
		for _, field := range strings.Fields(typed) {
			field = strings.Trim(field, " \t\r\n.,;:()[]{}\\\"")
			if strings.HasPrefix(field, "QSP-") {
				return field
			}
		}
	case []any:
		for _, item := range typed {
			if token := findProbeToken(item); token != "" {
				return token
			}
		}
	case map[string]any:
		for _, item := range typed {
			if token := findProbeToken(item); token != "" {
				return token
			}
		}
	}
	return ""
}

func TestDraftModelListFailureDoesNotMutateSavedProviderOrManualModel(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"private response","type":"server_error","code":"server_error"}}`))
	}))
	defer server.Close()
	s := &Service{config: config.Config{APIKey: "saved-key", BaseURL: "https://saved.invalid/v1", Model: "manual-model", APIProtocol: "openai_chat"}}
	s.updateProviderLocked()
	beforeProvider := s.GetProvider()
	_, err := s.ListModelsForConnection(context.Background(), config.Config{
		APIKey: "draft-key", BaseURL: server.URL + "/v1", Model: "draft-model", APIProtocol: "openai_chat",
	})
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != ErrorProviderUnavailable {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls.Load() != 1 || s.GetProvider() != beforeProvider || s.config.Model != "manual-model" || s.config.APIKey != "saved-key" {
		t.Fatalf("draft failure changed saved state: calls=%d config=%+v", calls.Load(), s.config)
	}
	if strings.Contains(err.Error(), "private response") {
		t.Fatal("provider response leaked")
	}
}

func TestServiceActiveProviderUsesExplicitProtocolFactory(t *testing.T) {
	tests := []struct {
		protocolValue string
		assertType    func(Provider) bool
	}{
		{"openai_chat", func(provider Provider) bool { _, ok := provider.(*OpenAIAdapter); return ok }},
		{"openai_responses", func(provider Provider) bool { _, ok := provider.(*OpenAIResponsesAdapter); return ok }},
		{"anthropic_messages", func(provider Provider) bool { _, ok := provider.(*AnthropicAdapter); return ok }},
	}
	for _, tt := range tests {
		service := NewService(config.Config{APIProtocol: tt.protocolValue}, nil)
		if provider := service.GetProvider(); provider == nil || !tt.assertType(provider) {
			t.Fatalf("protocol %q created %T", tt.protocolValue, provider)
		}
	}
}

func TestCCRModelCatalogUsesExactlyOneGETAndNoInferenceRequest(t *testing.T) {
	var calls atomic.Int32
	var posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method == http.MethodPost {
			posts.Add(1)
		}
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Errorf("unexpected CCR request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"route-a","object":"model"},{"id":"route-b","object":"model"},{"id":"route-a","object":"model"}],"object":"list"}`))
	}))
	defer server.Close()

	models, err := (&Service{}).ListModelsForConnection(context.Background(), config.Config{
		ConnectionMode: config.ConnectionModeCCR,
		APIKey:         "test-key",
		BaseURL:        server.URL,
		APIProtocol:    "anthropic_messages",
		Model:          "claude-name-must-not-select-protocol",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0] != "route-a" || models[1] != "route-b" {
		t.Fatalf("unexpected catalog: %#v", models)
	}
	if calls.Load() != 1 || posts.Load() != 0 {
		t.Fatalf("CCR catalog must be exactly one GET and zero POST: calls=%d posts=%d", calls.Load(), posts.Load())
	}
}

func TestCCRProbeIsRejectedBeforeAnyNetworkRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "must not be called", http.StatusInternalServerError)
	}))
	defer server.Close()

	for _, kind := range []string{string(ProbeText), string(ProbeVision)} {
		result := (&Service{}).ProbeConnection(context.Background(), config.Config{
			ConnectionMode: config.ConnectionModeCCR,
			APIKey:         "test-key",
			BaseURL:        server.URL,
			APIProtocol:    "anthropic_messages",
			Model:          "route-a",
		}, ProbeRequest{Kind: kind})
		if result.Kind != kind || result.Outcome != ProbeFailed || result.Protocol != "openai_chat" || result.RequestSent || result.Error == nil {
			t.Fatalf("unexpected CCR %s probe result: %+v", kind, result)
		}
		if result.Error.Code != ErrorProbeDisabled || result.Error.Stage != StageValidateProbe || result.Error.Retryable || result.Error.HTTPStatus != 0 {
			t.Fatalf("unexpected CCR %s probe error: %+v", kind, result.Error)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("CCR probes made %d network requests", calls.Load())
	}
}

func TestCCRProbeDisabledEvenWhenRequestFieldsAreIncomplete(t *testing.T) {
	result := (&Service{}).ProbeConnection(context.Background(), config.Config{
		ConnectionMode: config.ConnectionModeCCR,
	}, ProbeRequest{Kind: string(ProbeText)})
	if result.Error == nil || result.Error.Code != ErrorProbeDisabled || result.RequestSent || result.Protocol != "openai_chat" {
		t.Fatalf("CCR probe should stop before request validation: %+v", result)
	}
}
