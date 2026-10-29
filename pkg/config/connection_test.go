package config

import "testing"

func TestNormalizeConnectionMode(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "legacy empty", value: "", want: ConnectionModeDirect},
		{name: "direct", value: ConnectionModeDirect, want: ConnectionModeDirect},
		{name: "ccr", value: ConnectionModeCCR, want: ConnectionModeCCR},
		{name: "trimmed ccr", value: "  ccr  ", want: ConnectionModeCCR},
		{name: "model name is not a mode", value: "claude-opus-5", wantErr: true},
		{name: "brand is not a mode", value: "anthropic", wantErr: true},
		{name: "url is not a mode", value: "https://gateway.example.com/v1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeConnectionMode(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeConnectionMode(%q) accepted unknown mode %q", tt.value, got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("NormalizeConnectionMode(%q) = %q, %v; want %q", tt.value, got, err, tt.want)
			}
		})
	}
}

func TestNormalizeCCRBaseURL(t *testing.T) {
	accepted := map[string]string{
		"":                                   "",
		"http://127.0.0.1:3456":              "http://127.0.0.1:3456/v1",
		"http://127.0.0.1:3456/":             "http://127.0.0.1:3456/v1",
		"http://127.0.0.1:3456/v1":           "http://127.0.0.1:3456/v1",
		"http://127.0.0.1:3456/v1/":          "http://127.0.0.1:3456/v1",
		"https://gateway.example.com":        "https://gateway.example.com/v1",
		"https://gateway.example.com/v1":     "https://gateway.example.com/v1",
		"  https://gateway.example.com/v1/ ": "https://gateway.example.com/v1",
	}
	for input, want := range accepted {
		t.Run("accept_"+input, func(t *testing.T) {
			got, err := NormalizeCCRBaseURL(input)
			if err != nil || got != want {
				t.Fatalf("NormalizeCCRBaseURL(%q) = %q, %v; want %q", input, got, err, want)
			}
		})
	}

	rejected := []string{
		"127.0.0.1:3456",
		"file:///tmp/ccr",
		"https://user:password@gateway.example.com/v1",
		"https://gateway.example.com/v1?route=x",
		"https://gateway.example.com/v1#route",
		"https://gateway.example.com/v1/models",
		"https://gateway.example.com/v1/chat/completions",
		"https://gateway.example.com/v1/responses",
		"https://gateway.example.com/extra",
		"https:///v1",
	}
	for _, input := range rejected {
		t.Run("reject_"+input, func(t *testing.T) {
			if got, err := NormalizeCCRBaseURL(input); err == nil {
				t.Fatalf("NormalizeCCRBaseURL(%q) accepted as %q", input, got)
			}
		})
	}
}

func TestNormalizedConnectionSettings(t *testing.T) {
	direct := Config{
		ConnectionMode: ConnectionModeDirect,
		APIProtocol:    "anthropic_messages",
		BaseURL:        "https://direct.example.com/custom/path",
		Model:          "claude-opus-5",
	}
	normalizedDirect, err := direct.NormalizedConnectionSettings()
	if err != nil {
		t.Fatal(err)
	}
	if normalizedDirect.ConnectionMode != ConnectionModeDirect || normalizedDirect.APIProtocol != "anthropic_messages" || normalizedDirect.BaseURL != direct.BaseURL {
		t.Fatalf("direct settings changed unexpectedly: %+v", normalizedDirect)
	}

	ccr := Config{
		ConnectionMode: ConnectionModeCCR,
		APIProtocol:    "anthropic_messages",
		BaseURL:        "https://gateway.example.com/",
		Model:          "gpt-or-claude-name-does-not-select-protocol",
	}
	normalizedCCR, err := ccr.NormalizedConnectionSettings()
	if err != nil {
		t.Fatal(err)
	}
	if normalizedCCR.ConnectionMode != ConnectionModeCCR || normalizedCCR.APIProtocol != "openai_chat" || normalizedCCR.BaseURL != "https://gateway.example.com/v1" {
		t.Fatalf("CCR settings were not canonicalized: %+v", normalizedCCR)
	}
	if ccr.APIProtocol != "anthropic_messages" || ccr.BaseURL != "https://gateway.example.com/" {
		t.Fatal("normalization mutated the source config")
	}
}
