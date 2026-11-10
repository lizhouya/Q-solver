package llm

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"
)

type templatePayloads struct {
	Model             string
	SystemPrompt      string
	UserText          string
	OpenAIMessages    []any
	ResponsesInput    []any
	AnthropicMessages []any
	AnthropicSystem   []any
	ImagesDataURLs    []string
	ImagesBase64      []string
}

func buildTemplatePayloads(model string, messages []Message) (templatePayloads, error) {
	payloads := templatePayloads{
		Model:             model,
		OpenAIMessages:    make([]any, 0, len(messages)),
		ResponsesInput:    make([]any, 0, len(messages)),
		AnthropicMessages: make([]any, 0, len(messages)),
		AnthropicSystem:   make([]any, 0),
		ImagesDataURLs:    make([]string, 0),
		ImagesBase64:      make([]string, 0),
	}
	systemTexts := make([]string, 0)

	for _, message := range messages {
		role, err := templateRole(message.Role)
		if err != nil {
			return templatePayloads{}, err
		}
		text, err := templateMessageText(message)
		if err != nil {
			return templatePayloads{}, err
		}
		if message.Role == RoleSystem {
			systemTexts = append(systemTexts, text)
		}
		if message.Role == RoleUser {
			payloads.UserText = text
		}

		openAIContent, responsesContent, anthropicContent, err := projectTemplateContent(message, &payloads)
		if err != nil {
			return templatePayloads{}, err
		}
		payloads.OpenAIMessages = append(payloads.OpenAIMessages, map[string]any{
			"role": role, "content": openAIContent,
		})
		payloads.ResponsesInput = append(payloads.ResponsesInput, map[string]any{
			"role": role, "content": responsesContent,
		})
		if message.Role == RoleSystem {
			blocks, ok := anthropicContent.([]any)
			if !ok {
				return templatePayloads{}, templateRequestError(fmt.Errorf("system content has invalid shape"))
			}
			for _, block := range blocks {
				blockMap, ok := block.(map[string]any)
				if !ok || blockMap["type"] != "text" {
					return templatePayloads{}, templateRequestError(fmt.Errorf("system message contains non-text content"))
				}
				payloads.AnthropicSystem = append(payloads.AnthropicSystem, block)
			}
			continue
		}
		payloads.AnthropicMessages = append(payloads.AnthropicMessages, map[string]any{
			"role": role, "content": anthropicContent,
		})
	}
	payloads.SystemPrompt = strings.Join(systemTexts, "\n")
	return payloads, nil
}

func templateRole(role Role) (string, error) {
	switch role {
	case RoleSystem, RoleUser, RoleAssistant:
		return string(role), nil
	default:
		return "", templateRequestError(fmt.Errorf("unsupported message role"))
	}
}

