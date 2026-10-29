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
	"Q-Solver/pkg/llm/protocol"
)

func newAnthropicTestAdapter(server *httptest.Server, model string) *AnthropicAdapter {
	return NewAnthropicAdapter(config.Config{
		APIKey:      "explicit-test-key",
		BaseURL:     server.URL,
		APIProtocol: string(protocol.AnthropicMessages),
		Model:       model,
	})
}

func writeAnthropicMessage(w http.ResponseWriter, stopReason string, content string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"id":"msg-test","type":"message","role":"assistant","model":"test-model","stop_reason":"` + stopReason + `","stop_sequence":null,"content":` + content + `,"usage":{"input_tokens":1,"output_tokens":1}}`))
}

func anthropicTextContent(text string) string {
	encoded, _ := json.Marshal(text)
	return `[{"type":"text","text":` + string(encoded) + `}]`
}

func TestAnthropicRequestUsesNativeStatelessMultimodalShape(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString([]byte("png"))
	pdfData := base64.StdEncoding.EncodeToString([]byte("pdf"))
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer explicit-test-key" {
			t.Fatalf("unexpected credential header")
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		writeAnthropicMessage(w, "end_turn", anthropicTextContent("ok"))
	}))
	defer server.Close()

	got, err := newAnthropicTestAdapter(server, "test-model").GenerateContent(context.Background(), "", []Message{
		NewSystemMessage("system one"),
		NewSystemMessage("system two"),
		NewMultiPartMessage(RoleUser, []ContentPart{
			TextPart("first"),
			ImagePart("data:image/png;base64," + pngData),
			TextPart("second"),
			PDFPart(pdfData),
		}),
		NewAssistantMessage("prior"),
	})
	if err != nil || got.Content != "ok" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if body["model"] != "test-model" || body["max_tokens"] != float64(anthropicBusinessMaxTokens) {
		t.Fatalf("unexpected core params: %#v", body)
	}
	for _, forbidden := range []string{"tools", "thinking", "container", "cache_control"} {
		if _, ok := body[forbidden]; ok {
			t.Fatalf("unexpected field %q: %#v", forbidden, body)
		}
	}
	system := body["system"].([]any)
	if len(system) != 2 || system[0].(map[string]any)["text"] != "system one" || system[1].(map[string]any)["text"] != "system two" {
		t.Fatalf("unexpected system blocks: %#v", system)
	}
	messages := body["messages"].([]any)
	if len(messages) != 2 || messages[0].(map[string]any)["role"] != "user" || messages[1].(map[string]any)["role"] != "assistant" {
		t.Fatalf("history order changed: %#v", messages)
	}
	parts := messages[0].(map[string]any)["content"].([]any)
	if len(parts) != 4 || parts[0].(map[string]any)["type"] != "text" || parts[1].(map[string]any)["type"] != "image" || parts[2].(map[string]any)["type"] != "text" || parts[3].(map[string]any)["type"] != "document" {
		t.Fatalf("unexpected content order: %#v", parts)
	}
	imageSource := parts[1].(map[string]any)["source"].(map[string]any)
	if imageSource["type"] != "base64" || imageSource["media_type"] != "image/png" || imageSource["data"] != pngData {
		t.Fatalf("unexpected image source: %#v", imageSource)
	}
	pdfSource := parts[3].(map[string]any)["source"].(map[string]any)
	if pdfSource["type"] != "base64" || pdfSource["media_type"] != "application/pdf" || pdfSource["data"] != pdfData {
		t.Fatalf("unexpected document source: %#v", pdfSource)
	}
}

func TestAnthropicRejectsInvalidDataAndSystemPartsBeforeRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer server.Close()
	adapter := newAnthropicTestAdapter(server, "test-model")
	cases := [][]Message{
		{NewMultiPartMessage(RoleUser, []ContentPart{ImagePart("data:image/png;base64,%%%")})},
		{NewMultiPartMessage(RoleUser, []ContentPart{{Type: ContentPDF, Base64: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("bad"))}})},
		{NewMultiPartMessage(RoleSystem, []ContentPart{ImagePart("data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("bad")))})},
	}
	for _, messages := range cases {
		_, err := adapter.GenerateContent(context.Background(), "", messages)
		var providerErr *ProviderError
		if !errors.As(err, &providerErr) || providerErr.Code != ErrorInvalidRequest || providerErr.Stage != StageBuildRequest {
			t.Fatalf("unexpected validation error: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid input sent %d requests", calls.Load())
	}
}

func TestAnthropicNonStreamingTerminalValidation(t *testing.T) {
	tests := []struct {
		name, stopReason, content, want string
		code                            ErrorCode
	}{
		{name: "end turn", stopReason: "end_turn", content: anthropicTextContent("answer"), want: "answer"},
		{name: "stop sequence", stopReason: "stop_sequence", content: anthropicTextContent("answer"), want: "answer"},
		{name: "multiple text", stopReason: "end_turn", content: `[{"type":"text","text":"first"},{"type":"text","text":"second"}]`, want: "firstsecond"},
		{name: "empty", stopReason: "end_turn", content: `[]`, code: ErrorEmptyResponse},
		{name: "max tokens", stopReason: "max_tokens", content: anthropicTextContent("partial"), code: ErrorIncompleteResponse},
		{name: "context window", stopReason: "model_context_window_exceeded", content: anthropicTextContent("partial"), code: ErrorIncompleteResponse},
		{name: "refusal", stopReason: "refusal", content: anthropicTextContent("no"), code: ErrorRefused},
		{name: "tool use", stopReason: "tool_use", content: `[{"type":"tool_use","id":"tool-1","name":"x","input":{}}]`, code: ErrorInvalidResponse},
		{name: "pause turn", stopReason: "pause_turn", content: anthropicTextContent("partial"), code: ErrorInvalidResponse},
		{name: "unknown", stopReason: "future_reason", content: anthropicTextContent("partial"), code: ErrorInvalidResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeAnthropicMessage(w, tt.stopReason, tt.content)
			}))
			defer server.Close()
			got, err := newAnthropicTestAdapter(server, "test-model").GenerateContent(context.Background(), "", []Message{NewUserMessage("hi")})
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

func TestAnthropicStreamingTerminalValidation(t *testing.T) {
	tests := []struct {
		name   string
		events []string
		want   string
		think  string
		code   ErrorCode
	}{
		{
			name: "message stop",
			events: []string{
				`{"type":"message_start","message":{"id":"msg-test","type":"message","role":"assistant","model":"test-model","stop_reason":null,"stop_sequence":null,"content":[],"usage":{"input_tokens":1,"output_tokens":0}}}`,
				`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"reason"}}`,
				`{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"answer"}}`,
				`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":1}}`,
				`{"type":"message_stop"}`,
			},
			want: "answer", think: "reason",
		},
		{name: "eof without message stop", events: []string{`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial"}}`, `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":1}}`}, code: ErrorStreamInterrupted},
		{name: "missing message delta", events: []string{`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial"}}`, `{"type":"message_stop"}`}, code: ErrorInvalidResponse},
		{name: "incomplete", events: []string{`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial"}}`, `{"type":"message_delta","delta":{"stop_reason":"max_tokens","stop_sequence":null},"usage":{"output_tokens":1}}`, `{"type":"message_stop"}`}, code: ErrorIncompleteResponse},
		{name: "refusal", events: []string{`{"type":"message_delta","delta":{"stop_reason":"refusal","stop_sequence":null},"usage":{"output_tokens":1}}`, `{"type":"message_stop"}`}, code: ErrorRefused},
		{name: "error response", events: []string{`{"type":"error","error":{"type":"overloaded_error","message":"private payload"}}`}, code: ErrorProviderUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				for _, event := range tt.events {
					eventName := "event"
					if strings.Contains(event, `"type":"error"`) {
						eventName = "error"
					}
					_, _ = w.Write([]byte("event: " + eventName + "\ndata: " + event + "\n\n"))
				}
			}))
			defer server.Close()
			var chunks []StreamChunk
			got, err := newAnthropicTestAdapter(server, "test-model").GenerateContentStream(context.Background(), []Message{NewUserMessage("hi")}, func(chunk StreamChunk) {
				chunks = append(chunks, chunk)
			})
			if tt.code == "" {
				if err != nil || got.Content != tt.want || got.Thinking != tt.think {
					t.Fatalf("got=%+v err=%v", got, err)
				}
				if len(chunks) != 2 || chunks[0].Type != ChunkThinking || chunks[1].Type != ChunkContent {
					t.Fatalf("unexpected chunks: %#v", chunks)
				}
				return
			}
			var providerErr *ProviderError
			if !errors.As(err, &providerErr) || providerErr.Code != tt.code {
				t.Fatalf("want %s, got %v", tt.code, err)
			}
			if strings.Contains(err.Error(), "private payload") {
				t.Fatalf("stream error leaked response text: %v", err)
			}
		})
	}
}

