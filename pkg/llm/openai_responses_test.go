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

func newOpenAIResponsesTestAdapter(server *httptest.Server, model string) *OpenAIResponsesAdapter {
	return NewOpenAIResponsesAdapter(config.Config{
		APIKey:      "test-key",
		BaseURL:     server.URL + "/v1",
		APIProtocol: string(protocol.OpenAIResponses),
		Model:       model,
	})
}

func writeResponsesResponse(w http.ResponseWriter, status, output string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"id":"resp-test","object":"response","status":"` + status + `","model":"test","output":` + output + `}`))
}

func completedResponsesOutput(text string) string {
	encoded, _ := json.Marshal(text)
	return `[{"id":"msg-test","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":` + string(encoded) + `,"annotations":[]}]}]`
}

func TestOpenAIResponsesRequestUsesStatelessMultimodalInput(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString([]byte("png"))
	pdfData := base64.StdEncoding.EncodeToString([]byte("pdf"))
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		writeResponsesResponse(w, "completed", completedResponsesOutput("ok"))
	}))
	defer server.Close()

	adapter := newOpenAIResponsesTestAdapter(server, "test-model")
	got, err := adapter.GenerateContent(context.Background(), "", []Message{
		NewSystemMessage("system"),
		NewMultiPartMessage(RoleUser, []ContentPart{
			TextPart("question"),
			ImagePart("data:image/png;base64," + pngData),
			PDFPart(pdfData),
		}),
		NewAssistantMessage("prior"),
	})
	if err != nil || got.Content != "ok" {
		t.Fatalf("got=%+v err=%v", got, err)
	}

	if body["store"] != false {
		t.Fatalf("responses request must set store=false: %#v", body)
	}
	if _, ok := body["previous_response_id"]; ok {
		t.Fatalf("stateless request sent previous_response_id: %#v", body)
	}
	input := body["input"].([]any)
	if len(input) != 3 {
		t.Fatalf("unexpected input length: %#v", input)
	}
	if input[0].(map[string]any)["role"] != "system" || input[1].(map[string]any)["role"] != "user" || input[2].(map[string]any)["role"] != "assistant" {
		t.Fatalf("history order changed: %#v", input)
	}
	parts := input[1].(map[string]any)["content"].([]any)
	if parts[0].(map[string]any)["type"] != "input_text" || parts[1].(map[string]any)["type"] != "input_image" || parts[2].(map[string]any)["type"] != "input_file" {
		t.Fatalf("unexpected multipart shape: %#v", parts)
	}
	if parts[1].(map[string]any)["image_url"] != "data:image/png;base64,"+pngData {
		t.Fatalf("unexpected image part: %#v", parts[1])
	}
	file := parts[2].(map[string]any)
	if file["file_data"] != "data:application/pdf;base64,"+pdfData || file["filename"] != "document.pdf" {
		t.Fatalf("unexpected file part: %#v", file)
	}
}

func TestOpenAIResponsesRejectsInvalidDataBeforeRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer server.Close()
	adapter := newOpenAIResponsesTestAdapter(server, "test-model")

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

func TestOpenAIResponsesNonStreamingTerminalValidation(t *testing.T) {
	tests := []struct {
		name, status, output, want string
		code                       ErrorCode
	}{
		{name: "completed", status: "completed", output: completedResponsesOutput("answer"), want: "answer"},
		{name: "empty", status: "completed", output: `[]`, code: ErrorEmptyResponse},
		{name: "refusal", status: "completed", output: `[{"id":"msg-test","type":"message","role":"assistant","status":"completed","content":[{"type":"refusal","refusal":"no"}]}]`, code: ErrorRefused},
		{name: "failed", status: "failed", output: `[]`, code: ErrorResponseFailed},
		{name: "incomplete", status: "incomplete", output: completedResponsesOutput("partial"), code: ErrorIncompleteResponse},
		{name: "in progress", status: "in_progress", output: completedResponsesOutput("partial"), code: ErrorInvalidResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeResponsesResponse(w, tt.status, tt.output)
			}))
			defer server.Close()
			got, err := newOpenAIResponsesTestAdapter(server, "test-model").GenerateContent(context.Background(), "", []Message{NewUserMessage("hi")})
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

func TestOpenAIResponsesStreamingTerminalValidation(t *testing.T) {
	tests := []struct {
		name   string
		events []string
		want   string
		think  string
		code   ErrorCode
	}{
		{
			name: "completed",
			events: []string{
				`{"type":"response.reasoning_summary_text.delta","delta":"reason"}`,
				`{"type":"response.output_text.delta","delta":"answer"}`,
				`{"type":"response.completed","response":{"id":"resp-test","object":"response","status":"completed","output":` + completedResponsesOutput("answer") + `}}`,
			},
			want: "answer", think: "reason",
		},
		{name: "eof without terminal", events: []string{`{"type":"response.output_text.delta","delta":"partial"}`}, code: ErrorStreamInterrupted},
		{name: "failed", events: []string{`{"type":"response.failed","response":{"id":"resp-test","object":"response","status":"failed","output":[]}}`}, code: ErrorResponseFailed},
		{name: "incomplete", events: []string{`{"type":"response.incomplete","response":{"id":"resp-test","object":"response","status":"incomplete","output":[]}}`}, code: ErrorIncompleteResponse},
		{name: "error event", events: []string{`{"type":"error","code":"server_error","message":"private payload","param":""}`}, code: ErrorProviderUnavailable},
		{name: "refusal", events: []string{`{"type":"response.refusal.delta","delta":"no"}`, `{"type":"response.completed","response":{"id":"resp-test","object":"response","status":"completed","output":[]}}`}, code: ErrorRefused},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				for _, event := range tt.events {
					_, _ = w.Write([]byte("event: event\ndata: " + event + "\n\n"))
				}
			}))
			defer server.Close()
			var chunks []StreamChunk
			got, err := newOpenAIResponsesTestAdapter(server, "test-model").GenerateContentStream(context.Background(), []Message{NewUserMessage("hi")}, func(chunk StreamChunk) {
				chunks = append(chunks, chunk)
			})
			if tt.code == "" {
				if err != nil || got.Content != tt.want || got.Thinking != tt.think {
					t.Fatalf("got=%+v err=%v", got, err)
				}
				if len(chunks) != 2 || chunks[0].Type != ChunkThinking || chunks[1].Type != ChunkContent {
					t.Fatalf("unexpected chunks (completed response may have been duplicated): %#v", chunks)
				}
				return
			}
			var providerErr *ProviderError
			if !errors.As(err, &providerErr) || providerErr.Code != tt.code {
				t.Fatalf("want %s, got %v", tt.code, err)
			}
			if err != nil && strings.Contains(err.Error(), "private payload") {
				t.Fatalf("stream error leaked response text: %v", err)
			}
		})
	}
}

func TestOpenAIResponsesTestUsesSingleLowCostRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path == "/v1/chat/completions" {
			t.Fatal("responses failure must not fall back to Chat Completions")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["max_output_tokens"] != float64(32) || body["store"] != false {
			t.Fatalf("unexpected probe body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"private payload","type":"server_error","code":"server_error"}}`))
	}))
	defer server.Close()

	err := newOpenAIResponsesTestAdapter(server, "test-model").TestChat(context.Background())
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != ErrorProviderUnavailable {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("SDK retried or adapter fell back: %d requests", calls.Load())
	}
	if strings.Contains(err.Error(), "private payload") {
		t.Fatalf("response body leaked: %v", err)
	}
}

func TestNewProviderRoutesOpenAIResponses(t *testing.T) {
	provider, err := NewProvider(config.Config{APIProtocol: string(protocol.OpenAIResponses)})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := provider.(*OpenAIResponsesAdapter); !ok {
		t.Fatalf("expected Responses adapter, got %T", provider)
	}
}

func TestOpenAIResponsesAdapterKeepsConfigSnapshot(t *testing.T) {
	cfg := config.Config{APIProtocol: string(protocol.OpenAIResponses), APIKey: "first", BaseURL: "https://first.invalid/v1", Model: "first-model"}
	adapter := NewOpenAIResponsesAdapter(cfg)
	cfg.APIKey = "second"
	cfg.BaseURL = "https://second.invalid/v1"
	cfg.Model = "second-model"
	if adapter.config.APIKey != "first" || adapter.config.BaseURL != "https://first.invalid/v1" || adapter.config.Model != "first-model" {
		t.Fatalf("adapter observed mutable config: %+v", adapter.config)
	}
}
