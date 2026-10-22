// Package questionbank stores exam segments independently of model conversation history.
package questionbank

import "time"

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

type Record struct {
	ID            string     `json:"id"`
	CreatedAt     time.Time  `json:"createdAt"`
	Images        []string   `json:"images"`
	Model         string     `json:"model"`
	Scene         string     `json:"scene"`
	Answer        string     `json:"answer"`
	Status        string     `json:"status"`
	Questions     []Question `json:"questions,omitempty"`
	OrganizeError string     `json:"organizeError,omitempty"`
}

type Session struct {
	Version         int        `json:"version"`
	ID              string     `json:"id"`
	Company         string     `json:"company"`
	Type            string     `json:"type"`
	StartedAt       time.Time  `json:"startedAt"`
	EndedAt         *time.Time `json:"endedAt,omitempty"`
	ExportedAt      *time.Time `json:"exportedAt,omitempty"`
	ExportDirectory string     `json:"exportDirectory,omitempty"`
	DiscardedAt     *time.Time `json:"discardedAt,omitempty"`
	State           string     `json:"state"`
	Records         []Record   `json:"records"`
}

type SessionInfo struct {
	ID              string `json:"id"`
	Company         string `json:"company"`
	Type            string `json:"type"`
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

type Progress struct {
	Done   int `json:"done"`
	Total  int `json:"total"`
	Failed int `json:"failed"`
}
