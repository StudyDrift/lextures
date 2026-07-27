package inline_questions

// QuestionType is a formative question kind.
type QuestionType string

const (
	TypeSingle    QuestionType = "single"
	TypeMulti     QuestionType = "multi"
	TypeTrueFalse QuestionType = "true_false"
	TypeShortText QuestionType = "short_text"
	TypeNumeric   QuestionType = "numeric"
)

// RevealPolicy controls when the correct answer is shown.
type RevealPolicy string

const (
	RevealFirstAttempt RevealPolicy = "first_attempt"
	RevealLastAttempt  RevealPolicy = "last_attempt"
	RevealNever        RevealPolicy = "never"
)

// ScorePolicy selects which attempt contributes to the reported score.
type ScorePolicy string

const (
	ScoreLast  ScorePolicy = "last"
	ScoreBest  ScorePolicy = "best"
	ScoreFirst ScorePolicy = "first"
)

// ToleranceKind is absolute or relative numeric tolerance.
type ToleranceKind string

const (
	ToleranceAbsolute ToleranceKind = "absolute"
	ToleranceRelative ToleranceKind = "relative"
)

// Option is a choice for single / multi / true_false questions.
type Option struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Correct  bool   `json:"correct"`
	Feedback string `json:"feedback,omitempty"`
}

// Tolerance is a numeric grading window.
type Tolerance struct {
	Kind  ToleranceKind `json:"kind"`
	Value float64       `json:"value"`
}

// Question is one formative item in the check.
type Question struct {
	ID                   string       `json:"id"`
	Type                 QuestionType `json:"type"`
	Prompt               string       `json:"prompt"`
	Options              []Option     `json:"options,omitempty"`
	AcceptedAnswers      []string     `json:"acceptedAnswers,omitempty"`
	CaseSensitive        bool         `json:"caseSensitive,omitempty"`
	NormalizePunctuation bool         `json:"normalizePunctuation,omitempty"`
	CorrectValue         *float64     `json:"correctValue,omitempty"`
	Tolerance            *Tolerance   `json:"tolerance,omitempty"`
	Unit                 string       `json:"unit,omitempty"`
	Explanation          string       `json:"explanation,omitempty"`
	OutcomeID            string       `json:"outcomeId,omitempty"`
	Points               float64      `json:"points,omitempty"`
	PartialCredit        bool         `json:"partialCredit,omitempty"` // multi only; default strict
}

// Config is instructor-authored Inline Questions configuration.
type Config struct {
	Questions          []Question   `json:"questions"`
	Attempts           any          `json:"attempts"` // number | "unlimited"
	RevealCorrectAfter RevealPolicy `json:"revealCorrectAfter"`
	ShuffleOptions     bool         `json:"shuffleOptions"`
	Sequential         bool         `json:"sequential"`
	ScorePolicy        ScorePolicy  `json:"scorePolicy"`
	Label              string       `json:"label,omitempty"`
}

// Attempt is one recorded submission for a question.
type Attempt struct {
	Value   any    `json:"value"`
	Correct bool   `json:"correct"`
	At      string `json:"at"`
	Points  float64 `json:"points,omitempty"`
}

// QuestionAnswer is per-question learner progress.
type QuestionAnswer struct {
	Attempts []Attempt `json:"attempts"`
	Revealed bool      `json:"revealed"`
}

// State is per-enrollment Inline Questions state.
type State struct {
	V           int                       `json:"v"`
	Answers     map[string]QuestionAnswer `json:"answers"`
	Drafts      map[string]any            `json:"drafts,omitempty"`
	ScoreRaw    *float64                  `json:"scoreRaw,omitempty"`
	ScoreMax    *float64                  `json:"scoreMax,omitempty"`
	CompletedAt string                    `json:"completedAt,omitempty"`
}

// DefaultConfig returns manifest defaults.
func DefaultConfig() Config {
	return Config{
		Questions:          nil,
		Attempts:           2,
		RevealCorrectAfter: RevealLastAttempt,
		ShuffleOptions:     false,
		Sequential:         false,
		ScorePolicy:        ScoreLast,
	}
}

// EmptyState returns a fresh unanswered document.
func EmptyState() State {
	return State{V: 1, Answers: map[string]QuestionAnswer{}}
}

// PointsFor returns the point weight for a question (default 1).
func PointsFor(q Question) float64 {
	if q.Points > 0 {
		return q.Points
	}
	return 1
}

// MaxAttempts returns the attempt limit, or 0 for unlimited.
func MaxAttempts(cfg Config) int {
	switch v := cfg.Attempts.(type) {
	case float64:
		n := int(v)
		if n < 1 {
			return 2
		}
		if n > 5 {
			return 5
		}
		return n
	case int:
		if v < 1 {
			return 2
		}
		if v > 5 {
			return 5
		}
		return v
	case int64:
		n := int(v)
		if n < 1 {
			return 2
		}
		if n > 5 {
			return 5
		}
		return n
	case string:
		if v == "unlimited" {
			return 0
		}
	}
	return 2
}
