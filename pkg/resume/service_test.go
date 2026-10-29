package resume

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"Q-Solver/pkg/config"
)

func TestParseResumeUsesExplicitAnthropicProviderAndNativePDFBlock(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg-test","type":"message","role":"assistant","model":"test-model","stop_reason":"end_turn","stop_sequence":null,"content":[{"type":"text","text":"# Resume"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "resume.pdf")
	if err := os.WriteFile(path, []byte("synthetic-pdf"), 0600); err != nil {
		t.Fatal(err)
	}
	service := &Service{config: config.Config{
		APIKey: "test-key", BaseURL: server.URL, Model: "test-model", APIProtocol: "anthropic_messages", ResumePath: path,
	}}
	got, err := service.ParseResume(context.Background())
	if err != nil || got != "# Resume" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	messages := body["messages"].([]any)
	parts := messages[0].(map[string]any)["content"].([]any)
	if len(parts) != 2 || parts[1].(map[string]any)["type"] != "document" {
		t.Fatalf("native PDF block missing: %#v", body)
	}
	source := parts[1].(map[string]any)["source"].(map[string]any)
	if source["type"] != "base64" || source["media_type"] != "application/pdf" || source["data"] == "" {
		t.Fatalf("unexpected PDF source: %#v", source)
	}
}