func TestAnthropicIgnoresEnvironmentDefaults(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "environment-secret")
	t.Setenv("ANTHROPIC_BASE_URL", "https://environment.invalid")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer explicit-test-key" {
			t.Fatalf("environment credential was used")
		}
		writeAnthropicMessage(w, "end_turn", anthropicTextContent("ok"))
	}))
	defer server.Close()
	_, err := newAnthropicTestAdapter(server, "test-model").GenerateContent(context.Background(), "", []Message{NewUserMessage("hi")})
	if err != nil || calls.Load() != 1 {
		t.Fatalf("explicit connection not used: calls=%d err=%v", calls.Load(), err)
	}
}

func TestAnthropicDoesNotFollowRedirect(t *testing.T) {
	var targetCalls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetCalls.Add(1)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/stolen", http.StatusFound)
	}))
	defer source.Close()
	_, err := newAnthropicTestAdapter(source, "test-model").GenerateContent(context.Background(), "", []Message{NewUserMessage("hi")})
	if err == nil {
		t.Fatal("redirect response must not be accepted")
	}
	if targetCalls.Load() != 0 {
		t.Fatalf("redirect target received %d requests", targetCalls.Load())
	}
}

func TestAnthropicTestUsesSingleLowCostRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("must not fall back to another protocol: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["max_tokens"] != float64(anthropicProbeMaxTokens) {
			t.Fatalf("unexpected probe body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"private payload"}}`))
	}))
	defer server.Close()
	err := newAnthropicTestAdapter(server, "test-model").TestChat(context.Background())
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != ErrorProviderUnavailable || providerErr.HTTPStatus != 500 {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("SDK retried or adapter fell back: %d requests", calls.Load())
	}
	if strings.Contains(err.Error(), "private payload") {
		t.Fatalf("response body leaked: %v", err)
	}
}

func TestAnthropicGetModelsUsesNativeEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-test","type":"model","display_name":"Test","created_at":"2026-01-01T00:00:00Z","lifecycle":"active","line":"opus","max_input_tokens":1,"max_tokens":1,"capabilities":{}}],"has_more":false,"first_id":"claude-test","last_id":"claude-test"}`))
	}))
	defer server.Close()
	models, err := newAnthropicTestAdapter(server, "test-model").GetModels(context.Background())
	if err != nil || len(models) != 1 || models[0] != "claude-test" {
		t.Fatalf("models=%#v err=%v", models, err)
	}
}

func TestNewProviderRoutesAnthropicMessages(t *testing.T) {
	provider, err := NewProvider(config.Config{APIProtocol: string(protocol.AnthropicMessages)})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := provider.(*AnthropicAdapter); !ok {
		t.Fatalf("expected Anthropic adapter, got %T", provider)
	}
}

func TestAnthropicAdapterKeepsConfigSnapshot(t *testing.T) {
	cfg := config.Config{APIProtocol: string(protocol.AnthropicMessages), APIKey: "first", BaseURL: "https://first.invalid", Model: "first-model"}
	adapter := NewAnthropicAdapter(cfg)
	cfg.APIKey = "second"
	cfg.BaseURL = "https://second.invalid"
	cfg.Model = "second-model"
	if adapter.config.APIKey != "first" || adapter.config.BaseURL != "https://first.invalid" || adapter.config.Model != "first-model" {
		t.Fatalf("adapter observed mutable config: %+v", adapter.config)
	}
}
