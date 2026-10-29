package llm

import (
	"context"
	"strings"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

const responsesProbeMaxTokens int64 = 32

type OpenAIResponsesAdapter struct {
	client *openai.Client
	config config.Config
}

func NewOpenAIResponsesAdapter(cfg config.Config) *OpenAIResponsesAdapter {
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
	return &OpenAIResponsesAdapter{client: &client, config: cfg}
}

func (a *OpenAIResponsesAdapter) toResponsesInput(messages []Message) (responses.ResponseInputParam, error) {
	result := make(responses.ResponseInputParam, 0, len(messages))
	for _, msg := range messages {
		role, err := toResponsesRole(msg.Role)
		if err != nil {
			return nil, err
		}
		if len(msg.Parts) == 0 {
			result = append(result, responses.ResponseInputItemParamOfMessage(msg.Content, role))
			continue
		}
		parts, err := a.toResponsesParts(msg.Parts)
		if err != nil {
			return nil, err
		}
		result = append(result, responses.ResponseInputItemParamOfMessage(parts, role))
	}
	return result, nil
}

func toResponsesRole(role Role) (responses.EasyInputMessageRole, error) {
	switch role {
	case RoleSystem:
		return responses.EasyInputMessageRoleSystem, nil
	case RoleUser:
		return responses.EasyInputMessageRoleUser, nil
	case RoleAssistant:
		return responses.EasyInputMessageRoleAssistant, nil
	default:
		return "", NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.OpenAIResponses, nil)
	}
}

func (a *OpenAIResponsesAdapter) toResponsesParts(parts []ContentPart) (responses.ResponseInputMessageContentListParam, error) {
	result := make(responses.ResponseInputMessageContentListParam, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case ContentText:
			result = append(result, responses.ResponseInputContentParamOfInputText(part.Text))
		case ContentImage:
			mimeType, _, err := ParseBase64DataURLStrict(part.Base64)
			if err != nil || !isSupportedImageMIME(mimeType) {
				return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.OpenAIResponses, err)
			}
			result = append(result, responses.ResponseInputContentUnionParam{
				OfInputImage: &responses.ResponseInputImageParam{
					Detail:   responses.ResponseInputImageDetailAuto,
					ImageURL: openai.String(part.Base64),
				},
			})
		case ContentPDF:
			mimeType, _, err := ParseBase64DataURLStrict(part.Base64)
			if err != nil || mimeType != "application/pdf" {
				return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.OpenAIResponses, err)
			}
			result = append(result, responses.ResponseInputContentUnionParam{
				OfInputFile: &responses.ResponseInputFileParam{
					FileData: openai.String(part.Base64),
					Filename: openai.String("document.pdf"),
				},
			})
		default:
			return nil, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.OpenAIResponses, nil)
		}
	}
	return result, nil
}

func (a *OpenAIResponsesAdapter) newParams(model string, input responses.ResponseInputParam) responses.ResponseNewParams {
	return responses.ResponseNewParams{
		Model: shared.ResponsesModel(model),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: input,
		},
		Store: openai.Bool(false),
	}
}

func (a *OpenAIResponsesAdapter) GenerateContent(ctx context.Context, model string, messages []Message) (Message, error) {
	if strings.TrimSpace(model) == "" {
		model = a.config.Model
	}
	input, err := a.toResponsesInput(messages)
	if err != nil {
		return Message{}, err
	}
	resp, err := a.client.Responses.New(ctx, a.newParams(model, input))
	if err != nil {
		return Message{}, ClassifyProviderError(err, StageSendRequest, protocol.OpenAIResponses)
	}
	content, err := validateResponsesResponse(resp)
	if err != nil {
		return Message{}, err
	}
	return Message{Role: RoleAssistant, Content: content}, nil
}

func (a *OpenAIResponsesAdapter) GenerateContentStream(ctx context.Context, messages []Message, onChunk StreamCallback) (Message, error) {
	input, err := a.toResponsesInput(messages)
	if err != nil {
		return Message{}, err
	}
	stream := a.client.Responses.NewStreaming(ctx, a.newParams(a.config.Model, input))
	defer stream.Close()

	var content strings.Builder
	var thinking strings.Builder
	terminal := false
	refused := false
	var terminalErr error

	for stream.Next() {
		event := stream.Current()
		switch variant := event.AsAny().(type) {
		case responses.ResponseTextDeltaEvent:
			content.WriteString(variant.Delta)
			if onChunk != nil {
				onChunk(StreamChunk{Type: ChunkContent, Content: variant.Delta})
			}
		case responses.ResponseReasoningSummaryTextDeltaEvent:
			thinking.WriteString(variant.Delta)
			if onChunk != nil {
				onChunk(StreamChunk{Type: ChunkThinking, Content: variant.Delta})
			}
		case responses.ResponseRefusalDeltaEvent:
			refused = true
		case responses.ResponseCompletedEvent:
			terminal = true
			if content.Len() == 0 {
				content.WriteString(variant.Response.OutputText())
			}
			terminalErr = validateResponsesTerminal(variant.Response, content.String(), refused)
		case responses.ResponseFailedEvent:
			terminal = true
			terminalErr = classifyResponsesFailure(string(variant.Response.Error.Code), StageValidateResponse)
		case responses.ResponseIncompleteEvent:
			terminal = true
			terminalErr = NewProviderError(ErrorIncompleteResponse, StageValidateResponse, protocol.OpenAIResponses, nil)
		case responses.ResponseErrorEvent:
			terminal = true
			terminalErr = classifyResponsesEventError(variant.Code)
		}
		if terminal {
			break
		}
	}
	if err := stream.Err(); err != nil {
		return Message{}, ClassifyProviderError(err, StageStreamResponse, protocol.OpenAIResponses)
	}
	if !terminal {
		return Message{}, NewProviderError(ErrorStreamInterrupted, StageStreamResponse, protocol.OpenAIResponses, nil)
	}
	if terminalErr != nil {
		return Message{}, terminalErr
	}
	return Message{Role: RoleAssistant, Content: content.String(), Thinking: thinking.String()}, nil
}

