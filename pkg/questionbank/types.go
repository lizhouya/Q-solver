// Package questionbank stores exam segments independently of model conversation history.
package questionbank

import (
	"time"

	"Q-Solver/pkg/answersummary"
)

const (
	Active      = "active"
	Paused      = "paused"
	Ended       = "ended"
	Exported    = "exported"
	Discarded   = "discarded"
	Running     = "running"
	Completed   = "completed"
	Failed      = "failed"
	Cancelled   = "cancelled"
	Interrupted = "interrupted"
	Empty       = "empty"
)

type Question struct {
	Title           string   `json:"title"`
	Stem            string   `json:"stem"`
	Category        string   `json:"category,omitempty"`
	Type            string   `json:"type"`
	Options         []string `json:"options"`
	KnowledgePoints []string `json:"knowledgePoints"`
	KeyPoints       []string `json:"keyPoints"`
	ReferenceAnswer string   `json:"referenceAnswer,omitempty"`
	ImageIndexes    []int    `json:"imageIndexes"`
	ReviewNote      string   `json:"reviewNote"`
}

type SourceMetadataSuggestion struct {
	Company    string `json:"company,omitempty"`
	Issuer     string `json:"issuer,omitempty"`
	Type       string `json:"type,omitempty"`
	ReviewNote string `json:"reviewNote,omitempty"`
}

const (
	ConversationQualityPersisted     = "persisted"
	ConversationQualityReconstructed = "record-reconstructed"
	MaxConversationTurns             = 10
)

type ConversationTurnRef struct {
	SessionID string `json:"sessionId"`
	RecordID  string `json:"recordId"`
}

type ConversationSnapshot struct {
	Version   int                   `json:"version"`
	Quality   string                `json:"quality"`
	Turns     []ConversationTurnRef `json:"turns"`
	UpdatedAt time.Time             `json:"updatedAt"`
}

type ExportSnapshot struct {
	ExportedAt time.Time `json:"exportedAt"`
	Directory  string    `json:"directory"`
}

type Record struct {
	ID                 string                    `json:"id"`
	CreatedAt          time.Time                 `json:"createdAt"`
	Images             []string                  `json:"images"`
	Model              string                    `json:"model"`
	Scene              string                    `json:"scene"`
	Answer             string                    `json:"answer"`
	AnswerItems        []answersummary.Item      `json:"answerItems,omitempty"`
	Status             string                    `json:"status"`
	Questions          []Question                `json:"questions,omitempty"`
	OrganizeError      string                    `json:"organizeError,omitempty"`
	MetadataSuggestion *SourceMetadataSuggestion `json:"metadataSuggestion,omitempty"`
	KeepContext        *bool                     `json:"keepContext,omitempty"`
}

type Session struct {
	Version         int                   `json:"version"`
	ID              string                `json:"id"`
	Name            string                `json:"name,omitempty"`
	Company         string                `json:"company"`
	Issuer          string                `json:"issuer,omitempty"`
	Type            string                `json:"type"`
	NameCustomized  bool                  `json:"nameCustomized,omitempty"`
	StartedAt       time.Time             `json:"startedAt"`
	EndedAt         *time.Time            `json:"endedAt,omitempty"`
	ExportedAt      *time.Time            `json:"exportedAt,omitempty"`
	ExportDirectory string                `json:"exportDirectory,omitempty"`
	DiscardedAt     *time.Time            `json:"discardedAt,omitempty"`
	State           string                `json:"state"`
	Records         []Record              `json:"records"`
	Conversation    *ConversationSnapshot `json:"conversation,omitempty"`
	Exports         []ExportSnapshot      `json:"exports,omitempty"`
}

type SessionInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name,omitempty"`
	Company         string `json:"company"`
	Issuer          string `json:"issuer,omitempty"`
	Type            string `json:"type"`
	NameCustomized  bool   `json:"nameCustomized,omitempty"`
	State           string `json:"state"`
	Count           int    `json:"count"`
	StartedAt       string `json:"startedAt"`
	ExportDirectory string `json:"exportDirectory,omitempty"`
}

type ExportOptions struct {
	// GroupBy is retained for compatibility with Change 010 callers. Change 012
	// organizes Category and Type inside Markdown rather than in directories.
	GroupBy   string `json:"groupBy"`
	Name      string `json:"name"`
	Company   string `json:"company,omitempty"`
	Issuer    string `json:"issuer,omitempty"`
	Type      string `json:"type,omitempty"`
	Directory string `json:"directory,omitempty"`
	RawOnly   bool   `json:"rawOnly,omitempty"`
}

type ExportResult struct {
	Directory   string `json:"directory"`
	Markdown    string `json:"markdown"`
	RawMarkdown string `json:"rawMarkdown"`
	Records     int    `json:"records"`
	Questions   int    `json:"questions"`
}

type SealForNextResult struct {
	SessionID     string `json:"sessionId"`
	PreviousState string `json:"previousState"`
	SessionState  string `json:"sessionState"`
	Sealed        bool   `json:"sealed"`
}

type Progress struct {
	Done   int `json:"done"`
	Total  int `json:"total"`
	Failed int `json:"failed"`
}
