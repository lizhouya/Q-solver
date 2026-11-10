package config

import (
	"strings"
	"testing"
)

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
	if normalizedCCR.ConnectionMode != ConnectionModeDirect || normalizedCCR.APIProtocol != "openai_chat" || normalizedCCR.BaseURL != "https://gateway.example.com/v1" || normalizedCCR.EndpointPath != "/chat/completions" {
		t.Fatalf("legacy CCR settings were not migrated: %+v", normalizedCCR)
	}
	if ccr.APIProtocol != "anthropic_messages" || ccr.BaseURL != "https://gateway.example.com/" {
		t.Fatal("normalization mutated the source config")
	}
}

func TestUnifiedConnectionDefaultsEndpointByProtocol(t *testing.T) {
	tests := []struct {
		protocol string
		want     string
	}{
		{protocol: "openai_chat", want: "/chat/completions"},
		{protocol: "openai_responses", want: "/responses"},
		{protocol: "anthropic_messages", want: "/v1/messages"},
	}
	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			got, err := (Config{ConnectionMode: ConnectionModeDirect, APIProtocol: tt.protocol, BaseURL: "http://127.0.0.1:1234/v1"}).NormalizedConnectionSettings()
			if err != nil {
				t.Fatal(err)
			}
			if got.EndpointPath != tt.want || got.ResponseFormat != ResponseFormatText {
				t.Fatalf("unexpected normalized connection: %+v", got)
			}
		})
	}
}

func TestUnifiedConnectionAcceptsKeylessLocalHTTP(t *testing.T) {
	got, err := (Config{
		ConnectionMode: ConnectionModeDirect,
		APIProtocol:    "openai_chat",
		BaseURL:        " http://127.0.0.1:11434/v1/ ",
		Model:          "local-model",
	}).NormalizedConnectionSettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.APIKey != "" || got.BaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("keyless local connection changed unexpectedly: %+v", got)
	}
}

func TestUnifiedConnectionRejectsInvalidEndpoint(t *testing.T) {
	for _, endpoint := range []string{
		"chat/completions",
		"https://evil.invalid/chat",
		"/chat/completions?route=x",
		"/chat/completions#route",
		"/chat\r\nX-Test: value",
	} {
		t.Run(endpoint, func(t *testing.T) {
			_, err := (Config{APIProtocol: "openai_chat", BaseURL: "https://api.example.com/v1", EndpointPath: endpoint}).NormalizedConnectionSettings()
			if err == nil {
				t.Fatalf("accepted unsafe endpoint %q", endpoint)
			}
		})
	}
}

func TestUnifiedConnectionRejectsUnsafeHeaders(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		headers map[string]string
	}{
		{name: "host", headers: map[string]string{"Host": "other.invalid"}},
		{name: "content length", headers: map[string]string{"Content-Length": "10"}},
		{name: "connection", headers: map[string]string{"Connection": "close"}},
		{name: "invalid name", headers: map[string]string{"Bad Header": "value"}},
		{name: "newline value", headers: map[string]string{"X-Test": "one\r\ntwo"}},
		{name: "authorization conflict", apiKey: "SET", headers: map[string]string{"Authorization": "SET"}},
		{name: "anthropic key conflict", apiKey: "SET", headers: map[string]string{"x-api-key": "SET"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (Config{APIKey: tt.apiKey, APIProtocol: "openai_chat", BaseURL: "https://api.example.com/v1", ExtraHeaders: tt.headers}).NormalizedConnectionSettings()
			if err == nil {
				t.Fatalf("accepted unsafe headers for %s", tt.name)
			}
		})
	}
}

func TestUnifiedConnectionAllowsCustomAuthorizationWhenKeyEmpty(t *testing.T) {
	input := map[string]string{"Authorization": "SET", "X-Route": "route-a"}
	got, err := (Config{APIProtocol: "openai_chat", BaseURL: "http://127.0.0.1:1234/v1", ExtraHeaders: input}).NormalizedConnectionSettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.ExtraHeaders["Authorization"] != "SET" || got.ExtraHeaders["X-Route"] != "route-a" {
		t.Fatalf("custom auth headers were not preserved: %#v", got.ExtraHeaders)
	}
	got.ExtraHeaders["X-Route"] = "changed"
	if input["X-Route"] != "route-a" {
		t.Fatal("normalization must deep-copy extra headers")
	}
}

