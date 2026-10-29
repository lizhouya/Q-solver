package llm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"strings"

	"Q-Solver/pkg/llm/protocol"

	anthropic "github.com/anthropics/anthropic-sdk-go/v2"
	openai "github.com/openai/openai-go"
)

type ErrorCode string

const (
	ErrorUnknown                    ErrorCode = "unknown"
	ErrorResponseFailed             ErrorCode = "response_failed"
	ErrorInvalidConfig              ErrorCode = "invalid_config"
	ErrorCancelled                  ErrorCode = "cancelled"
	ErrorTimeout                    ErrorCode = "timeout"
	ErrorNetwork                    ErrorCode = "network_error"
	ErrorTLS                        ErrorCode = "tls_error"
	ErrorAuthentication             ErrorCode = "authentication_failed"
	ErrorPermission                 ErrorCode = "permission_denied"
	ErrorNotFoundOrProtocolMismatch ErrorCode = "not_found_or_protocol_mismatch"
	ErrorInvalidRequest             ErrorCode = "invalid_request"
	ErrorRateLimited                ErrorCode = "rate_limited"
	ErrorQuotaExhausted             ErrorCode = "quota_exhausted"
	ErrorRequestTooLarge            ErrorCode = "request_too_large"
	ErrorProviderUnavailable        ErrorCode = "provider_unavailable"
	ErrorEmptyResponse              ErrorCode = "empty_response"
	ErrorInvalidResponse            ErrorCode = "invalid_response"
	ErrorIncompleteResponse         ErrorCode = "incomplete_response"
	ErrorStreamInterrupted          ErrorCode = "stream_interrupted"
	ErrorRefused                    ErrorCode = "refused"
	ErrorVisionMismatch             ErrorCode = "vision_mismatch"
	ErrorVisionInconclusive         ErrorCode = "vision_inconclusive"
	ErrorProbeDisabled              ErrorCode = "probe_disabled"
)

type ErrorStage string

const (
	StageValidateConfig   ErrorStage = "validate_config"
	StageBuildRequest     ErrorStage = "build_request"
	StageSendRequest      ErrorStage = "send_request"
	StageReadResponse     ErrorStage = "read_response"
	StageStreamResponse   ErrorStage = "stream_response"
	StageValidateResponse ErrorStage = "validate_response"
	StageValidateProbe    ErrorStage = "validate_probe"
)

type ProviderError struct {
	Code       ErrorCode            `json:"code"`
	Stage      ErrorStage           `json:"stage"`
	Message    string               `json:"message"`
	Retryable  bool                 `json:"retryable"`
	HTTPStatus int                  `json:"httpStatus"`
	Protocol   protocol.APIProtocol `json:"protocol"`
	cause      error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "模型请求失败"
	}
	return e.Message
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func NewProviderError(code ErrorCode, stage ErrorStage, p protocol.APIProtocol, cause error) *ProviderError {
	return &ProviderError{Code: code, Stage: stage, Message: safeErrorMessage(code), Retryable: defaultRetryable(code), Protocol: p, cause: cause}
}

func ClassifyProviderError(err error, stage ErrorStage, p protocol.APIProtocol) *ProviderError {
	if err == nil {
		return nil
	}
	var providerErr *ProviderError
	if errors.As(err, &providerErr) && providerErr != nil {
		return providerErr
	}
	if errors.Is(err, context.Canceled) {
		return NewProviderError(ErrorCancelled, stage, p, err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return NewProviderError(ErrorTimeout, stage, p, err)
	}

	var unknownAuthority x509.UnknownAuthorityError
	var hostnameError x509.HostnameError
	var certificateInvalid x509.CertificateInvalidError
	var tlsRecord tls.RecordHeaderError
	if errors.As(err, &unknownAuthority) || errors.As(err, &hostnameError) || errors.As(err, &certificateInvalid) || errors.As(err, &tlsRecord) {
		return NewProviderError(ErrorTLS, stage, p, err)
	}

	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		result := NewProviderError(errorCodeForHTTP(apiErr.StatusCode, apiErr.Code), stage, p, err)
		result.HTTPStatus = apiErr.StatusCode
		return result
	}

	var anthropicErr *anthropic.Error
	if errors.As(err, &anthropicErr) {
		result := NewProviderError(errorCodeForAnthropic(anthropicErr.StatusCode, string(anthropicErr.Type())), stage, p, err)
		result.HTTPStatus = anthropicErr.StatusCode
		return result
	}

	var networkErr net.Error
	if errors.As(err, &networkErr) {
		if networkErr.Timeout() {
			return NewProviderError(ErrorTimeout, stage, p, err)
		}
		return NewProviderError(ErrorNetwork, stage, p, err)
	}
	return NewProviderError(ErrorUnknown, stage, p, err)
}

