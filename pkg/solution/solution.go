package solution

import (
	"Q-Solver/pkg/answersummary"
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/domain"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// MaxConversationRounds is the maximum number of conversation rounds to keep.
const MaxConversationRounds = 10

type Callbacks struct {
	EmitEvent func(event string, data ...interface{})
}

type Request struct {
	Config           config.Config
	Screenshots      []string
	NumberingContext answersummary.NumberingContext
}

// ConversationTurn is one retained user/assistant exchange. System messages are
// intentionally excluded because every Solve rebuilds them from the active configuration.
type ConversationTurn struct {
	User      llm.Message `json:"user"`
	Assistant llm.Message `json:"assistant"`
}

// ConversationState is a portable snapshot of the retained conversation.
type ConversationState struct {
	Turns []ConversationTurn `json:"turns"`
}

type Solver struct {
	mu sync.Mutex

	llmProvider llm.Provider
	// chatHistory contains only complete user/assistant pairs and never a system message.
	chatHistory []llm.Message
	// generation invalidates in-flight Solve commits after a clear or replacement.
	generation uint64
}

func NewSolver(provider llm.Provider) *Solver {
	return &Solver{
		llmProvider: provider,
		chatHistory: make([]llm.Message, 0),
	}
}

func (s *Solver) SetProvider(provider llm.Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.llmProvider = provider
}

func (s *Solver) ClearHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearHistoryLocked()
}

// SnapshotConversation returns a deep copy of the retained non-system turns.
func (s *Solver) SnapshotConversation() ConversationState {
	s.mu.Lock()
	defer s.mu.Unlock()

	turnCount := len(s.chatHistory) / 2
	state := ConversationState{Turns: make([]ConversationTurn, 0, turnCount)}
	for i := 0; i+1 < len(s.chatHistory); i += 2 {
		state.Turns = append(state.Turns, ConversationTurn{
			User:      cloneMessage(s.chatHistory[i]),
			Assistant: cloneMessage(s.chatHistory[i+1]),
		})
	}
	return state
}

// ReplaceConversation atomically replaces retained history after validating its
// user/assistant pair invariant. The input is deep-copied before it is retained.
func (s *Solver) ReplaceConversation(state ConversationState) error {
	if len(state.Turns) > MaxConversationRounds {
		return fmt.Errorf("conversation has %d rounds, maximum is %d", len(state.Turns), MaxConversationRounds)
	}

	history := make([]llm.Message, 0, len(state.Turns)*2)
	for i, turn := range state.Turns {
		if turn.User.Role == llm.RoleSystem || turn.Assistant.Role == llm.RoleSystem {
			return fmt.Errorf("conversation turn %d must not contain a system message", i)
		}
		if turn.User.Role != llm.RoleUser || turn.Assistant.Role != llm.RoleAssistant {
			return fmt.Errorf("conversation turn %d must be ordered user/assistant", i)
		}
		if strings.TrimSpace(turn.Assistant.Content) == "" {
			return fmt.Errorf("conversation turn %d has an empty assistant message", i)
		}
		history = append(history, cloneMessage(turn.User), cloneMessage(turn.Assistant))
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.chatHistory = history
	s.generation++
	return nil
}

func (s *Solver) Solve(ctx context.Context, req Request, cb Callbacks) bool {
	if req.Config.APIKey == "" {
		if cb.EmitEvent != nil {
			cb.EmitEvent("require-api-key")
		}
		return false
	}

	logger.Println("开始解题流程")
	systemPrompt := buildSystemPrompt(req.Config)
	logger.Printf("system 提示词长度: %d", len(systemPrompt))

	numberingContext := answersummary.FormatNumberingContext(req.NumberingContext)
	partCapacity := len(req.Screenshots)
	if numberingContext != "" {
		partCapacity++
	}
	userParts := make([]llm.ContentPart, 0, partCapacity)
	if numberingContext != "" {
		userParts = append(userParts, llm.TextPart(numberingContext))
	}
	for _, screenshot := range req.Screenshots {
		userParts = append(userParts, llm.ImagePart(screenshot))
	}
	currentUserMsg := llm.NewMultiPartMessage(llm.RoleUser, userParts)

	var (
		provider        llm.Provider
		solveGeneration uint64
		history         []llm.Message
	)
	s.mu.Lock()
	provider = s.llmProvider
	if req.Config.KeepContext {
		s.trimChatHistoryLocked()
		history = cloneMessages(s.chatHistory)
		solveGeneration = s.generation
	} else {
		// An OFF request is a context boundary. Advancing the generation prevents
		// an older KeepContext request from restoring history after this clear.
		s.clearHistoryLocked()
		solveGeneration = s.generation
	}
	s.mu.Unlock()

	messagesToSend := make([]llm.Message, 0, len(history)+2)
	messagesToSend = append(messagesToSend, llm.NewSystemMessage(systemPrompt))
	messagesToSend = append(messagesToSend, trimHistoryForSend(history)...)
	messagesToSend = append(messagesToSend, cloneMessage(currentUserMsg))

	if cb.EmitEvent != nil {
		cb.EmitEvent("solution-stream-start")
	}

	response, err := provider.GenerateContentStream(ctx, messagesToSend, func(chunk llm.StreamChunk) {
		if cb.EmitEvent == nil {
			return
		}

		switch chunk.Type {
		case llm.ChunkThinking:
			cb.EmitEvent("solution-stream-thinking", chunk.Content)
		case llm.ChunkContent:
			cb.EmitEvent("solution-stream-chunk", chunk.Content)
		}
	})

	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			logger.Println("当前任务已中断（用户产生新输入）")
			if cb.EmitEvent != nil {
				detail := llm.DiagnoseError(llm.NewProviderError(llm.ErrorCancelled, llm.StageStreamResponse, "", nil))
				cb.EmitEvent("solution-error", "context canceled", detail)
			}
			return false
		}

		logger.Println("LLM 请求失败")
		if cb.EmitEvent != nil {
			detail := llm.DiagnoseError(err)
			cb.EmitEvent("solution-error", detail.Message, detail)
		}
		return false
	}

	logger.Printf("[解题] 模型返回内容长度: %d", len(response.Content))
	logger.Printf("[解题] 模型返回思考链长度: %d", len(response.Thinking))

	if response.Content == "" && response.Thinking == "" {
		logger.Println("[解题] 警告: 模型返回内容为空")
		if cb.EmitEvent != nil {
			detail := llm.DiagnoseError(llm.NewProviderError(llm.ErrorEmptyResponse, llm.StageValidateResponse, "", nil))
			cb.EmitEvent("solution-error", "模型返回内容为空，请检查模型配置或稍后重试", detail)
		}
		return false
	}

	if cb.EmitEvent != nil {
		cb.EmitEvent("solution", response.Content)
	}

	if req.Config.KeepContext && strings.TrimSpace(response.Content) != "" {
		s.mu.Lock()
		if s.generation == solveGeneration {
			s.chatHistory = append(s.chatHistory, cloneMessage(currentUserMsg), llm.NewAssistantMessage(response.Content))
			s.trimChatHistoryLocked()
		}
		s.mu.Unlock()
	}
	return true
}

