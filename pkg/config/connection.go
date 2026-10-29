package config

import (
	"net/url"
	"strings"

	"Q-Solver/pkg/llm/protocol"
)

const (
	ConnectionModeDirect = "direct"
	ConnectionModeCCR    = "ccr"
)

func NormalizeConnectionMode(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", ConnectionModeDirect:
		return ConnectionModeDirect, nil
	case ConnectionModeCCR:
		return ConnectionModeCCR, nil
	default:
		return "", &ValidationError{Field: "connectionMode", Message: "连接模式必须是 direct 或 ccr"}
	}
}

func NormalizeCCRBaseURL(value string) (string, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return "", nil
	}
	if strings.ContainsAny(raw, "?#") {
		return "", invalidCCRBaseURLError()
	}
	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" || u.Host == "" || u.Hostname() == "" || u.User != nil {
		return "", invalidCCRBaseURLError()
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", invalidCCRBaseURLError()
	}
	if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" {
		return "", invalidCCRBaseURLError()
	}
	switch u.EscapedPath() {
	case "", "/", "/v1", "/v1/":
	default:
		return "", invalidCCRBaseURLError()
	}
	u.Scheme = scheme
	u.Path = "/v1"
	u.RawPath = ""
	return u.String(), nil
}

func invalidCCRBaseURLError() error {
	return &ValidationError{Field: "baseURL", Message: "CCR 地址必须是 http(s) 根地址或 /v1 API 根地址"}
}

func (c Config) NormalizedConnectionSettings() (Config, error) {
	mode, err := NormalizeConnectionMode(c.ConnectionMode)
	if err != nil {
		return Config{}, err
	}
	c.ConnectionMode = mode
	if mode == ConnectionModeCCR {
		baseURL, normalizeErr := NormalizeCCRBaseURL(c.BaseURL)
		if normalizeErr != nil {
			return Config{}, normalizeErr
		}
		c.BaseURL = baseURL
		c.APIProtocol = string(protocol.OpenAIChat)
		return c, nil
	}
	p, normalizeErr := protocol.Normalize(c.APIProtocol)
	if normalizeErr != nil {
		return Config{}, &ValidationError{Field: "apiProtocol", Message: "API 协议必须是 openai_chat、openai_responses 或 anthropic_messages"}
	}
	c.APIProtocol = string(p)
	return c, nil
}
