package llm

import (
	"context"
	"strings"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"

	anthropic "github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

const (
	anthropicBusinessMaxTokens int64 = 8192
	anthropicProbeMaxTokens    int64 = 32
	defaultAnthropicModel            = "claude-opus-5"
)

type AnthropicAdapter struct {
	client *anthropic.Client
	config config.Config
}

func NewAnthropicAdapter(cfg config.Config) *AnthropicAdapter {
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaultAnthropicModel
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://api.anthropic.com"
	} else {
		cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	}

	client := anthropic.NewClient(
		option.WithoutEnvironmentDefaults(),
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(cfg.BaseURL),
		option.WithHTTPClient(newProviderHTTPClient()),
		option.WithMaxRetries(0),
	)
	return &AnthropicAdapter{client: &client, config: cfg}
}

func (a *AnthropicAdapter) newParams(model string, messages []Message, maxTokens int64) (anthropic.MessageNewParams, error) {
	if strings.TrimSpace(model) == "" {
		model = a.config.Model
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		Messages:  make([]anthropic.MessageParam, 0, len(messages)),
	}
	for _, message := range messages {
		if message.Role == RoleSystem {
			blocks, err := toAnthropicSystemBlocks(message)
			if err != nil {
				return anthropic.MessageNewParams{}, err
			}
			params.System = append(params.System, blocks...)
			continue
		}
		blocks, err := toAnthropicContentBlocks(message)
		if err != nil {
			return anthropic.MessageNewParams{}, err
		}
		switch message.Role {
		case RoleUser:
			params.Messages = append(params.Messages, anthropic.NewUserMessage(blocks...))
		case RoleAssistant:
			params.Messages = append(params.Messages, anthropic.NewAssistantMessage(blocks...))
		default:
			return anthropic.MessageNewParams{}, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.AnthropicMessages, nil)
		}
	}
	return params, nil
}

func toAnthropicSystemBlocks(message Message) ([]anthropic.TextBlockParam, error) {
	if len(message.Parts) == 0 {
		return []anthropic.TextBlockParam{{Text: message.Content}}, nil
	}
	blocks := make([]anthropic.TextBlockParam, 0, len(message.Parts))
	for _, part := range message.Parts {
		if part.Type != ContentText {
			return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.AnthropicMessages, nil)
		}
		blocks = append(blocks, anthropic.TextBlockParam{Text: part.Text})
	}
	return blocks, nil
}

func toAnthropicContentBlocks(message Message) ([]anthropic.ContentBlockParamUnion, error) {
	if len(message.Parts) == 0 {
		return []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(message.Content)}, nil
	}
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(message.Parts))
	for _, part := range message.Parts {
		switch part.Type {
		case ContentText:
			blocks = append(blocks, anthropic.NewTextBlock(part.Text))
		case ContentImage:
			mimeType, data, err := ParseBase64DataURLStrict(part.Base64)
			if err != nil || !isSupportedImageMIME(mimeType) {
				return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.AnthropicMessages, err)
			}
			blocks = append(blocks, anthropic.NewImageBlockBase64(mimeType, data))
		case ContentPDF:
			mimeType, data, err := ParseBase64DataURLStrict(part.Base64)
			if err != nil || mimeType != "application/pdf" {
				return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.AnthropicMessages, err)
			}
			blocks = append(blocks, anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{Data: data}))
		default:
			return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.AnthropicMessages, nil)
		}
	}
	return blocks, nil
}

func (a *AnthropicAdapter) GenerateContent(ctx context.Context, model string, messages []Message) (Message, error) {
	params, err := a.newParams(model, messages, anthropicBusinessMaxTokens)
	if err != nil {
		return Message{}, err
	}
	response, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return Message{}, ClassifyProviderError(err, StageSendRequest, protocol.AnthropicMessages)
	}
	return validateAnthropicResponse(response)
}

func validateAnthropicResponse(response *anthropic.Message) (Message, error) {
	if response == nil {
		return Message{}, NewProviderError(ErrorEmptyResponse, StageValidateResponse, protocol.AnthropicMessages, nil)
	}
	var content strings.Builder
	var thinking strings.Builder
	for _, block := range response.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.TextBlock:
			content.WriteString(variant.Text)
		case anthropic.ThinkingBlock:
			thinking.WriteString(variant.Thinking)
		default:
			return Message{}, NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.AnthropicMessages, nil)
		}
	}
	if err := validateAnthropicTerminal(response.StopReason, content.String()); err != nil {
		return Message{}, err
	}
	return Message{Role: RoleAssistant, Content: content.String(), Thinking: thinking.String()}, nil
}

