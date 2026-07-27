package sort_sequence

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
)

// CorrectBucketsFor returns the set of acceptable bucket ids for an item.
func CorrectBucketsFor(cfg Config, itemID string) []string {
	raw, ok := cfg.CorrectBucketByItem[itemID]
	if !ok || len(raw) == 0 {
		return nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		single = strings.TrimSpace(single)
		if single == "" {
			return nil
		}
		return []string{single}
	}
	var multi []string
	if err := json.Unmarshal(raw, &multi); err == nil {
		out := make([]string, 0, len(multi))
		for _, s := range multi {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// GradePlacement grades a learner placement against the answer key.
func GradePlacement(cfg Config, placement json.RawMessage) GradeResult {
	itemIDs := ItemIDs(cfg)
	perItem := make(map[string]PerItemResult, len(itemIDs))
	correctIDs := make([]string, 0, len(itemIDs))

	switch cfg.Mode {
	case ModeOrder:
		order, ok := ParseOrderPlacement(placement)
		if !ok {
			order = nil
		}
		posByItem := map[string]int{}
		for i, id := range order {
			posByItem[id] = i
		}
		expected := cfg.CorrectOrder
		if len(expected) == 0 {
			expected = itemIDs
		}
		tieOf := buildTieIndex(cfg.TieGroups)
		for _, id := range itemIDs {
			okItem := orderItemCorrect(id, expected, posByItem, tieOf)
			fb := ""
			if cfg.ItemFeedback != nil {
				fb = cfg.ItemFeedback[id]
			}
			perItem[id] = PerItemResult{Correct: okItem, Feedback: fb}
			if okItem {
				correctIDs = append(correctIDs, id)
			}
		}
	default: // categorize
		place, ok := ParseCategorizePlacement(placement)
		if !ok {
			place = map[string]*string{}
		}
		for _, id := range itemIDs {
			okItem := false
			if bucketPtr, has := place[id]; has && bucketPtr != nil {
				got := strings.TrimSpace(*bucketPtr)
				for _, want := range CorrectBucketsFor(cfg, id) {
					if got == want {
						okItem = true
						break
					}
				}
			}
			fb := ""
			if cfg.ItemFeedback != nil {
				fb = cfg.ItemFeedback[id]
			}
			perItem[id] = PerItemResult{Correct: okItem, Feedback: fb}
			if okItem {
				correctIDs = append(correctIDs, id)
			}
		}
	}

	sort.Strings(correctIDs)
	n := float64(len(itemIDs))
	if n == 0 {
		return GradeResult{
			PerItem:        perItem,
			CorrectItemIDs: correctIDs,
			ScorePct:       0,
			ScoreRaw:       0,
			ScoreMax:       0,
		}
	}
	raw := float64(len(correctIDs))
	max := n
	pct := (raw / max) * 100
	if cfg.ScoreMode == ScoreAllOrNothing {
		if len(correctIDs) == len(itemIDs) {
			pct = 100
			raw = max
		} else {
			pct = 0
			raw = 0
		}
	}
	pct = math.Round(pct*100) / 100
	return GradeResult{
		PerItem:        perItem,
		CorrectItemIDs: correctIDs,
		ScorePct:       pct,
		ScoreRaw:       raw,
		ScoreMax:       max,
	}
}

func buildTieIndex(groups [][]string) map[string]int {
	out := map[string]int{}
	for i, g := range groups {
		for _, id := range g {
			id = strings.TrimSpace(id)
			if id != "" {
				out[id] = i
			}
		}
	}
	return out
}

// orderItemCorrect: an item is correct when its position matches the expected
// position, or when it and the expected occupant at its position are in the
// same tie group (order-insensitive within the group).
func orderItemCorrect(itemID string, expected []string, posByItem map[string]int, tieOf map[string]int) bool {
	gotPos, has := posByItem[itemID]
	if !has {
		return false
	}
	expPos := -1
	for i, id := range expected {
		if id == itemID {
			expPos = i
			break
		}
	}
	if expPos < 0 {
		return false
	}
	if gotPos == expPos {
		return true
	}
	// Tie-group: any permutation within the group is correct if every group
	// member occupies the group's expected slots.
	tieA, okA := tieOf[itemID]
	if !okA {
		return false
	}
	if gotPos < 0 || gotPos >= len(expected) {
		return false
	}
	occupant := expected[gotPos]
	tieB, okB := tieOf[occupant]
	if !okB || tieA != tieB {
		return false
	}
	// Also require that gotPos is one of the expected slots for this tie group.
	expectedSlots := map[int]struct{}{}
	for i, id := range expected {
		if t, ok := tieOf[id]; ok && t == tieA {
			expectedSlots[i] = struct{}{}
		}
	}
	_, inSlot := expectedSlots[gotPos]
	return inSlot
}

// AllItemsPlaced reports whether every configured item has a placement.
func AllItemsPlaced(cfg Config, placement json.RawMessage) bool {
	ids := ItemIDs(cfg)
	if len(ids) == 0 {
		return false
	}
	switch cfg.Mode {
	case ModeOrder:
		order, ok := ParseOrderPlacement(placement)
		if !ok {
			return false
		}
		seen := map[string]struct{}{}
		for _, id := range order {
			seen[id] = struct{}{}
		}
		for _, id := range ids {
			if _, ok := seen[id]; !ok {
				return false
			}
		}
		return true
	default:
		place, ok := ParseCategorizePlacement(placement)
		if !ok {
			return false
		}
		for _, id := range ids {
			v, has := place[id]
			if !has || v == nil || strings.TrimSpace(*v) == "" {
				return false
			}
		}
		return true
	}
}

// ValidatePlacementIDs ensures placement only references known items/buckets.
func ValidatePlacementIDs(cfg Config, placement json.RawMessage) error {
	knownItems := map[string]struct{}{}
	for _, id := range ItemIDs(cfg) {
		knownItems[id] = struct{}{}
	}
	buckets := BucketIDs(cfg)
	switch cfg.Mode {
	case ModeOrder:
		order, ok := ParseOrderPlacement(placement)
		if !ok {
			return errInvalidPlacement
		}
		seen := map[string]struct{}{}
		for _, id := range order {
			if _, ok := knownItems[id]; !ok {
				return errUnknownItem
			}
			if _, dup := seen[id]; dup {
				return errDuplicateItem
			}
			seen[id] = struct{}{}
		}
		return nil
	default:
		place, ok := ParseCategorizePlacement(placement)
		if !ok {
			return errInvalidPlacement
		}
		for itemID, bucket := range place {
			if _, ok := knownItems[itemID]; !ok {
				return errUnknownItem
			}
			if bucket == nil {
				continue
			}
			if _, ok := buckets[*bucket]; !ok {
				return errUnknownBucket
			}
		}
		return nil
	}
}

var (
	errInvalidPlacement = &placementError{msg: "invalid placement"}
	errUnknownItem      = &placementError{msg: "unknown item id"}
	errUnknownBucket    = &placementError{msg: "unknown bucket id"}
	errDuplicateItem    = &placementError{msg: "duplicate item in order"}
)

type placementError struct{ msg string }

func (e *placementError) Error() string { return e.msg }
