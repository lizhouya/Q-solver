package llm

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"
)

const maxCustomJSONResponseBytes int64 = 4 * 1024 * 1024

type CustomJSONAdapter struct {
	client *http.Client
	config config.Config
}

func NewCustomJSONAdapter(cfg config.Config) *CustomJSONAdapter {
	return &CustomJSONAdapter{
		client: &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   120 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		config: cfg,
	}
}

func (a *CustomJSONAdapter) GenerateContent(ctx context.Context, model string, messages []Message) (Message, error) {
	if strings.TrimSpace(model) == "" {
		model = a.config.Model
	}
	return a.execute(ctx, model, messages)
}

func (a *CustomJSONAdapter) GenerateContentStream(ctx context.Context, messages []Message, onChunk StreamCallback) (Message, error) {
	result, err := a.execute(ctx, a.config.Model, messages)
	if err != nil {
		return Message{}, err
	}
	if err := ctx.Err(); err != nil {
		return Message{}, ClassifyProviderError(err, StageReadResponse, protocol.CustomJSON)
	}
	if onChunk != nil {
		onChunk(StreamChunk{Type: ChunkContent, Content: result.Content})
	}
	return result, nil
}

func (a *CustomJSONAdapter) TestChat(ctx context.Context) error {
	_, err := a.execute(ctx, a.config.Model, []Message{NewUserMessage("hi")})
	return err
}

func (a *CustomJSONAdapter) GetModels(context.Context) ([]string, error) {
	return nil, NewProviderError(ErrorCatalogUnsupported, StageValidateConfig, protocol.CustomJSON, nil)
}

func (a *CustomJSONAdapter) generateTextProbe(ctx context.Context, model string, messages []Message) (Message, error) {
	return a.GenerateContent(ctx, model, messages)
}

func (a *CustomJSONAdapter) execute(ctx context.Context, model string, messages []Message) (Message, error) {
	payloads, err := buildTemplatePayloads(model, messages)
	if err != nil {
		return Message{}, err
	}
	body, err := renderRequestTemplate(a.config.RequestTemplate, payloads)
	if err != nil {
		return Message{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.config.BaseURL, bytes.NewReader(body))
	if err != nil {
		return Message{}, NewProviderError(ErrorInvalidRequest, StageBuildRequest, protocol.CustomJSON, err)
	}
	request.Header.Set("Content-Type", "application/json")
	for name, value := range a.config.ExtraHeaders {
		request.Header.Set(name, value)
	}
	if apiKey := strings.TrimSpace(a.config.APIKey); apiKey != "" {
		request.Header.Set(a.config.AuthHeader, a.config.AuthPrefix+apiKey)
	}

	response, err := a.client.Do(request)
	if err != nil {
		return Message{}, ClassifyProviderError(err, StageSendRequest, protocol.CustomJSON)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		providerErr := NewProviderError(errorCodeForHTTP(response.StatusCode, ""), StageValidateResponse, protocol.CustomJSON, nil)
		providerErr.HTTPStatus = response.StatusCode
		return Message{}, providerErr
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxCustomJSONResponseBytes+1))
	if err != nil {
		return Message{}, ClassifyProviderError(err, StageReadResponse, protocol.CustomJSON)
	}
	if int64(len(data)) > maxCustomJSONResponseBytes {
		return Message{}, NewProviderError(ErrorRequestTooLarge, StageReadResponse, protocol.CustomJSON, nil)
	}
	content, err := extractJSONPointer(data, a.config.ResponseJSONPointer)
	if err != nil {
		return Message{}, err
	}
	return Message{Role: RoleAssistant, Content: content}, nil
}
