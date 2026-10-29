package llm

import (
	"context"
	"encoding/json"
	"strings"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const chatProbeMaxTokens int64 = 32

type OpenAIAdapter struct {
	client *openai.Client
	config config.Config
}

func NewOpenAIAdapter(cfg config.Config) *OpenAIAdapter {
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = openai.ChatModelGPT4o
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	} else {
		cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	}

	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(cfg.BaseURL),
		option.WithHTTPClient(newProviderHTTPClient()),
		option.WithMaxRetries(0),
	)
	return &OpenAIAdapter{client: &client, config: cfg}
}

func (a *OpenAIAdapter) toOpenAIMessages(messages []Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	result := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case RoleSystem:
			result = append(result, openai.SystemMessage(msg.Content))
		case RoleUser:
			if len(msg.Parts) == 0 {
				result = append(result, openai.UserMessage(msg.Content))
				continue
			}
			parts, err := a.toOpenAIParts(msg.Parts)
			if err != nil {
				return nil, err
			}
			result = append(result, openai.UserMessage(parts))
		case RoleAssistant:
			result = append(result, openai.AssistantMessage(msg.Content))
		default:
			return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.OpenAIChat, nil)
		}
	}
	return result, nil
}

func (a *OpenAIAdapter) toOpenAIParts(parts []ContentPart) ([]openai.ChatCompletionContentPartUnionParam, error) {
	result := make([]openai.ChatCompletionContentPartUnionParam, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case ContentText:
			result = append(result, openai.TextContentPart(part.Text))
		case ContentImage:
			mimeType, _, err := ParseBase64DataURLStrict(part.Base64)
			if err != nil || !isSupportedImageMIME(mimeType) {
				return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.OpenAIChat, err)
			}
			result = append(result, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: part.Base64}))
		case ContentPDF:
			mimeType, _, err := ParseBase64DataURLStrict(part.Base64)
			if err != nil || mimeType != "application/pdf" {
				return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.OpenAIChat, err)
			}
			result = append(result, openai.FileContentPart(openai.ChatCompletionContentPartFileFileParam{
				FileData: openai.String(part.Base64), Filename: openai.String("document.pdf"),
			}))
		default:
			return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.OpenAIChat, nil)
		}
	}
	return result, nil
}

func (a *OpenAIAdapter) GenerateContentStream(ctx context.Context, messages []Message, onChunk StreamCallback) (Message, error) {
	openaiMessages, err := a.toOpenAIMessages(messages)
	if err != nil {
		return Message{}, err
	}
	stream := a.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{Model: a.config.Model, Messages: openaiMessages})
	defer stream.Close()

	var fullContent strings.Builder
	var fullThinking strings.Builder
	finishReason := ""
	refused := false
	for stream.Next() {
		evt := stream.Current()
		for _, choice := range evt.Choices {
			if choice.Index != 0 {
				continue
			}
			if choice.Delta.Refusal != "" {
				refused = true
			}
			if thinkingRaw, ok := choice.Delta.JSON.ExtraFields["reasoning"]; ok {
				a.handleThinkingChunk(thinkingRaw.Raw(), &fullThinking, onChunk)
			} else if thinkingRaw, ok := choice.Delta.JSON.ExtraFields["reasoning_content"]; ok {
				a.handleThinkingChunk(thinkingRaw.Raw(), &fullThinking, onChunk)
			}
			if choice.Delta.Content != "" {
				fullContent.WriteString(choice.Delta.Content)
				if onChunk != nil {
					onChunk(StreamChunk{Type: ChunkContent, Content: choice.Delta.Content})
				}
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}
	}
	if err := stream.Err(); err != nil {
		return Message{}, ClassifyProviderError(err, StageStreamResponse, protocol.OpenAIChat)
	}
	if refused {
		return Message{}, NewProviderError(ErrorRefused, StageValidateResponse, protocol.OpenAIChat, nil)
	}
	if err := validateChatTerminal(finishReason, fullContent.String(), protocol.OpenAIChat); err != nil {
		return Message{}, err
	}
	return Message{Role: RoleAssistant, Content: fullContent.String(), Thinking: fullThinking.String()}, nil
}

