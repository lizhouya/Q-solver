package protocol

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    APIProtocol
		wantErr bool
	}{
		{name: "legacy empty defaults to chat", input: "", want: OpenAIChat},
		{name: "chat", input: "openai_chat", want: OpenAIChat},
		{name: "responses", input: "openai_responses", want: OpenAIResponses},
		{name: "anthropic", input: "anthropic_messages", want: AnthropicMessages},
		{name: "model name is not a protocol", input: "claude-opus-5", wantErr: true},
		{name: "provider name is not a protocol", input: "openai", wantErr: true},
		{name: "url is not a protocol", input: "https://api.openai.com/v1", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Normalize(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("Normalize(%q) 应拒绝未知协议", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q) 返回错误: %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("Normalize(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
