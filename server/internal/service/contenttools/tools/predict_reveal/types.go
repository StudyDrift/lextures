package predict_reveal

// Mode is how the learner records a prediction.
type Mode string

const (
	ModeChoice Mode = "choice"
	ModeOpen   Mode = "open"
)

// ConfidenceScale selects how confidence is collected.
type ConfidenceScale string

const (
	ScaleNone    ConfidenceScale = "none"
	ScaleThree   ConfidenceScale = "three"
	ScaleFive    ConfidenceScale = "five"
	ScalePercent ConfidenceScale = "percent"
)

// Outcome is one choice-mode prediction option.
type Outcome struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Correct bool   `json:"correct,omitempty"`
}

// Reveal is instructor-authored answer content (x-lex-sensitive until commit).
type Reveal struct {
	Markdown string `json:"markdown"`
	ImageURL string `json:"imageUrl,omitempty"`
}

// Config is instructor-authored Predict & Reveal configuration.
type Config struct {
	Question           string            `json:"question"`
	Mode               Mode              `json:"mode"`
	Outcomes           []Outcome         `json:"outcomes,omitempty"`
	OpenPlaceholder    string            `json:"openPlaceholder,omitempty"`
	ConfidenceScale    ConfidenceScale   `json:"confidenceScale"`
	ConfidenceRequired bool              `json:"confidenceRequired"`
	Reveal             Reveal            `json:"reveal"`
	ReflectionPrompt   string            `json:"reflectionPrompt,omitempty"`
	ShowPeerResults    bool              `json:"showPeerResults"`
	ConfidenceLabels   map[string]string `json:"confidenceLabels,omitempty"`
}

// Prediction is the learner's committed (or draft) guess.
type Prediction struct {
	OutcomeID string `json:"outcomeId,omitempty"`
	Text      string `json:"text,omitempty"`
}

// Draft is uncommitted local draft fields persisted via PUT state.
type Draft struct {
	OutcomeID  string   `json:"outcomeId,omitempty"`
	Text       string   `json:"text,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"` // raw scale value before commit
}

// State is per-enrollment Predict & Reveal state.
type State struct {
	V                int         `json:"v"`
	Prediction       *Prediction `json:"prediction,omitempty"`
	Confidence       *float64    `json:"confidence,omitempty"` // normalized 0..1
	ConfidenceBucket string      `json:"confidenceBucket,omitempty"`
	CommittedAt      string      `json:"committedAt,omitempty"`
	RevealedAt       string      `json:"revealedAt,omitempty"`
	Correct          *bool       `json:"correct,omitempty"`
	Reflection       string      `json:"reflection,omitempty"`
	Draft            *Draft      `json:"draft,omitempty"`
}

// DefaultConfig returns manifest defaults.
func DefaultConfig() Config {
	return Config{
		Mode:               ModeChoice,
		ConfidenceScale:    ScaleThree,
		ConfidenceRequired: true,
		ShowPeerResults:    false,
	}
}

// EmptyState returns a fresh uncommitted document.
func EmptyState() State {
	return State{V: 1}
}

// IsCommitted reports whether the learner has committed a prediction.
func (s State) IsCommitted() bool {
	return s.CommittedAt != ""
}
