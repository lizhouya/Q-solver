package config

import (
	"encoding/json"
	"net/textproto"
	"net/url"
	"path"
	"strings"

	"Q-Solver/pkg/llm/protocol"
)

const (
	ConnectionModeDirect = "direct"
	ConnectionModeCCR    = "ccr"

	ResponseFormatText       = "text"
	ResponseFormatJSONObject = "json_object"

	DefaultCustomAuthHeader  = "Authorization"
	DefaultCustomAuthPrefix  = "Bearer "
	MaxRequestTemplateBytes  = 64 * 1024
	maxCustomAuthPrefixBytes = 128

	maxExtraHeaderCount = 32
	maxExtraHeaderBytes = 16 * 1024
)

var forbiddenExtraHeaders = map[string]struct{}{
	"connection":        {},
	"content-length":    {},
	"host":              {},
	"proxy-connection":  {},
	"te":                {},
	"trailer":           {},
	"transfer-encoding": {},
	"upgrade":           {},
}

var allowedRequestTemplatePlaceholders = map[string]struct{}{
	"{{model}}":              {},
	"{{system_prompt}}":      {},
	"{{user_text}}":          {},
	"{{openai_messages}}":    {},
	"{{responses_input}}":    {},
	"{{anthropic_messages}}": {},
	"{{anthropic_system}}":   {},
	"{{images_data_urls}}":   {},
	"{{images_base64}}":      {},
}

var contentRequestTemplatePlaceholders = map[string]struct{}{
	"{{system_prompt}}":      {},
	"{{user_text}}":          {},
	"{{openai_messages}}":    {},
	"{{responses_input}}":    {},
	"{{anthropic_messages}}": {},
	"{{anthropic_system}}":   {},
	"{{images_data_urls}}":   {},
	"{{images_base64}}":      {},
}

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

func NormalizeCustomRequestURL(value string) (string, error) {
	raw := strings.TrimSpace(value)
	u, err := url.Parse(raw)
	if err != nil || raw == "" || strings.ContainsRune(value, 13) || strings.ContainsRune(value, 10) || strings.ContainsAny(raw, "?#") || u.Opaque != "" || u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", &ValidationError{Field: "baseURL", Message: "自定义请求地址必须是合法且不含凭据、query 或 fragment 的 http(s) URL"}
	}
	return u.String(), nil
}

func NormalizeOpenAIChatAddress(value string) (baseURL string, endpointPath string, err error) {
	baseURL, err = NormalizeBaseURL(value)
	if err != nil || baseURL == "" {
		return baseURL, "/chat/completions", err
	}
	const chatEndpoint = "/chat/completions"
	if strings.HasSuffix(baseURL, chatEndpoint) {
		baseURL = strings.TrimSuffix(baseURL, chatEndpoint)
		if baseURL == "" {
			return "", "", &ValidationError{Field: "baseURL", Message: "OpenAI 兼容请求地址缺少 API root"}
		}
	}
	return baseURL, chatEndpoint, nil
}

func NormalizeBaseURL(value string) (string, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return "", nil
	}
	if strings.ContainsAny(raw, "\r\n") {
		return "", &ValidationError{Field: "baseURL", Message: "接口地址必须是合法的 http(s) API root"}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" || u.Host == "" || u.Hostname() == "" || u.User != nil {
		return "", &ValidationError{Field: "baseURL", Message: "接口地址必须是合法的 http(s) API root"}
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", &ValidationError{Field: "baseURL", Message: "接口地址必须使用 http 或 https"}
	}
	if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" {
		return "", &ValidationError{Field: "baseURL", Message: "接口地址不能包含 query 或 fragment"}
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	return u.String(), nil
}

func DefaultEndpointPath(value string) string {
	p, _ := protocol.Normalize(value)
	switch p {
	case protocol.OpenAIResponses:
		return "/responses"
	case protocol.AnthropicMessages:
		return "/v1/messages"
	default:
		return "/chat/completions"
	}
}

