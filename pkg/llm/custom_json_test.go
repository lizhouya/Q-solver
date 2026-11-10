package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"Q-Solver/pkg/config"
)

func customJSONTestConfig(requestURL string) config.Config {
	return config.Config{
		ConnectionMode:      config.ConnectionModeDirect,
		APIProtocol:         "custom_json",
		BaseURL:             requestURL,
		Model:               "test-model",
		RequestTemplate:     `{"model":"{{model}}","messages":"{{openai_messages}}"}`,
		ResponseJSONPointer: "/result/text",
		AuthHeader:          "Authorization",
		AuthPrefix:          "Bearer ",
	}
}

func TestCustomJSONGenerateContentPostsRenderedJSONAndExtractsPointer(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/generate" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type=%q", got)
		}
		body := readRequestBodyForTest(t, r)
		if !strings.Contains(body, `"model":"test-model"`) || !strings.Contains(body, `"content":"hello"`) {
			t.Errorf("unexpected body: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"text":"answer"}}`))
	}))
	defer server.Close()

	provider, err := NewProvider(customJSONTestConfig(server.URL + "/generate"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.GenerateContent(context.Background(), "", []Message{NewUserMessage("hello")})
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 || result.Content != "answer" {
		t.Fatalf("requests=%d result=%+v", requests.Load(), result)
	}
}

func TestCustomJSONGenerateContentStreamEmitsOneContentChunk(t *testing.T) {
	server := newCustomJSONResponseServer(t, http.StatusOK, `{"result":{"text":"complete answer"}}`)
	defer server.Close()
	provider, err := NewProvider(customJSONTestConfig(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	var chunks []StreamChunk
	result, err := provider.GenerateContentStream(context.Background(), []Message{NewUserMessage("hello")}, func(chunk StreamChunk) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "complete answer" || len(chunks) != 1 || chunks[0].Type != ChunkContent || chunks[0].Content != result.Content {
		t.Fatalf("result=%+v chunks=%+v", result, chunks)
	}
}

func TestCustomJSONUsesConfigurableAuthHeaderAndPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test-Key"); got != "Token test-only-key" {
			t.Errorf("auth header=%q", got)
		}
		_, _ = w.Write([]byte(`{"result":{"text":"answer"}}`))
	}))
	defer server.Close()
	cfg := customJSONTestConfig(server.URL)
	cfg.APIKey = "test-only-key"
	cfg.AuthHeader = "X-Test-Key"
	cfg.AuthPrefix = "Token "
	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.GenerateContent(context.Background(), "", []Message{NewUserMessage("hello")}); err != nil {
		t.Fatal(err)
	}
}

func TestCustomJSONKeylessOmitsAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-Api-Key") != "" {
			t.Errorf("unexpected auth headers: %#v", r.Header)
		}
		_, _ = w.Write([]byte(`{"result":{"text":"answer"}}`))
	}))
	defer server.Close()
	provider, err := NewProvider(customJSONTestConfig(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.GenerateContent(context.Background(), "", []Message{NewUserMessage("hello")}); err != nil {
		t.Fatal(err)
	}
}

func TestCustomJSONInjectsExtraHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Route"); got != "test-route" {
			t.Errorf("X-Route=%q", got)
		}
		_, _ = w.Write([]byte(`{"result":{"text":"answer"}}`))
	}))
	defer server.Close()
	cfg := customJSONTestConfig(server.URL)
	cfg.ExtraHeaders = map[string]string{"X-Route": "test-route"}
	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.GenerateContent(context.Background(), "", []Message{NewUserMessage("hello")}); err != nil {
		t.Fatal(err)
	}
}

func TestCustomJSONRejectsRedirectWithoutForwardingCredentials(t *testing.T) {
	var targetRequests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequests.Add(1)
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	cfg := customJSONTestConfig(redirect.URL)
	cfg.APIKey = "test-only-key"
	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, requestErr := provider.GenerateContent(context.Background(), "", []Message{NewUserMessage("hello")})
	if requestErr == nil {
		t.Fatal("redirect response was accepted")
	}
	if targetRequests.Load() != 0 {
		t.Fatalf("credentials may have been forwarded to redirect target: requests=%d", targetRequests.Load())
	}
}

