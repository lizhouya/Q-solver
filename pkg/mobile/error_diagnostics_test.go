package mobile

import (
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/llm/protocol"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMobileErrorDiagnosticSnapshotCompatibility(t *testing.T) {
	b, err := NewBridge(1)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	pe := llm.NewProviderError(llm.ErrorRateLimited, llm.StageStreamResponse, protocol.OpenAIChat, errors.New("TEST_ONLY_PRIVATE_TOKEN"))
	pe.HTTPStatus = 429
	detail := llm.DiagnoseError(pe)
	b.Publish("solution-stream-start")
	ch, _ := b.Subscribe()
	defer b.Unsubscribe(ch)
	b.Publish("solution-error", detail.Message, detail)
	s := <-ch
	if s.Status != StatusError || s.Error != detail.Message || s.ErrorDetail == nil || s.ErrorDetail.HTTPStatus == nil || *s.ErrorDetail.HTTPStatus != 429 {
		t.Fatal("lost atomic error diagnostic")
	}
	raw, _ := json.Marshal(s)
	if strings.Contains(string(raw), "TEST_ONLY_PRIVATE") {
		t.Fatal("secret leaked to SSE snapshot")
	}
	*s.ErrorDetail.HTTPStatus = 999
	*s.ErrorDetail.Retryable = false
	saved := b.GetState()
	if *saved.ErrorDetail.HTTPStatus != 429 || !*saved.ErrorDetail.Retryable {
		t.Fatal("subscriber mutated bridge state")
	}
	_, initial := b.Subscribe()
	if initial.ErrorDetail == nil {
		t.Fatal("reconnect lost error detail")
	}
	b.Publish("solution-stream-start")
	if b.GetState().ErrorDetail != nil {
		t.Fatal("old error leaked into next request")
	}
	b.Publish("solution-error", "legacy error")
	if b.GetState().Error != "legacy error" || b.GetState().ErrorDetail != nil {
		t.Fatal("legacy error must remain compatible")
	}
	b.Publish("solution-error", detail.Message, detail)
	b.Publish("solution", "answer")
	if b.GetState().ErrorDetail != nil {
		t.Fatal("success retained stale error")
	}
	b.ClearDisplay(2)
	if b.PublishIfCurrent(1, "solution-error", detail.Message, detail) || b.GetState().ErrorDetail != nil {
		t.Fatal("old task crossed clear boundary")
	}
}

func TestMobileErrorDiagnosticOverRealSSE(t *testing.T) {
	b := newTestBridge(t)
	server, err := NewServer(b, testToken)
	if err != nil {
		t.Fatal(err)
	}
	endpoint := httptest.NewServer(server.routes(http.NewServeMux()))
	defer endpoint.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.URL+"/sse?token="+server.connectToken(), nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := endpoint.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)
	readSSEFrame(t, reader)
	b.Publish("solution-stream-start")
	readSSEFrame(t, reader)
	pe := llm.NewProviderError(llm.ErrorAuthentication, llm.StageStreamResponse, protocol.OpenAIChat, errors.New("TEST_ONLY_PRIVATE_AUTHORIZATION"))
	pe.HTTPStatus = 401
	detail := llm.DiagnoseError(pe)
	b.Publish("solution-error", detail.Message, detail)
	_, snap, _ := readSSEFrame(t, reader)
	if snap.Status != StatusError || snap.Error != detail.Message || snap.ErrorDetail == nil || snap.ErrorDetail.HTTPStatus == nil || *snap.ErrorDetail.HTTPStatus != 401 {
		t.Fatal("SSE lost structured error")
	}
	raw, _ := json.Marshal(snap)
	if strings.Contains(string(raw), "TEST_ONLY_PRIVATE") {
		t.Fatal("SSE exposed raw error")
	}
	// 旧客户端只解析既有字段，新字段不会改变错误字符串或 SSE 事件类型。
	var legacy struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if json.Unmarshal(raw, &legacy) != nil || legacy.Status != StatusError || legacy.Error != detail.Message {
		t.Fatal("legacy SSE client failed")
	}
}