func validateAnthropicTerminal(reason anthropic.StopReason, content string) error {
	switch reason {
	case anthropic.StopReasonEndTurn, anthropic.StopReasonStopSequence:
		if strings.TrimSpace(content) == "" {
			return NewProviderError(ErrorEmptyResponse, StageValidateResponse, protocol.AnthropicMessages, nil)
		}
		return nil
	case anthropic.StopReasonMaxTokens, anthropic.StopReasonModelContextWindowExceeded:
		return NewProviderError(ErrorIncompleteResponse, StageValidateResponse, protocol.AnthropicMessages, nil)
	case anthropic.StopReasonRefusal:
		return NewProviderError(ErrorRefused, StageValidateResponse, protocol.AnthropicMessages, nil)
	default:
		return NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.AnthropicMessages, nil)
	}
}

func (a *AnthropicAdapter) GenerateContentStream(ctx context.Context, messages []Message, onChunk StreamCallback) (Message, error) {
	params, err := a.newParams(a.config.Model, messages, anthropicBusinessMaxTokens)
	if err != nil {
		return Message{}, err
	}
	stream := a.client.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	var content strings.Builder
	var thinking strings.Builder
	var stopReason anthropic.StopReason
	stopReasonSeen := false
	messageStopped := false

	for stream.Next() {
		event := stream.Current()
		switch variant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch delta := variant.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				content.WriteString(delta.Text)
				if onChunk != nil {
					onChunk(StreamChunk{Type: ChunkContent, Content: delta.Text})
				}
			case anthropic.ThinkingDelta:
				thinking.WriteString(delta.Thinking)
				if onChunk != nil {
					onChunk(StreamChunk{Type: ChunkThinking, Content: delta.Thinking})
				}
			}
		case anthropic.MessageDeltaEvent:
			stopReason = variant.Delta.StopReason
			stopReasonSeen = true
		case anthropic.MessageStopEvent:
			messageStopped = true
		}
		if messageStopped {
			break
		}
	}
	if err := stream.Err(); err != nil {
		return Message{}, ClassifyProviderError(err, StageStreamResponse, protocol.AnthropicMessages)
	}
	if !messageStopped {
		return Message{}, NewProviderError(ErrorStreamInterrupted, StageStreamResponse, protocol.AnthropicMessages, nil)
	}
	if !stopReasonSeen {
		return Message{}, NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.AnthropicMessages, nil)
	}
	if err := validateAnthropicTerminal(stopReason, content.String()); err != nil {
		return Message{}, err
	}
	return Message{Role: RoleAssistant, Content: content.String(), Thinking: thinking.String()}, nil
}

func (a *AnthropicAdapter) TestChat(ctx context.Context) error {
	params, err := a.newParams(a.config.Model, []Message{NewUserMessage("hi")}, anthropicProbeMaxTokens)
	if err != nil {
		return err
	}
	response, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return ClassifyProviderError(err, StageSendRequest, protocol.AnthropicMessages)
	}
	_, err = validateAnthropicResponse(response)
	return err
}

func (a *AnthropicAdapter) GetModels(ctx context.Context) ([]string, error) {
	page, err := a.client.Models.List(ctx, anthropic.ModelListParams{})
	if err != nil {
		return nil, ClassifyProviderError(err, StageSendRequest, protocol.AnthropicMessages)
	}
	models := make([]string, 0, len(page.Data))
	for _, model := range page.Data {
		models = append(models, model.ID)
	}
	return models, nil
}

func (a *AnthropicAdapter) generateTextProbe(ctx context.Context, model string, messages []Message) (Message, error) {
	params, err := a.newParams(model, messages, anthropicProbeMaxTokens)
	if err != nil {
		return Message{}, err
	}
	response, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return Message{}, ClassifyProviderError(err, StageSendRequest, protocol.AnthropicMessages)
	}
	return validateAnthropicResponse(response)
}
