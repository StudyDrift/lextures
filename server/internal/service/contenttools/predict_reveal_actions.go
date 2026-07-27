package contenttools

import (
	"encoding/json"
	"fmt"
	"strings"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/predict_reveal"
)

func init() {
	RegisterActionHandler(predict_reveal.ID, "commit", handlePredictRevealCommit)
	RegisterActionHandler(predict_reveal.ID, "reflect", handlePredictRevealReflect)
}

func handlePredictRevealCommit(ctx ActionContext) (*ActionResult, error) {
	cfg := predict_reveal.ParseConfig(ctx.ConfigJSON)
	st := predict_reveal.ParseState(ctx.StateJSON)

	var in struct {
		Prediction *predict_reveal.Prediction `json:"prediction"`
		Confidence *float64                   `json:"confidence"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid commit input: %w", err)
		}
	}

	// Idempotent recommit: return reveal again; refuse mutation (AC-2 / AC-3).
	if st.IsCommitted() {
		if in.Prediction != nil || in.Confidence != nil {
			ObservePredictRevealCommit("already_committed")
			return &ActionResult{
				Result: map[string]any{
					"error":   "already_committed",
					"message": "You already committed a prediction. It cannot be changed.",
				},
			}, nil
		}
		result := map[string]any{
			"reveal": revealPayload(cfg.Reveal),
			"state":  committedView(st),
		}
		if peer := peerResultsFor(ctx, cfg, &st); peer != nil {
			result["peerResults"] = peer
		}
		ObservePredictRevealCommit("idempotent")
		return &ActionResult{
			Result: result,
			Status: StatusCompleted,
		}, nil
	}

	pred := in.Prediction
	if pred == nil && st.Draft != nil {
		pred = &predict_reveal.Prediction{
			OutcomeID: st.Draft.OutcomeID,
			Text:      st.Draft.Text,
		}
	}
	if pred == nil {
		return nil, fmt.Errorf("prediction is required")
	}

	switch cfg.Mode {
	case predict_reveal.ModeChoice:
		pred.OutcomeID = strings.TrimSpace(pred.OutcomeID)
		pred.Text = ""
		if pred.OutcomeID == "" {
			return nil, fmt.Errorf("prediction.outcomeId is required")
		}
		if predict_reveal.FindOutcome(cfg, pred.OutcomeID) == nil {
			return nil, fmt.Errorf("unknown outcomeId")
		}
	case predict_reveal.ModeOpen:
		pred.Text = strings.TrimSpace(pred.Text)
		pred.OutcomeID = ""
		if pred.Text == "" {
			return nil, fmt.Errorf("prediction.text is required")
		}
		screen := ScreenFreeText(pred.Text, FilterActionFlag, true)
		if screen.Action == FilterActionBlock {
			ObservePredictRevealCommit("filtered")
			return &ActionResult{
				Result: map[string]any{
					"error":         "filtered",
					"message":       screen.Guidance,
					"preserveInput": true,
				},
			}, nil
		}
		if screen.Crisis {
			ObservePredictRevealCommit("crisis")
			return &ActionResult{
				Result: map[string]any{
					"error":         "filtered",
					"message":       screen.Guidance,
					"crisis":        true,
					"preserveInput": true,
				},
			}, nil
		}
	default:
		return nil, fmt.Errorf("unsupported mode")
	}

	var confNorm *float64
	bucket := ""
	if cfg.ConfidenceScale == predict_reveal.ScaleNone {
		bucket = "none"
	} else {
		raw := (*float64)(nil)
		if in.Confidence != nil {
			raw = in.Confidence
		} else if st.Draft != nil && st.Draft.Confidence != nil {
			raw = st.Draft.Confidence
		}
		if raw == nil {
			if cfg.ConfidenceRequired {
				ObservePredictRevealCommit("confidence_required")
				return &ActionResult{
					Result: map[string]any{
						"error":   "confidence_required",
						"message": "Select how sure you are before committing.",
					},
				}, nil
			}
		} else {
			n, b, err := predict_reveal.NormalizeConfidence(cfg.ConfidenceScale, *raw)
			if err != nil {
				return nil, err
			}
			confNorm = &n
			bucket = b
		}
	}

	now := predict_reveal.NowRFC3339()
	st.Prediction = pred
	st.Confidence = confNorm
	st.ConfidenceBucket = bucket
	st.CommittedAt = now
	st.RevealedAt = now
	st.Correct = predict_reveal.TagCorrectness(cfg, pred.OutcomeID)
	st.Draft = nil
	st.V = 1

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"reveal": revealPayload(cfg.Reveal),
		"state":  committedView(st),
	}
	if peer := peerResultsFor(ctx, cfg, &st); peer != nil {
		result["peerResults"] = peer
	}

	if st.Correct != nil && !*st.Correct && isConfidentlyWrong(bucket) {
		ObservePredictRevealConfidentlyWrong()
	}
	ObservePredictRevealCommit("ok")
	if st.Confidence != nil {
		ObservePredictRevealConfidence(*st.Confidence)
	}

	return &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     StatusCompleted,
	}, nil
}

func handlePredictRevealReflect(ctx ActionContext) (*ActionResult, error) {
	cfg := predict_reveal.ParseConfig(ctx.ConfigJSON)
	st := predict_reveal.ParseState(ctx.StateJSON)
	if !st.IsCommitted() {
		return &ActionResult{
			Result: map[string]any{
				"error":   "not_committed",
				"message": "Commit a prediction before reflecting.",
			},
		}, nil
	}
	var in struct {
		Text string `json:"text"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid reflect input: %w", err)
		}
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	screen := ScreenFreeText(text, FilterActionFlag, true)
	if screen.Action == FilterActionBlock || screen.Crisis {
		return &ActionResult{
			Result: map[string]any{
				"error":         "filtered",
				"message":       screen.Guidance,
				"crisis":        screen.Crisis,
				"preserveInput": true,
			},
		}, nil
	}
	_ = cfg // reflection prompt is client-displayed; no server gate beyond commit
	st.Reflection = text
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	return &ActionResult{
		Result:     map[string]any{"state": committedView(st)},
		StatePatch: patch,
		Status:     StatusCompleted,
	}, nil
}

