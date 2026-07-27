package code_sandbox

// EditorMode selects the learner editor surface.
type EditorMode string

const (
	EditorRich       EditorMode = "rich"
	EditorPlain      EditorMode = "plain"
	EditorUserChoice EditorMode = "user_choice"
)

// ScoringMode controls whether Check writes a gradebook score.
type ScoringMode string

const (
	ScoringAuto ScoringMode = "auto"
	ScoringNone ScoringMode = "none"
)

// RunAction is run vs check.
type RunAction string

const (
	ActionRun   RunAction = "run"
	ActionCheck RunAction = "check"
)

// RunStatus is the stored outcome of a run/check.
type RunStatus string

const (
	StatusOK           RunStatus = "ok"
	StatusCompileError RunStatus = "compile_error"
	StatusRuntimeError RunStatus = "runtime_error"
	StatusTimeout      RunStatus = "timeout"
	StatusMemory       RunStatus = "memory"
	StatusError        RunStatus = "error"
)

// TestCase is one instructor-authored I/O pair.
type TestCase struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expectedOutput"`
	Hidden         bool   `json:"hidden"`
	Feedback       string `json:"feedback,omitempty"`
}

// ErrorHint maps a stderr substring to a plain-language hint.
type ErrorHint struct {
	Match string `json:"match"`
	Hint  string `json:"hint"`
}

// Config is instructor-authored Code Sandbox configuration.
type Config struct {
	Language           string      `json:"language"`
	Prompt             string      `json:"prompt"`
	StarterCode        string      `json:"starterCode"`
	PrefixCode         string      `json:"prefixCode,omitempty"`
	SuffixCode         string      `json:"suffixCode,omitempty"`
	SampleInput        string      `json:"sampleInput,omitempty"`
	Tests              []TestCase  `json:"tests,omitempty"`
	RunLimitPerHour    int         `json:"runLimitPerHour"`
	CheckLimitPerHour  int         `json:"checkLimitPerHour"`
	EditorMode         EditorMode  `json:"editorMode"`
	ScoringMode        ScoringMode `json:"scoringMode"`
	ErrorHints         []ErrorHint `json:"errorHints,omitempty"`
	OutputLimitBytes   int         `json:"outputLimitBytes,omitempty"`
	MaxRunHistory      int         `json:"maxRunHistory,omitempty"`
}

// TestOutcome is one test result stored in history (no secrets).
type TestOutcome struct {
	ID     string `json:"id"`
	Passed bool   `json:"passed"`
}

// RunRecord is one capped history entry.
type RunRecord struct {
	At     string        `json:"at"`
	Action RunAction     `json:"action"`
	Status RunStatus     `json:"status"`
	Stdout string        `json:"stdout,omitempty"`
	Stderr string        `json:"stderr,omitempty"`
	Tests  []TestOutcome `json:"tests,omitempty"`
}

// BestResult tracks the best check score.
type BestResult struct {
	Passed int    `json:"passed"`
	Total  int    `json:"total"`
	At     string `json:"at"`
}

// RateWindow tracks per-hour run/check counts beyond truncated history.
type RateWindow struct {
	HourKey string `json:"hourKey"`
	Runs    int    `json:"runs"`
	Checks  int    `json:"checks"`
}

// State is per-enrollment Code Sandbox state.
type State struct {
	V           int         `json:"v"`
	Code        string      `json:"code"`
	Runs        []RunRecord `json:"runs"`
	Best        *BestResult `json:"best,omitempty"`
	CompletedAt string      `json:"completedAt,omitempty"`
	Rate        *RateWindow `json:"rate,omitempty"`
	EditorMode  string      `json:"editorMode,omitempty"` // learner override when user_choice
}

// CheckTestResult is returned to the learner for one test (secrets redacted).
type CheckTestResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Feedback string `json:"feedback,omitempty"`
	Hidden   bool   `json:"hidden"`
}

// DefaultConfig returns manifest defaults.
func DefaultConfig() Config {
	return Config{
		Language:          "python",
		RunLimitPerHour:   30,
		CheckLimitPerHour: 20,
		EditorMode:        EditorUserChoice,
		ScoringMode:       ScoringAuto,
		OutputLimitBytes:  DefaultOutputLimitBytes,
		MaxRunHistory:     DefaultMaxRunHistory,
	}
}

// EmptyState returns a fresh document (code filled from starter by handlers).
func EmptyState() State {
	return State{
		V:    1,
		Runs: []RunRecord{},
	}
}

const (
	// DefaultOutputLimitBytes is the per-stream truncation cap (FR-7).
	DefaultOutputLimitBytes = 8 * 1024
	// DefaultMaxRunHistory is the capped append-only run log length (FR-6).
	DefaultMaxRunHistory = 10
	// MaxTests is the authoring ceiling (FR-1).
	MaxTests = 10
	// TruncationMarker is appended when output is cut (FR-7 / AC-9).
	TruncationMarker = "\n… [output truncated]"
)
