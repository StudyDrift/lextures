package learnerprofile

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	lprepo "github.com/lextures/lextures/server/internal/repos/learnerprofile"
	"github.com/lextures/lextures/server/internal/redisclient"
)

const adaptiveContextCacheTTL = 15 * time.Minute

// PeakWindowFacet is a dominant study window from the profile.
type PeakWindowFacet struct {
	Dow        string  `json:"dow"`
	HourBucket string  `json:"hourBucket"`
	Share      float64 `json:"share"`
}

// AdaptiveContext is the read-optimised facet bundle adaptive consumers use (LP09 FR-1).
type AdaptiveContext struct {
	Active            bool              `json:"active"`
	EvalCohort        string            `json:"evalCohort"`
	Interests         []string          `json:"interests,omitempty"`
	GrowthAreas       []string          `json:"growthAreas,omitempty"`
	NeedsReview       []string          `json:"needsReview,omitempty"`
	ModalityAffinity  map[string]float64 `json:"modalityAffinity,omitempty"`
	PreferredModality string            `json:"preferredModality,omitempty"`
	PeakWindows       []PeakWindowFacet `json:"peakWindows,omitempty"`
	HelpSeekingStyle  string            `json:"helpSeekingStyle,omitempty"`
}

// ProfileRationale explains a profile-driven adaptation for the UI (LP09 FR-6).
type ProfileRationale struct {
	Text       string `json:"text"`
	FacetKey   string `json:"facetKey"`
	InsightKey string `json:"insightKey"`
}

// SetRedis wires the shared Redis client used to cache adaptive context (LP09 NFR).
func (s *Service) SetRedis(redis *redisclient.Client) {
	s.redis = redis
}

func adaptiveContextCacheKey(userID uuid.UUID) string {
	return "lp:adaptive_ctx:" + userID.String()
}

// GetAdaptiveContext returns cached adaptive facets for a user, or builds them from stored profile data.
func (s *Service) GetAdaptiveContext(ctx context.Context, userID uuid.UUID) (AdaptiveContext, error) {
	if s == nil || s.Pool == nil {
		return AdaptiveContext{}, nil
	}
	if s.redis != nil {
		raw, err := s.redis.Get(ctx, adaptiveContextCacheKey(userID))
		if err == nil && raw != "" {
			var cached AdaptiveContext
			if json.Unmarshal([]byte(raw), &cached) == nil {
				return cached, nil
			}
		}
	}
	ctxOut, err := s.buildAdaptiveContext(ctx, userID)
	if err != nil {
		return AdaptiveContext{}, err
	}
	if s.redis != nil {
		if b, err := json.Marshal(ctxOut); err == nil {
			_ = s.redis.Set(ctx, adaptiveContextCacheKey(userID), string(b), adaptiveContextCacheTTL)
		}
	}
	return ctxOut, nil
}

// InvalidateAdaptiveContext drops the cached adaptive context for a user (call after recompute).
func (s *Service) InvalidateAdaptiveContext(ctx context.Context, userID uuid.UUID) {
	if s == nil || s.redis == nil {
		return
	}
	_ = s.redis.Del(ctx, adaptiveContextCacheKey(userID))
}

