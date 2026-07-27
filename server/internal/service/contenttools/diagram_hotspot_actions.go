package contenttools

import (
	"encoding/json"
	"fmt"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/diagram_hotspot"
)

func init() {
	RegisterActionHandler(diagram_hotspot.ID, "check", handleDiagramHotspotCheck)
	RegisterActionHandler(diagram_hotspot.ID, "reset_attempt", handleDiagramHotspotResetAttempt)
}

func handleDiagramHotspotCheck(ctx ActionContext) (*ActionResult, error) {
	cfg := diagram_hotspot.ParseConfig(ctx.ConfigJSON)
	st := diagram_hotspot.ParseState(ctx.StateJSON)

	var in struct {
		Assignments  json.RawMessage `json:"assignments"`
		UsedListMode *bool           `json:"usedListMode"`
		Clicks       map[string]struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"clicks"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid check input: %w", err)
		}
	}

	// Authoring-quality gate (descriptions + alt) — reject badly saved configs early.
	if err := diagram_hotspot.ValidateConfigForAuthoring(cfg); err != nil {
		ObserveDiagramHotspotCheck(string(cfg.Mode), "invalid_config")
		return &ActionResult{
			Result: map[string]any{
				"error":   "invalid_config",
				"message": err.Error(),
			},
		}, nil
	}

	var assignments map[string]*string
	if len(in.Assignments) > 0 {
		parsed, ok := diagram_hotspot.ParseAssignments(in.Assignments)
		if !ok {
			ObserveDiagramHotspotCheck(string(cfg.Mode), "invalid_placement")
			return &ActionResult{
				Result: map[string]any{
					"error":   "invalid_placement",
					"message": "invalid assignments",
				},
			}, nil
		}
		assignments = parsed
	} else {
		assignments = diagram_hotspot.AssignmentsFromState(st)
	}

	// Optional pointer clicks: map normalized coords → smallest containing region
	// and merge into assignments (spatial path equivalence to list mode).
	if len(in.Clicks) > 0 {
		for itemID, pt := range in.Clicks {
			regionID := diagram_hotspot.SmallestContainingRegion(cfg.Regions, pt.X, pt.Y)
			if regionID == "" {
				continue
			}
			cp := regionID
			assignments[itemID] = &cp
		}
	}

	remaining := diagram_hotspot.AttemptsRemaining(cfg, st)
	if remaining == 0 {
		ObserveDiagramHotspotCheck(string(cfg.Mode), "max_attempts")
		return &ActionResult{
			Result: map[string]any{
				"error":             "max_attempts",
				"message":           "No attempts remaining.",
				"attemptsRemaining": 0,
			},
		}, nil
	}

	if err := diagram_hotspot.ValidateAssignmentIDs(cfg, assignments); err != nil {
		ObserveDiagramHotspotCheck(string(cfg.Mode), "invalid_placement")
		return &ActionResult{
			Result: map[string]any{
				"error":   "invalid_placement",
				"message": err.Error(),
			},
		}, nil
	}

	if !diagram_hotspot.AllItemsAssigned(cfg, assignments) {
		ObserveDiagramHotspotCheck(string(cfg.Mode), "incomplete")
		return &ActionResult{
			Result: map[string]any{
				"error":   "incomplete",
				"message": "Assign every label or prompt before checking.",
			},
		}, nil
	}

	if cfg.LockCorrect && len(st.LockedIDs) > 0 {
		assignments = mergeLockedAssignments(cfg, st, assignments)
	}

	grade := diagram_hotspot.GradeAssignments(cfg, assignments)
	flat := diagram_hotspot.FlatAssignments(assignments)
	attempt := diagram_hotspot.Attempt{
		At:          diagram_hotspot.NowRFC3339(),
		CorrectIDs:  grade.CorrectIDs,
		ScorePct:    grade.ScorePct,
		Assignments: flat,
		HeatCells:   diagram_hotspot.HeatCellsForAssignments(cfg, assignments),
	}
	st.Attempts = append(st.Attempts, attempt)
	st.Assignments = assignments
	st.LastPerItem = map[string]bool{}
	for id, r := range grade.PerItem {
		st.LastPerItem[id] = r.Correct
	}
	if in.UsedListMode != nil && *in.UsedListMode {
		st.UsedListMode = true
	}

	if cfg.LockCorrect {
		locked := map[string]struct{}{}
		for _, id := range diagram_hotspot.ItemIDs(cfg) {
			if diagram_hotspot.IsLocked(st, id) {
				locked[id] = struct{}{}
			}
		}
		for _, id := range grade.CorrectIDs {
			locked[id] = struct{}{}
		}
		st.LockedIDs = make([]string, 0, len(locked))
		for id := range locked {
			st.LockedIDs = append(st.LockedIDs, id)
		}
	}

	status := StatusSubmitted
	left := diagram_hotspot.AttemptsRemaining(cfg, st)
	allCorrect := len(grade.CorrectIDs) == len(diagram_hotspot.ItemIDs(cfg)) && len(diagram_hotspot.ItemIDs(cfg)) > 0
	if allCorrect || left == 0 {
		status = StatusCompleted
		st.CompletedAt = diagram_hotspot.NowRFC3339()
	}

	perItemOut := map[string]any{}
	for id, r := range grade.PerItem {
		if !cfg.ShowPerItemCorrectness {
			perItemOut[id] = map[string]any{}
			continue
		}
		entry := map[string]any{"correct": r.Correct}
		if r.Feedback != "" {
			entry["feedback"] = r.Feedback
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
	} else if len(grade.CorrectIDs) > 0 {
		outcome = "partial"
	}
	ObserveDiagramHotspotCheck(string(cfg.Mode), outcome)
	if st.UsedListMode {
		ObserveDiagramHotspotListMode()
	}

	return &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     status,
		ScoreRaw:   &grade.ScoreRaw,
		ScoreMax:   &grade.ScoreMax,
	}, nil
}

func handleDiagramHotspotResetAttempt(ctx ActionContext) (*ActionResult, error) {
	cfg := diagram_hotspot.ParseConfig(ctx.ConfigJSON)
	st := diagram_hotspot.ParseState(ctx.StateJSON)
	st = diagram_hotspot.ResetUnlocked(cfg, st)
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveDiagramHotspotCheck(string(cfg.Mode), "reset_attempt")
	return &ActionResult{
		Result: map[string]any{
			"ok":                true,
			"attemptsRemaining": diagram_hotspot.AttemptsRemaining(cfg, st),
		},
		StatePatch: patch,
	}, nil
}

func mergeLockedAssignments(
	cfg diagram_hotspot.Config,
	st diagram_hotspot.State,
	incoming map[string]*string,
) map[string]*string {
	locked := map[string]struct{}{}
	for _, id := range st.LockedIDs {
		locked[id] = struct{}{}
	}
	if len(locked) == 0 {
		return incoming
	}
	merged := map[string]*string{}
	for _, id := range diagram_hotspot.ItemIDs(cfg) {
		if _, isLocked := locked[id]; isLocked {
			if v, ok := st.Assignments[id]; ok {
				merged[id] = v
				continue
			}
		}
		if v, ok := incoming[id]; ok {
			merged[id] = v
		} else {
			merged[id] = nil
		}
	}
	return merged
}
