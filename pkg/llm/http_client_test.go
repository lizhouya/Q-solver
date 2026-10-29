package llm

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestProviderHTTPClientDoesNotFollowRedirects(t *testing.T) {
	var targetCalls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetCalls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	for _, status := range []int{301, 302, 307, 308} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var sourceCalls atomic.Int32
			source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sourceCalls.Add(1)
				http.Redirect(w, r, target.URL, status)
			}))
			defer source.Close()

			resp, err := newProviderHTTPClient().Get(source.URL)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != status || sourceCalls.Load() != 1 || targetCalls.Load() != 0 {
				t.Fatalf("redirect followed: status=%d source=%d target=%d", resp.StatusCode, sourceCalls.Load(), targetCalls.Load())
			}
		})
	}
}
