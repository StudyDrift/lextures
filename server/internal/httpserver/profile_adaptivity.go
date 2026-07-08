package httpserver

import (
	"context"

	"github.com/google/uuid"

	lpsvc "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

type profileRationaleJSON struct {
	Text       string `json:"text"`
	FacetKey   string `json:"facetKey"`
	InsightKey string `json:"insightKey"`
}

func rationaleToJSON(r *lpsvc.ProfileRationale) *profileRationaleJSON {
	if r == nil {
		return nil
	}
	return &profileRationaleJSON{
		Text:       r.Text,
		FacetKey:   r.FacetKey,
		InsightKey: r.InsightKey,
	}
}

func (d Deps) profileAdaptEnabled(consumer string) bool {
	cfg := d.effectiveConfig()
	if !cfg.LearnerProfileEnabled || d.LearnerProfileService == nil {
		return false
	}
	switch consumer {
	case "recommendations":
		return cfg.LpAdaptRecommendationsEnabled
	case "review":
		return cfg.LpAdaptReviewEnabled
	case "modality":
		return cfg.LpAdaptModalityEnabled
	case "tutor":
		return cfg.LpAdaptTutorEnabled
	default:
		return false
	}
}

func (d Deps) loadAdaptiveContext(ctx context.Context, userID uuid.UUID) (lpsvc.AdaptiveContext, error) {
	if d.LearnerProfileService == nil {
		return lpsvc.AdaptiveContext{}, nil
	}
	return d.LearnerProfileService.GetAdaptiveContext(ctx, userID)
}