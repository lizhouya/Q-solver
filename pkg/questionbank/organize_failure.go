package questionbank

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	OrganizeOutcomePending    = "pending"
	OrganizeOutcomeComplete   = "complete"
	OrganizeOutcomePartial    = "partial"
	OrganizeOutcomeAllFailed  = "all_failed"
	OrganizeOutcomeNoEligible = "no_eligible"
)

const (
	organizeCodeInputInvalid   = "input_invalid"
	organizeCodeRequestTimeout = "request_timeout"
	organizeCodeRequestFailed  = "request_failed"
	organizeCodeEmptyResponse  = "empty_response"
	organizeCodeResponseLarge  = "response_too_large"
	organizeCodeInvalidJSON    = "invalid_json"
	organizeCodeExtraContent   = "extra_content"
	organizeCodeSourceMismatch = "source_mismatch"
	organizeCodeSchemaInvalid  = "schema_invalid"
	organizeCodeLegacyUnknown  = "legacy_unknown"
)

type organizeFailureError struct {
	code      string
	stage     string
	retryable bool
	message   string
}

func (e *organizeFailureError) Error() string { return e.message }

func newOrganizeFailureError(code, stage, message string, retryable bool) error {
	return &organizeFailureError{code: code, stage: stage, message: message, retryable: retryable}
}

func safeOrganizeFailure(err error, model string, providerCalled bool, responseBytes int, attemptedAt time.Time) (*OrganizeFailure, string) {
	if err == nil {
		return nil, ""
	}
	failure := &organizeFailureError{
		code: organizeCodeLegacyUnknown, stage: "unknown", retryable: true,
		message: "整理失败；原稿已保留，可只重试失败记录",
	}
	var typed *organizeFailureError
	if errors.As(err, &typed) {
		failure = typed
	} else if errors.Is(err, context.DeadlineExceeded) {
		failure = &organizeFailureError{
			code: organizeCodeRequestTimeout, stage: "request", retryable: true,
			message: "模型整理请求超时；原稿已保留，可只重试失败记录",
		}
	}
	return &OrganizeFailure{
		Code: failure.code, Stage: failure.stage, Retryable: failure.retryable,
		ProviderCalled: providerCalled, AttemptedAt: attemptedAt, Model: strings.TrimSpace(model),
		ResponseBytes: responseBytes,
	}, failure.message
}

type OrganizeSummary struct {
	Outcome             string `json:"outcome"`
	EligibleRecords     int    `json:"eligibleRecords"`
	OrganizedRecords    int    `json:"organizedRecords"`
	StructuredQuestions int    `json:"structuredQuestions"`
	PendingRecords      int    `json:"pendingRecords"`
	FailedRecords       int    `json:"failedRecords"`
}

func SummarizeOrganizeState(session *Session) OrganizeSummary {
	summary := OrganizeSummary{}
	if session == nil {
		summary.Outcome = OrganizeOutcomeNoEligible
		return summary
	}
	for _, record := range session.Records {
		if record.Status != Completed {
			continue
		}
		summary.EligibleRecords++
		if len(record.Questions) > 0 {
			summary.OrganizedRecords++
			summary.StructuredQuestions += len(record.Questions)
			continue
		}
		if record.OrganizeFailure != nil || strings.TrimSpace(record.OrganizeError) != "" {
			summary.FailedRecords++
		} else {
			summary.PendingRecords++
		}
	}
	switch {
	case summary.EligibleRecords == 0:
		summary.Outcome = OrganizeOutcomeNoEligible
	case summary.PendingRecords > 0:
		summary.Outcome = OrganizeOutcomePending
	case summary.StructuredQuestions == 0 && summary.FailedRecords > 0:
		summary.Outcome = OrganizeOutcomeAllFailed
	case summary.StructuredQuestions > 0 && summary.FailedRecords > 0:
		summary.Outcome = OrganizeOutcomePartial
	case summary.StructuredQuestions > 0:
		summary.Outcome = OrganizeOutcomeComplete
	default:
		summary.Outcome = OrganizeOutcomePending
	}
	return summary
}

func organizeBatchError(failed int) error {
	if failed == 0 {
		return nil
	}
	return fmt.Errorf("%d 条记录整理失败；原稿和已完成结果已保留，可只重试失败记录", failed)
}
