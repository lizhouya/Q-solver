package llm

import (
	"errors"

	"Q-Solver/pkg/llm/protocol"
)

// ErrorDiagnostic 只传递白名单元数据，不包含 SDK 文本、cause 或请求内容。
// 未知 HTTP 状态省略；无法给出重试建议时 Retryable 为 null。
type ErrorDiagnostic struct {
	Code       ErrorCode `json:"code"`
	Stage      string    `json:"stage"`
	HTTPStatus *int      `json:"httpStatus,omitempty"`
	Protocol   string    `json:"protocol"`
	Retryable  *bool     `json:"retryable"`
	Message    string    `json:"message"`
}

func DiagnoseError(err error) ErrorDiagnostic {
	result := ErrorDiagnostic{Code: ErrorUnknown, Stage: "unknown", Protocol: "unknown"}
	var pe *ProviderError
	if errors.As(err, &pe) && pe != nil {
		result.Code = pe.Code
		result.Stage = string(pe.Stage)
		result.Protocol = string(pe.Protocol)
		status, retry := pe.HTTPStatus, pe.Retryable
		result.HTTPStatus = &status
		result.Retryable = &retry
	}
	return *CleanDiagnostic(&result)
}

// CleanDiagnostic 在传输和快照复制边界重新筛选字段，并隔离可变指针。
func CleanDiagnostic(value *ErrorDiagnostic) *ErrorDiagnostic {
	if value == nil {
		return nil
	}
	result := &ErrorDiagnostic{Code: safeDiagnosticCode(value.Code), Stage: "unknown", Protocol: "unknown"}
	switch ErrorStage(value.Stage) {
	case StageValidateConfig, StageBuildRequest, StageSendRequest, StageReadResponse, StageStreamResponse, StageValidateResponse, StageValidateProbe:
		result.Stage = value.Stage
	}
	switch protocol.APIProtocol(value.Protocol) {
	case protocol.OpenAIChat, protocol.OpenAIResponses, protocol.AnthropicMessages:
		result.Protocol = value.Protocol
	}
	if value.HTTPStatus != nil && *value.HTTPStatus >= 100 && *value.HTTPStatus <= 599 {
		status := *value.HTTPStatus
		result.HTTPStatus = &status
	}
	if result.Code != ErrorUnknown && result.Code != ErrorResponseFailed && value.Retryable != nil {
		retry := *value.Retryable
		result.Retryable = &retry
	}
	result.Message = safeErrorMessage(result.Code)
	return result
}

func safeDiagnosticCode(code ErrorCode) ErrorCode {
	switch code {
	case ErrorInvalidConfig, ErrorCancelled, ErrorTimeout, ErrorNetwork, ErrorTLS,
		ErrorAuthentication, ErrorPermission, ErrorNotFoundOrProtocolMismatch, ErrorInvalidRequest,
		ErrorRateLimited, ErrorQuotaExhausted, ErrorRequestTooLarge, ErrorProviderUnavailable,
		ErrorEmptyResponse, ErrorInvalidResponse, ErrorIncompleteResponse, ErrorStreamInterrupted,
		ErrorRefused, ErrorVisionMismatch, ErrorVisionInconclusive, ErrorProbeDisabled, ErrorResponseFailed:
		return code
	default:
		return ErrorUnknown
	}
}
