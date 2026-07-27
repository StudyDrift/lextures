package worked_example

import (
	"encoding/json"
	"hash/fnv"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/service/mathnorm"
)

// ParseConfig unmarshals instructor config with defaults applied.
func ParseConfig(raw json.RawMessage) Config {
	cfg := DefaultConfig()
	if len(raw) == 0 {
		return cfg
	}
	var overlay struct {
		Title            *string      `json:"title"`
		Problem          *string      `json:"problem"`
		Variables        []string     `json:"variables"`
		Steps            []Step       `json:"steps"`
		BlankPolicy      *BlankPolicy `json:"blankPolicy"`
		AttemptsPerStep  *int         `json:"attemptsPerStep"`
		HintsAffectScore *bool        `json:"hintsAffectScore"`
		PracticeOnly     *bool        `json:"practiceOnly"`
		ShowAllSteps     *bool        `json:"showAllSteps"`
		AllowRevealAll   *bool        `json:"allowRevealAll"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Title != nil {
		cfg.Title = *overlay.Title
	}
	if overlay.Problem != nil {
		cfg.Problem = *overlay.Problem
	}
	if overlay.Variables != nil {
		cfg.Variables = overlay.Variables
	}
	if overlay.Steps != nil {
		cfg.Steps = overlay.Steps
	}
	if overlay.BlankPolicy != nil {
		switch *overlay.BlankPolicy {
		case BlankAuthor, BlankProgressive, BlankAll:
			cfg.BlankPolicy = *overlay.BlankPolicy
		}
	}
	if overlay.AttemptsPerStep != nil {
		n := *overlay.AttemptsPerStep
		if n < 1 {
			n = 1
		}
		if n > 10 {
			n = 10
		}
		cfg.AttemptsPerStep = n
	}
	if overlay.HintsAffectScore != nil {
		cfg.HintsAffectScore = *overlay.HintsAffectScore
	}
	if overlay.PracticeOnly != nil {
		cfg.PracticeOnly = *overlay.PracticeOnly
	}
	if overlay.ShowAllSteps != nil {
		cfg.ShowAllSteps = *overlay.ShowAllSteps
	}
	if overlay.AllowRevealAll != nil {
		cfg.AllowRevealAll = *overlay.AllowRevealAll
	}
	return cfg
}

// ParseState unmarshals learner state with defaults.
func ParseState(raw json.RawMessage) State {
	st := EmptyState()
	if len(raw) == 0 {
		return st
	}
	_ = json.Unmarshal(raw, &st)
	if st.V == 0 {
		st.V = 1
	}
	if st.Steps == nil {
		st.Steps = map[string]StepProgress{}
	}
	return st
}

// FindStep returns a step by id.
func FindStep(cfg Config, id string) *Step {
	id = strings.TrimSpace(id)
	for i := range cfg.Steps {
		if cfg.Steps[i].ID == id {
			return &cfg.Steps[i]
		}
	}
	return nil
}

// StepIndex returns the 0-based index of a step, or -1.
func StepIndex(cfg Config, id string) int {
	id = strings.TrimSpace(id)
	for i := range cfg.Steps {
		if cfg.Steps[i].ID == id {
			return i
		}
	}
	return -1
}

// AttemptsUsed returns how many attempts the learner has for a step.
func AttemptsUsed(st State, stepID string) int {
	sp, ok := st.Steps[stepID]
	if !ok {
		return 0
	}
	return len(sp.Attempts)
}

// AttemptsRemaining returns remaining attempts for a step.
func AttemptsRemaining(cfg Config, st State, stepID string) int {
	left := cfg.AttemptsPerStep - AttemptsUsed(st, stepID)
	if left < 0 {
		return 0
	}
	return left
}

// StepCompleted reports whether the step is done (correct, needs_review, or revealed).
func StepCompleted(st State, stepID string) bool {
	sp, ok := st.Steps[stepID]
	if !ok {
		return false
	}
	if sp.Revealed || sp.CompletedAt != "" {
		return true
	}
	if len(sp.Attempts) == 0 {
		return false
	}
	last := sp.Attempts[len(sp.Attempts)-1]
	return last.Result == ResultCorrect || last.Result == ResultNeedsReview
}

// StepUnlocked reports whether sequential mode allows interacting with this step.
// Blanked steps must be answered/skipped before later blanked steps; non-blanked
// steps are always "complete" for gating purposes.
func StepUnlocked(cfg Config, st State, stepID string, blanked map[string]bool) bool {
	if cfg.ShowAllSteps {
		return true
	}
	for _, step := range cfg.Steps {
		if step.ID == stepID {
			return true
		}
		if !blanked[step.ID] {
			continue
		}
		if !StepCompleted(st, step.ID) {
			return false
		}
	}
	return false
}

// NextBlankedStep returns the next blanked incomplete step after stepID, or "".
func NextBlankedStep(cfg Config, st State, afterID string, blanked map[string]bool) string {
	start := StepIndex(cfg, afterID)
	for i, step := range cfg.Steps {
		if start >= 0 && i <= start {
			continue
		}
		if !blanked[step.ID] {
			continue
		}
		if !StepCompleted(st, step.ID) {
			return step.ID
		}
	}
	return ""
}

// FirstIncompleteBlanked returns the first incomplete blanked step id.
func FirstIncompleteBlanked(cfg Config, st State, blanked map[string]bool) string {
	for _, step := range cfg.Steps {
		if !blanked[step.ID] {
			continue
		}
		if !StepCompleted(st, step.ID) {
			return step.ID
		}
	}
	return ""
}

// ComputeScore returns steps correct without reveal / total blanked steps.
func ComputeScore(cfg Config, st State, blanked map[string]bool) (raw, max float64) {
	for _, step := range cfg.Steps {
		if !blanked[step.ID] {
			continue
		}
		max++
		sp := st.Steps[step.ID]
		if sp.Revealed {
			if cfg.HintsAffectScore {
				continue
			}
			// Revealed steps do not count as correct.
			continue
		}
		if len(sp.Attempts) == 0 {
			continue
		}
		last := sp.Attempts[len(sp.Attempts)-1]
		if last.Result == ResultCorrect || last.Result == ResultNeedsReview {
			if cfg.HintsAffectScore && sp.HintsUsed > 0 {
				continue
			}
			raw++
		}
	}
	return raw, max
}

// AllBlankedComplete reports whether every blanked step is completed.
func AllBlankedComplete(cfg Config, st State, blanked map[string]bool) bool {
	any := false
	for _, step := range cfg.Steps {
		if !blanked[step.ID] {
			continue
		}
		any = true
		if !StepCompleted(st, step.ID) {
			return false
		}
	}
	return any || len(cfg.Steps) == 0
}

// EnrollmentSeed derives a deterministic uint64 from enrollment id string.
func EnrollmentSeed(enrollmentID string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(enrollmentID))
	return h.Sum64()
}

// NowRFC3339 returns UTC now for timestamps.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ExpectedDisplay returns a learner-safe display of the expected answer.
func ExpectedDisplay(step Step) string {
	if step.Blank == nil {
		return ""
	}
	b := step.Blank
	switch b.Type {
	case BlankChoice:
		for _, o := range b.Options {
			if o.ID == b.CorrectOptionID {
				return o.Text
			}
		}
		return b.CorrectOptionID
	case BlankNumeric:
		if b.Expected != nil {
			return stringify(b.Expected)
		}
		if len(b.AcceptedAnswers) > 0 {
			return b.AcceptedAnswers[0]
		}
	case BlankExpression:
		raw := ""
		if b.Expected != nil {
			raw = stringify(b.Expected)
		} else if len(b.AcceptedAnswers) > 0 {
			raw = b.AcceptedAnswers[0]
		}
		if raw == "" {
			return ""
		}
		if canon, ok := mathnorm.NormalizeCanonical(raw, nil); ok {
			return canon
		}
		return raw
	case BlankText:
		if b.Expected != nil {
			return stringify(b.Expected)
		}
		if len(b.AcceptedAnswers) > 0 {
			return b.AcceptedAnswers[0]
		}
	}
	return ""
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		b, _ := json.Marshal(t)
		return string(b)
	case json.Number:
		return t.String()
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
