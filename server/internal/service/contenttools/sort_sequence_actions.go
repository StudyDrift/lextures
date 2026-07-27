package contenttools

import (
	"encoding/json"
	"fmt"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/sort_sequence"
)

func init() {
	RegisterActionHandler(sort_sequence.ID, "check", handleSortSequenceCheck)
	RegisterActionHandler(sort_sequence.ID, "reset_attempt", handleSortSequenceResetAttempt)
}

func handleSortSequenceCheck(ctx ActionContext) (*ActionResult, error) {
	cfg := sort_sequence.ParseConfig(ctx.ConfigJSON)
	st := sort_sequence.ParseState(ctx.StateJSON)

	var in struct {
		Placement json.RawMessage `json:"placement"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid check input: %w", err)
		}
	}
	placement := in.Placement
	if len(placement) == 0 {
		placement = st.Placement
	}
	if len(placement) == 0 {
		return nil, fmt.Errorf("placement is required")
	}

	remaining := sort_sequence.AttemptsRemaining(cfg, st)
	if remaining == 0 {
		ObserveSortSequenceCheck(string(cfg.Mode), "max_attempts")
		return &ActionResult{
			Result: map[string]any{
				"error":             "max_attempts",
				"message":           "No attempts remaining.",
				"attemptsRemaining": 0,
			},
		}, nil
	}

	if err := sort_sequence.ValidatePlacementIDs(cfg, placement); err != nil {
		ObserveSortSequenceCheck(string(cfg.Mode), "invalid_placement")
		return &ActionResult{
			Result: map[string]any{
				"error":   "invalid_placement",
				"message": err.Error(),
			},
		}, nil
	}

	if !sort_sequence.AllItemsPlaced(cfg, placement) {
		ObserveSortSequenceCheck(string(cfg.Mode), "incomplete")
		return &ActionResult{
			Result: map[string]any{
				"error":   "incomplete",
				"message": "Place every item before checking.",
			},
		}, nil
	}

	// Preserve locked items' previous correct placements when lockCorrect is on.
	if cfg.LockCorrect && len(st.LockedItemIDs) > 0 {
		placement = mergeLockedPlacement(cfg, st, placement)
	}

	grade := sort_sequence.GradePlacement(cfg, placement)
	attempt := sort_sequence.Attempt{
		At:             sort_sequence.NowRFC3339(),
		CorrectItemIDs: grade.CorrectItemIDs,
		ScorePct:       grade.ScorePct,
		Placement:      placement,
	}
	st.Attempts = append(st.Attempts, attempt)
	st.Placement = placement
	st.LastPerItem = map[string]bool{}
	for id, r := range grade.PerItem {
		st.LastPerItem[id] = r.Correct
	}

	if cfg.LockCorrect {
		locked := map[string]struct{}{}
		for _, id := range sort_sequence.ItemIDs(cfg) {
			if sort_sequence.IsLocked(st, id) {
				locked[id] = struct{}{}
			}
		}
		for _, id := range grade.CorrectItemIDs {
			locked[id] = struct{}{}
		}
		st.LockedItemIDs = make([]string, 0, len(locked))
		for id := range locked {
			st.LockedItemIDs = append(st.LockedItemIDs, id)
		}
	}

	status := StatusSubmitted
	left := sort_sequence.AttemptsRemaining(cfg, st)
	allCorrect := len(grade.CorrectItemIDs) == len(sort_sequence.ItemIDs(cfg)) && len(sort_sequence.ItemIDs(cfg)) > 0
	if allCorrect || left == 0 {
		status = StatusCompleted
		st.CompletedAt = sort_sequence.NowRFC3339()
	}

	perItemOut := map[string]any{}
	for id, r := range grade.PerItem {
		entry := map[string]any{"correct": r.Correct}
		if cfg.ShowPerItemCorrectness && r.Feedback != "" && !r.Correct {
			entry["feedback"] = r.Feedback
		} else if cfg.ShowPerItemCorrectness && r.Feedback != "" && r.Correct {
			entry["feedback"] = r.Feedback
		}
		if !cfg.ShowPerItemCorrectness {
			delete(entry, "feedback")
			// Still include correct only when showPerItemCorrectness.
			perItemOut[id] = map[string]any{}
			continue
		}
		perItemOut[id] = entry
	}

	result := map[string]any{
		"perItem":           perItemOut,
		"scorePct":          grade.ScorePct,
		"attemptsRemaining": left,
		"showPerItem":       cfg.ShowPerItemCorrectness,
	}

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	outcome := "incorrect"
	if allCorrect {
		outcome = "correct"
	} else if len(grade.CorrectItemIDs) > 0 {
		outcome = "partial"
	}
	ObserveSortSequenceCheck(string(cfg.Mode), outcome)
	ObserveSortSequenceAttemptCount(len(st.Attempts))

	return &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     status,
		ScoreRaw:   &grade.ScoreRaw,
		ScoreMax:   &grade.ScoreMax,
	}, nil
}

func handleSortSequenceResetAttempt(ctx ActionContext) (*ActionResult, error) {
	cfg := sort_sequence.ParseConfig(ctx.ConfigJSON)
	st := sort_sequence.ParseState(ctx.StateJSON)
	st = sort_sequence.ResetUnlockedToTray(cfg, st)
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveSortSequenceCheck(string(cfg.Mode), "reset_attempt")
	// Do not demote lifecycle status — CT.4 owns resets; this only clears unlocked placement.
	return &ActionResult{
		Result: map[string]any{
			"ok":                true,
			"attemptsRemaining": sort_sequence.AttemptsRemaining(cfg, st),
		},
		StatePatch: patch,
	}, nil
}

func mergeLockedPlacement(cfg sort_sequence.Config, st sort_sequence.State, incoming json.RawMessage) json.RawMessage {
	locked := map[string]struct{}{}
	for _, id := range st.LockedItemIDs {
		locked[id] = struct{}{}
	}
	if len(locked) == 0 {
		return incoming
	}
	switch cfg.Mode {
	case sort_sequence.ModeOrder:
		prev, okPrev := sort_sequence.ParseOrderPlacement(st.Placement)
		next, okNext := sort_sequence.ParseOrderPlacement(incoming)
		if !okPrev || !okNext {
			return incoming
		}
		prevPos := map[string]int{}
		for i, id := range prev {
			prevPos[id] = i
		}
		// Build result: keep locked items at prior indices; fill rest from next unlocked.
		result := make([]string, len(sort_sequence.ItemIDs(cfg)))
		used := map[string]struct{}{}
		for id := range locked {
			if i, ok := prevPos[id]; ok && i >= 0 && i < len(result) {
				result[i] = id
				used[id] = struct{}{}
			}
		}
		fill := 0
		for _, id := range next {
			if _, u := used[id]; u {
				continue
			}
			for fill < len(result) && result[fill] != "" {
				fill++
			}
			if fill >= len(result) {
				break
			}
			result[fill] = id
			used[id] = struct{}{}
			fill++
		}
		// Append any remaining item ids not yet placed.
		for _, id := range sort_sequence.ItemIDs(cfg) {
			if _, u := used[id]; u {
				continue
			}
			for fill < len(result) && result[fill] != "" {
				fill++
			}
			if fill >= len(result) {
				break
			}
			result[fill] = id
			fill++
		}
		compact := make([]string, 0, len(result))
		for _, id := range result {
			if id != "" {
				compact = append(compact, id)
			}
		}
		return sort_sequence.MarshalOrderPlacement(compact)
	default:
		prev, okPrev := sort_sequence.ParseCategorizePlacement(st.Placement)
		next, okNext := sort_sequence.ParseCategorizePlacement(incoming)
		if !okPrev || !okNext {
			return incoming
		}
		merged := map[string]*string{}
		for _, id := range sort_sequence.ItemIDs(cfg) {
			if _, isLocked := locked[id]; isLocked {
				if v, ok := prev[id]; ok {
					merged[id] = v
					continue
				}
			}
			if v, ok := next[id]; ok {
				merged[id] = v
			} else {
				merged[id] = nil
			}
		}
		return sort_sequence.MarshalCategorizePlacement(merged)
	}
}
