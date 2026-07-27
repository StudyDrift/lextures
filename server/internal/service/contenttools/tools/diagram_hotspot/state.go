package diagram_hotspot

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
		Mode                   *Mode              `json:"mode"`
		Prompt                 *string            `json:"prompt"`
		Image                  *ImageRef          `json:"image"`
		Regions                []Region           `json:"regions"`
		Labels                 []LabelChip        `json:"labels"`
		CorrectRegionByLabel   map[string]string  `json:"correctRegionByLabel"`
		Prompts                []Prompt           `json:"prompts"`
		CorrectRegionByPrompt  map[string]string  `json:"correctRegionByPrompt"`
		FeedbackByRegion       map[string]string  `json:"feedbackByRegion"`
		Attempts               any                `json:"attempts"`
		LockCorrect            *bool              `json:"lockCorrect"`
		ShowPerItemCorrectness *bool              `json:"showPerItemCorrectness"`
		ShowRegionOutlines     *OutlineVisibility `json:"showRegionOutlines"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Mode != nil {
		switch *overlay.Mode {
		case ModeLabel, ModeHotspot:
			cfg.Mode = *overlay.Mode
		}
	}
	if overlay.Prompt != nil {
		cfg.Prompt = *overlay.Prompt
	}
	if overlay.Image != nil {
		cfg.Image = *overlay.Image
	}
	if overlay.Regions != nil {
		cfg.Regions = overlay.Regions
		if len(cfg.Regions) > 40 {
			cfg.Regions = cfg.Regions[:40]
		}
	}
	if overlay.Labels != nil {
		cfg.Labels = overlay.Labels
		if len(cfg.Labels) > 40 {
			cfg.Labels = cfg.Labels[:40]
		}
	}
	if overlay.CorrectRegionByLabel != nil {
		cfg.CorrectRegionByLabel = overlay.CorrectRegionByLabel
	}
	if overlay.Prompts != nil {
		cfg.Prompts = overlay.Prompts
		if len(cfg.Prompts) > 20 {
			cfg.Prompts = cfg.Prompts[:20]
		}
	}
	if overlay.CorrectRegionByPrompt != nil {
		cfg.CorrectRegionByPrompt = overlay.CorrectRegionByPrompt
	}
	if overlay.FeedbackByRegion != nil {
		cfg.FeedbackByRegion = overlay.FeedbackByRegion
	}
	if overlay.Attempts != nil {
		cfg.Attempts = overlay.Attempts
	}
	if overlay.LockCorrect != nil {
		cfg.LockCorrect = *overlay.LockCorrect
	}
	if overlay.ShowPerItemCorrectness != nil {
		cfg.ShowPerItemCorrectness = *overlay.ShowPerItemCorrectness
	}
	if overlay.ShowRegionOutlines != nil {
		switch *overlay.ShowRegionOutlines {
		case OutlineAlways, OutlineOnFocus, OutlineAfterCheck:
			cfg.ShowRegionOutlines = *overlay.ShowRegionOutlines
		}
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
	if st.Attempts == nil {
		st.Attempts = []Attempt{}
	}
	if st.LockedIDs == nil {
		st.LockedIDs = []string{}
	}
	if st.Assignments == nil {
		st.Assignments = map[string]*string{}
	}
	return st
}

// MaxAttempts returns the attempt limit, or 0 for unlimited.
func MaxAttempts(cfg Config) int {
	switch v := cfg.Attempts.(type) {
	case float64:
		n := int(v)
		if n < 1 {
			return 3
		}
		if n > 5 {
			return 5
		}
		return n
	case int:
		if v < 1 {
			return 3
		}
		if v > 5 {
			return 5
		}
		return v
	case json.Number:
		n, err := v.Int64()
		if err != nil || n < 1 {
			return 3
		}
		if n > 5 {
			return 5
		}
		return int(n)
	case string:
		if strings.EqualFold(strings.TrimSpace(v), "unlimited") {
			return 0
		}
		return 3
	default:
		return 3
	}
}

// AttemptsRemaining returns remaining checks, or -1 for unlimited.
func AttemptsRemaining(cfg Config, st State) int {
	max := MaxAttempts(cfg)
	if max == 0 {
		return -1
	}
	left := max - len(st.Attempts)
	if left < 0 {
		return 0
	}
	return left
}

// IsLocked reports whether an item id is locked.
func IsLocked(st State, id string) bool {
	for _, locked := range st.LockedIDs {
		if locked == id {
			return true
		}
	}
	return false
}

// ItemIDs returns label ids (label mode) or prompt ids (hotspot mode).
func ItemIDs(cfg Config) []string {
	switch cfg.Mode {
	case ModeHotspot:
		out := make([]string, 0, len(cfg.Prompts))
		for _, p := range cfg.Prompts {
			if strings.TrimSpace(p.ID) != "" {
				out = append(out, p.ID)
			}
		}
		return out
	default:
		out := make([]string, 0, len(cfg.Labels))
		for _, l := range cfg.Labels {
			if strings.TrimSpace(l.ID) != "" {
				out = append(out, l.ID)
			}
		}
		return out
	}
}

// RegionIDs returns configured region ids.
func RegionIDs(cfg Config) map[string]struct{} {
	out := map[string]struct{}{}
	for _, r := range cfg.Regions {
		if strings.TrimSpace(r.ID) != "" {
			out[r.ID] = struct{}{}
		}
	}
	return out
}

// RegionByID looks up a region.
func RegionByID(cfg Config, id string) (Region, bool) {
	for _, r := range cfg.Regions {
		if r.ID == id {
			return r, true
		}
	}
	return Region{}, false
}

// NowRFC3339 returns the current UTC time as RFC3339.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ParseAssignments decodes itemId→regionId (null = unassigned).
func ParseAssignments(raw json.RawMessage) (map[string]*string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]*string{}, true
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, false
	}
	out := make(map[string]*string, len(obj))
	for k, v := range obj {
		if v == nil {
			out[k] = nil
			continue
		}
		s, ok := v.(string)
		if !ok {
			return nil, false
		}
		cp := s
		out[k] = &cp
	}
	return out, true
}

// AssignmentsFromState returns a non-nil copy of state assignments.
func AssignmentsFromState(st State) map[string]*string {
	out := make(map[string]*string, len(st.Assignments))
	for k, v := range st.Assignments {
		if v == nil {
			out[k] = nil
			continue
		}
		cp := *v
		out[k] = &cp
	}
	return out
}

// FlatAssignments converts pointer map to plain string map (skips nulls).
func FlatAssignments(m map[string]*string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		if v != nil && strings.TrimSpace(*v) != "" {
			out[k] = *v
		}
	}
	return out
}

// ResetUnlocked clears unlocked assignments (returns labels/prompts to tray).
func ResetUnlocked(cfg Config, st State) State {
	locked := map[string]struct{}{}
	for _, id := range st.LockedIDs {
		locked[id] = struct{}{}
	}
	next := map[string]*string{}
	for _, id := range ItemIDs(cfg) {
		if _, ok := locked[id]; ok {
			if v, has := st.Assignments[id]; has {
				next[id] = v
			}
		} else {
			next[id] = nil
		}
	}
	st.Assignments = next
	st.LastPerItem = nil
	return st
}

// HeatCellsForAssignments maps each assigned region to a coarse grid cell.
func HeatCellsForAssignments(cfg Config, assignments map[string]*string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, regionID := range FlatAssignments(assignments) {
		r, ok := RegionByID(cfg, regionID)
		if !ok {
			continue
		}
		cx, cy := Centroid(r.Shape)
		cell := HeatCellForPoint(cx, cy)
		if _, dup := seen[cell]; dup {
			continue
		}
		seen[cell] = struct{}{}
		out = append(out, cell)
	}
	return out
}
