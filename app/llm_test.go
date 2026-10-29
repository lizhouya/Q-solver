package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"Q-Solver/pkg/llm"
)

func TestListModelsForDraftReturnsStructuredResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"one","object":"model"},{"id":"one","object":"model"},{"id":"two","object":"model"}],"object":"list"}`))
	}))
	defer server.Close()

	result := (&App{}).ListModelsForDraft(ConnectionDraft{
		APIKey: "test-key", BaseURL: server.URL + "/v1", Model: "manual-model", APIProtocol: "openai_chat",
	})
	if result.Error != nil || len(result.Models) != 2 || result.Models[0] != "one" || result.Models[1] != "two" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestProbeModelConnectionReturnsSafeLocalValidationDTO(t *testing.T) {
	result := (&App{}).ProbeModelConnection(ModelProbeRequest{
		Connection: ConnectionDraft{APIKey: "private-key", BaseURL: "file:///private", Model: "test-model", APIProtocol: "openai_responses"},
		Kind:       "text",
	})
	if result.RequestSent || result.Outcome != "failed" || result.Protocol != "openai_responses" || result.Error == nil {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Error.Code != "invalid_config" || result.Error.Stage != "validate_config" || result.Error.Message == "" {
		t.Fatalf("unexpected error DTO: %+v", result.Error)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range []string{"private-key", "file:///private"} {
		if strings.Contains(string(encoded), sentinel) {
			t.Fatalf("private draft value leaked: %s", sentinel)
		}
	}
}

func TestProviderErrorDTOTypedNilReturnsNil(t *testing.T) {
	var typedNil *llm.ProviderError
	var err error = typedNil
	if result := providerErrorDTO(err); result != nil {
		t.Fatalf("typed nil should map to nil DTO: %+v", result)
	}
}

func TestProbeModelConnectionSuccessReturnsNilErrorDTO(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		encoded, _ := json.Marshal(body)
		token := regexp.MustCompile(`QSP-[A-F0-9]{12}`).FindString(string(encoded))
		if token == "" {
			t.Error("probe token missing from request")
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"probe","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"` + token + `"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	result := (&App{}).ProbeModelConnection(ModelProbeRequest{
		Connection: ConnectionDraft{APIKey: "test-key", BaseURL: server.URL + "/v1", Model: "test-model", APIProtocol: "openai_chat"},
		Kind:       "text",
	})
	if calls != 1 {
		t.Fatalf("expected one request, got %d", calls)
	}
	if result.Outcome != "passed" || result.Error != nil || !result.RequestSent {
		t.Fatalf("unexpected success DTO: %+v", result)
	}
}

func TestConnectionDraftMapsConnectionModeWithoutDroppingFields(t *testing.T) {
	draft := ConnectionDraft{
		ConnectionMode: "ccr",
		APIKey:         "test-only-key",
		BaseURL:        "http://127.0.0.1:3456",
		Model:          "route-a",
		APIProtocol:    "anthropic_messages",
	}
	got := draft.config()
	if got.ConnectionMode != draft.ConnectionMode || got.APIKey != draft.APIKey || got.BaseURL != draft.BaseURL || got.Model != draft.Model || got.APIProtocol != draft.APIProtocol {
		t.Fatalf("draft mapping dropped a field: %+v", got)
	}
}
