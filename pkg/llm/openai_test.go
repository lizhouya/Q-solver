package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"Q-Solver/pkg/config"
)

func newOpenAITestAdapter(server *httptest.Server, model string) *OpenAIAdapter {
	return NewOpenAIAdapter(config.Config{APIKey: "test-key", BaseURL: server.URL + "/v1", Model: model})
}

func writeChatResponse(w http.ResponseWriter, choices string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"id":"test","object":"chat.completion","model":"test","choices":` + choices + `}`))
}

func TestOpenAIChatRequestUsesImageAndFileParts(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString([]byte("png"))
	pdfData := base64.StdEncoding.EncodeToString([]byte("pdf"))
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		writeChatResponse(w, `[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]`)
	}))
	defer server.Close()

	adapter := newOpenAITestAdapter(server, "test-model")
	_, err := adapter.GenerateContent(context.Background(), "", []Message{
		NewSystemMessage("system"),
		NewMultiPartMessage(RoleUser, []ContentPart{
			TextPart("question"),
			ImagePart("data:image/png;base64," + pngData),
			PDFPart(pdfData),
		}),
		NewAssistantMessage("prior"),
	})
	if err != nil {
		t.Fatal(err)
	}

	messages := body["messages"].([]any)
	parts := messages[1].(map[string]any)["content"].([]any)
	if parts[0].(map[string]any)["type"] != "text" || parts[1].(map[string]any)["type"] != "image_url" || parts[2].(map[string]any)["type"] != "file" {
		t.Fatalf("unexpected multipart shape: %#v", parts)
	}
	file := parts[2].(map[string]any)["file"].(map[string]any)
	if file["file_data"] != "data:application/pdf;base64,"+pdfData || file["filename"] != "document.pdf" {
		t.Fatalf("unexpected file part: %#v", file)
	}
	if _, ok := body["max_tokens"]; ok {
		t.Fatal("normal request must not set deprecated max_tokens")
	}
	if _, ok := body["max_completion_tokens"]; ok {
		t.Fatal("normal request must not cap output")
	}
}

func TestOpenAIChatRejectsInvalidDataBeforeRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer server.Close()
	adapter := newOpenAITestAdapter(server, "test-model")

	for _, part := range []ContentPart{
		ImagePart("data:image/png;base64,%%%"),
		ImagePart("data:text/plain;base64," + base64.StdEncoding.EncodeToString([]byte("not image"))),
		{Type: ContentPDF, Base64: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not pdf"))},
	} {
		_, err := adapter.GenerateContent(context.Background(), "", []Message{NewMultiPartMessage(RoleUser, []ContentPart{part})})
		var providerErr *ProviderError
		if !errors.As(err, &providerErr) || providerErr.Code != ErrorInvalidRequest || providerErr.Stage != StageBuildRequest {
			t.Fatalf("unexpected validation error for %#v: %v", part, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid input sent %d requests", calls.Load())
	}
}

func TestOpenAIChatNonStreamingTerminalValidation(t *testing.T) {
	tests := []struct {
		name, choices, want string
		code                ErrorCode
	}{
		{name: "stop", choices: `[{"index":0,"message":{"role":"assistant","content":"answer"},"finish_reason":"stop"}]`, want: "answer"},
		{name: "empty choices", choices: `[]`, code: ErrorEmptyResponse},
		{name: "empty content", choices: `[{"index":0,"message":{"role":"assistant","content":""},"finish_reason":"stop"}]`, code: ErrorEmptyResponse},
		{name: "refusal", choices: `[{"index":0,"message":{"role":"assistant","content":"","refusal":"no"},"finish_reason":"stop"}]`, code: ErrorRefused},
		{name: "length", choices: `[{"index":0,"message":{"role":"assistant","content":"partial"},"finish_reason":"length"}]`, code: ErrorIncompleteResponse},
		{name: "content filter", choices: `[{"index":0,"message":{"role":"assistant","content":""},"finish_reason":"content_filter"}]`, code: ErrorRefused},
		{name: "tool calls", choices: `[{"index":0,"message":{"role":"assistant","content":""},"finish_reason":"tool_calls"}]`, code: ErrorInvalidResponse},
		{name: "unknown", choices: `[{"index":0,"message":{"role":"assistant","content":"answer"},"finish_reason":"other"}]`, code: ErrorInvalidResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { writeChatResponse(w, tt.choices) }))
			defer server.Close()
			got, err := newOpenAITestAdapter(server, "test-model").GenerateContent(context.Background(), "", []Message{NewUserMessage("hi")})
			if tt.code == "" {
				if err != nil || got.Content != tt.want {
					t.Fatalf("got=%+v err=%v", got, err)
				}
				return
			}
			var providerErr *ProviderError
			if !errors.As(err, &providerErr) || providerErr.Code != tt.code {
				t.Fatalf("want %s, got %v", tt.code, err)
			}
		})
	}
}

func TestOpenAIChatStreamingRequiresTerminalEvent(t *testing.T) {
	tests := []struct {
		name   string
		events []string
		want   string
		code   ErrorCode
	}{
		{name: "stop", events: []string{`{"choices":[{"index":0,"delta":{"content":"answer"},"finish_reason":""}]}`, `{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`}, want: "answer"},
		{name: "eof without terminal", events: []string{`{"choices":[{"index":0,"delta":{"content":"partial"},"finish_reason":""}]}`}, code: ErrorStreamInterrupted},
		{name: "length", events: []string{`{"choices":[{"index":0,"delta":{"content":"partial"},"finish_reason":"length"}]}`}, code: ErrorIncompleteResponse},
		{name: "refusal", events: []string{`{"choices":[{"index":0,"delta":{"refusal":"no"},"finish_reason":"stop"}]}`}, code: ErrorRefused},
		{name: "empty stop", events: []string{`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`}, code: ErrorEmptyResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				for _, event := range tt.events {
					_, _ = w.Write([]byte("data: " + event + "\n\n"))
				}
				_, _ = w.Write([]byte("data: [DONE]\n\n"))
			}))
			defer server.Close()
			got, err := newOpenAITestAdapter(server, "test-model").GenerateContentStream(context.Background(), []Message{NewUserMessage("hi")}, nil)
			if tt.code == "" {
				if err != nil || got.Content != tt.want {
					t.Fatalf("got=%+v err=%v", got, err)
				}
				return
			}
			var providerErr *ProviderError
			if !errors.As(err, &providerErr) || providerErr.Code != tt.code {
				t.Fatalf("want %s, got %v", tt.code, err)
			}
		})
	}
}

func TestOpenAIChatTestUsesSingleLowCostRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["max_completion_tokens"] != float64(32) {
			t.Fatalf("unexpected token limit: %#v", body)
		}
		if _, ok := body["max_tokens"]; ok {
			t.Fatal("deprecated max_tokens was sent")
		}
		writeChatResponse(w, `[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]`)
	}))
	defer server.Close()
	if err := newOpenAITestAdapter(server, "test-model").TestChat(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("got %d requests", calls.Load())
	}
}

func TestOpenAIChatDisablesSDKRetries(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"private payload","type":"server_error","code":"server_error"}}`))
	}))
	defer server.Close()
	err := newOpenAITestAdapter(server, "test-model").TestChat(context.Background())
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != ErrorProviderUnavailable {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("SDK retried request %d times", calls.Load())
	}
	if strings.Contains(err.Error(), "private payload") {
		t.Fatalf("response body leaked: %v", err)
	}
}

func TestOpenAIChatDefaultsModelWithoutMutatingCaller(t *testing.T) {
	var model string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		model, _ = body["model"].(string)
		writeChatResponse(w, `[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]`)
	}))
	defer server.Close()
	cfg := config.Config{APIKey: "test", BaseURL: server.URL + "/v1"}
	adapter := NewOpenAIAdapter(cfg)
	if err := adapter.TestChat(context.Background()); err != nil {
		t.Fatal(err)
	}
	if model == "" || adapter.config.Model == "" || cfg.Model != "" {
		t.Fatalf("default model snapshot failed: sent=%q adapter=%q caller=%q", model, adapter.config.Model, cfg.Model)
	}
}
