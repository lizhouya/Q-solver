package llm

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm/protocol"
)

type connectionRoundTripper struct {
	base            http.RoundTripper
	basePath        string
	defaultEndpoint string
	endpointPath    string
	modelsPath      string
	extraHeaders    map[string]string
	omitDefaultAuth bool
}

func newConnectionRoundTripper(cfg config.Config, p protocol.APIProtocol) http.RoundTripper {
	basePath := ""
	if parsed, err := url.Parse(strings.TrimSpace(cfg.BaseURL)); err == nil {
		basePath = strings.TrimRight(parsed.Path, "/")
	}
	defaultEndpoint := config.DefaultEndpointPath(string(p))
	endpointPath := strings.TrimSpace(cfg.EndpointPath)
	if endpointPath == "" {
		endpointPath = defaultEndpoint
	}
	modelsEndpoint := "/models"
	if p == protocol.AnthropicMessages {
		modelsEndpoint = "/v1/models"
	}
	return &connectionRoundTripper{
		base:            http.DefaultTransport,
		basePath:        basePath,
		defaultEndpoint: defaultEndpoint,
		endpointPath:    endpointPath,
		modelsPath:      joinAPIPath(basePath, modelsEndpoint),
		extraHeaders:    cloneStringMap(cfg.ExtraHeaders),
		omitDefaultAuth: strings.TrimSpace(cfg.APIKey) == "",
	}
}

func (t *connectionRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()

	if clone.Method == http.MethodPost && strings.HasSuffix(clone.URL.Path, t.defaultEndpoint) {
		clone.URL.Path = joinAPIPath(t.basePath, t.endpointPath)
		clone.URL.RawPath = ""
	}
	if clone.Method == http.MethodGet && path.Clean(clone.URL.Path) == joinAPIPath(t.basePath, "") {
		clone.URL.Path = t.modelsPath
		clone.URL.RawPath = ""
	}
	if t.omitDefaultAuth {
		clone.Header.Del("Authorization")
		clone.Header.Del("X-Api-Key")
	}
	for key, value := range t.extraHeaders {
		clone.Header.Set(key, value)
	}

	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

func joinAPIPath(basePath, endpointPath string) string {
	basePath = strings.TrimSpace(basePath)
	endpointPath = strings.TrimSpace(endpointPath)
	if endpointPath == "" {
		return path.Clean("/" + strings.TrimPrefix(basePath, "/"))
	}
	return path.Join("/", basePath, endpointPath)
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
