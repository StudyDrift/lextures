package contenttools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/highlight_annotate"
)

func init() {
	RegisterActionHandler(highlight_annotate.ID, "filter_note", handleHighlightAnnotateFilterNote)
}

func handleHighlightAnnotateFilterNote(ctx ActionContext) (*ActionResult, error) {
	var in struct {
		Note string `json:"note"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid filter_note input: %w", err)
		}
	}
	note := strings.TrimSpace(in.Note)
	if note == "" {
		ObserveHighlightAnnotateFilter("empty")
		return &ActionResult{
			Result: map[string]any{
				"ok": true,
			},
		}, nil
	}

	screen := ScreenFreeText(note, FilterActionBlock, true)
	if screen.Action == FilterActionBlock {
		ObserveHighlightAnnotateFilter("blocked")
		return &ActionResult{
			Result: map[string]any{
				"error":         "filtered",
				"message":       screen.Guidance,
				"preserveInput": true,
			},
		}, nil
	}
	if screen.Crisis {
		ObserveHighlightAnnotateFilter("crisis")
		return &ActionResult{
			Result: map[string]any{
				"error":         "filtered",
				"message":       screen.Guidance,
				"crisis":        true,
				"preserveInput": true,
			},
		}, nil
	}
	if screen.Action == FilterActionFlag {
		ObserveHighlightAnnotateFilter("flagged")
		return &ActionResult{
			Result: map[string]any{
				"ok":      true,
				"flagged": true,
				"message": screen.Guidance,
			},
		}, nil
	}
	ObserveHighlightAnnotateFilter("ok")
	return &ActionResult{
		Result: map[string]any{
			"ok": true,
		},
	}, nil
}

// DeriveHighlightAnnotateStatus advances status from annotation count vs min (AC-4).
// Returns empty string when the tool id does not match.
func DeriveHighlightAnnotateStatus(toolID string, configJSON, stateJSON json.RawMessage, current string) string {
	if toolID != highlight_annotate.ID {
		return ""
	}
	cfg := highlight_annotate.ParseConfig(configJSON)
	st := highlight_annotate.ParseState(stateJSON)
	st = highlight_annotate.DropUnknownTags(cfg, st)
	st = highlight_annotate.CapAnnotations(st, cfg.MaxAnnotations)
	derived := highlight_annotate.DeriveStatus(cfg, st, current)
	if derived == "" {
		return current
	}
	// Lifecycle is forward-only; never demote completed without a reset.
	if current == StatusCompleted {
		return StatusCompleted
	}
	if !CanTransitionStateStatus(current, derived) {
		return current
	}
	return derived
}

// NormalizeHighlightAnnotateState caps annotations and stamps completedAt when min is met.
func NormalizeHighlightAnnotateState(toolID string, configJSON, stateJSON json.RawMessage) (json.RawMessage, bool) {
	if toolID != highlight_annotate.ID || len(stateJSON) == 0 {
		return stateJSON, false
	}
	cfg := highlight_annotate.ParseConfig(configJSON)
	st := highlight_annotate.ParseState(stateJSON)
	st = highlight_annotate.DropUnknownTags(cfg, st)
	before := len(st.Annotations)
	st = highlight_annotate.CapAnnotations(st, cfg.MaxAnnotations)
	orphaned := 0
	for _, a := range st.Annotations {
		if a.Orphaned {
			orphaned++
		}
	}
	if st.MeetsMinimum(cfg) {
		if st.CompletedAt == "" {
			st.CompletedAt = highlight_annotate.NowRFC3339()
		}
	} else {
		st.CompletedAt = ""
	}
	raw, err := json.Marshal(st)
	if err != nil {
		return stateJSON, false
	}
	if before != len(st.Annotations) || orphaned > 0 {
		ObserveHighlightAnnotateOrphans(orphaned)
	}
	return raw, true
}
