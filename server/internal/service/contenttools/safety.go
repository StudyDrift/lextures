package contenttools

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/service/boardfilter"
)

const (
	FilterActionAllow = "allow"
	FilterActionFlag  = "flag"
	FilterActionBlock = "block"

	FilterCategoryProfanity = "profanity"
	FilterCategoryCrisis    = "crisis"
)

// crisisSignals are plain-language self-harm / abuse indicators (CT.8 FR-9).
var crisisSignals = []string{
	"kill myself", "suicide", "want to die", "end my life", "self harm", "self-harm",
	"cutting myself", "hurt myself",
}

// FreeTextScreenResult is the outcome of screening student free text.
type FreeTextScreenResult struct {
	Action      string // allow | flag | block
	Category    string
	MatchedTerm string
	Crisis      bool
	Guidance    string
}

// ScreenFreeText applies the org filter action + crisis detection (FR-8 / FR-9).
// On filter unavailability, callers should treat boardfilter as always available (in-process).
func ScreenFreeText(text string, filterAction string, crisisEnabled bool) FreeTextScreenResult {
	out := FreeTextScreenResult{Action: FilterActionAllow}
	if crisisEnabled {
		lower := strings.ToLower(text)
		for _, sig := range crisisSignals {
			if strings.Contains(lower, sig) {
				out.Crisis = true
				out.Category = FilterCategoryCrisis
				out.MatchedTerm = sig
				out.Action = FilterActionFlag
				out.Guidance = "If you are in crisis, please reach out to a trusted adult or local emergency services. Your instructor has been notified so they can help."
				IncCrisisEscalation()
				return out
			}
		}
	}
	res := boardfilter.Match(text, nil)
	if !res.Matched {
		return out
	}
	out.Category = FilterCategoryProfanity
	out.MatchedTerm = res.Term
	switch strings.ToLower(strings.TrimSpace(filterAction)) {
	case FilterActionBlock:
		out.Action = FilterActionBlock
		out.Guidance = "Please revise your response using respectful language. Your draft was not saved."
		IncContentFilterFlag(FilterCategoryProfanity)
	case FilterActionAllow:
		out.Action = FilterActionAllow
	default:
		out.Action = FilterActionFlag
		IncContentFilterFlag(FilterCategoryProfanity)
	}
	return out
}

// ExtractFreeTextFromState pulls common free-text fields from state JSON.
func ExtractFreeTextFromState(state json.RawMessage) string {
	if len(state) == 0 {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal(state, &obj); err != nil {
		return string(state)
	}
	parts := make([]string, 0, 4)
	for _, key := range []string{"response", "text", "comment", "body", "message", "answer", "draft"} {
		if v, ok := obj[key].(string); ok && strings.TrimSpace(v) != "" {
			parts = append(parts, v)
		}
	}
	if drafts, ok := obj["drafts"].(map[string]any); ok {
		for _, v := range drafts {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				parts = append(parts, s)
			}
		}
	}
	if posts, ok := obj["posts"].([]any); ok {
		for _, p := range posts {
			if m, ok := p.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
	}
	if annotations, ok := obj["annotations"].([]any); ok {
		for _, a := range annotations {
			if m, ok := a.(map[string]any); ok {
				if note, ok := m["note"].(string); ok && strings.TrimSpace(note) != "" {
					parts = append(parts, note)
				}
			}
		}
	}
	return strings.Join(parts, "\n")
}

// CrisisEscalation records a crisis signal and notifies via event insert (FR-9 / AC-5).
// Notification delivery is best-effort through content_tool_events; operators alert on the metric.
func RecordCrisisEscalation(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, instanceID uuid.UUID,
	actor *uuid.UUID,
	toolID string,
) error {
	payload := map[string]any{
		"category": FilterCategoryCrisis,
		"at":       time.Now().UTC().Format(time.RFC3339),
	}
	return ctrepo.InsertEvent(ctx, pool, courseID, &instanceID, nil, actor, toolID, "crisis_escalation", payload)
}

// RecordFilterFlag stores an aggregate flag without raw text (AC-4).
func RecordFilterFlag(
	ctx context.Context,
	pool *pgxpool.Pool,
	instanceID, courseID uuid.UUID,
	userID *uuid.UUID,
	category, action string,
) error {
	return ctrepo.InsertFilterFlag(ctx, pool, instanceID, courseID, userID, category, action)
}
