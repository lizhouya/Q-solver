package solution

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/llm/protocol"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSolveErrorCarriesAtomicSafeDiagnostic(t *testing.T) {
	pe := llm.NewProviderError(llm.ErrorProviderUnavailable, llm.StageStreamResponse, protocol.OpenAIChat, errors.New("TEST_ONLY_PRIVATE_RESPONSE"))
	pe.HTTPStatus = 502
	for _, err := range []error{fmt.Errorf("TEST_ONLY_PRIVATE_KEY: %w", pe), errors.New("TEST_ONLY_PRIVATE_TOKEN")} {
		solver := NewSolver(&fakeProvider{err: err})
		count := 0
		ok := solver.Solve(context.Background(), Request{Config: config.Config{APIKey: "TEST_ONLY_PRIVATE_KEY"}, Screenshots: []string{"TEST_ONLY_PRIVATE_IMAGE"}}, Callbacks{EmitEvent: func(event string, data ...interface{}) {
			if event != "solution-error" {
				return
			}
			count++
			if len(data) != 2 {
				t.Fatalf("need legacy string plus diagnostic, got %d args", len(data))
			}
			text, isString := data[0].(string)
			detail, isDetail := data[1].(llm.ErrorDiagnostic)
			if !isString || !isDetail || text != detail.Message {
				t.Fatal("error contract changed")
			}
			raw, _ := json.Marshal(data)
			if strings.Contains(string(raw), "TEST_ONLY_PRIVATE") {
				t.Fatal("private data leaked in event")
			}
			if errors.Is(err, pe) && (detail.HTTPStatus == nil || *detail.HTTPStatus != 502 || detail.Protocol != "openai_chat") {
				t.Fatal("lost metadata")
			}
		}})
		if ok || count != 1 {
			t.Fatalf("wrong terminal outcome: ok=%v errors=%d", ok, count)
		}
	}
}

func TestSolveDiagnosticForCancelledAndEmptyTerminal(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		provider := &fakeProvider{}
		if cancelled {
			cancel()
			provider.err = context.Canceled
		}
		var args []interface{}
		NewSolver(provider).Solve(ctx, Request{Config: config.Config{APIKey: "test-key"}}, Callbacks{EmitEvent: func(event string, data ...interface{}) {
			if event == "solution-error" {
				args = data
			}
		}})
		cancel()
		if len(args) != 2 {
			t.Fatalf("missing terminal diagnostic: cancelled=%v", cancelled)
		}
		detail, ok := args[1].(llm.ErrorDiagnostic)
		if !ok {
			t.Fatal("invalid diagnostic type")
		}
		if cancelled && (args[0] != "context canceled" || detail.Code != llm.ErrorCancelled) {
			t.Fatal("cancellation contract changed")
		}
		if !cancelled && detail.Code != llm.ErrorEmptyResponse {
			t.Fatal("empty response classification changed")
		}
	}
}
