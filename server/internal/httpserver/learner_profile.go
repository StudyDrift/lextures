package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	lprepo "github.com/lextures/lextures/server/internal/repos/learnerprofile"
	lpsvc "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

func (d Deps) learnerProfileEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().LearnerProfileEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Learner profile is not enabled.")
		return false
	}
	return true
}

func (d Deps) learnerProfileService() *lpsvc.Service {
	if d.LearnerProfileService != nil {
		return d.LearnerProfileService
	}
	return nil
}

func (d Deps) registerLearnerProfileRoutes(r chi.Router) {
	r.Get("/api/v1/me/learner-profile", d.handleGetLearnerProfile())
	r.Get("/api/v1/me/learner-profile/facets/{facetKey}", d.handleGetLearnerProfileFacet())
	r.Get("/api/v1/me/learner-profile/facets/{facetKey}/evidence", d.handleGetLearnerProfileFacetEvidence())
	r.Post("/api/v1/me/learner-profile/pause", d.handlePostLearnerProfilePause())
	r.Post("/api/v1/me/learner-profile/resume", d.handlePostLearnerProfileResume())
	r.Post("/api/v1/me/learner-profile/reset", d.handlePostLearnerProfileReset())
	r.Get("/api/v1/me/learner-profile/export", d.handleGetLearnerProfileExport())
}

func (d Deps) handleGetLearnerProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.learnerProfileEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		svc := d.learnerProfileService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Learner profile service unavailable.")
			return
		}
		profile, err := svc.Get(r.Context(), userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load learner profile.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"profile": learnerProfileToJSON(profile)})
	}
}

func (d Deps) handleGetLearnerProfileFacet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.learnerProfileEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		facetKey := chi.URLParam(r, "facetKey")
		if _, known := lprepo.ValidFacetKeys[facetKey]; !known {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unknown facet.")
			return
		}
		svc := d.learnerProfileService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Learner profile service unavailable.")
			return
		}
		detail, err := svc.GetFacet(r.Context(), userID, facetKey)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load facet.")
			return
		}
		if detail == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Facet not derived yet.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"facet":    learnerFacetSummaryToJSON(detail.Facet),
			"insights": learnerInsightsToJSON(detail.Insights),
		})
	}
}

func (d Deps) handleGetLearnerProfileFacetEvidence() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.learnerProfileEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		facetKey := chi.URLParam(r, "facetKey")
		if _, known := lprepo.ValidFacetKeys[facetKey]; !known {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unknown facet.")
			return
		}
		svc := d.learnerProfileService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Learner profile service unavailable.")
			return
		}
		evidence, err := svc.GetFacetEvidence(r.Context(), userID, facetKey)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load evidence.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(evidence)
	}
}

func learnerProfileToJSON(p lpsvc.ProfileView) map[string]any {
	out := map[string]any{
		"status": p.Status,
		"facets": learnerFacetSummariesToJSON(p.Facets),
	}
	if p.LastComputedAt != nil {
		out["lastComputedAt"] = p.LastComputedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func learnerFacetSummariesToJSON(facets []lpsvc.FacetSummary) []map[string]any {
	out := make([]map[string]any, 0, len(facets))
	for _, f := range facets {
		out = append(out, learnerFacetSummaryToJSON(f))
	}
	return out
}

func learnerFacetSummaryToJSON(f lpsvc.FacetSummary) map[string]any {
	var summary any
	_ = json.Unmarshal(f.Summary, &summary)
	if summary == nil {
		summary = map[string]any{}
	}
	return map[string]any{
		"facetKey":        f.FacetKey,
		"state":           f.State,
		"summary":         summary,
		"confidence":      f.Confidence,
		"computedVersion": f.ComputedVersion,
		"updatedAt":       f.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func learnerInsightsToJSON(insights []lpsvc.InsightView) []map[string]any {
	out := make([]map[string]any, 0, len(insights))
	for _, ins := range insights {
		var value any
		_ = json.Unmarshal(ins.Value, &value)
		if value == nil {
			value = map[string]any{}
		}
		out = append(out, map[string]any{
			"insightKey": ins.InsightKey,
			"label":      ins.Label,
			"value":      value,
			"confidence": ins.Confidence,
			"salience":   ins.Salience,
			"evidence":   learnerEvidenceViewsToJSON(ins.Evidence),
		})
	}
	return out
}

func learnerEvidenceViewsToJSON(rows []lpsvc.EvidenceView) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, ev := range rows {
		item := map[string]any{
			"sourceKind":       ev.SourceKind,
			"sourceTable":      ev.SourceTable,
			"observationCount": ev.ObservationCount,
		}
		if ev.CourseID != nil {
			item["courseId"] = *ev.CourseID
		}
		if ev.WindowStart != nil {
			item["windowStart"] = *ev.WindowStart
		}
		if ev.WindowEnd != nil {
			item["windowEnd"] = *ev.WindowEnd
		}
		if ev.Contribution != nil {
			item["contribution"] = *ev.Contribution
		}
		out = append(out, item)
	}
	return out
}