func (a *OpenAIAdapter) handleThinkingChunk(rawJSON string, builder *strings.Builder, onChunk StreamCallback) {
	var decoded string
	if err := json.Unmarshal([]byte(rawJSON), &decoded); err != nil {
		return
	}
	builder.WriteString(decoded)
	if onChunk != nil {
		onChunk(StreamChunk{Type: ChunkThinking, Content: decoded})
	}
}

func (a *OpenAIAdapter) TestChat(ctx context.Context) error {
	resp, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:               a.config.Model,
		Messages:            []openai.ChatCompletionMessageParamUnion{openai.UserMessage("hi")},
		MaxCompletionTokens: openai.Int(chatProbeMaxTokens),
	})
	if err != nil {
		return ClassifyProviderError(err, StageSendRequest, protocol.OpenAIChat)
	}
	_, err = validateChatResponse(resp, protocol.OpenAIChat)
	return err
}

func (a *OpenAIAdapter) GenerateContent(ctx context.Context, model string, messages []Message) (Message, error) {
	if strings.TrimSpace(model) == "" {
		model = a.config.Model
	}
	openaiMessages, err := a.toOpenAIMessages(messages)
	if err != nil {
		return Message{}, err
	}
	resp, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{Model: model, Messages: openaiMessages})
	if err != nil {
		return Message{}, ClassifyProviderError(err, StageSendRequest, protocol.OpenAIChat)
	}
	content, err := validateChatResponse(resp, protocol.OpenAIChat)
	if err != nil {
		return Message{}, err
	}
	return Message{Role: RoleAssistant, Content: content}, nil
}

func validateChatResponse(resp *openai.ChatCompletion, p protocol.APIProtocol) (string, error) {
	if resp == nil || len(resp.Choices) == 0 {
		return "", NewProviderError(ErrorEmptyResponse, StageValidateResponse, p, nil)
	}
	choice := resp.Choices[0]
	if choice.Message.Refusal != "" {
		return "", NewProviderError(ErrorRefused, StageValidateResponse, p, nil)
	}
	if err := validateChatTerminal(choice.FinishReason, choice.Message.Content, p); err != nil {
		return "", err
	}
	return choice.Message.Content, nil
}

func validateChatTerminal(reason, content string, p protocol.APIProtocol) error {
	switch reason {
	case "stop":
		if strings.TrimSpace(content) == "" {
			return NewProviderError(ErrorEmptyResponse, StageValidateResponse, p, nil)
		}
		return nil
	case "":
		return NewProviderError(ErrorStreamInterrupted, StageStreamResponse, p, nil)
	case "length":
		return NewProviderError(ErrorIncompleteResponse, StageValidateResponse, p, nil)
	case "content_filter":
		return NewProviderError(ErrorRefused, StageValidateResponse, p, nil)
	case "tool_calls", "function_call":
		return NewProviderError(ErrorInvalidResponse, StageValidateResponse, p, nil)
	default:
		return NewProviderError(ErrorInvalidResponse, StageValidateResponse, p, nil)
	}
}

func (a *OpenAIAdapter) GetModels(ctx context.Context) ([]string, error) {
	resp, err := a.client.Models.List(ctx)
	if err != nil {
		return nil, ClassifyProviderError(err, StageSendRequest, protocol.OpenAIChat)
	}
	models := make([]string, 0, len(resp.Data))
	for _, model := range resp.Data {
		models = append(models, model.ID)
	}
	return models, nil
}

func (a *OpenAIAdapter) generateTextProbe(ctx context.Context, model string, messages []Message) (Message, error) {
	if strings.TrimSpace(model) == "" {
		model = a.config.Model
	}
	openaiMessages, err := a.toOpenAIMessages(messages)
	if err != nil {
		return Message{}, err
	}
	resp, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:               model,
		Messages:            openaiMessages,
		MaxCompletionTokens: openai.Int(chatProbeMaxTokens),
	})
	if err != nil {
		return Message{}, ClassifyProviderError(err, StageSendRequest, protocol.OpenAIChat)
	}
	content, err := validateChatResponse(resp, protocol.OpenAIChat)
	if err != nil {
		return Message{}, err
	}
	return Message{Role: RoleAssistant, Content: content}, nil
}
