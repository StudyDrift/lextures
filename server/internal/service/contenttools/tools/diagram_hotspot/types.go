package diagram_hotspot

// Mode is label placement or hotspot identification.
type Mode string

const (
	ModeLabel   Mode = "label"
	ModeHotspot Mode = "hotspot"
)

// OutlineVisibility controls when region outlines are shown.
type OutlineVisibility string

const (
	OutlineAlways     OutlineVisibility = "always"
	OutlineOnFocus    OutlineVisibility = "on_focus"
	OutlineAfterCheck OutlineVisibility = "after_check"
)

// ImageRef is the diagram image with required alt text.
type ImageRef struct {
	URL           string `json:"url"`
	Alt           string `json:"alt"`
	NaturalWidth  int    `json:"naturalWidth"`
	NaturalHeight int    `json:"naturalHeight"`
}

// Shape is a normalized (0–1) region geometry.
type Shape struct {
	Kind   string      `json:"kind"` // rect | circle | polygon
	X      float64     `json:"x,omitempty"`
	Y      float64     `json:"y,omitempty"`
	W      float64     `json:"w,omitempty"`
	H      float64     `json:"h,omitempty"`
	CX     float64     `json:"cx,omitempty"`
	CY     float64     `json:"cy,omitempty"`
	R      float64     `json:"r,omitempty"`
	Points [][]float64 `json:"points,omitempty"`
}

// Region is one labelled interactive area on the image.
type Region struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Shape       Shape  `json:"shape"`
}

// LabelChip is a placeable label in label mode.
type LabelChip struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Prompt is one hotspot identification question.
type Prompt struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Config is instructor-authored Diagram & Hotspot configuration.
type Config struct {
	Mode                 Mode                   `json:"mode"`
	Prompt               string                 `json:"prompt"`
	Image                ImageRef               `json:"image"`
	Regions              []Region               `json:"regions"`
	Labels               []LabelChip            `json:"labels,omitempty"`
	CorrectRegionByLabel map[string]string      `json:"correctRegionByLabel,omitempty"` // x-lex-sensitive
	Prompts              []Prompt               `json:"prompts,omitempty"`
	CorrectRegionByPrompt map[string]string     `json:"correctRegionByPrompt,omitempty"` // x-lex-sensitive
	FeedbackByRegion     map[string]string      `json:"feedbackByRegion,omitempty"` // x-lex-sensitive
	Attempts             any                    `json:"attempts"` // number | "unlimited"
	LockCorrect          bool                   `json:"lockCorrect"`
	ShowPerItemCorrectness bool                 `json:"showPerItemCorrectness"`
	ShowRegionOutlines   OutlineVisibility      `json:"showRegionOutlines"`
}

// Attempt is one recorded check.
type Attempt struct {
	At          string             `json:"at"`
	CorrectIDs  []string           `json:"correctIds"`
	ScorePct    float64            `json:"scorePct"`
	Assignments map[string]string  `json:"assignments,omitempty"`
	HeatCells   []string           `json:"heatCells,omitempty"` // coarse grid cells for CT.7
}

// State is per-enrollment Diagram & Hotspot state.
type State struct {
	V            int                `json:"v"`
	Assignments  map[string]*string `json:"assignments"` // labelId|promptId → regionId | null
	Attempts     []Attempt          `json:"attempts"`
	LockedIDs    []string           `json:"lockedIds"`
	LastPerItem  map[string]bool    `json:"lastPerItem,omitempty"`
	UsedListMode bool               `json:"usedListMode,omitempty"`
	CompletedAt  string             `json:"completedAt,omitempty"`
}

// PerItemResult is correctness (+ optional feedback) for one label/prompt after check.
type PerItemResult struct {
	Correct  bool   `json:"correct"`
	Feedback string `json:"feedback,omitempty"`
}

// GradeResult is the outcome of grading assignments.
type GradeResult struct {
	PerItem    map[string]PerItemResult
	CorrectIDs []string
	ScorePct   float64
	ScoreRaw   float64
	ScoreMax   float64
}

// DefaultConfig returns manifest defaults.
func DefaultConfig() Config {
	return Config{
		Mode:                   ModeLabel,
		Attempts:               3,
		LockCorrect:            true,
		ShowPerItemCorrectness: true,
		ShowRegionOutlines:     OutlineOnFocus,
	}
}

// EmptyState returns a fresh unassigned document.
func EmptyState() State {
	return State{
		V:           1,
		Assignments: map[string]*string{},
		Attempts:    []Attempt{},
		LockedIDs:   []string{},
	}
}
