package sort_sequence

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
		Mode                   *Mode                      `json:"mode"`
		Prompt                 *string                    `json:"prompt"`
		Items                  []Item                     `json:"items"`
		Buckets                []Bucket                   `json:"buckets"`
		CorrectBucketByItem    map[string]json.RawMessage `json:"correctBucketByItem"`
		CorrectOrder           []string                   `json:"correctOrder"`
		TieGroups              [][]string                 `json:"tieGroups"`
		ItemFeedback           map[string]string          `json:"itemFeedback"`
		Attempts               any                        `json:"attempts"`
		ShowPerItemCorrectness *bool                      `json:"showPerItemCorrectness"`
		LockCorrect            *bool                      `json:"lockCorrect"`
		ScoreMode              *ScoreMode                 `json:"scoreMode"`
		ShuffleItems           *bool                      `json:"shuffleItems"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Mode != nil {
		switch *overlay.Mode {
		case ModeCategorize, ModeOrder:
			cfg.Mode = *overlay.Mode
		}
	}
	if overlay.Prompt != nil {
		cfg.Prompt = *overlay.Prompt
	}
	if overlay.Items != nil {
		cfg.Items = overlay.Items
		if len(cfg.Items) > 30 {
			cfg.Items = cfg.Items[:30]
		}
	}
	if overlay.Buckets != nil {
		cfg.Buckets = overlay.Buckets
		if len(cfg.Buckets) > 6 {
			cfg.Buckets = cfg.Buckets[:6]
		}
	}
	if overlay.CorrectBucketByItem != nil {
		cfg.CorrectBucketByItem = overlay.CorrectBucketByItem
	}
	if overlay.CorrectOrder != nil {
		cfg.CorrectOrder = overlay.CorrectOrder
	}
	if overlay.TieGroups != nil {
		cfg.TieGroups = overlay.TieGroups
	}
	if overlay.ItemFeedback != nil {
		cfg.ItemFeedback = overlay.ItemFeedback
	}
	if overlay.Attempts != nil {
		cfg.Attempts = overlay.Attempts
	}
	if overlay.ShowPerItemCorrectness != nil {
		cfg.ShowPerItemCorrectness = *overlay.ShowPerItemCorrectness
	}
	if overlay.LockCorrect != nil {
		cfg.LockCorrect = *overlay.LockCorrect
	}
	if overlay.ScoreMode != nil {
		switch *overlay.ScoreMode {
		case ScorePerItem, ScoreAllOrNothing:
			cfg.ScoreMode = *overlay.ScoreMode
		}
	}
	if overlay.ShuffleItems != nil {
		cfg.ShuffleItems = *overlay.ShuffleItems
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
	if st.LockedItemIDs == nil {
		st.LockedItemIDs = []string{}
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

// IsLocked reports whether an item is locked after a correct placement.
func IsLocked(st State, itemID string) bool {
	for _, id := range st.LockedItemIDs {
		if id == itemID {
			return true
		}
	}
	return false
}

// ItemIDs returns configured item ids.
func ItemIDs(cfg Config) []string {
	out := make([]string, 0, len(cfg.Items))
	for _, it := range cfg.Items {
		if strings.TrimSpace(it.ID) != "" {
			out = append(out, it.ID)
		}
	}
	return out
}

// BucketIDs returns configured bucket ids.
func BucketIDs(cfg Config) map[string]struct{} {
	out := map[string]struct{}{}
	for _, b := range cfg.Buckets {
		if strings.TrimSpace(b.ID) != "" {
			out[b.ID] = struct{}{}
		}
	}
	return out
}

// NowRFC3339 returns the current UTC time as RFC3339.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ParseCategorizePlacement decodes itemId→bucketId (null = tray).
func ParseCategorizePlacement(raw json.RawMessage) (map[string]*string, bool) {
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

// ParseOrderPlacement decodes ordered item ids.
func ParseOrderPlacement(raw json.RawMessage) ([]string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return []string{}, true
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, false
	}
	return arr, true
}

// MarshalCategorizePlacement encodes a categorize placement map.
func MarshalCategorizePlacement(m map[string]*string) json.RawMessage {
	raw, err := json.Marshal(m)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

// MarshalOrderPlacement encodes an order placement list.
func MarshalOrderPlacement(order []string) json.RawMessage {
	raw, err := json.Marshal(order)
	if err != nil {
		return json.RawMessage(`[]`)
	}
	return raw
}

// ResetUnlockedToTray clears unlocked categorize placements (null) or removes
// unlocked items from an order list (they return to the conceptual tray).
func ResetUnlockedToTray(cfg Config, st State) State {
	locked := map[string]struct{}{}
	for _, id := range st.LockedItemIDs {
		locked[id] = struct{}{}
	}
	switch cfg.Mode {
	case ModeOrder:
		order, ok := ParseOrderPlacement(st.Placement)
		if !ok {
			st.Placement = MarshalOrderPlacement(nil)
			return st
		}
		kept := make([]string, 0, len(order))
		for _, id := range order {
			if _, ok := locked[id]; ok {
				kept = append(kept, id)
			}
		}
		st.Placement = MarshalOrderPlacement(kept)
	default:
		place, ok := ParseCategorizePlacement(st.Placement)
		if !ok {
			st.Placement = MarshalCategorizePlacement(map[string]*string{})
			return st
		}
		next := map[string]*string{}
		for _, id := range ItemIDs(cfg) {
			if _, ok := locked[id]; ok {
				if v, has := place[id]; has {
					next[id] = v
				}
			} else {
				next[id] = nil
			}
		}
		st.Placement = MarshalCategorizePlacement(next)
	}
	st.LastPerItem = nil
	return st
}