func (s *Service) buildAdaptiveContext(ctx context.Context, userID uuid.UUID) (AdaptiveContext, error) {
	out := AdaptiveContext{EvalCohort: evalCohortForUser(userID)}
	p, err := lprepo.GetProfileByUserID(ctx, s.Pool, userID)
	if err != nil {
		return out, err
	}
	if p == nil || p.Status == "paused" {
		recordAdaptation("context", "suppressed")
		return out, nil
	}
	profileID, err := lprepo.EnsureProfile(ctx, s.Pool, userID)
	if err != nil {
		return out, err
	}
	facets, err := lprepo.ListFacets(ctx, s.Pool, profileID)
	if err != nil {
		return out, err
	}
	if len(facets) == 0 {
		recordAdaptation("context", "suppressed")
		return out, nil
	}
	okCount := 0
	for _, f := range facets {
		if f.State != "ok" {
			continue
		}
		okCount++
		switch f.FacetKey {
		case "interests":
			out.Interests = parseInterestTopics(f.Summary)
		case "strengths_growth":
			out.GrowthAreas, out.NeedsReview = parseStrengthsGrowth(f.Summary)
		case "content_modality":
			out.ModalityAffinity, out.PreferredModality = parseModalityAffinity(f.Summary)
		case "study_rhythm":
			out.PeakWindows = parsePeakWindows(f.Summary)
		case "learning_approach":
			out.HelpSeekingStyle = parseHelpSeekingStyle(f.Summary)
		}
	}
	if okCount == 0 {
		recordAdaptation("context", "suppressed")
		return out, nil
	}
	out.Active = true
	recordAdaptation("context", "applied")
	slog.Debug("learner_profile.adaptive_context",
		"user_hash", hashUserID(userID),
		"active", out.Active,
		"cohort", out.EvalCohort,
		"preferred_modality", out.PreferredModality,
		"help_seeking", out.HelpSeekingStyle)
	return out, nil
}

func evalCohortForUser(userID uuid.UUID) string {
	// Stable 50/50 eval assignment for LP09 pilot metrics (not persisted).
	if userID[15]&1 == 0 {
		return "personalised"
	}
	return "control"
}

func parseInterestTopics(summary json.RawMessage) []string {
	var payload struct {
		Topics []struct {
			Topic string `json:"topic"`
		} `json:"topics"`
	}
	if json.Unmarshal(summary, &payload) != nil {
		return nil
	}
	out := make([]string, 0, len(payload.Topics))
	for _, t := range payload.Topics {
		topic := strings.TrimSpace(t.Topic)
		if topic != "" {
			out = append(out, topic)
		}
	}
	return out
}

func parseStrengthsGrowth(summary json.RawMessage) (growth, needsReview []string) {
	var payload struct {
		Growth []struct {
			Concept string `json:"concept"`
		} `json:"growth"`
		NeedsReview []struct {
			Concept string `json:"concept"`
		} `json:"needsReview"`
	}
	if json.Unmarshal(summary, &payload) != nil {
		return nil, nil
	}
	for _, g := range payload.Growth {
		if c := strings.TrimSpace(g.Concept); c != "" {
			growth = append(growth, c)
		}
	}
	for _, n := range payload.NeedsReview {
		if c := strings.TrimSpace(n.Concept); c != "" {
			needsReview = append(needsReview, c)
		}
	}
	return growth, needsReview
}

func parseModalityAffinity(summary json.RawMessage) (map[string]float64, string) {
	var payload struct {
		ModalityAffinity map[string]float64 `json:"modalityAffinity"`
	}
	if json.Unmarshal(summary, &payload) != nil || len(payload.ModalityAffinity) == 0 {
		return nil, ""
	}
	best := ""
	bestScore := -1.0
	for mod, score := range payload.ModalityAffinity {
		if score > bestScore {
			bestScore = score
			best = mod
		}
	}
	return payload.ModalityAffinity, best
}

func parsePeakWindows(summary json.RawMessage) []PeakWindowFacet {
	var payload struct {
		PeakWindows []PeakWindowFacet `json:"peakWindows"`
	}
	if json.Unmarshal(summary, &payload) != nil {
		return nil
	}
	return payload.PeakWindows
}

func parseHelpSeekingStyle(summary json.RawMessage) string {
	var payload struct {
		HelpSeeking struct {
			Style string `json:"style"`
		} `json:"helpSeeking"`
	}
	if json.Unmarshal(summary, &payload) != nil {
		return ""
	}
	return strings.TrimSpace(payload.HelpSeeking.Style)
}

// Usable reports whether consumers should apply profile personalisation.
func (c AdaptiveContext) Usable(cohortGate bool) bool {
	if !c.Active {
		return false
	}
	if cohortGate && c.EvalCohort == "control" {
		return false
	}
	return true
}