func TestUnifiedConnectionRejectsResponseFormatOutsideChat(t *testing.T) {
	for _, protocolValue := range []string{"openai_responses", "anthropic_messages"} {
		_, err := (Config{
			APIProtocol:    protocolValue,
			BaseURL:        "https://api.example.com/v1",
			ResponseFormat: ResponseFormatJSONObject,
		}).NormalizedConnectionSettings()
		if err == nil {
			t.Fatalf("%s accepted json_object response format", protocolValue)
		}
	}
}

func TestCustomJSONDefaultsAuthAndRequiresTemplate(t *testing.T) {
	cfg := Config{
		ConnectionMode:      ConnectionModeDirect,
		APIProtocol:         "custom_json",
		BaseURL:             "https://gateway.example.com/generate",
		Model:               "test-model",
		RequestTemplate:     `{"model":"{{model}}","messages":"{{openai_messages}}"}`,
		ResponseJSONPointer: "/answer/text",
	}
	got, err := cfg.NormalizedConnectionSettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.BaseURL != cfg.BaseURL || got.EndpointPath != "" || got.ResponseFormat != ResponseFormatText {
		t.Fatalf("unexpected custom connection: %+v", got)
	}
	if got.AuthHeader != DefaultCustomAuthHeader || got.AuthPrefix != DefaultCustomAuthPrefix {
		t.Fatalf("unexpected auth defaults: header=%q prefix=%q", got.AuthHeader, got.AuthPrefix)
	}

	cfg.RequestTemplate = ""
	if _, err := cfg.NormalizedConnectionSettings(); err == nil {
		t.Fatal("custom JSON accepted an empty request template")
	}
}

func TestCustomJSONAcceptsKeylessHTTPAndHTTPS(t *testing.T) {
	for _, address := range []string{
		"http://127.0.0.1:11434/generate",
		"https://gateway.example.com/v2/generate",
	} {
		t.Run(address, func(t *testing.T) {
			got, err := (Config{
				APIProtocol:         "custom_json",
				BaseURL:             address,
				Model:               "test-model",
				RequestTemplate:     `{"messages":"{{openai_messages}}"}`,
				ResponseJSONPointer: "/answer",
			}).NormalizedConnectionSettings()
			if err != nil {
				t.Fatal(err)
			}
			if got.APIKey != "" || got.BaseURL != address {
				t.Fatalf("keyless custom connection changed unexpectedly: %+v", got)
			}
		})
	}
}

func TestCustomJSONRejectsUnsafeRequestURLs(t *testing.T) {
	for _, address := range []string{
		"gateway.example.com/generate",
		"file:///tmp/generate",
		"https://user:password@gateway.example.com/generate",
		"https://gateway.example.com/generate?token=value",
		"https://gateway.example.com/generate#fragment",
		"https://gateway.example.com/generate\r\nX-Test: value",
	} {
		t.Run(address, func(t *testing.T) {
			_, err := (Config{
				APIProtocol:         "custom_json",
				BaseURL:             address,
				RequestTemplate:     `{"messages":"{{openai_messages}}"}`,
				ResponseJSONPointer: "/answer",
			}).NormalizedConnectionSettings()
			if err == nil {
				t.Fatalf("accepted unsafe custom request URL %q", address)
			}
		})
	}
}

func TestCustomJSONRejectsInvalidAuthAndHeaderConflict(t *testing.T) {
	tests := []Config{
		{AuthHeader: "Bad Header"},
		{AuthHeader: "Host"},
		{AuthPrefix: "Bearer\r\nX-Test: value"},
		{APIKey: "test-only-key", AuthHeader: "X-Custom-Key", ExtraHeaders: map[string]string{"x-custom-key": "test-only-duplicate"}},
	}
	for i, partial := range tests {
		partial.APIProtocol = "custom_json"
		partial.BaseURL = "https://gateway.example.com/generate"
		partial.RequestTemplate = `{"messages":"{{openai_messages}}"}`
		partial.ResponseJSONPointer = "/answer"
		if _, err := partial.NormalizedConnectionSettings(); err == nil {
			t.Fatalf("case %d accepted invalid auth settings", i)
		}
	}
}

