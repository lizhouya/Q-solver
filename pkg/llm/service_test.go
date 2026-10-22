package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"Q-Solver/pkg/config"
)

func TestDraftConnectionUsesOwnEndpointWithoutMutatingSavedProvider(t *testing.T) {
	var oldCalls, draftCalls atomic.Int32
	old := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oldCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"saved-model","object":"model"}],"object":"list"}`))
	}))
	defer old.Close()
	draft := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		draftCalls.Add(1)
		if r.Header.Get("Authorization") != "Bearer draft-test-key" {
			t.Error("unexpected synthetic credential")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/models" {
			w.Write([]byte(`{"data":[{"id":"draft-model","object":"model"}],"object":"list"}`))
		} else if r.URL.Path == "/v1/chat/completions" {
			w.Write([]byte(`{"id":"test","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
		} else {
			t.Error("unexpected route")
			w.WriteHeader(404)
		}
	}))
	defer draft.Close()
	s := &Service{config: config.Config{APIKey: "saved-test-key", BaseURL: old.URL + "/v1", Model: "saved-model"}}
	s.updateProviderLocked()
	before := s.GetProvider()
	models, err := s.GetModelsForConnection(context.Background(), "draft-test-key", draft.URL+"/v1")
	if err != nil || len(models) != 1 || models[0] != "draft-model" {
		t.Fatal("draft model request failed")
	}
	if result := s.TestConnectionForConnection(context.Background(), "draft-test-key", draft.URL+"/v1", "draft-model"); result != "" {
		t.Fatal("draft connection test failed")
	}
	if draftCalls.Load() != 2 || oldCalls.Load() != 0 {
		t.Fatal("draft request used saved endpoint")
	}
	if s.GetProvider() != before || s.config.BaseURL != old.URL+"/v1" {
		t.Fatal("saved provider changed")
	}
	if _, err := s.GetModels(context.Background(), "saved-test-key"); err != nil || oldCalls.Load() != 1 {
		t.Fatal("legacy route changed")
	}
}

func TestDraftConnectionRejectsInvalidInput(t *testing.T) {
	s := &Service{}
	for _, u := range []string{"", "file:///tmp/models", "invalid", "https://user:password@example.invalid/v1"} {
		if _, err := s.GetModelsForConnection(context.Background(), "test-key", u); err == nil {
			t.Fatal("invalid endpoint accepted")
		}
	}
	if _, err := s.GetModelsForConnection(context.Background(), "", "http://localhost/v1"); err == nil {
		t.Fatal("empty key accepted")
	}
}
