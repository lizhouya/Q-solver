package llm

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func captureRequest(t *testing.T, transport http.RoundTripper, method, rawURL string, headers http.Header) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), method, rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header = headers.Clone()
	var captured *http.Request
	wrapper := transport.(*connectionRoundTripper)
	wrapper.base = roundTripFunc(func(got *http.Request) (*http.Response, error) {
		captured = got
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Request:    got,
		}, nil
	})
	if _, err := wrapper.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if captured == nil {
		t.Fatal("request was not captured")
	}
	return captured
}

func TestConnectionTransportRewritesOnlyInferenceEndpoint(t *testing.T) {
	transport := newConnectionRoundTripper(config.Config{
		BaseURL:      "https://gateway.example.com/v1",
		EndpointPath: "/custom/chat",
	}, protocol.OpenAIChat)

	got := captureRequest(t, transport, http.MethodPost, "https://gateway.example.com/v1/chat/completions", make(http.Header))
	if got.URL.Path != "/v1/custom/chat" {
		t.Fatalf("unexpected rewritten path: %s", got.URL.Path)
	}
}

func TestConnectionTransportUsesProtocolDefaultWhenEndpointEmpty(t *testing.T) {
	transport := newConnectionRoundTripper(config.Config{
		BaseURL: "https://gateway.example.com/v1",
	}, protocol.OpenAIChat)

	got := captureRequest(t, transport, http.MethodPost, "https://gateway.example.com/v1/chat/completions", make(http.Header))
	if got.URL.Path != "/v1/chat/completions" {
		t.Fatalf("unexpected default path: %s", got.URL.Path)
	}
}

func TestConnectionTransportLeavesModelsPathUntouched(t *testing.T) {
	transport := newConnectionRoundTripper(config.Config{
		BaseURL:      "https://gateway.example.com/v1",
		EndpointPath: "/custom/chat",
	}, protocol.OpenAIChat)

	got := captureRequest(t, transport, http.MethodGet, "https://gateway.example.com/v1/models", make(http.Header))
	if got.URL.Path != "/v1/models" {
		t.Fatalf("models path changed: %s", got.URL.Path)
	}
}

func TestConnectionTransportCompletesSDKModelsPathAtAPIRoot(t *testing.T) {
	tests := []struct {
		name, baseURL, requestURL, want string
		protocol                        protocol.APIProtocol
	}{
		{name: "openai", baseURL: "https://gateway.example.com/v1", requestURL: "https://gateway.example.com/v1", want: "/v1/models", protocol: protocol.OpenAIChat},
		{name: "anthropic", baseURL: "https://gateway.example.com", requestURL: "https://gateway.example.com/", want: "/v1/models", protocol: protocol.AnthropicMessages},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := newConnectionRoundTripper(config.Config{BaseURL: tt.baseURL}, tt.protocol)
			got := captureRequest(t, transport, http.MethodGet, tt.requestURL, make(http.Header))
			if got.URL.Path != tt.want {
				t.Fatalf("models path=%s want=%s", got.URL.Path, tt.want)
			}
		})
	}
}

func TestConnectionTransportInjectsExtraHeaders(t *testing.T) {
	transport := newConnectionRoundTripper(config.Config{
		APIKey:       "SET",
		BaseURL:      "https://gateway.example.com/v1",
		EndpointPath: "/chat/completions",
		ExtraHeaders: map[string]string{"X-Route": "route-a", "X-Title": "Q-Solver"},
	}, protocol.OpenAIChat)

	headers := make(http.Header)
	headers.Set("Authorization", "Bearer SET")
	got := captureRequest(t, transport, http.MethodPost, "https://gateway.example.com/v1/chat/completions", headers)
	if got.Header.Get("Authorization") != "Bearer SET" || got.Header.Get("X-Route") != "route-a" || got.Header.Get("X-Title") != "Q-Solver" {
		t.Fatalf("headers not applied: %#v", got.Header)
	}
}

func TestConnectionTransportRemovesEmptyDefaultAuthorization(t *testing.T) {
	transport := newConnectionRoundTripper(config.Config{
		BaseURL:      "http://127.0.0.1:11434/v1",
		EndpointPath: "/chat/completions",
	}, protocol.OpenAIChat)

	headers := make(http.Header)
	headers.Set("Authorization", "Bearer ")
	headers.Set("X-Api-Key", "")
	got := captureRequest(t, transport, http.MethodPost, "http://127.0.0.1:11434/v1/chat/completions", headers)
	if got.Header.Get("Authorization") != "" || got.Header.Get("X-Api-Key") != "" {
		t.Fatalf("empty default auth was not removed: %#v", got.Header)
	}
}

func TestConnectionTransportAllowsCustomAuthorizationWhenKeyEmpty(t *testing.T) {
	transport := newConnectionRoundTripper(config.Config{
		BaseURL:      "http://127.0.0.1:11434/v1",
		EndpointPath: "/chat/completions",
		ExtraHeaders: map[string]string{"Authorization": "Custom SET"},
	}, protocol.OpenAIChat)

	headers := make(http.Header)
	headers.Set("Authorization", "Bearer ")
	got := captureRequest(t, transport, http.MethodPost, "http://127.0.0.1:11434/v1/chat/completions", headers)
	if got.Header.Get("Authorization") != "Custom SET" {
		t.Fatalf("custom authorization missing: %#v", got.Header)
	}
}

func TestProviderHTTPClientStillRejectsRedirects(t *testing.T) {
	client := newProviderHTTPClient(config.Config{
		BaseURL:      "https://gateway.example.com/v1",
		EndpointPath: "/chat/completions",
	}, protocol.OpenAIChat)
	request, err := http.NewRequest(http.MethodGet, "https://gateway.example.com/v1/models", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.CheckRedirect(request, nil); err != http.ErrUseLastResponse {
		t.Fatalf("redirect policy changed: %v", err)
	}
}
