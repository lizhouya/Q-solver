package answersummary

const (
	KindSingleChoice   = "single_choice"
	KindMultipleChoice = "multiple_choice"
	KindFill           = "fill"
	KindJudgment       = "judgment"
	KindShortAnswer    = "short_answer"
	KindCalculation    = "calculation"
	KindCode           = "code"
	KindOther          = "other"

	StatusReady  = "ready"
	StatusReview = "review"
)

type Item struct {
	Label  string `json:"label"`
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Status string `json:"status"`
}

type Result struct {
	Items    []Item `json:"items,omitempty"`
	CopyText string `json:"copyText,omitempty"`
}

type NumberingContext struct {
	PreviousLabels     []string `json:"previousLabels,omitempty"`
	SuggestedNextLabel string   `json:"suggestedNextLabel,omitempty"`
}