func NormalizeEndpointPath(value, protocolValue string) (string, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return DefaultEndpointPath(protocolValue), nil
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "?#\r\n") {
		return "", &ValidationError{Field: "endpointPath", Message: "Endpoint Path 必须是以 / 开头且不含 query 或 fragment 的相对路径"}
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", &ValidationError{Field: "endpointPath", Message: "Endpoint Path 必须是合法相对路径"}
	}
	decoded, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return "", &ValidationError{Field: "endpointPath", Message: "Endpoint Path 编码无效"}
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == "." || segment == ".." {
			return "", &ValidationError{Field: "endpointPath", Message: "Endpoint Path 不能包含相对目录段"}
		}
	}
	cleaned := path.Clean(raw)
	if cleaned == "." || cleaned == "/" {
		return "", &ValidationError{Field: "endpointPath", Message: "Endpoint Path 不能为空"}
	}
	return cleaned, nil
}

func validHeaderFieldName(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		default:
			return false
		}
	}
	return true
}

func NormalizeAuthHeader(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		name = DefaultCustomAuthHeader
	}
	canonicalName := textproto.CanonicalMIMEHeaderKey(name)
	if !validHeaderFieldName(name) || canonicalName == "" || len(name) > 128 {
		return "", &ValidationError{Field: "authHeader", Message: "鉴权 Header 名称无效"}
	}
	if _, forbidden := forbiddenExtraHeaders[strings.ToLower(canonicalName)]; forbidden {
		return "", &ValidationError{Field: "authHeader", Message: "鉴权 Header 不能使用传输控制头"}
	}
	return canonicalName, nil
}

func NormalizeAuthPrefix(value string) (string, error) {
	if !validHeaderValue(value) || len(value) > maxCustomAuthPrefixBytes {
		return "", &ValidationError{Field: "authPrefix", Message: "鉴权前缀无效或超过 128 字节"}
	}
	return value, nil
}

func ValidateJSONPointer(value string) error {
	if value == "" || !strings.HasPrefix(value, "/") {
		return &ValidationError{Field: "responseJSONPointer", Message: "响应 JSON Pointer 必须以 / 开头"}
	}
	for index := 0; index < len(value); index++ {
		if value[index] != '~' {
			continue
		}
		if index+1 >= len(value) || (value[index+1] != '0' && value[index+1] != '1') {
			return &ValidationError{Field: "responseJSONPointer", Message: "响应 JSON Pointer 转义无效"}
		}
		index++
	}
	return nil
}

func ValidateRequestTemplateConfig(value, pointer string) error {
	if len(value) == 0 {
		return &ValidationError{Field: "requestTemplate", Message: "自定义请求模板不能为空"}
	}
	if len(value) > MaxRequestTemplateBytes {
		return &ValidationError{Field: "requestTemplate", Message: "自定义请求模板不能超过 64 KiB"}
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return &ValidationError{Field: "requestTemplate", Message: "自定义请求模板必须是合法 JSON"}
	}
	object, ok := decoded.(map[string]any)
	if !ok || object == nil {
		return &ValidationError{Field: "requestTemplate", Message: "自定义请求模板顶层必须是 JSON object"}
	}
	hasContent := false
	if err := validateRequestTemplateValue(object, &hasContent, 0); err != nil {
		return err
	}
	if !hasContent {
		return &ValidationError{Field: "requestTemplate", Message: "自定义请求模板必须包含消息、文字或图片占位符"}
	}
	return ValidateJSONPointer(strings.TrimSpace(pointer))
}

