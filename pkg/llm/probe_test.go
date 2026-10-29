package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"
)

type probeFakeProvider struct {
	generateCalls int
	streamCalls   int
	messages      []Message
	response      Message
	err           error
}

func (p *probeFakeProvider) GenerateContent(ctx context.Context, model string, messages []Message) (Message, error) {
	p.generateCalls++
	p.messages = messages
	return p.response, p.err
}

func (p *probeFakeProvider) GenerateContentStream(ctx context.Context, messages []Message, onChunk StreamCallback) (Message, error) {
	p.streamCalls++
	p.messages = messages
	return p.response, p.err
}

func (p *probeFakeProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p *probeFakeProvider) TestChat(context.Context) error              { return nil }

func TestTextProbeRequiresExactRandomTokenAndOneNonStreamingRequest(t *testing.T) {
	const token = "QSP-1234ABCD"
	provider := &probeFakeProvider{response: Message{Role: RoleAssistant, Content: "  " + token + "\n"}}
	result := probeTextWithToken(context.Background(), provider, protocol.OpenAIChat, "test-model", token)
	if result.Outcome != ProbePassed || !result.RequestSent || result.Error != nil {
		t.Fatalf("unexpected result: %+v", result)
	}
	if provider.generateCalls != 1 || provider.streamCalls != 0 || len(provider.messages) != 1 {
		t.Fatalf("unexpected calls: generate=%d stream=%d messages=%d", provider.generateCalls, provider.streamCalls, len(provider.messages))
	}
	if !strings.Contains(provider.messages[0].Content, token) {
		t.Fatal("text probe prompt must contain its random challenge token")
	}

	provider = &probeFakeProvider{response: Message{Role: RoleAssistant, Content: token + " extra"}}
	result = probeTextWithToken(context.Background(), provider, protocol.OpenAIChat, "test-model", token)
	if result.Outcome != ProbeFailed || result.Error == nil || result.Error.Code != ErrorInvalidResponse {
		t.Fatalf("non-exact token accepted: %+v", result)
	}
}

func TestVisionProbeUsesSyntheticPNGStreamingAndDoesNotRevealAnswer(t *testing.T) {
	imageData, err := renderSevenSegmentPNG("42")
	if err != nil {
		t.Fatal(err)
	}
	provider := &probeFakeProvider{response: Message{Role: RoleAssistant, Content: "DIGITS: ４２"}}
	result := probeVisionWithChallenge(context.Background(), provider, protocol.OpenAIResponses, "test-model", "42", imageData)
	if result.Outcome != ProbePassed || !result.RequestSent || result.Error != nil {
		t.Fatalf("unexpected result: %+v", result)
	}
	if provider.generateCalls != 0 || provider.streamCalls != 1 || len(provider.messages) != 1 {
		t.Fatalf("vision probe did not use exactly one streaming request: %+v", provider)
	}
	msg := provider.messages[0]
	if strings.Contains(msg.Content, "42") || len(msg.Parts) != 2 || strings.Contains(msg.Parts[0].Text, "42") {
		t.Fatalf("vision prompt leaked the answer: %+v", msg)
	}
	if msg.Parts[1].Type != ContentImage || !strings.HasPrefix(msg.Parts[1].Base64, "data:image/png;base64,") {
		t.Fatalf("missing synthetic PNG image part: %+v", msg.Parts)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(msg.Parts[1].Base64, "data:image/png;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() < 40 || decoded.Bounds().Dy() < 40 {
		t.Fatalf("generated PNG too small: %v", decoded.Bounds())
	}
}

func TestVisionProbeSeparatesMismatchAndInconclusive(t *testing.T) {
	imageData, err := renderSevenSegmentPNG("42")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		response string
		outcome  ProbeOutcome
		code     ErrorCode
	}{
		{response: "DIGITS: 17", outcome: ProbeFailed, code: ErrorVisionMismatch},
		{response: "I think it might be 42.", outcome: ProbeInconclusive, code: ErrorVisionInconclusive},
		{response: "DIGITS: 4", outcome: ProbeInconclusive, code: ErrorVisionInconclusive},
	}
	for _, tt := range tests {
		provider := &probeFakeProvider{response: Message{Role: RoleAssistant, Content: tt.response}}
		got := probeVisionWithChallenge(context.Background(), provider, protocol.AnthropicMessages, "test-model", "42", imageData)
		if got.Outcome != tt.outcome || got.Error == nil || got.Error.Code != tt.code || provider.streamCalls != 1 {
			t.Fatalf("response=%q result=%+v calls=%d", tt.response, got, provider.streamCalls)
		}
	}
}

func TestProbeProviderRejectsInvalidKindBeforeRequest(t *testing.T) {
	provider := &probeFakeProvider{}
	result := ProbeProvider(context.Background(), provider, protocol.OpenAIChat, "test-model", ProbeRequest{Kind: "audio"})
	if result.RequestSent || result.Outcome != ProbeFailed || result.Error == nil || result.Error.Code != ErrorInvalidRequest {
		t.Fatalf("unexpected local validation result: %+v", result)
	}
	if provider.generateCalls != 0 || provider.streamCalls != 0 {
		t.Fatal("invalid probe kind reached provider")
	}
}

func TestVisionProbeTraversesChatStreamingMultimodalAdapterOnce(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		messages := body["messages"].([]any)
		parts := messages[0].(map[string]any)["content"].([]any)
		if len(parts) != 2 || parts[0].(map[string]any)["type"] != "text" || parts[1].(map[string]any)["type"] != "image_url" {
			t.Fatalf("unexpected multimodal body: %#v", body)
		}
		prompt := parts[0].(map[string]any)["text"].(string)
		if strings.Contains(prompt, "42") || strings.Contains(prompt, "private-user-history") {
			t.Fatalf("probe prompt leaked answer or user data: %q", prompt)
		}
		imageURL := parts[1].(map[string]any)["image_url"].(map[string]any)["url"].(string)
		if !strings.HasPrefix(imageURL, "data:image/png;base64,") {
			t.Fatal("probe image was not transmitted")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"DIGITS: 42\"},\"finish_reason\":\"\"}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	imageData, err := renderSevenSegmentPNG("42")
	if err != nil {
		t.Fatal(err)
	}
	adapter := NewOpenAIAdapter(config.Config{APIKey: "test-key", BaseURL: server.URL + "/v1", Model: "test-model", APIProtocol: "openai_chat"})
	result := probeVisionWithChallenge(context.Background(), adapter, protocol.OpenAIChat, "test-model", "42", imageData)
	if result.Outcome != ProbePassed || !result.RequestSent || result.Error != nil || calls.Load() != 1 {
		t.Fatalf("result=%+v calls=%d", result, calls.Load())
	}
}
