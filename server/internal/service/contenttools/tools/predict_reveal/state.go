package predict_reveal

import (
	"encoding/json"
	"strings"
	"time"
)

// ParseConfig unmarshals instructor config with defaults applied.
func ParseConfig(raw json.RawMessage) Config {
	cfg := DefaultConfig()
	if len(raw) == 0 {
		return cfg
	}
	var overlay struct {
		Question           *string           `json:"question"`
		Mode               *Mode             `json:"mode"`
		Outcomes           []Outcome         `json:"outcomes"`
		OpenPlaceholder    *string           `json:"openPlaceholder"`
		ConfidenceScale    *ConfidenceScale  `json:"confidenceScale"`
		ConfidenceRequired *bool             `json:"confidenceRequired"`
		Reveal             *Reveal           `json:"reveal"`
		ReflectionPrompt   *string           `json:"reflectionPrompt"`
		ShowPeerResults    *bool             `json:"showPeerResults"`
		ConfidenceLabels   map[string]string `json:"confidenceLabels"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Question != nil {
		cfg.Question = *overlay.Question
	}
	if overlay.Mode != nil {
		switch *overlay.Mode {
		case ModeChoice, ModeOpen:
			cfg.Mode = *overlay.Mode
		}
	}
	if overlay.Outcomes != nil {
		cfg.Outcomes = overlay.Outcomes
	}
	if overlay.OpenPlaceholder != nil {
		cfg.OpenPlaceholder = *overlay.OpenPlaceholder
	}
	if overlay.ConfidenceScale != nil {
		switch *overlay.ConfidenceScale {
		case ScaleNone, ScaleThree, ScaleFive, ScalePercent:
			cfg.ConfidenceScale = *overlay.ConfidenceScale
		}
	}
	if overlay.ConfidenceRequired != nil {
		cfg.ConfidenceRequired = *overlay.ConfidenceRequired
	}
	if overlay.Reveal != nil {
		cfg.Reveal = *overlay.Reveal
	}
	if overlay.ReflectionPrompt != nil {
		cfg.ReflectionPrompt = *overlay.ReflectionPrompt
	}
	if overlay.ShowPeerResults != nil {
		cfg.ShowPeerResults = *overlay.ShowPeerResults
	}
	if overlay.ConfidenceLabels != nil {
		cfg.ConfidenceLabels = overlay.ConfidenceLabels
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
	return st
}

// FindOutcome returns an outcome by id.
func FindOutcome(cfg Config, id string) *Outcome {
	id = strings.TrimSpace(id)
	for i := range cfg.Outcomes {
		if cfg.Outcomes[i].ID == id {
			return &cfg.Outcomes[i]
		}
	}
	return nil
}

// TagCorrectness records analytics-only correctness for choice mode.
func TagCorrectness(cfg Config, outcomeID string) *bool {
	if cfg.Mode != ModeChoice {
		return nil
	}
	o := FindOutcome(cfg, outcomeID)
	if o == nil {
		return nil
	}
	anyMarked := false
	for _, out := range cfg.Outcomes {
		if out.Correct {
			anyMarked = true
			break
		}
	}
	if !anyMarked {
		return nil
	}
	v := o.Correct
	return &v
}

// NowRFC3339 returns UTC now for commit/reveal timestamps.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// GuardStatePut refuses mutation of committed prediction fields via PUT.
// Reflection must go through the reflect action. Draft-only updates are allowed
// before commit; after commit PUT is refused entirely.
func GuardStatePut(current, _ json.RawMessage) (blocked bool, message string) {
	st := ParseState(current)
	if st.IsCommitted() {
		return true, "Prediction is locked after commit; use reset to start over."
	}
	return false, ""
}
