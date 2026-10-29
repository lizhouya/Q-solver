package llm

import (
	"Q-Solver/pkg/llm/protocol"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrorDiagnosticPreservesWrappedMetadata(t *testing.T) {
	for _, retry := range []bool{false, true} {
		original := NewProviderError(ErrorPermission, StageStreamResponse, protocol.OpenAIChat, errors.New("TEST_ONLY_PRIVATE_TOKEN"))
		original.HTTPStatus = 403
		original.Retryable = retry
		err := fmt.Errorf("TEST_ONLY_PRIVATE_KEY: %w", fmt.Errorf("TEST_ONLY_PRIVATE_PROMPT: %w", original))
		got := DiagnoseError(err)
		if got.Code != ErrorPermission || got.Stage != string(StageStreamResponse) || got.Protocol != string(protocol.OpenAIChat) || got.HTTPStatus == nil || *got.HTTPStatus != 403 || got.Retryable == nil || *got.Retryable != retry {
			t.Fatalf("lost metadata: %+v", got)
		}
		if got.Message != safeErrorMessage(ErrorPermission) {
			t.Fatal("unexpected friendly message")
		}
		raw, _ := json.Marshal(got)
		if strings.Contains(string(raw), "TEST_ONLY_PRIVATE") {
			t.Fatal("diagnostic leaked wrapped cause")
		}
		classified := ClassifyProviderError(err, StageSendRequest, protocol.OpenAIResponses)
		if classified != original {
			t.Fatal("reclassification discarded original provider metadata")
		}
	}
}

func TestErrorDiagnosticUnknownAndMissingValues(t *testing.T) {
	var typedNil *ProviderError
	for _, err := range []error{nil, typedNil, errors.New("TEST_ONLY_PRIVATE_KEY"), fmt.Errorf("wrapped: %w", typedNil)} {
		got := DiagnoseError(err)
		if got.Code != ErrorUnknown || got.Stage != "unknown" || got.Protocol != "unknown" || got.HTTPStatus != nil || got.Retryable != nil {
			t.Fatalf("invented metadata: %+v", got)
		}
	}
	e := NewProviderError(ErrorTimeout, StageSendRequest, protocol.AnthropicMessages, nil)
	if got := DiagnoseError(e); got.HTTPStatus != nil || got.Retryable == nil || !*got.Retryable {
		t.Fatal("missing HTTP status must remain unknown")
	}
	if got := ClassifyProviderError(errors.New("TEST_ONLY_PRIVATE_BODY"), StageReadResponse, protocol.OpenAIChat); got.Code != ErrorUnknown {
		t.Fatal("opaque errors must not be called network failures")
	}
}

func TestErrorDiagnosticWhitelistsAllUntrustedStrings(t *testing.T) {
	e := &ProviderError{Code: "TEST_ONLY_PRIVATE_KEY", Stage: "TEST_ONLY_PRIVATE_TOKEN", Protocol: "TEST_ONLY_PRIVATE_URL", Message: "TEST_ONLY_PRIVATE_PROMPT", HTTPStatus: 999, Retryable: true}
	got := DiagnoseError(e)
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), "TEST_ONLY_PRIVATE") || got.Code != ErrorUnknown || got.HTTPStatus != nil || got.Retryable != nil {
		t.Fatalf("unsafe summary: %s", raw)
	}
	e.Code = ErrorAuthentication
	e.Stage = StageSendRequest
	e.Protocol = protocol.OpenAIResponses
	e.HTTPStatus = 401
	got = DiagnoseError(e)
	if got.Message != safeErrorMessage(ErrorAuthentication) {
		t.Fatal("provider Message must not bypass fixed safe messages")
	}
	copy := CleanDiagnostic(&got)
	*copy.HTTPStatus = 500
	*copy.Retryable = false
	if *got.HTTPStatus != 401 || !*got.Retryable {
		t.Fatal("diagnostic copy shares mutable fields")
	}
	if CleanDiagnostic(nil) != nil {
		t.Fatal("legacy missing diagnostic must remain absent")
	}
}

func TestUnknownFailureDiagnosticDoesNotInventRetryAdvice(t *testing.T) {
	for _, code := range []ErrorCode{ErrorUnknown, ErrorResponseFailed} {
		pe := NewProviderError(code, StageValidateResponse, protocol.OpenAIResponses, nil)
		got := DiagnoseError(pe)
		if got.Retryable != nil || got.HTTPStatus != nil {
			t.Fatal("unknown failure must not invent retry advice or HTTP status")
		}
		raw, _ := json.Marshal(got)
		if strings.Contains(string(raw), "httpStatus") || !strings.Contains(string(raw), `"retryable":null`) {
			t.Fatalf("unknown field wire format: %s", raw)
		}
	}
}