func buildSystemPrompt(cfg config.Config) string {
	var systemPrompt bytes.Buffer
	systemPrompt.WriteString(domain.GetSystemBehaviorPrompt())

	if cfg.DomainId != "" {
		if prompt := domain.GetPrompt(cfg.DomainId); prompt != "" {
			systemPrompt.WriteString("\n\n<ScenePrompt>\n")
			systemPrompt.WriteString(prompt)
			systemPrompt.WriteString("\n</ScenePrompt>")
			logger.Printf("已注入场景提示词 (DomainID: %s)", cfg.DomainId)
		}
	}

	if cfg.ResumeContent != "" {
		logger.Println("使用 Markdown 简历内容")
		systemPrompt.WriteString("\n\n<CandidateProfile>\n")
		systemPrompt.WriteString("  <ResumeSummary>\n")
		systemPrompt.WriteString(cfg.ResumeContent)
		systemPrompt.WriteString("\n  </ResumeSummary>\n")
		systemPrompt.WriteString("</CandidateProfile>\n")
	}

	return systemPrompt.String()
}

func (s *Solver) clearHistoryLocked() {
	s.chatHistory = make([]llm.Message, 0)
	s.generation++
}

// trimChatHistory keeps only the most recent complete conversation rounds.
func (s *Solver) trimChatHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trimChatHistoryLocked()
}

// trimHistoryForSend 把历史裁剪到 MaxConversationRounds-1 轮，
// 为紧随其后的当前轮预留位置，使单次请求的总轮数不超过 MaxConversationRounds。
// 仅在发送侧生效，不改变 chatHistory 的存储内容。
func trimHistoryForSend(history []llm.Message) []llm.Message {
	maxMessages := (MaxConversationRounds - 1) * 2
	if len(history) <= maxMessages {
		return history
	}

	start := len(history) - maxMessages
	// Preserve the pair boundary even if malformed state was assigned internally.
	if start%2 != 0 {
		start++
	}
	return history[start:]
}

func (s *Solver) trimChatHistoryLocked() {
	maxMessages := MaxConversationRounds * 2
	if len(s.chatHistory) <= maxMessages {
		return
	}

	oldLen := len(s.chatHistory)
	start := len(s.chatHistory) - maxMessages
	// Preserve the pair boundary even if malformed state was assigned internally.
	if start%2 != 0 {
		start++
	}
	s.chatHistory = cloneMessages(s.chatHistory[start:])
	logger.Printf("裁剪对话历史: %d -> %d 条消息 (保留最近 %d 轮对话)", oldLen, len(s.chatHistory), MaxConversationRounds)
}

func cloneMessages(messages []llm.Message) []llm.Message {
	if messages == nil {
		return nil
	}
	cloned := make([]llm.Message, len(messages))
	for i := range messages {
		cloned[i] = cloneMessage(messages[i])
	}
	return cloned
}

func cloneMessage(message llm.Message) llm.Message {
	cloned := message
	if message.Parts != nil {
		cloned.Parts = append([]llm.ContentPart(nil), message.Parts...)
	}
	return cloned
}