func validateRequestTemplateValue(value any, hasContent *bool, depth int) error {
	if depth > 64 {
		return &ValidationError{Field: "requestTemplate", Message: "请求模板嵌套不能超过 64 层"}
	}
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			if strings.Contains(key, "{{") || strings.Contains(key, "}}") {
				return &ValidationError{Field: "requestTemplate", Message: "自定义请求模板的字段名不能包含占位符"}
			}
			if err := validateRequestTemplateValue(child, hasContent, depth+1); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range current {
			if err := validateRequestTemplateValue(child, hasContent, depth+1); err != nil {
				return err
			}
		}
	case string:
		_, allowed := allowedRequestTemplatePlaceholders[current]
		if strings.Contains(current, "{{") || strings.Contains(current, "}}") {
			if !allowed {
				return &ValidationError{Field: "requestTemplate", Message: "自定义请求模板包含未知或嵌入式占位符"}
			}
		}
		if _, content := contentRequestTemplatePlaceholders[current]; content {
			*hasContent = true
		}
	}
	return nil
}

func normalizeExtraHeadersWithAuth(headers map[string]string, apiKey, authHeader string) (map[string]string, error) {
	if len(headers) == 0 {
		return map[string]string{}, nil
	}
	if len(headers) > maxExtraHeaderCount {
		return nil, &ValidationError{Field: "extraHeaders", Message: "额外 Headers 最多允许 32 项"}
	}
	result := make(map[string]string, len(headers))
	totalBytes := 0
	for name, value := range headers {
		trimmedName := strings.TrimSpace(name)
		canonicalName := textproto.CanonicalMIMEHeaderKey(trimmedName)
		if !validHeaderFieldName(trimmedName) || canonicalName == "" || !validHeaderValue(value) {
			return nil, &ValidationError{Field: "extraHeaders", Message: "额外 Header 名称或值无效"}
		}

		if _, exists := result[canonicalName]; exists {
			return nil, &ValidationError{Field: "extraHeaders", Message: "Header 名称不能大小写重复"}
		}
		lowerName := strings.ToLower(canonicalName)
		if _, forbidden := forbiddenExtraHeaders[lowerName]; forbidden {
			return nil, &ValidationError{Field: "extraHeaders", Message: "额外 Headers 不能覆盖传输控制头"}
		}
		if strings.TrimSpace(apiKey) != "" {
			conflictsWithDefault := lowerName == "authorization" || lowerName == "x-api-key"
			conflictsWithConfigured := authHeader != "" && strings.EqualFold(canonicalName, authHeader)
			if conflictsWithDefault || conflictsWithConfigured {
				return nil, &ValidationError{Field: "extraHeaders", Message: "API Key 与额外鉴权 Header 不能同时配置"}
			}
		}
		totalBytes += len(canonicalName) + len(value)
		if totalBytes > maxExtraHeaderBytes {
			return nil, &ValidationError{Field: "extraHeaders", Message: "额外 Headers 总大小不能超过 16 KiB"}
		}
		result[canonicalName] = value
	}
	return result, nil
}

func NormalizeExtraHeaders(headers map[string]string, apiKey string, _ string) (map[string]string, error) {
	return normalizeExtraHeadersWithAuth(headers, apiKey, "")
}

func NormalizeResponseFormat(value, protocolValue string) (string, error) {
	format := strings.TrimSpace(value)
	if format == "" {
		format = ResponseFormatText
	}
	if format != ResponseFormatText && format != ResponseFormatJSONObject {
		return "", &ValidationError{Field: "responseFormat", Message: "响应格式必须是 text 或 json_object"}
	}
	p, err := protocol.Normalize(protocolValue)
	if err != nil {
		return "", &ValidationError{Field: "apiProtocol", Message: "API 协议必须是 openai_chat、openai_responses 或 anthropic_messages"}
	}
	if format == ResponseFormatJSONObject && p != protocol.OpenAIChat {
		return "", &ValidationError{Field: "responseFormat", Message: "json_object 仅适用于 OpenAI Chat Completions"}
	}
	return format, nil
}

