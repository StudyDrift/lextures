package sort_sequence

import "encoding/json"

// Mode is categorize (buckets) or order (sequence).
type Mode string

const (
	ModeCategorize Mode = "categorize"
	ModeOrder      Mode = "order"
)

// ScoreMode selects how scorePct is computed.
type ScoreMode string

const (
	ScorePerItem       ScoreMode = "per_item"
	ScoreAllOrNothing  ScoreMode = "all_or_nothing"
)

// Item is one sortable chip.
type Item struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	ImageURL string `json:"imageUrl,omitempty"`
	ImageAlt string `json:"imageAlt,omitempty"`
}

// Bucket is a categorize-mode drop target.
type Bucket struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Config is instructor-authored Sort & Sequence configuration.
type Config struct {
	Mode                   Mode                       `json:"mode"`
	Prompt                 string                     `json:"prompt"`
	Items                  []Item                     `json:"items"`
	Buckets                []Bucket                   `json:"buckets,omitempty"`
	CorrectBucketByItem    map[string]json.RawMessage `json:"correctBucketByItem,omitempty"` // string | string[]
	CorrectOrder           []string                   `json:"correctOrder,omitempty"`
	TieGroups              [][]string                 `json:"tieGroups,omitempty"`
	ItemFeedback           map[string]string          `json:"itemFeedback,omitempty"`
	Attempts               any                        `json:"attempts"` // number | "unlimited"
	ShowPerItemCorrectness bool                       `json:"showPerItemCorrectness"`
	LockCorrect            bool                       `json:"lockCorrect"`
	ScoreMode              ScoreMode                  `json:"scoreMode"`
	ShuffleItems           bool                       `json:"shuffleItems"`
}

// Attempt is one recorded check.
type Attempt struct {
	At             string          `json:"at"`
	CorrectItemIDs []string        `json:"correctItemIds"`
	ScorePct       float64         `json:"scorePct"`
	Placement      json.RawMessage `json:"placement,omitempty"`
}

// State is per-enrollment Sort & Sequence state.
type State struct {
	V             int             `json:"v"`
	Placement     json.RawMessage `json:"placement,omitempty"` // map or ordered ids
	Attempts      []Attempt       `json:"attempts"`
	LockedItemIDs []string        `json:"lockedItemIds"`
	TrayOrder     []string        `json:"trayOrder,omitempty"` // stable shuffled tray order
	LastPerItem   map[string]bool `json:"lastPerItem,omitempty"`
	CompletedAt   string          `json:"completedAt,omitempty"`
}

// PerItemResult is correctness (+ optional feedback) for one item after check.
type PerItemResult struct {
	Correct  bool   `json:"correct"`
	Feedback string `json:"feedback,omitempty"`
}

// GradeResult is the outcome of grading a full placement.
type GradeResult struct {
	PerItem        map[string]PerItemResult
	CorrectItemIDs []string
	ScorePct       float64
	ScoreRaw       float64
	ScoreMax       float64
}

// DefaultConfig returns manifest defaults.
func DefaultConfig() Config {
	return Config{
		Mode:                   ModeCategorize,
		Attempts:               3,
		ShowPerItemCorrectness: true,
		LockCorrect:            true,
		ScoreMode:              ScorePerItem,
		ShuffleItems:           true,
	}
}

// EmptyState returns a fresh unplaced document.
func EmptyState() State {
	return State{
		V:             1,
		Attempts:      []Attempt{},
		LockedItemIDs: []string{},
	}
}
