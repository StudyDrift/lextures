package explain_it_back

import (
	"encoding/json"
	"fmt"
	"strings"
)

// modelFeedbackWire is the strict JSON schema expected from the model.
type modelFeedbackWire struct {
	Covered    []string `json:"covered"`
	Missing    []string `json:"missing"`
	Strength   string   `json:"strength"`
	Suggestion string   `json:"suggestion"`
	Probe      string   `json:"probe"`
}

// ParseModelFeedback validates structured model output against configured key points.
// Extra fields are ignored by encoding/json; unknown key-point ids are dropped.
func ParseModelFeedback(raw string, cfg Config) (*Feedback, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, fmt.Errorf("empty model output")
	}
	// Strip optional markdown fences.
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```JSON")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(strings.TrimSpace(text), "```")
		text = strings.TrimSpace(text)
	}
	var wire modelFeedbackWire
	dec := json.NewDecoder(strings.NewReader(text))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&wire); err != nil {
		// Retry without DisallowUnknownFields for providers that add noise keys.
		if err2 := json.Unmarshal([]byte(text), &wire); err2 != nil {
			return nil, fmt.Errorf("malformed feedback json: %w", err)
		}
	}
	strength := strings.TrimSpace(wire.Strength)
	suggestion := strings.TrimSpace(wire.Suggestion)
	if strength == "" || suggestion == "" {
		return nil, fmt.Errorf("strength and suggestion are required")
	}
	allowed := map[string]struct{}{}
	for _, kp := range cfg.KeyPoints {
		allowed[kp.ID] = struct{}{}
	}
	covered, err := filterIDs(wire.Covered, allowed)
	if err != nil {
		return nil, err
	}
	missing, err := filterIDs(wire.Missing, allowed)
	if err != nil {
		return nil, err
	}
	// Ensure partition: ids in both lists prefer covered; fill missing with remainder.
	coveredSet := map[string]struct{}{}
	for _, id := range covered {
		coveredSet[id] = struct{}{}
	}
	missingOut := make([]string, 0, len(missing))
	missingSeen := map[string]struct{}{}
	for _, id := range missing {
		if _, ok := coveredSet[id]; ok {
			continue
		}
		if _, ok := missingSeen[id]; ok {
			continue
		}
		missingSeen[id] = struct{}{}
		missingOut = append(missingOut, id)
	}
	for _, kp := range cfg.KeyPoints {
		if _, ok := coveredSet[kp.ID]; ok {
			continue
		}
		if _, ok := missingSeen[kp.ID]; ok {
			continue
		}
		missingOut = append(missingOut, kp.ID)
	}
	fb := &Feedback{
		Covered:    covered,
		Missing:    missingOut,
		Strength:   strength,
		Suggestion: suggestion,
		Mode:       FeedbackModeAI,
	}
	if cfg.IncludeProbeQuestion {
		if probe := strings.TrimSpace(wire.Probe); probe != "" {
			fb.Probe = probe
		}
	}
	return fb, nil
}

func filterIDs(ids []string, allowed map[string]struct{}) ([]string, error) {
	out := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := allowed[id]; !ok {
			continue // drop unknown ids rather than failing the whole response
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

// SynthesizeDryRunFeedback builds deterministic AI-shaped feedback for local dry-run providers.
func SynthesizeDryRunFeedback(cfg Config, learnerText string) Feedback {
	words := CountWords(learnerText)
	covered := []string{}
	missing := []string{}
	lower := strings.ToLower(learnerText)
	for i, kp := range cfg.KeyPoints {
		label := strings.ToLower(kp.Label)
		// Heuristic: mark covered when the label token appears or every other point for variety.
		if label != "" && strings.Contains(lower, label) {
			covered = append(covered, kp.ID)
			continue
		}
		if i%2 == 0 && words >= cfg.MinWords {
			covered = append(covered, kp.ID)
		} else {
			missing = append(missing, kp.ID)
		}
	}
	fb := Feedback{
		Covered:    covered,
		Missing:    missing,
		Strength:   "Dry-run: you restated the idea in your own words.",
		Suggestion: "Dry-run: add one concrete detail from the activity without copying it.",
		Mode:       FeedbackModeAI,
	}
	if cfg.IncludeProbeQuestion {
		fb.Probe = "What would change if one assumption in your explanation were false?"
	}
	return fb
}

// IsDryRunText reports whether provider output looks like the synthetic dry-run provider.
func IsDryRunText(text string) bool {
	t := strings.TrimSpace(text)
	return strings.HasPrefix(t, "Dry-run") || strings.HasPrefix(t, "Dry-run tool")
}
