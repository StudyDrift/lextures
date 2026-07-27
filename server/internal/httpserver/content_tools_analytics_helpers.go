package httpserver

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	ctanalytics "github.com/lextures/lextures/server/internal/service/contenttools/analytics"
)

const (
	contentToolsAnalyticsRatePerMin = 120
	contentToolsExportRatePerMin    = 5
)

// contentToolsAnalyticsRateLimited returns true when the request was rejected (and a response written).
func (d Deps) contentToolsAnalyticsRateLimited(w http.ResponseWriter, r *http.Request, viewer uuid.UUID) bool {
	return !d.contentToolsRateLimit(w, r, viewer, "analytics", contentToolsAnalyticsRatePerMin)
}

// contentToolsExportRateLimited returns true when the request was rejected (and a response written).
func (d Deps) contentToolsExportRateLimited(w http.ResponseWriter, r *http.Request, viewer uuid.UUID) bool {
	return !d.contentToolsRateLimit(w, r, viewer, "analytics_export", contentToolsExportRatePerMin)
}

func (d Deps) buildInstanceAnalytics(ctx context.Context, courseID, instanceID uuid.UUID, suppressSmallN bool) (*ctmodel.InstanceAnalytics, error) {
	cacheKey := ctanalytics.CacheKeyInstance(instanceID.String())
	if !suppressSmallN {
		// Instructor instance view: identified roster — do not suppress (FR-6 exception).
		if cached, ok := ctanalytics.DefaultCache().Get(cacheKey + ":full"); ok {
			if ia, ok := cached.(ctmodel.InstanceAnalytics); ok {
				return &ia, nil
			}
		}
	} else if cached, ok := ctanalytics.DefaultCache().Get(cacheKey + ":sup"); ok {
		if ia, ok := cached.(ctmodel.InstanceAnalytics); ok {
			return &ia, nil
		}
	}

	inst, err := ctrepo.GetInstance(ctx, d.Pool, courseID, instanceID)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, nil
	}
	rows, err := ctrepo.ListSummariesForInstance(ctx, d.Pool, instanceID)
	if err != nil {
		return nil, err
	}
	agg := ctanalytics.AggregateInstance(ctanalytics.ToAggregateRows(rows), inst.ToolID, ctanalytics.DefaultSmallN, suppressSmallN)
	link, _ := ctrepo.GetGradeLink(ctx, d.Pool, instanceID)
	counts := link != nil && link.CountsForGrade

	ia := ctmodel.InstanceAnalytics{
		InstanceID:       inst.ID,
		ToolID:           inst.ToolID,
		Title:            inst.Title,
		Learners:         agg.Learners,
		Engaged:          agg.Engaged,
		Completed:        agg.Completed,
		Suppressed:       agg.Suppressed,
		MedianDurationMs: agg.MedianDurationMs,
		Facets:           make([]ctmodel.FacetAggregate, 0, len(agg.Facets)),
		NeedsAttention:   make([]ctmodel.NeedsAttention, 0, len(agg.NeedsAttention)),
		CountsForGrade:   counts,
	}
	if agg.ScoreMean != nil && !agg.Suppressed {
		dist := make([]ctmodel.ScoreDistributionBucket, 0, len(agg.ScoreDistribution))
		for _, b := range agg.ScoreDistribution {
			dist = append(dist, ctmodel.ScoreDistributionBucket{Bucket: b.Bucket, Count: b.Count})
		}
		ia.Score = &ctmodel.InstanceScoreStats{
			Mean:         *agg.ScoreMean,
			Median:       *agg.ScoreMedian,
			Distribution: dist,
		}
	}
	for _, f := range agg.Facets {
		vals := make([]ctmodel.FacetValue, 0, len(f.Values))
		for _, v := range f.Values {
			vals = append(vals, ctmodel.FacetValue{Value: v.Value, Count: v.Count, Correct: v.Correct})
		}
		ia.Facets = append(ia.Facets, ctmodel.FacetAggregate{Key: f.Key, Label: f.Label, Values: vals})
	}
	for _, n := range agg.NeedsAttention {
		ia.NeedsAttention = append(ia.NeedsAttention, ctmodel.NeedsAttention{
			EnrollmentID: n.EnrollmentID,
			DisplayName:  n.DisplayName,
			Reason:       string(n.Reason),
		})
	}
	if suppressSmallN {
		ctanalytics.DefaultCache().Set(cacheKey+":sup", ia)
	} else {
		ctanalytics.DefaultCache().Set(cacheKey+":full", ia)
	}
	return &ia, nil
}