func errorCodeForAnthropic(status int, sdkType string) ErrorCode {
	switch strings.ToLower(strings.TrimSpace(sdkType)) {
	case "authentication_error":
		return ErrorAuthentication
	case "permission_error":
		return ErrorPermission
	case "not_found_error":
		return ErrorNotFoundOrProtocolMismatch
	case "rate_limit_error":
		return ErrorRateLimited
	case "timeout_error":
		return ErrorTimeout
	case "overloaded_error", "api_error":
		return ErrorProviderUnavailable
	case "billing_error":
		return ErrorQuotaExhausted
	case "invalid_request_error":
		if status == http.StatusRequestEntityTooLarge {
			return ErrorRequestTooLarge
		}
		return ErrorInvalidRequest
	default:
		return errorCodeForHTTP(status, "")
	}
}

func errorCodeForHTTP(status int, sdkCode string) ErrorCode {
	switch status {
	case 400, 422:
		return ErrorInvalidRequest
	case 401:
		return ErrorAuthentication
	case 403:
		return ErrorPermission
	case 404:
		return ErrorNotFoundOrProtocolMismatch
	case 413:
		return ErrorRequestTooLarge
	case 429:
		if strings.EqualFold(strings.TrimSpace(sdkCode), "insufficient_quota") {
			return ErrorQuotaExhausted
		}
		return ErrorRateLimited
	default:
		if status >= 500 && status <= 599 {
			return ErrorProviderUnavailable
		}
		return ErrorInvalidResponse
	}
}

func defaultRetryable(code ErrorCode) bool {
	switch code {
	case ErrorTimeout, ErrorNetwork, ErrorRateLimited, ErrorProviderUnavailable, ErrorStreamInterrupted:
		return true
	default:
		return false
	}
}

func safeErrorMessage(code ErrorCode) string {
	switch code {
	case ErrorInvalidConfig:
		return "模型连接配置无效，请检查后重试"
	case ErrorProbeDisabled:
		return "CCR 模式不执行付费模型实测，请通过模型目录选择路由"
	case ErrorCancelled:
		return "模型请求已取消"
	case ErrorTimeout:
		return "模型请求超时，请稍后重试"
	case ErrorTLS:
		return "安全连接建立失败，请检查接口证书或网络环境"
	case ErrorAuthentication:
		return "API Key 无效或鉴权失败"
	case ErrorPermission:
		return "当前凭据没有访问该模型或接口的权限"
	case ErrorNotFoundOrProtocolMismatch:
		return "未找到请求资源，请检查端点、协议或模型是否匹配"
	case ErrorInvalidRequest:
		return "模型服务拒绝了当前请求，请检查协议和模型配置"
	case ErrorRateLimited:
		return "请求过于频繁，请稍后重试"
	case ErrorQuotaExhausted:
		return "当前账号额度不足，请检查服务商额度"
	case ErrorRequestTooLarge:
		return "本次请求内容过大，请减少图片或文档内容"
	case ErrorProviderUnavailable:
		return "模型服务暂时不可用，请稍后重试"
	case ErrorEmptyResponse:
		return "模型返回了空内容"
	case ErrorInvalidResponse:
		return "模型返回格式无效"
	case ErrorIncompleteResponse:
		return "模型未完整完成本次回答"
	case ErrorStreamInterrupted:
		return "模型流式响应意外中断，请重试"
	case ErrorRefused:
		return "模型拒绝了本次请求"
	case ErrorVisionMismatch:
		return "图片识别结果与测试内容不一致"
	case ErrorVisionInconclusive:
		return "无法确认当前通道是否成功识别图片"
	case ErrorNetwork:
		return "无法连接模型服务，请检查网络和接口地址"
	case ErrorResponseFailed:
		return "模型响应失败，暂无法确定原因，请查看诊断详情"
	default:
		return "模型请求失败，原因未知，请查看诊断详情"
	}
}
