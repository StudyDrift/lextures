package highlight_annotate

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
		Prompt          *string          `json:"prompt"`
		PassageSource   *PassageSource   `json:"passageSource"`
		PassageMarkdown *string          `json:"passageMarkdown"`
		SectionAnchor   *string          `json:"sectionAnchor"`
		UnitGranularity *UnitGranularity `json:"unitGranularity"`
		Tags            []Tag            `json:"tags"`
		MinAnnotations  *int             `json:"minAnnotations"`
		MaxAnnotations  *int             `json:"maxAnnotations"`
		RequireNote     *bool            `json:"requireNote"`
		ExpectedRegions []ExpectedRegion `json:"expectedRegions"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Prompt != nil {
		cfg.Prompt = *overlay.Prompt
	}
	if overlay.PassageSource != nil {
		switch *overlay.PassageSource {
		case PassagePrecedingBlock, PassageInline, PassageSectionAnchor:
			cfg.PassageSource = *overlay.PassageSource
		}
	}
	if overlay.PassageMarkdown != nil {
		cfg.PassageMarkdown = *overlay.PassageMarkdown
	}
	if overlay.SectionAnchor != nil {
		cfg.SectionAnchor = *overlay.SectionAnchor
	}
	if overlay.UnitGranularity != nil {
		switch *overlay.UnitGranularity {
		case UnitSentence, UnitParagraph, UnitLine:
			cfg.UnitGranularity = *overlay.UnitGranularity
		}
	}
	if overlay.Tags != nil {
		cfg.Tags = overlay.Tags
	}
	if overlay.MinAnnotations != nil && *overlay.MinAnnotations >= 0 {
		cfg.MinAnnotations = *overlay.MinAnnotations
	}
	if overlay.MaxAnnotations != nil && *overlay.MaxAnnotations > 0 {
		cfg.MaxAnnotations = *overlay.MaxAnnotations
	}
	if overlay.RequireNote != nil {
		cfg.RequireNote = *overlay.RequireNote
	}
	if overlay.ExpectedRegions != nil {
		cfg.ExpectedRegions = overlay.ExpectedRegions
	}
	if cfg.MinAnnotations < 1 {
		cfg.MinAnnotations = 1
	}
	if cfg.MaxAnnotations < 1 {
		cfg.MaxAnnotations = 20
	}
	if cfg.MaxAnnotations > 50 {
		cfg.MaxAnnotations = 50
	}
	if cfg.MinAnnotations > cfg.MaxAnnotations {
		cfg.MinAnnotations = cfg.MaxAnnotations
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
	if st.Annotations == nil {
		st.Annotations = []Annotation{}
	}
	return st
}

// FindTag returns a tag by id.
func FindTag(cfg Config, id string) *Tag {
	id = strings.TrimSpace(id)
	for i := range cfg.Tags {
		if cfg.Tags[i].ID == id {
			return &cfg.Tags[i]
		}
	}
	return nil
}

// NowRFC3339 returns UTC now for timestamps.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// MergeAnnotationsByID merges client and server annotation arrays by id.
// Prefer the newer createdAt when both exist; otherwise keep both sides' unique ids.
func MergeAnnotationsByID(client, server []Annotation) []Annotation {
	byID := map[string]Annotation{}
	order := make([]string, 0, len(client)+len(server))
	seen := map[string]bool{}

	add := func(a Annotation) {
		id := strings.TrimSpace(a.ID)
		if id == "" {
			return
		}
		if existing, ok := byID[id]; ok {
			if a.CreatedAt > existing.CreatedAt {
				byID[id] = a
			} else if a.CreatedAt == existing.CreatedAt {
				// Prefer client when timestamps tie ( fresher local edit).
				byID[id] = a
			}
			return
		}
		byID[id] = a
		if !seen[id] {
			order = append(order, id)
			seen[id] = true
		}
	}
	for _, a := range server {
		add(a)
	}
	for _, a := range client {
		id := strings.TrimSpace(a.ID)
		if id == "" {
			continue
		}
		if existing, ok := byID[id]; ok {
			// Client wins on same id (edit/delete semantics: omit = delete handled by full replace).
			_ = existing
			byID[id] = a
		} else {
			byID[id] = a
			if !seen[id] {
				order = append(order, id)
				seen[id] = true
			}
		}
	}
	out := make([]Annotation, 0, len(order))
	for _, id := range order {
		if a, ok := byID[id]; ok {
			out = append(out, a)
		}
	}
	return out
}

// DeriveStatus returns the lifecycle status for the given state/config.
// empty → not_started; some progress → in_progress; min met → completed.
func DeriveStatus(cfg Config, st State, current string) string {
	n := st.ActiveAnnotationCount()
	if n == 0 && st.CompletedAt == "" {
		if current == "" || current == "not_started" {
			return "not_started"
		}
		// After reset, empty state is not_started; if somehow mid-flight, allow in_progress.
		return current
	}
	if st.MeetsMinimum(cfg) {
		return "completed"
	}
	return "in_progress"
}

// CapAnnotations truncates to maxAnnotations (keeps earliest).
func CapAnnotations(st State, max int) State {
	if max < 1 {
		max = 20
	}
	if len(st.Annotations) > max {
		st.Annotations = st.Annotations[:max]
	}
	return st
}
