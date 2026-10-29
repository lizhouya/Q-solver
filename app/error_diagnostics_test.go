package app

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/llm/protocol"
	"Q-Solver/pkg/questionbank"
	"Q-Solver/pkg/solution"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type diagnosticFailureProvider struct{ err error }

func (p diagnosticFailureProvider) GenerateContentStream(context.Context, []llm.Message, llm.StreamCallback) (llm.Message, error) {
	return llm.Message{}, p.err
}
func (p diagnosticFailureProvider) GenerateContent(context.Context, string, []llm.Message) (llm.Message, error) {
	return llm.Message{}, p.err
}
func (p diagnosticFailureProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (p diagnosticFailureProvider) TestChat(context.Context) error              { return p.err }

func TestSolveDiagnosticSurvivesCaptureAndDesktopEnvelope(t *testing.T) {
	a, store, img := archiveApp(t)
	pe := llm.NewProviderError(llm.ErrorPermission, llm.StageStreamResponse, protocol.OpenAIChat, errors.New("TEST_ONLY_PRIVATE_RESPONSE"))
	pe.HTTPStatus = 403
	cfg := config.Config{APIKey: "TEST_ONLY_PRIVATE_KEY", Model: "test-model"}
	var envelopes []DesktopSolveEvent
	target := desktopSolveTarget{epoch: 7, sessionID: "session-test", recordID: "record-test"}
	capture := a.beginQuestionSolveCapture(context.Background(), cfg, []string{img}, func(event string, data ...interface{}) {
		if event == "solution-error" {
			if _, ok := data[0].(string); !ok {
				t.Fatal("legacy consumer no longer sees string")
			}
			envelopes = append(envelopes, newDesktopSolveEvent(target, event, data...))
		}
	})
	solver := solution.NewSolver(diagnosticFailureProvider{fmt.Errorf("TEST_ONLY_PRIVATE_TOKEN: %w", pe)})
	if solver.Solve(context.Background(), solution.Request{Config: cfg, Screenshots: []string{img}}, solution.Callbacks{EmitEvent: capture.Emit}) {
		t.Fatal("failure became success")
	}
	capture.Complete(false, nil)
	if len(envelopes) != 1 {
		t.Fatalf("expected one terminal envelope, got %d", len(envelopes))
	}
	e := envelopes[0]
	if e.ErrorDetail == nil || e.ErrorDetail.Code != llm.ErrorPermission || e.ErrorDetail.HTTPStatus == nil || *e.ErrorDetail.HTTPStatus != 403 || e.Epoch != 7 || e.RecordID != "record-test" {
		t.Fatal("lost scoped diagnostic")
	}
	raw, _ := json.Marshal(e)
	if strings.Contains(string(raw), "TEST_ONLY_PRIVATE") {
		t.Fatal("private data leaked into desktop envelope")
	}
	snap, dirty := store.Snapshot()
	if dirty || snap.Records[0].Status != questionbank.Failed {
		t.Fatal("capture failure state changed")
	}
	raw, _ = json.Marshal(snap)
	if strings.Contains(string(raw), "TEST_ONLY_PRIVATE") || strings.Contains(string(raw), "errorDetail") {
		t.Fatal("unexpected diagnostic/private persistence")
	}
	old := newDesktopSolveEvent(target, "solution-error", "legacy error")
	if old.Text != "legacy error" || old.ErrorDetail != nil {
		t.Fatal("legacy envelope changed")
	}
}
