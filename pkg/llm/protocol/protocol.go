package protocol

import (
	"fmt"
	"strings"
)

type APIProtocol string

const (
	OpenAIChat        APIProtocol = "openai_chat"
	OpenAIResponses   APIProtocol = "openai_responses"
	AnthropicMessages APIProtocol = "anthropic_messages"
)

func Normalize(value string) (APIProtocol, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return OpenAIChat, nil
	}

	protocol := APIProtocol(value)
	switch protocol {
	case OpenAIChat, OpenAIResponses, AnthropicMessages:
		return protocol, nil
	default:
		return "", fmt.Errorf("不支持的 API 协议")
	}
}