func validateResponsesResponse(resp *responses.Response) (string, error) {
	if resp == nil {
		return "", NewProviderError(ErrorEmptyResponse, StageValidateResponse, protocol.OpenAIResponses, nil)
	}
	content := resp.OutputText()
	if err := validateResponsesTerminal(*resp, content, responseContainsRefusal(*resp)); err != nil {
		return "", err
	}
	return content, nil
}

func validateResponsesTerminal(resp responses.Response, content string, refused bool) error {
	if refused || responseContainsRefusal(resp) {
		return NewProviderError(ErrorRefused, StageValidateResponse, protocol.OpenAIResponses, nil)
	}
	switch resp.Status {
	case responses.ResponseStatusCompleted:
		if strings.TrimSpace(content) == "" {
			return NewProviderError(ErrorEmptyResponse, StageValidateResponse, protocol.OpenAIResponses, nil)
		}
		return nil
	case responses.ResponseStatusFailed:
		return classifyResponsesFailure(string(resp.Error.Code), StageValidateResponse)
	case responses.ResponseStatusIncomplete:
		return NewProviderError(ErrorIncompleteResponse, StageValidateResponse, protocol.OpenAIResponses, nil)
	case responses.ResponseStatusCancelled:
		return NewProviderError(ErrorCancelled, StageValidateResponse, protocol.OpenAIResponses, nil)
	default:
		return NewProviderError(ErrorInvalidResponse, StageValidateResponse, protocol.OpenAIResponses, nil)
	}
}

func responseContainsRefusal(resp responses.Response) bool {
	for _, item := range resp.Output {
		for _, part := range item.Content {
			if part.Type == "refusal" || strings.TrimSpace(part.Refusal) != "" {
				return true
			}
		}
	}
	return false
}

func classifyResponsesEventError(code string) error {
	return classifyResponsesFailure(code, StageStreamResponse)
}

// 仅映射明确的业务错误码；事件没有 HTTP 状态，不能从业务码合成状态码。
func classifyResponsesFailure(code string, stage ErrorStage) *ProviderError {
	category := ErrorResponseFailed
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "rate_limit_exceeded":
		category = ErrorRateLimited
	case "server_error":
		category = ErrorProviderUnavailable
	case "vector_store_timeout":
		category = ErrorTimeout
	case "image_too_large", "image_file_too_large":
		category = ErrorRequestTooLarge
	case "image_content_policy_violation":
		category = ErrorRefused
	case "invalid_prompt", "invalid_image", "invalid_image_format", "invalid_base64_image", "invalid_image_url",
		"image_too_small", "image_parse_error", "invalid_image_mode", "unsupported_image_media_type",
		"empty_image_file", "failed_to_download_image", "image_file_not_found", "invalid_request_error":
		category = ErrorInvalidRequest
	// SDK 的 Code 是字符串类型；这些是兼容端点的显式扩展码，不推测模型或服务商。
	case "invalid_api_key", "authentication_error":
		category = ErrorAuthentication
	case "permission_denied", "permission_error":
		category = ErrorPermission
	case "insufficient_quota":
		category = ErrorQuotaExhausted
	}
	return NewProviderError(category, stage, protocol.OpenAIResponses, nil)
}

func (a *OpenAIResponsesAdapter) TestChat(ctx context.Context) error {
	input := responses.ResponseInputParam{
		responses.ResponseInputItemParamOfMessage("hi", responses.EasyInputMessageRoleUser),
	}
	params := a.newParams(a.config.Model, input)
	params.MaxOutputTokens = openai.Int(responsesProbeMaxTokens)
	resp, err := a.client.Responses.New(ctx, params)
	if err != nil {
		return ClassifyProviderError(err, StageSendRequest, protocol.OpenAIResponses)
	}
	_, err = validateResponsesResponse(resp)
	return err
}

func (a *OpenAIResponsesAdapter) GetModels(ctx context.Context) ([]string, error) {
	resp, err := a.client.Models.List(ctx)
	if err != nil {
		return nil, ClassifyProviderError(err, StageSendRequest, protocol.OpenAIResponses)
	}
	models := make([]string, 0, len(resp.Data))
	for _, model := range resp.Data {
		models = append(models, model.ID)
	}
	return models, nil
}

func (a *OpenAIResponsesAdapter) generateTextProbe(ctx context.Context, model string, messages []Message) (Message, error) {
	if strings.TrimSpace(model) == "" {
		model = a.config.Model
	}
	input, err := a.toResponsesInput(messages)
	if err != nil {
		return Message{}, err
	}
	params := a.newParams(model, input)
	params.MaxOutputTokens = openai.Int(responsesProbeMaxTokens)
	resp, err := a.client.Responses.New(ctx, params)
	if err != nil {
		return Message{}, ClassifyProviderError(err, StageSendRequest, protocol.OpenAIResponses)
	}
	content, err := validateResponsesResponse(resp)
	if err != nil {
		return Message{}, err
	}
	return Message{Role: RoleAssistant, Content: content}, nil
}