func TestCustomJSONClassifiesHTTPFailuresWithoutLeakingBody(t *testing.T) {
	server := newCustomJSONResponseServer(t, http.StatusUnauthorized, `{"error":"test-only-secret-body"}`)
	defer server.Close()
	provider, err := NewProvider(customJSONTestConfig(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, requestErr := provider.GenerateContent(context.Background(), "", []Message{NewUserMessage("hello")})
	assertProviderErrorCode(t, requestErr, ErrorAuthentication)
	if strings.Contains(requestErr.Error(), "test-only-secret-body") {
		t.Fatal("safe error leaked response body")
	}
}

func TestCustomJSONRejectsOversizeInvalidMissingAndEmptyResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
		code ErrorCode
	}{
		{name: "oversize", body: `{"result":{"text":"` + strings.Repeat("x", int(maxCustomJSONResponseBytes)) + `"}}`, code: ErrorRequestTooLarge},
		{name: "invalid json", body: `{"result":`, code: ErrorInvalidResponse},
		{name: "missing pointer", body: `{"result":{}}`, code: ErrorInvalidResponse},
		{name: "empty content", body: `{"result":{"text":"   "}}`, code: ErrorInvalidResponse},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newCustomJSONResponseServer(t, http.StatusOK, test.body)
			defer server.Close()
			provider, err := NewProvider(customJSONTestConfig(server.URL))
			if err != nil {
				t.Fatal(err)
			}
			_, requestErr := provider.GenerateContent(context.Background(), "", []Message{NewUserMessage("hello")})
			assertProviderErrorCode(t, requestErr, test.code)
		})
	}
}

func TestCustomJSONHonorsContextCancellation(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	provider, err := NewProvider(customJSONTestConfig(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, requestErr := provider.GenerateContent(ctx, "", []Message{NewUserMessage("hello")})
	if requestErr == nil {
		t.Fatal("cancelled request returned nil error")
	}
	var providerErr *ProviderError
	if !errors.As(requestErr, &providerErr) || (providerErr.Code != ErrorTimeout && providerErr.Code != ErrorCancelled) {
		t.Fatalf("unexpected cancellation error: %#v", requestErr)
	}
}

func TestCustomJSONGetModelsMakesNoRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()
	provider, err := NewProvider(customJSONTestConfig(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, requestErr := provider.GetModels(context.Background())
	assertProviderErrorCode(t, requestErr, ErrorCatalogUnsupported)
	if requests.Load() != 0 {
		t.Fatalf("GetModels sent %d requests", requests.Load())
	}
}

func newCustomJSONResponseServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func assertProviderErrorCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error", want)
	}
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr == nil || providerErr.Code != want {
		t.Fatalf("error=%v (%T), want code=%s", err, err, want)
	}
}

func readRequestBodyForTest(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestCustomJSONRejectsImagesOmittedByTemplateBeforeNetwork(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"result":{"text":"answer"}}`))
	}))
	defer server.Close()
	cfg := customJSONTestConfig(server.URL)
	cfg.RequestTemplate = `{"prompt":"{{user_text}}","note":"messages not included"}`
	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, err = provider.GenerateContent(context.Background(), "", []Message{NewMultiPartMessage(RoleUser, []ContentPart{ImagePart(templateTestPNG)})})
	if err == nil || calls.Load() != 0 {
		t.Fatalf("omitted image request not rejected: err=%v requests=%d", err, calls.Load())
	}
}

func TestCustomJSONProbeUsesDraftAndReportsLocalRejection(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		token := regexp.MustCompile(`QSP-[A-F0-9]{12}`).Find(body)
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]string{"text": string(token)}})
	}))
	defer server.Close()
	cfg := customJSONTestConfig(server.URL)
	result := (&Service{}).ProbeConnection(context.Background(), cfg, ProbeRequest{Kind: "text"})
	if result.Outcome != ProbePassed || !result.RequestSent || calls.Load() != 1 {
		t.Fatalf("probe=%+v count=%d", result, calls.Load())
	}
	_, err := (&Service{}).ListModelsForConnection(context.Background(), cfg)
	assertProviderErrorCode(t, err, ErrorCatalogUnsupported)
	if calls.Load() != 1 {
		t.Fatal("directory called server")
	}
	cfg.RequestTemplate = `{"text":"{{user_text}}"}`
	result = (&Service{}).ProbeConnection(context.Background(), cfg, ProbeRequest{Kind: "vision"})
	if result.Outcome != ProbeFailed || result.RequestSent || calls.Load() != 1 {
		t.Fatalf("local rejection misreported: %+v", result)
	}
}