func TestOpenAIChatAddressAcceptsRootOrFullEndpoint(t *testing.T) {
	tests := []struct {
		address  string
		wantBase string
	}{
		{address: "https://api.example.com/v1", wantBase: "https://api.example.com/v1"},
		{address: "https://api.example.com/v1/chat/completions", wantBase: "https://api.example.com/v1"},
	}
	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			got, err := (Config{APIProtocol: "openai_chat", BaseURL: tt.address}).NormalizedConnectionSettings()
			if err != nil {
				t.Fatal(err)
			}
			if got.BaseURL != tt.wantBase || got.EndpointPath != "/chat/completions" {
				t.Fatalf("got base=%q endpoint=%q", got.BaseURL, got.EndpointPath)
			}
		})
	}
}

func TestLegacyProtocolsRemainValidBeforeTemplateMigration(t *testing.T) {
	for _, protocolValue := range []string{"openai_responses", "anthropic_messages"} {
		cfg := Config{APIProtocol: protocolValue, BaseURL: "https://api.example.com", Model: "test-model"}
		got, err := cfg.NormalizedConnectionSettings()
		if err != nil {
			t.Fatal(err)
		}
		if got.APIProtocol != protocolValue {
			t.Fatalf("legacy protocol migrated during normalization: %q", got.APIProtocol)
		}
	}
}

func TestRequestTemplateRejectsOversizeNonObjectAndInvalidJSON(t *testing.T) {
	tests := []string{
		`[]`,
		`{"messages":`,
		`{"messages":"{{openai_messages}}"} trailing`,
		strings.Repeat("x", MaxRequestTemplateBytes+1),
	}
	for i, template := range tests {
		if err := ValidateRequestTemplateConfig(template, "/answer"); err == nil {
			t.Fatalf("case %d accepted invalid template", i)
		}
	}
}

func TestRequestTemplateRejectsUnknownEmbeddedAndKeyPlaceholders(t *testing.T) {
	tests := []string{
		`{"body":"{{unknown}}"}`,
		`{"body":"prefix-{{model}}","messages":"{{openai_messages}}"}`,
		`{"{{model}}":"value","messages":"{{openai_messages}}"}`,
	}
	for i, template := range tests {
		if err := ValidateRequestTemplateConfig(template, "/answer"); err == nil {
			t.Fatalf("case %d accepted unsafe placeholder", i)
		}
	}
}

func TestRequestTemplateRequiresContentPlaceholder(t *testing.T) {
	if err := ValidateRequestTemplateConfig(`{"model":"{{model}}"}`, "/answer"); err == nil {
		t.Fatal("template without content placeholder was accepted")
	}
	if err := ValidateRequestTemplateConfig(`{"model":"{{model}}","messages":"{{openai_messages}}"}`, "/answer"); err != nil {
		t.Fatalf("valid template rejected: %v", err)
	}
}

func TestJSONPointerValidation(t *testing.T) {
	for _, pointer := range []string{"/answer", "/items/0/text", "/a~1b/~0value"} {
		if err := ValidateJSONPointer(pointer); err != nil {
			t.Fatalf("valid pointer %q rejected: %v", pointer, err)
		}
	}
	for _, pointer := range []string{"", "answer/text", "/bad~2escape", "/trailing~"} {
		if err := ValidateJSONPointer(pointer); err == nil {
			t.Fatalf("invalid pointer %q accepted", pointer)
		}
	}
}

func TestCustomURLPreservesEscapesAndTrailingSlash(t *testing.T) {
	for _, raw := range []string{"https://example.invalid/api%2Froute/", "https://example.invalid/generate/"} {
		got, err := NormalizeCustomRequestURL(raw)
		if err != nil || got != raw {
			t.Fatalf("path changed: %q %v", got, err)
		}
	}
}
func TestTemplateRejectsDeepNesting(t *testing.T) {
	raw := `{"messages":"{{openai_messages}}","deep":` + strings.Repeat("[", 70) + "0" + strings.Repeat("]", 70) + "}"
	if ValidateRequestTemplateConfig(raw, "/answer") == nil {
		t.Fatal("unbounded template depth accepted")
	}
}
func TestExtraHeadersRejectsDuplicateCaseAndControlBytes(t *testing.T) {
	for _, h := range []map[string]string{{"X-Route": "a", "x-route": "b"}, {"X-Route": "a\x00b"}} {
		if _, err := NormalizeExtraHeaders(h, "", "openai_chat"); err == nil {
			t.Fatal("ambiguous or invalid header accepted")
		}
	}
}