func revealPayload(r predict_reveal.Reveal) map[string]any {
	out := map[string]any{"markdown": r.Markdown}
	if r.ImageURL != "" {
		out["imageUrl"] = r.ImageURL
	}
	return out
}

func committedView(st predict_reveal.State) map[string]any {
	out := map[string]any{
		"v":           st.V,
		"committedAt": st.CommittedAt,
		"revealedAt":  st.RevealedAt,
	}
	if st.Prediction != nil {
		out["prediction"] = st.Prediction
	}
	if st.Confidence != nil {
		out["confidence"] = *st.Confidence
	}
	if st.ConfidenceBucket != "" {
		out["confidenceBucket"] = st.ConfidenceBucket
	}
	if st.Correct != nil {
		out["correct"] = *st.Correct
	}
	if st.Reflection != "" {
		out["reflection"] = st.Reflection
	}
	return out
}

func peerResultsFor(ctx ActionContext, cfg predict_reveal.Config, committed *predict_reveal.State) *predict_reveal.PeerResults {
	if !cfg.ShowPeerResults {
		return nil
	}
	if ctx.Pool == nil || ctx.Ctx == nil {
		pr := predict_reveal.PeerResults{Suppressed: true, Reason: "unavailable", Learners: 0}
		return &pr
	}
	raws, err := ctrepo.ListEnrollmentStateJSONForInstance(ctx.Ctx, ctx.Pool, ctx.InstanceID)
	if err != nil {
		pr := predict_reveal.PeerResults{Suppressed: true, Reason: "unavailable", Learners: 0}
		return &pr
	}
	// Replace caller's prior row (if any) with the committed snapshot so counts are current
	// even though the action runs before the state upsert.
	if committed != nil && committed.IsCommitted() {
		selfJSON, _ := json.Marshal(committed)
		filtered := make([]json.RawMessage, 0, len(raws)+1)
		for _, raw := range raws {
			st := predict_reveal.ParseState(raw)
			// Drop uncommitted self drafts; keep other enrollments' commits.
			if !st.IsCommitted() {
				continue
			}
			filtered = append(filtered, raw)
		}
		// Always append this enrollment's committed state (may double-count if already
		// persisted on idempotent recommit — dedupe by preferring DB rows when present).
		already := false
		prior := predict_reveal.ParseState(ctx.StateJSON)
		if prior.IsCommitted() {
			already = true
		}
		if !already {
			filtered = append(filtered, selfJSON)
		}
		raws = filtered
	}
	pr := predict_reveal.BuildPeerResults(raws, cfg.Mode)
	return &pr
}

func isConfidentlyWrong(bucket string) bool {
	switch bucket {
	case "certain", "5", "80_100", "4":
		return true
	default:
		return false
	}
}

// GuardPredictRevealStatePut refuses PUT after commit (AC-3).
func GuardPredictRevealStatePut(toolID string, current json.RawMessage) (blocked bool, message string) {
	if toolID != predict_reveal.ID {
		return false, ""
	}
	return predict_reveal.GuardStatePut(current, nil)
}