func templateMessageText(message Message) (string, error) {
	if len(message.Parts) == 0 {
		return message.Content, nil
	}
	texts := make([]string, 0, len(message.Parts))
	for _, part := range message.Parts {
		if part.Type == ContentText {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n"), nil
}

func projectTemplateContent(message Message, payloads *templatePayloads) (openAI any, responses any, anthropic any, err error) {
	if len(message.Parts) == 0 {
		return message.Content, message.Content, []any{map[string]any{"type": "text", "text": message.Content}}, nil
	}

	openAIParts := make([]any, 0, len(message.Parts))
	responsesParts := make([]any, 0, len(message.Parts))
	anthropicParts := make([]any, 0, len(message.Parts))
	for _, part := range message.Parts {
		switch part.Type {
		case ContentText:
			openAIParts = append(openAIParts, map[string]any{"type": "text", "text": part.Text})
			responsesParts = append(responsesParts, map[string]any{"type": "input_text", "text": part.Text})
			anthropicParts = append(anthropicParts, map[string]any{"type": "text", "text": part.Text})
		case ContentImage:
			mimeType, data, parseErr := ParseBase64DataURLStrict(part.Base64)
			if parseErr != nil || !isSupportedImageMIME(mimeType) {
				return nil, nil, nil, templateRequestError(parseErr)
			}
			payloads.ImagesDataURLs = append(payloads.ImagesDataURLs, part.Base64)
			payloads.ImagesBase64 = append(payloads.ImagesBase64, data)
			openAIParts = append(openAIParts, map[string]any{
				"type": "image_url", "image_url": map[string]any{"url": part.Base64},
			})
			responsesParts = append(responsesParts, map[string]any{
				"type": "input_image", "image_url": part.Base64, "detail": "auto",
			})
			anthropicParts = append(anthropicParts, map[string]any{
				"type":   "image",
				"source": map[string]any{"type": "base64", "media_type": mimeType, "data": data},
			})
		case ContentPDF:
			mimeType, data, parseErr := ParseBase64DataURLStrict(part.Base64)
			if parseErr != nil || mimeType != "application/pdf" {
				return nil, nil, nil, templateRequestError(parseErr)
			}
			openAIParts = append(openAIParts, map[string]any{
				"type": "file", "file": map[string]any{"file_data": part.Base64, "filename": "document.pdf"},
			})
			responsesParts = append(responsesParts, map[string]any{
				"type": "input_file", "file_data": part.Base64, "filename": "document.pdf",
			})
			anthropicParts = append(anthropicParts, map[string]any{
				"type":   "document",
				"source": map[string]any{"type": "base64", "media_type": mimeType, "data": data},
			})
		default:
			return nil, nil, nil, templateRequestError(fmt.Errorf("unsupported content type"))
		}
	}
	return openAIParts, responsesParts, anthropicParts, nil
}

func renderRequestTemplate(value string, payloads templatePayloads) ([]byte, error) {
	if err := config.ValidateRequestTemplateConfig(value, "/answer"); err != nil {
		return nil, templateRequestError(err)
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return nil, templateRequestError(err)
	}
	if len(payloads.ImagesDataURLs) > 0 && !templateCarriesImages(decoded) {
		return nil, NewProviderError(ErrorInvalidConfig, StageBuildRequest, protocol.CustomJSON, nil)
	}
	rendered, err := renderTemplateValue(decoded, payloads)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(rendered)
	if err != nil {
		return nil, templateRequestError(err)
	}
	return body, nil
}

func renderTemplateValue(value any, payloads templatePayloads) (any, error) {
	switch current := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(current))
		for key, child := range current {
			rendered, err := renderTemplateValue(child, payloads)
			if err != nil {
				return nil, err
			}
			result[key] = rendered
		}
		return result, nil
	case []any:
		result := make([]any, len(current))
		for index, child := range current {
			rendered, err := renderTemplateValue(child, payloads)
			if err != nil {
				return nil, err
			}
			result[index] = rendered
		}
		return result, nil
	case string:
		switch current {
		case "{{model}}":
			return payloads.Model, nil
		case "{{system_prompt}}":
			return payloads.SystemPrompt, nil
		case "{{user_text}}":
			return payloads.UserText, nil
		case "{{openai_messages}}":
			return payloads.OpenAIMessages, nil
		case "{{responses_input}}":
			return payloads.ResponsesInput, nil
		case "{{anthropic_messages}}":
			return payloads.AnthropicMessages, nil
		case "{{anthropic_system}}":
			return payloads.AnthropicSystem, nil
		case "{{images_data_urls}}":
			return payloads.ImagesDataURLs, nil
		case "{{images_base64}}":
			return payloads.ImagesBase64, nil
		default:
			if strings.Contains(current, "{{") || strings.Contains(current, "}}") {
				return nil, templateRequestError(fmt.Errorf("unknown or embedded placeholder"))
			}
			return current, nil
		}
	default:
		return current, nil
	}
}

func extractJSONPointer(body []byte, pointer string) (string, error) {
	if err := config.ValidateJSONPointer(pointer); err != nil {
		return "", NewProviderError(ErrorInvalidConfig, StageValidateConfig, protocol.CustomJSON, err)
	}
	var current any
	if err := json.Unmarshal(body, &current); err != nil {
		return "", NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.CustomJSON, err)
	}
	for _, encoded := range strings.Split(pointer[1:], "/") {
		segment := strings.ReplaceAll(strings.ReplaceAll(encoded, "~1", "/"), "~0", "~")
		switch node := current.(type) {
		case map[string]any:
			value, ok := node[segment]
			if !ok {
				return "", NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.CustomJSON, nil)
			}
			current = value
		case []any:
			if segment == "" || (len(segment) > 1 && segment[0] == '0') {
				return "", NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.CustomJSON, nil)
			}
			for _, c := range segment {
				if c < '0' || c > '9' {
					return "", NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.CustomJSON, nil)
				}
			}
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(node) {
				return "", NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.CustomJSON, err)
			}
			current = node[index]
		default:
			return "", NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.CustomJSON, nil)
		}
	}
	content, ok := current.(string)
	if !ok || strings.TrimSpace(content) == "" {
		return "", NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.CustomJSON, nil)
	}
	return content, nil
}

func templateRequestError(err error) *ProviderError {
	return NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.CustomJSON, err)
}

func templateCarriesImages(value any) bool {
	switch v := value.(type) {
	case string:
		switch v {
		case "{{openai_messages}}", "{{responses_input}}", "{{anthropic_messages}}", "{{images_data_urls}}", "{{images_base64}}":
			return true
		}
	case map[string]any:
		for _, child := range v {
			if templateCarriesImages(child) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if templateCarriesImages(child) {
				return true
			}
		}
	}
	return false
}
