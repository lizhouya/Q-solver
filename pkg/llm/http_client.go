package llm

import (
	"net/http"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"
)

func newProviderHTTPClient(cfg config.Config, p protocol.APIProtocol) *http.Client {
	return &http.Client{
		Transport: newConnectionRoundTripper(cfg, p),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
