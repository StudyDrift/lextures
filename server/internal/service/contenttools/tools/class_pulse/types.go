package class_pulse

// RevealCorrect controls when concept-poll correctness is shown to learners.
type RevealCorrect string

const (
	RevealAfterVote   RevealCorrect = "after_vote"
	RevealAfterRevote RevealCorrect = "after_revote"
	RevealNever       RevealCorrect = "never"
)

// Option is one poll choice.
type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Config is instructor-authored Class Pulse configuration.
type Config struct {
	Question        string        `json:"question"`
	Options         []Option      `json:"options"`
	CorrectOptionID string        `json:"correctOptionId,omitempty"`
	Explanation     string        `json:"explanation,omitempty"`
	AllowSecondVote bool          `json:"allowSecondVote"`
	RevealCorrect   RevealCorrect `json:"revealCorrect"`
	MinRespondents  int           `json:"minRespondents"`
	ScopeToSection  bool          `json:"scopeToSection"`
	ShowPercentages bool          `json:"showPercentages"`
}

// Vote is one committed vote for a round.
type Vote struct {
	Round    int    `json:"round"` // 1 or 2
	OptionID string `json:"optionId"`
	At       string `json:"at"`
}

// Draft is uncommitted selection persisted via PUT state.
type Draft struct {
	OptionID string `json:"optionId,omitempty"`
	Round    int    `json:"round,omitempty"`
}

// State is per-enrollment Class Pulse state.
type State struct {
	V              int     `json:"v"`
	Votes          []Vote  `json:"votes,omitempty"`
	SawAggregateAt string  `json:"sawAggregateAt,omitempty"`
	CompletedAt    string  `json:"completedAt,omitempty"`
	Draft          *Draft  `json:"draft,omitempty"`
	Correct        *bool   `json:"correct,omitempty"` // analytics when concept poll revealed/scored
}

// DefaultConfig returns manifest defaults.
func DefaultConfig() Config {
	return Config{
		AllowSecondVote: false,
		RevealCorrect:   RevealNever,
		MinRespondents:  5,
		ScopeToSection:  true,
		ShowPercentages: true,
	}
}

// EmptyState returns a fresh unvoted document.
func EmptyState() State {
	return State{V: 1}
}
