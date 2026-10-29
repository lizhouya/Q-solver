package llm

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"

	"Q-Solver/pkg/llm/protocol"

	anthropic "github.com/anthropics/anthropic-sdk-go/v2"
	openai "github.com/openai/openai-go"
)

func TestClassifyProviderError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   ErrorCode
		wantStatus int
		wantRetry  bool
	}{
		{name: "cancelled", err: context.Canceled, wantCode: ErrorCancelled},
		{name: "timeout", err: context.DeadlineExceeded, wantCode: ErrorTimeout, wantRetry: true},
		{name: "tls", err: x509.UnknownAuthorityError{}, wantCode: ErrorTLS},
		{name: "dns", err: &net.DNSError{IsNotFound: true}, wantCode: ErrorNetwork, wantRetry: true},
		{name: "401", err: &openai.Error{StatusCode: 401}, wantCode: ErrorAuthentication, wantStatus: 401},
		{name: "403", err: &openai.Error{StatusCode: 403}, wantCode: ErrorPermission, wantStatus: 403},
		{name: "404", err: &openai.Error{StatusCode: 404}, wantCode: ErrorNotFoundOrProtocolMismatch, wantStatus: 404},
		{name: "413", err: &openai.Error{StatusCode: 413}, wantCode: ErrorRequestTooLarge, wantStatus: 413},
		{name: "429", err: &openai.Error{StatusCode: 429}, wantCode: ErrorRateLimited, wantStatus: 429, wantRetry: true},
		{name: "quota", err: &openai.Error{StatusCode: 429, Code: "insufficient_quota"}, wantCode: ErrorQuotaExhausted, wantStatus: 429},
		{name: "500", err: &openai.Error{StatusCode: 500}, wantCode: ErrorProviderUnavailable, wantStatus: 500, wantRetry: true},
		{name: "unknown", err: errors.New("opaque failure"), wantCode: ErrorUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyProviderError(tt.err, StageSendRequest, protocol.OpenAIChat)
			if got.Code != tt.wantCode || got.HTTPStatus != tt.wantStatus || got.Retryable != tt.wantRetry {
				t.Fatalf("unexpected classification: %+v", got)
			}
			if got.Stage != StageSendRequest || got.Protocol != protocol.OpenAIChat {
				t.Fatalf("lost error context: %+v", got)
			}
		})
	}
}

func TestProviderErrorNeverExposesCauseOrSDKPayload(t *testing.T) {
	const sentinel = `sk-secret https://private.invalid Authorization:{token} data:image/png;base64,AAAA C:Usersprivate`
	err := &openai.Error{StatusCode: 404, Code: sentinel, Message: sentinel, Type: sentinel}
	got := ClassifyProviderError(err, StageSendRequest, protocol.OpenAIResponses)

	encoded, marshalErr := json.Marshal(got)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	visible := got.Error() + string(encoded)
	if strings.Contains(visible, sentinel) || strings.Contains(visible, "private.invalid") || strings.Contains(visible, "base64") {
		t.Fatalf("sensitive cause leaked: %s", visible)
	}
	if !strings.Contains(got.Error(), "端点、协议或模型") {
		t.Fatalf("404 message assumes a single cause: %q", got.Error())
	}
}

func TestNewProviderErrorUsesFixedMessage(t *testing.T) {
	got := NewProviderError(ErrorInvalidConfig, StageValidateConfig, protocol.AnthropicMessages, errors.New("sk-secret"))
	if got.Error() == "" || strings.Contains(got.Error(), "sk-secret") {
		t.Fatalf("unsafe error text: %q", got.Error())
	}
}

func TestClassifyAnthropicTypedError(t *testing.T) {
	tests := []struct {
		name, errorType string
		status          int
		wantCode        ErrorCode
		wantRetry       bool
	}{
		{name: "401", status: 401, errorType: "authentication_error", wantCode: ErrorAuthentication},
		{name: "403", status: 403, errorType: "permission_error", wantCode: ErrorPermission},
		{name: "404", status: 404, errorType: "not_found_error", wantCode: ErrorNotFoundOrProtocolMismatch},
		{name: "413", status: 413, errorType: "invalid_request_error", wantCode: ErrorRequestTooLarge},
		{name: "429", status: 429, errorType: "rate_limit_error", wantCode: ErrorRateLimited, wantRetry: true},
		{name: "billing", status: 400, errorType: "billing_error", wantCode: ErrorQuotaExhausted},
		{name: "overloaded", status: 529, errorType: "overloaded_error", wantCode: ErrorProviderUnavailable, wantRetry: true},
		{name: "500", status: 500, errorType: "api_error", wantCode: ErrorProviderUnavailable, wantRetry: true},
		{name: "timeout type", status: 408, errorType: "timeout_error", wantCode: ErrorTimeout, wantRetry: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := &anthropic.Error{StatusCode: tt.status}
			payload := []byte(`{"type":"error","error":{"type":"` + tt.errorType + `","message":"private payload"}}`)
			if err := json.Unmarshal(payload, apiErr); err != nil {
				t.Fatal(err)
			}
			got := ClassifyProviderError(apiErr, StageSendRequest, protocol.AnthropicMessages)
			if got.Code != tt.wantCode || got.HTTPStatus != tt.status || got.Retryable != tt.wantRetry {
				t.Fatalf("unexpected classification: %+v", got)
			}
		})
	}
}

func TestAnthropicTypedErrorNeverExposesPayload(t *testing.T) {
	const sentinel = `sk-ant-secret https://private.invalid Authorization:{token} data:image/png;base64,AAAA C:\Users\private`
	apiErr := &anthropic.Error{StatusCode: 404}
	payload, err := json.Marshal(map[string]any{
		"type":  "error",
		"error": map[string]string{"type": "not_found_error", "message": sentinel},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, apiErr); err != nil {
		t.Fatal(err)
	}
	got := ClassifyProviderError(apiErr, StageSendRequest, protocol.AnthropicMessages)
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	visible := got.Error() + string(encoded)
	for _, fragment := range []string{"sk-ant-secret", "private.invalid", "Authorization", "base64", `C:\Users\private`, "private payload"} {
		if strings.Contains(visible, fragment) {
			t.Fatalf("Anthropic payload leaked fragment %q: %s", fragment, visible)
		}
	}
}

func TestProbeDisabledErrorUsesSafeFixedMessage(t *testing.T) {
	got := NewProviderError(ErrorProbeDisabled, StageValidateProbe, protocol.OpenAIChat, errors.New("sk-secret https://private.invalid"))
	if got.Code != ErrorProbeDisabled || got.Stage != StageValidateProbe || got.Protocol != protocol.OpenAIChat || got.Retryable || got.HTTPStatus != 0 {
		t.Fatalf("unexpected probe-disabled error: %+v", got)
	}
	if got.Message != "CCR 模式不执行付费模型实测，请通过模型目录选择路由" {
		t.Fatalf("unexpected fixed message: %q", got.Message)
	}
	if strings.Contains(got.Message, "secret") || strings.Contains(got.Message, "private.invalid") {
		t.Fatalf("probe-disabled message leaked cause: %q", got.Message)
	}
}
