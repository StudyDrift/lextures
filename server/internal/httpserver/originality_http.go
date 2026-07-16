package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/originalityreports"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/plagiarism"
)

func (d Deps) requirePlagiarismWorkflow(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFPlagiarismChecks && !d.effectiveConfig().FFPlagiarismChecks {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Plagiarism workflow is not enabled.")
		return false
	}
	if !cfg.OriginalityDetectionEnabled && !d.effectiveConfig().OriginalityDetectionEnabled {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Originality detection is not enabled.")
		return false
	}
	return true
}

func (d Deps) plagiarismService(orgID *uuid.UUID) *plagiarism.Service {
	cfg := d.effectiveConfig()
	return &plagiarism.Service{
		Pool:         d.Pool,
		Config:       cfg,
		FilesRoot:    d.Config.CourseFilesRoot,
		AI:           aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID},
		StubExternal: cfg.OriginalityStubExternal,
	}
}

func reportToJSON(r originalityreports.Report) map[string]any {
	out := map[string]any{
		"provider": r.Provider,
		"status":   r.Status,
	}
	if r.SimilarityPct != nil {
		out["similarityPct"] = *r.SimilarityPct
	} else {
		out["similarityPct"] = nil
	}
	if r.AIProbability != nil {
		out["aiProbability"] = *r.AIProbability
	} else {
		out["aiProbability"] = nil
	}
	if r.ReportURL != nil {
		out["reportUrl"] = *r.ReportURL
	} else {
		out["reportUrl"] = nil
	}
	if r.ReportToken != nil {
		out["reportToken"] = *r.ReportToken
	}
	if r.ErrorMessage != nil {
		out["errorMessage"] = *r.ErrorMessage
	}
	return out
}

func summaryFromReports(reports []originalityreports.Report) map[string]any {
	var best *originalityreports.Report
	for i := range reports {
		r := &reports[i]
		if r.Status != "done" {
			continue
		}
		if best == nil || r.UpdatedAt.After(best.UpdatedAt) {
			best = r
		}
	}
	if best == nil {
		return map[string]any{
			"provider":                "",
			"similarityPct":           nil,
			"aiProbability":           nil,
			"fullReportUnavailable":   true,
			"fullReportUnavailableMessage": "Originality scan is not complete yet.",
		}
	}
	summary := map[string]any{
		"provider":      best.Provider,
		"detectedAt":    best.UpdatedAt.UTC().Format(time.RFC3339),
		"fullReportUnavailable": false,
	}
	if best.SimilarityPct != nil {
		summary["similarityPct"] = *best.SimilarityPct
	} else {
		summary["similarityPct"] = nil
	}
	if best.AIProbability != nil {
		summary["aiProbability"] = *best.AIProbability
	} else {
		summary["aiProbability"] = nil
	}
	if best.ReportURL == nil || strings.TrimSpace(*best.ReportURL) == "" {
		summary["fullReportUnavailable"] = true
		summary["fullReportUnavailableMessage"] = "Full provider report link is unavailable."
	}
	return summary
}

// studentMayViewOriginality reports whether the submitting student may view originality reports.
func studentMayViewOriginality(visibility string, gradePosted bool) bool {
	switch visibility {
	case "show":
		return true
	case "show_after_grading":
		return gradePosted
	default:
		return false
	}
}

func (d Deps) canViewOriginality(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID, sc *originalityreports.SubmissionContext) bool {
	if sc == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Submission not found.")
		return false
	}
	if sc.OriginalityMode == "disabled" || !sc.PlagiarismEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	if sc.SubmittedBy == viewer {
		var gradePosted bool
		if sc.StudentVisibility == "show_after_grading" {
			cell, err := coursegrades.GetCell(r.Context(), d.Pool, sc.CourseID, sc.SubmittedBy, sc.ModuleItemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify grade status.")
				return false
			}
			gradePosted = cell != nil && cell.PostedAt != nil
		}
		if studentMayViewOriginality(sc.StudentVisibility, gradePosted) {
			return true
		}
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
		return false
	}
	ok, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !ok {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
		return false
	}
	return true
}

func (d Deps) parseSubmissionID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "submission_id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid submission id.")
		return uuid.Nil, false
	}
	return id, true
}

// handleGetSubmissionOriginality is GET .../submissions/{submission_id}/originality
func (d Deps) handleGetSubmissionOriginality() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requirePlagiarismWorkflow(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		submissionID, ok := d.parseSubmissionID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		sc, err := originalityreports.GetSubmissionContext(r.Context(), d.Pool, courseCode, submissionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submission.")
			return
		}
		if !d.canViewOriginality(w, r, courseCode, viewer, sc) {
			return
		}
		reports, err := originalityreports.ListBySubmission(r.Context(), d.Pool, submissionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load originality reports.")
			return
		}
		items := make([]map[string]any, 0, len(reports))
		for _, rep := range reports {
			items = append(items, reportToJSON(rep))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"reports": items})
	}
}

// handleGetSubmissionOriginalityEmbed is GET .../originality/embed-url
func (d Deps) handleGetSubmissionOriginalityEmbed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requirePlagiarismWorkflow(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		submissionID, ok := d.parseSubmissionID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		sc, err := originalityreports.GetSubmissionContext(r.Context(), d.Pool, courseCode, submissionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submission.")
			return
		}
		if !d.canViewOriginality(w, r, courseCode, viewer, sc) {
			return
		}
		reports, err := originalityreports.ListBySubmission(r.Context(), d.Pool, submissionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load originality reports.")
			return
		}
		var embedURL *string
		for _, rep := range reports {
			if rep.ReportURL != nil && strings.TrimSpace(*rep.ReportURL) != "" {
				u := strings.TrimSpace(*rep.ReportURL)
				embedURL = &u
				break
			}
		}
		out := map[string]any{"summary": summaryFromReports(reports)}
		if embedURL != nil {
			out["embedUrl"] = *embedURL
		} else {
			out["embedUrl"] = nil
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleGetSubmissionOriginalitySummary is GET .../originality/summary
func (d Deps) handleGetSubmissionOriginalitySummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requirePlagiarismWorkflow(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		submissionID, ok := d.parseSubmissionID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		sc, err := originalityreports.GetSubmissionContext(r.Context(), d.Pool, courseCode, submissionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submission.")
			return
		}
		if !d.canViewOriginality(w, r, courseCode, viewer, sc) {
			return
		}
		reports, err := originalityreports.ListBySubmission(r.Context(), d.Pool, submissionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load originality reports.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"summary": summaryFromReports(reports)})
	}
}

// handlePostSubmissionOriginalityRetry is POST .../originality/retry
func (d Deps) handlePostSubmissionOriginalityRetry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requirePlagiarismWorkflow(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		submissionID, ok := d.parseSubmissionID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		canGrade, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canGrade {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		orgID := d.orgIDPtrForUser(r.Context(), viewer)
		n, err := d.plagiarismService(orgID).RetryFailed(r.Context(), submissionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to retry originality scan.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"retried": n})
	}
}