func normalizeConnectionFields(c Config) (Config, error) {
	mode, err := NormalizeConnectionMode(c.ConnectionMode)
	if err != nil {
		return Config{}, err
	}
	if mode == ConnectionModeCCR {
		baseURL, normalizeErr := NormalizeCCRBaseURL(c.BaseURL)
		if normalizeErr != nil {
			return Config{}, normalizeErr
		}
		c.ConnectionMode = ConnectionModeDirect
		c.BaseURL = baseURL
		c.APIProtocol = string(protocol.OpenAIChat)
	} else {
		c.ConnectionMode = ConnectionModeDirect
		p, normalizeErr := protocol.Normalize(c.APIProtocol)
		if normalizeErr != nil {
			return Config{}, &ValidationError{Field: "apiProtocol", Message: "API 协议必须是 openai_chat、openai_responses、anthropic_messages 或 custom_json"}
		}
		c.APIProtocol = string(p)
	}

	p, err := protocol.Normalize(c.APIProtocol)
	if err != nil {
		return Config{}, &ValidationError{Field: "apiProtocol", Message: "API 协议无效"}
	}
	if p == protocol.CustomJSON {
		baseURL, normalizeErr := NormalizeCustomRequestURL(c.BaseURL)
		if normalizeErr != nil {
			return Config{}, normalizeErr
		}
		bothAuthFieldsEmpty := strings.TrimSpace(c.AuthHeader) == "" && c.AuthPrefix == ""
		authHeader, normalizeErr := NormalizeAuthHeader(c.AuthHeader)
		if normalizeErr != nil {
			return Config{}, normalizeErr
		}
		authPrefix := c.AuthPrefix
		if bothAuthFieldsEmpty {
			authPrefix = DefaultCustomAuthPrefix
		}
		authPrefix, normalizeErr = NormalizeAuthPrefix(authPrefix)
		if normalizeErr != nil {
			return Config{}, normalizeErr
		}
		extraHeaders, normalizeErr := normalizeExtraHeadersWithAuth(c.ExtraHeaders, c.APIKey, authHeader)
		if normalizeErr != nil {
			return Config{}, normalizeErr
		}
		if validateErr := ValidateRequestTemplateConfig(c.RequestTemplate, c.ResponseJSONPointer); validateErr != nil {
			return Config{}, validateErr
		}
		c.BaseURL = baseURL
		c.EndpointPath = ""
		c.ExtraHeaders = extraHeaders
		c.ResponseFormat = ResponseFormatText
		c.AuthHeader = authHeader
		c.AuthPrefix = authPrefix
		c.ResponseJSONPointer = strings.TrimSpace(c.ResponseJSONPointer)
		return c, nil
	}

	baseURL, err := NormalizeBaseURL(c.BaseURL)
	if err != nil {
		return Config{}, err
	}
	if p == protocol.OpenAIChat {
		baseURL, _, err = NormalizeOpenAIChatAddress(baseURL)
		if err != nil {
			return Config{}, err
		}
	}
	endpointPath, err := NormalizeEndpointPath(c.EndpointPath, c.APIProtocol)
	if err != nil {
		return Config{}, err
	}
	extraHeaders, err := NormalizeExtraHeaders(c.ExtraHeaders, c.APIKey, c.APIProtocol)
	if err != nil {
		return Config{}, err
	}
	responseFormat, err := NormalizeResponseFormat(c.ResponseFormat, c.APIProtocol)
	if err != nil {
		return Config{}, err
	}
	c.BaseURL = baseURL
	c.EndpointPath = endpointPath
	c.ExtraHeaders = extraHeaders
	c.ResponseFormat = responseFormat
	c.RequestTemplate = ""
	c.ResponseJSONPointer = ""
	c.AuthHeader = ""
	c.AuthPrefix = ""
	return c, nil
}

func (c Config) NormalizedConnectionSettings() (Config, error) {
	normalized, err := normalizeConnectionFields(c)
	if err != nil {
		return Config{}, err
	}
	return NormalizeConnectionProfiles(normalized)
}

func validHeaderValue(value string) bool {
	for _, b := range []byte(value) {
		if (b < 32 && b != '	') || b == 127 {
			return false
		}
	}
	return true
}
