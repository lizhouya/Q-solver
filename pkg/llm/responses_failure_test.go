package llm

import (
	"Q-Solver/pkg/llm/protocol"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/openai/openai-go/responses"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestResponsesFailureCodeMapping(t *testing.T) {
	tests := []struct {
		code  string
		want  ErrorCode
		retry bool
	}{
		{string(responses.ResponseErrorCodeRateLimitExceeded), ErrorRateLimited, true},
		{string(responses.ResponseErrorCodeInvalidPrompt), ErrorInvalidRequest, false},
		{string(responses.ResponseErrorCodeServerError), ErrorProviderUnavailable, true},
		{string(responses.ResponseErrorCodeVectorStoreTimeout), ErrorTimeout, true},
		{string(responses.ResponseErrorCodeImageTooLarge), ErrorRequestTooLarge, false},
		{string(responses.ResponseErrorCodeImageFileTooLarge), ErrorRequestTooLarge, false},
		{string(responses.ResponseErrorCodeImageContentPolicyViolation), ErrorRefused, false},
		{string(responses.ResponseErrorCodeImageFileNotFound), ErrorInvalidRequest, false},
		{"invalid_api_key", ErrorAuthentication, false},
		{"authentication_error", ErrorAuthentication, false},
		{"permission_denied", ErrorPermission, false},
		{"permission_error", ErrorPermission, false},
		{"insufficient_quota", ErrorQuotaExhausted, false},
		{"invalid_request_error", ErrorInvalidRequest, false},
		{"TEST_ONLY_PRIVATE_UNKNOWN_CODE", ErrorCode("response_failed"), false},
		{"", ErrorCode("response_failed"), false},
	}
	for _, tt := range tests {
		for _, kind := range []string{"non-stream", "failed-event", "error-event"} {
			t.Run(kind+"/"+tt.code, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Path != "/v1/responses" {
						t.Errorf("unexpected endpoint: %s", r.URL.Path)
					}
					// Code is a string-backed SDK type; vendor extensions use the real error.code field.
					response := responses.Response{Status: responses.ResponseStatusFailed, Error: responses.ResponseError{Code: responses.ResponseErrorCode(tt.code), Message: "TEST_ONLY_PRIVATE_RESPONSE"}}
					raw, _ := json.Marshal(response)
					if kind == "non-stream" {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write(raw)
						return
					}
					w.Header().Set("Content-Type", "text/event-stream")
					if kind == "failed-event" {
						fmt.Fprintf(w, "event: response.failed\ndata: {\"type\":\"response.failed\",\"response\":%s}\n\n", raw)
					} else {
						event, _ := json.Marshal(map[string]any{"type": "error", "code": tt.code, "message": "TEST_ONLY_PRIVATE_RESPONSE", "param": ""})
						fmt.Fprintf(w, "event: error\ndata: %s\n\n", event)
					}
				}))
				defer server.Close()
				adapter := newOpenAIResponsesTestAdapter(server, "test-model")
				var err error
				if kind == "non-stream" {
					_, err = adapter.GenerateContent(context.Background(), "", []Message{NewUserMessage("TEST_ONLY_PRIVATE_PROMPT")})
				} else {
					_, err = adapter.GenerateContentStream(context.Background(), []Message{NewUserMessage("TEST_ONLY_PRIVATE_PROMPT")}, nil)
				}
				var pe *ProviderError
				if !errors.As(err, &pe) || pe == nil {
					t.Fatal("missing structured provider error")
				}
				if pe.Code != tt.want || pe.Retryable != tt.retry || pe.HTTPStatus != 0 || pe.Protocol != protocol.OpenAIResponses {
					t.Fatalf("wrong classification: %+v", pe)
				}
				expectedStage := StageValidateResponse
				if kind == "error-event" {
					expectedStage = StageStreamResponse
				}
				if pe.Stage != expectedStage {
					t.Fatalf("wrong stage: %s", pe.Stage)
				}
				if strings.Contains(pe.Error(), "TEST_ONLY_PRIVATE") {
					t.Fatal("raw response leaked")
				}
				if calls.Load() != 1 {
					t.Fatalf("unexpected retry/fallback: %d", calls.Load())
				}
			})
		}
	}
}

func TestResponsesHTTPFailureKeepsActualStatus(t *testing.T) {
	for _, status := range []int{401, 403, 429, 503} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/stream=%v", status, stream), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					fmt.Fprint(w, `{"error":{"message":"TEST_ONLY_PRIVATE_RESPONSE","code":"test_error","type":"test_error"}}`)
				}))
				defer server.Close()
				a := newOpenAIResponsesTestAdapter(server, "test-model")
				var err error
				if stream {
					_, err = a.GenerateContentStream(context.Background(), []Message{NewUserMessage("hi")}, nil)
				} else {
					_, err = a.GenerateContent(context.Background(), "", []Message{NewUserMessage("hi")})
				}
				detail := DiagnoseError(err)
				want := map[int]ErrorCode{401: ErrorAuthentication, 403: ErrorPermission, 429: ErrorRateLimited, 503: ErrorProviderUnavailable}[status]
				if detail.Code != want || detail.HTTPStatus == nil || *detail.HTTPStatus != status {
					t.Fatalf("wrong HTTP classification: %+v", detail)
				}
			})
		}
	}
}
