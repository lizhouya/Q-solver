package llm

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const templateTestPNG = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9ZQmcAAAAASUVORK5CYII="

func TestBuildTemplatePayloadsProjectsFullConversation(t *testing.T) {
	messages := []Message{
		NewSystemMessage("system-one"),
		NewUserMessage("earlier question"),
		NewAssistantMessage("earlier answer"),
		NewMultiPartMessage(RoleUser, []ContentPart{TextPart("current question"), ImagePart(templateTestPNG)}),
	}
	payloads, err := buildTemplatePayloads("test-model", messages)
	if err != nil {
		t.Fatal(err)
	}
	if payloads.Model != "test-model" || payloads.SystemPrompt != "system-one" || payloads.UserText != "current question" {
		t.Fatalf("unexpected scalar payloads: %+v", payloads)
	}
	if len(payloads.OpenAIMessages) != 4 || len(payloads.ResponsesInput) != 4 {
		t.Fatalf("conversation projection lost messages: openai=%d responses=%d", len(payloads.OpenAIMessages), len(payloads.ResponsesInput))
	}
	if len(payloads.AnthropicSystem) != 1 || len(payloads.AnthropicMessages) != 3 {
		t.Fatalf("anthropic projection split incorrectly: system=%d messages=%d", len(payloads.AnthropicSystem), len(payloads.AnthropicMessages))
	}
	lastOpenAI := payloads.OpenAIMessages[3].(map[string]any)
	content := lastOpenAI["content"].([]any)
	if len(content) != 2 || content[0].(map[string]any)["type"] != "text" || content[1].(map[string]any)["type"] != "image_url" {
		t.Fatalf("unexpected OpenAI multipart projection: %#v", content)
	}
}

func TestBuildTemplatePayloadsProjectsImagesAsDataURLAndBase64(t *testing.T) {
	messages := []Message{NewMultiPartMessage(RoleUser, []ContentPart{ImagePart(templateTestPNG)})}
	payloads, err := buildTemplatePayloads("test-model", messages)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(payloads.ImagesDataURLs, []string{templateTestPNG}) {
		t.Fatalf("unexpected data URLs: %#v", payloads.ImagesDataURLs)
	}
	if len(payloads.ImagesBase64) != 1 || strings.HasPrefix(payloads.ImagesBase64[0], "data:") || payloads.ImagesBase64[0] == "" {
		t.Fatalf("unexpected raw base64: %#v", payloads.ImagesBase64)
	}
	anthropic := payloads.AnthropicMessages[0].(map[string]any)
	block := anthropic["content"].([]any)[0].(map[string]any)
	source := block["source"].(map[string]any)
	if source["type"] != "base64" || source["media_type"] != "image/png" || source["data"] != payloads.ImagesBase64[0] {
		t.Fatalf("unexpected Anthropic image projection: %#v", source)
	}
}

func TestBuildTemplatePayloadsRejectsInvalidImageAndPDF(t *testing.T) {
	tests := []ContentPart{
		ImagePart("data:image/png;base64,not-base64"),
		ImagePart("data:text/plain;base64,dGVzdA=="),
		{Type: ContentPDF, Base64: "data:text/plain;base64,dGVzdA=="},
	}
	for i, part := range tests {
		_, err := buildTemplatePayloads("test-model", []Message{NewMultiPartMessage(RoleUser, []ContentPart{part})})
		if err == nil {
			t.Fatalf("case %d accepted invalid content", i)
		}
	}
}

func TestRenderRequestTemplateInjectsTypedValues(t *testing.T) {
	payloads, err := buildTemplatePayloads("test-model", []Message{NewUserMessage("hello")})
	if err != nil {
		t.Fatal(err)
	}
	body, err := renderRequestTemplate(`{"model":"{{model}}","messages":"{{openai_messages}}","images":"{{images_data_urls}}"}`, payloads)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["model"] != "test-model" {
		t.Fatalf("model was not injected as a string: %#v", decoded["model"])
	}
	if _, ok := decoded["messages"].([]any); !ok {
		t.Fatalf("messages were not injected as an array: %#v", decoded["messages"])
	}
	if _, ok := decoded["images"].([]any); !ok {
		t.Fatalf("images were not injected as an array: %#v", decoded["images"])
	}
}

func TestRenderRequestTemplateDoesNotMutatePayloads(t *testing.T) {
	payloads, err := buildTemplatePayloads("test-model", []Message{NewUserMessage("hello")})
	if err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(payloads.OpenAIMessages)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := renderRequestTemplate(`{"messages":"{{openai_messages}}"}`, payloads); err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(payloads.OpenAIMessages)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("rendering mutated message payloads")
	}
}

func TestExtractJSONPointerSupportsObjectsArraysAndEscapes(t *testing.T) {
	body := []byte(`{"a/b":{"~value":[{"text":"answer"}]}}`)
	got, err := extractJSONPointer(body, "/a~1b/~0value/0/text")
	if err != nil || got != "answer" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestExtractJSONPointerRejectsMissingNonStringEmptyAndTrailingJSON(t *testing.T) {
	tests := []struct {
		body    string
		pointer string
	}{
		{body: `{"answer":"ok"}`, pointer: "/missing"},
		{body: `{"answer":42}`, pointer: "/answer"},
		{body: `{"answer":"   "}`, pointer: "/answer"},
		{body: `{"answer":"ok"} trailing`, pointer: "/answer"},
		{body: `{"items":[]}`, pointer: "/items/0"},
	}
	for i, test := range tests {
		if got, err := extractJSONPointer([]byte(test.body), test.pointer); err == nil {
			t.Fatalf("case %d returned %q without error", i, got)
		}
	}
}

func TestPointerRejectsNonCanonicalArrayIndex(t *testing.T) {
	for _, pointer := range []string{"/a/+0", "/a/-0", "/a/00"} {
		if _, err := extractJSONPointer([]byte(`{"a":["answer"]}`), pointer); err == nil {
			t.Fatalf("accepted %s", pointer)
		}
	}
}
