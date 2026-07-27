package worked_example

// BlankType is the input kind for a blanked step.
type BlankType string

const (
	BlankNumeric    BlankType = "numeric"
	BlankExpression BlankType = "expression"
	BlankChoice     BlankType = "choice"
	BlankText       BlankType = "text"
)

// BlankPolicy controls which steps are blanked for the learner.
type BlankPolicy string

const (
	BlankAuthor      BlankPolicy = "author"
	BlankProgressive BlankPolicy = "progressive"
	BlankAll         BlankPolicy = "all"
)

// ToleranceKind is absolute or relative numeric tolerance.
type ToleranceKind string

const (
	ToleranceAbsolute ToleranceKind = "absolute"
	ToleranceRelative ToleranceKind = "relative"
)

// Tolerance is a numeric grading window.
type Tolerance struct {
	Kind  ToleranceKind `json:"kind"`
	Value float64       `json:"value"`
}

// ChoiceOption is one choice for a choice blank.
type ChoiceOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Blank is the learner-facing input for a step (sensitive fields redacted).
type Blank struct {
	Type            BlankType      `json:"type"`
	Expected        any            `json:"expected,omitempty"` // x-lex-sensitive
	Tolerance       *Tolerance     `json:"tolerance,omitempty"`
	AcceptedAnswers []string       `json:"acceptedAnswers,omitempty"` // x-lex-sensitive
	Options         []ChoiceOption `json:"options,omitempty"`
	CorrectOptionID string         `json:"correctOptionId,omitempty"` // x-lex-sensitive
	Unit            string         `json:"unit,omitempty"`
}

// Step is one ordered derivation step.
type Step struct {
	ID          string   `json:"id"`
	Label       string   `json:"label,omitempty"`
	Text        string   `json:"text"`
	Blank       *Blank   `json:"blank,omitempty"`
	Hints       []string `json:"hints,omitempty"`       // x-lex-sensitive
	Explanation string   `json:"explanation,omitempty"` // x-lex-sensitive
}

// Config is instructor-authored Worked Example configuration.
type Config struct {
	Title           string      `json:"title,omitempty"`
	Problem         string      `json:"problem"`
	Variables       []string    `json:"variables,omitempty"`
	Steps           []Step      `json:"steps"`
	BlankPolicy     BlankPolicy `json:"blankPolicy"`
	AttemptsPerStep int         `json:"attemptsPerStep"`
	HintsAffectScore bool       `json:"hintsAffectScore"`
	PracticeOnly    bool        `json:"practiceOnly"`
	ShowAllSteps    bool        `json:"showAllSteps"`
	AllowRevealAll  bool        `json:"allowRevealAll"`
}

// AttemptResult is the grade outcome for one attempt.
type AttemptResult string

const (
	ResultCorrect     AttemptResult = "correct"
	ResultIncorrect   AttemptResult = "incorrect"
	ResultNeedsReview AttemptResult = "needs_review"
)

// Attempt is one recorded submission for a step.
type Attempt struct {
	Value  string        `json:"value"`
	Result AttemptResult `json:"result"`
	At     string        `json:"at"`
}

// StepProgress is per-step learner progress.
type StepProgress struct {
	Attempts    []Attempt `json:"attempts"`
	HintsUsed   int       `json:"hintsUsed"`
	Revealed    bool      `json:"revealed"`
	Draft       string    `json:"draft,omitempty"`
	CompletedAt string    `json:"completedAt,omitempty"`
	StartedAt   string    `json:"startedAt,omitempty"`
}

// State is per-enrollment Worked Example state.
type State struct {
	V              int                     `json:"v"`
	Steps          map[string]StepProgress `json:"steps"`
	BlankedStepIDs []string                `json:"blankedStepIds,omitempty"`
	CurrentStepID  string                  `json:"currentStepId,omitempty"`
	ScoreRaw       *float64                `json:"scoreRaw,omitempty"`
	ScoreMax       *float64                `json:"scoreMax,omitempty"`
	CompletedAt    string                  `json:"completedAt,omitempty"`
	RevealAllAt    string                  `json:"revealAllAt,omitempty"`
}

// DefaultConfig returns manifest defaults.
func DefaultConfig() Config {
	return Config{
		BlankPolicy:      BlankAuthor,
		AttemptsPerStep:  3,
		HintsAffectScore: false,
		PracticeOnly:     true,
		ShowAllSteps:     false,
		AllowRevealAll:   false,
	}
}

// EmptyState returns a fresh unanswered document.
func EmptyState() State {
	return State{V: 1, Steps: map[string]StepProgress{}}
}
