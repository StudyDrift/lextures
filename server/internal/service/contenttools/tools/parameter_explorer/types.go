package parameter_explorer

// Parameter kinds.
const (
	ParamNumber  = "number"
	ParamBoolean = "boolean"
	ParamChoice  = "choice"
)

// NumberParam is a numeric slider parameter.
type NumberParam struct {
	ID          string  `json:"id"`
	Kind        string  `json:"kind"` // number
	Label       string  `json:"label"`
	Unit        string  `json:"unit,omitempty"`
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Step        float64 `json:"step"`
	Default     float64 `json:"default"`
	Description string  `json:"description,omitempty"`
}

// BooleanParam is a toggle.
type BooleanParam struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"` // boolean
	Label       string `json:"label"`
	Default     bool   `json:"default"`
	Description string `json:"description,omitempty"`
}

// ChoiceOption is one enumerated choice.
type ChoiceOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ChoiceParam is an enumerated select.
type ChoiceParam struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"` // choice
	Label       string         `json:"label"`
	Options     []ChoiceOption `json:"options"`
	Default     string         `json:"default"`
	Description string         `json:"description,omitempty"`
}

// Parameter is one declared control (decoded loosely then normalized).
type Parameter struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	Label       string         `json:"label"`
	Unit        string         `json:"unit,omitempty"`
	Min         float64        `json:"min,omitempty"`
	Max         float64        `json:"max,omitempty"`
	Step        float64        `json:"step,omitempty"`
	DefaultNum  *float64       `json:"-"`
	DefaultBool *bool          `json:"-"`
	DefaultStr  *string        `json:"-"`
	Default     any            `json:"default"`
	Options     []ChoiceOption `json:"options,omitempty"`
	Description string         `json:"description,omitempty"`
}

// SweepSpec sweeps one numeric axis for plots.
type SweepSpec struct {
	ParamID string  `json:"paramId"`
	From    float64 `json:"from"`
	To      float64 `json:"to"`
	Points  int     `json:"points"`
}

// ModelPreset is a built-in model with slot→param binds.
type ModelPreset struct {
	Kind   string            `json:"kind"` // preset
	Preset string            `json:"preset"`
	Bind   map[string]string `json:"bind"`
}

// ModelExpression is an author expression over parameters.
type ModelExpression struct {
	Kind       string    `json:"kind"` // expression
	Expression string    `json:"expression"`
	Sweep      SweepSpec `json:"sweep"`
}

// Model is either preset or expression (decoded as raw then inspected).
type Model struct {
	Kind       string            `json:"kind"`
	Preset     string            `json:"preset,omitempty"`
	Bind       map[string]string `json:"bind,omitempty"`
	Expression string            `json:"expression,omitempty"`
	Sweep      *SweepSpec        `json:"sweep,omitempty"`
}

// OutputView declares how results are shown.
type OutputView struct {
	Kind   string `json:"kind"` // plot | readout | table
	Label  string `json:"label"`
	YLabel string `json:"yLabel,omitempty"`
	XLabel string `json:"xLabel,omitempty"`
}

// NoticingPrompt is a guided noticing question.
type NoticingPrompt struct {
	ID         string         `json:"id"`
	Text       string         `json:"text"`
	Kind       string         `json:"kind"` // text | choice
	Options    []ChoiceOption `json:"options,omitempty"`
	Required   bool           `json:"required,omitempty"`
	UnlockWhen string         `json:"unlockWhen,omitempty"`
}

// Config is instructor-authored Parameter Explorer configuration.
type Config struct {
	Prompt                string          `json:"prompt"`
	Hint                  string          `json:"hint,omitempty"`
	Parameters            []Parameter     `json:"parameters"`
	Model                 Model           `json:"model"`
	Outputs               []OutputView    `json:"outputs"`
	NoticingPrompts       []NoticingPrompt `json:"noticingPrompts,omitempty"`
	RequireAllCheckpoints bool            `json:"requireAllCheckpoints,omitempty"`
}

// TraceEntry is one downsampled exploration snapshot.
type TraceEntry struct {
	At     string         `json:"at"`
	Params map[string]any `json:"params"`
}

// State is per-enrollment Parameter Explorer state.
type State struct {
	V           int               `json:"v"`
	Params      map[string]any    `json:"params"`
	Trace       []TraceEntry      `json:"trace"`
	Checkpoints map[string]string `json:"checkpoints"` // promptId → first-hit ISO
	Answers     map[string]string `json:"answers"`
	CompletedAt string            `json:"completedAt,omitempty"`
}

// MaxTraceEntries caps distinct configurations stored.
const MaxTraceEntries = 200

// MaxSweepPoints caps plotted points.
const MaxSweepPoints = 500

// DefaultConfig returns sensible authoring defaults (quadratic demo).
func DefaultConfig() Config {
	a, b, c := 1.0, 0.0, 0.0
	return Config{
		Prompt: "Explore how the coefficients change the parabola.",
		Hint:   "Try increasing a and watch the curve steepen.",
		Parameters: []Parameter{
			{ID: "a", Kind: ParamNumber, Label: "a", Min: -3, Max: 3, Step: 0.1, Default: a},
			{ID: "b", Kind: ParamNumber, Label: "b", Min: -5, Max: 5, Step: 0.1, Default: b},
			{ID: "c", Kind: ParamNumber, Label: "c", Min: -5, Max: 5, Step: 0.1, Default: c},
		},
		Model: Model{
			Kind:   "preset",
			Preset: string(PresetQuadratic),
			Bind:   map[string]string{"a": "a", "b": "b", "c": "c"},
		},
		Outputs: []OutputView{
			{Kind: "plot", Label: "Curve", XLabel: "x", YLabel: "y"},
			{Kind: "readout", Label: "Vertex / values"},
			{Kind: "table", Label: "Data table"},
		},
	}
}

// EmptyState returns a fresh state document (params filled by client from config defaults).
func EmptyState() State {
	return State{
		V:           1,
		Params:      map[string]any{},
		Trace:       []TraceEntry{},
		Checkpoints: map[string]string{},
		Answers:     map[string]string{},
	}
}
