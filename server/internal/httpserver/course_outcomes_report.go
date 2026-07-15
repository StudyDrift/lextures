package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	outrep "github.com/lextures/lextures/server/internal/repos/outcomesreport"
	badgesvc "github.com/lextures/lextures/server/internal/service/badges"
)

func (d Deps) outcomesReportEnabled() bool {
	return d.effectiveConfig().OutcomesReportEnabled
}

// maybeAutoAwardBadgesAfterOutcomesRefresh awards auto_award definitions for students who met mastery (plan B1).
func (d Deps) maybeAutoAwardBadgesAfterOutcomesRefresh(ctx context.Context, courseID uuid.UUID) {
	cfg := d.effectiveConfig()
	if !cfg.FFCompetencyBadges || d.Pool == nil {
		return
	}
	rows, err := d.Pool.Query(ctx, `
SELECT DISTINCT user_id, outcome_id
FROM analytics.outcomes_report_student
WHERE course_id = $1 AND assessed = TRUE AND met = TRUE
`, courseID)
	if err != nil {
		slog.Warn("badges.auto_award.list_failed", "course_id", courseID.String(), "err", err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var userID, outcomeID uuid.UUID
		if err := rows.Scan(&userID, &outcomeID); err != nil {
			continue
		}
		if err := badgesvc.MaybeAutoAward(ctx, d.Pool, cfg, courseID, userID, outcomeID); err != nil {
			slog.Warn("badges.auto_award.failed",
				"course_id", courseID.String(),
				"user_id", userID.String(),
				"outcome_id", outcomeID.String(),
				"err", err.Error(),
			)
		}
	}
}

func (d Deps) guardOutcomesReport(w http.ResponseWriter) bool {
	if !d.outcomesReportEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	return true
}

func (d Deps) requireCourseStaffForOutcomesReport(w http.ResponseWriter, r *http.Request) (courseCode string, userID uuid.UUID, ok bool) {
	userID, ok = d.meUserID(w, r)
	if !ok {
		return "", uuid.Nil, false
	}
	courseCode = chi.URLParam(r, "course_code")
	ctx := r.Context()
	isStaff, err := enrollment.UserIsCourseStaff(ctx, d.Pool, courseCode, userID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check course access.")
		return "", uuid.Nil, false
	}
	if !isStaff {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
		return "", uuid.Nil, false
	}
	return courseCode, userID, true
}

type outcomesReportOutcomeJSON struct {
	OutcomeID       string   `json:"outcomeId"`
	Title           string   `json:"title"`
	SortOrder       int      `json:"sortOrder"`
	NStudents       int      `json:"nStudents"`
	NAssessed       int      `json:"nAssessed"`
	MeanScore       *float64 `json:"meanScore"`
	PctMet          float64  `json:"pctMet"`
	PctNotMet       float64  `json:"pctNotMet"`
	Threshold       float64  `json:"threshold"`
	AlignmentCount  int      `json:"alignmentCount"`
	ImprovementNote string   `json:"improvementNote"`
	NoAlignments    bool     `json:"noAlignments"`
}

type outcomesReportJSON struct {
	CourseID         string                      `json:"courseId"`
	MasteryThreshold float64                     `json:"masteryThreshold"`
	DataAsOf         string                      `json:"dataAsOf"`
	StaleMinutes     int                         `json:"staleMinutes"`
	Outcomes         []outcomesReportOutcomeJSON `json:"outcomes"`
}

func outcomeRowToJSON(row outrep.OutcomeRow) outcomesReportOutcomeJSON {
	var mean *float64
	if row.MeanScore != nil {
		v := float64(*row.MeanScore)
		mean = &v
	}
	return outcomesReportOutcomeJSON{
		OutcomeID:       row.OutcomeID.String(),
		Title:           row.Title,
		SortOrder:       int(row.SortOrder),
		NStudents:       row.NStudents,
		NAssessed:       row.NAssessed,
		MeanScore:       mean,
		PctMet:          row.PctMet,
		PctNotMet:       row.PctNotMet,
		Threshold:       float64(row.Threshold),
		AlignmentCount:  row.AlignmentCount,
		ImprovementNote: row.ImprovementNote,
		NoAlignments:    row.NoAlignments,
	}
}

func parseOutcomesReportFilters(r *http.Request) (sectionID, groupID *uuid.UUID, err error) {
	if s := strings.TrimSpace(r.URL.Query().Get("sectionId")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return nil, nil, e
		}
		sectionID = &id
	}
	if g := strings.TrimSpace(r.URL.Query().Get("groupId")); g != "" {
		id, e := uuid.Parse(g)
		if e != nil {
			return nil, nil, e
		}
		groupID = &id
	}
	return sectionID, groupID, nil
}

// handleCourseOutcomesAnalyticsGet is GET /api/v1/courses/{course_code}/analytics/outcomes.
func (d Deps) handleCourseOutcomesAnalyticsGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.guardOutcomesReport(w) {
			return
		}
		courseCode, userID, ok := d.requireCourseStaffForOutcomesReport(w, r)
		if !ok {
			return
		}
		sectionID, groupID, err := parseOutcomesReportFilters(r)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid sectionId or groupId.")
			return
		}

		ctx := r.Context()
		crow, err := course.GetPublicByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || crow == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID, err := uuid.Parse(crow.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}

		go func() {
			bg := contextWithoutCancel(ctx)
			if refreshed, err := outrep.RefreshViewIfStale(bg, d.Pool); err != nil {
				slog.Warn("outcomes_report.refresh_failed", "err", err, "course_id", courseID)
			} else if refreshed {
				slog.Info("outcomes_report.refresh", "course_id", courseID, "trigger", "stale")
			}
		}()

		// Ensure this course has been computed at least once.
		var hasRows bool
		_ = d.Pool.QueryRow(ctx, `
SELECT EXISTS(SELECT 1 FROM analytics.outcomes_report_student WHERE course_id = $1 LIMIT 1)
`, courseID).Scan(&hasRows)
		if !hasRows {
			if err := outrep.RefreshCourseNow(ctx, d.Pool, courseID); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load outcomes report.")
				return
			}
		}

		report, err := outrep.ReportForCourse(ctx, d.Pool, courseID, sectionID, groupID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load outcomes report.")
			return
		}

		slog.Info("outcomes_report.view",
			"course_id", courseID.String(),
			"outcome_count", len(report.Outcomes),
			"viewer_id", userID.String(),
		)

		out := make([]outcomesReportOutcomeJSON, 0, len(report.Outcomes))
		for _, row := range report.Outcomes {
			out = append(out, outcomeRowToJSON(row))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(outcomesReportJSON{
			CourseID:         courseID.String(),
			MasteryThreshold: float64(report.MasteryThreshold),
			DataAsOf:         report.DataAsOf.UTC().Format(time.RFC3339),
			StaleMinutes:     report.StaleMinutes,
			Outcomes:         out,
		})
	}
}

type outcomesReportSettingsBody struct {
	MasteryThreshold *float64 `json:"masteryThreshold"`
}

// handleCourseOutcomesAnalyticsSettingsPut is PUT /api/v1/courses/{course_code}/analytics/outcomes/settings.
func (d Deps) handleCourseOutcomesAnalyticsSettingsPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.guardOutcomesReport(w) {
			return
		}
		courseCode, _, ok := d.requireCourseStaffForOutcomesReport(w, r)
		if !ok {
			return
		}
		var body outcomesReportSettingsBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.MasteryThreshold == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "masteryThreshold is required.")
			return
		}
		th := float32(*body.MasteryThreshold)
		if th <= 0 || th > 100 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "masteryThreshold must be between 0 and 100.")
			return
		}
		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if err := outrep.SetMasteryThreshold(ctx, d.Pool, *cid, th); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save threshold.")
			return
		}
		go func() {
			bg := contextWithoutCancel(ctx)
			if err := outrep.RefreshCourseNow(bg, d.Pool, *cid); err != nil {
				slog.Warn("outcomes_report.refresh_failed", "err", err)
			}
		}()
		w.WriteHeader(http.StatusNoContent)
	}
}

type outcomeImprovementNoteBody struct {
	NoteText string `json:"noteText"`
}

// handleCourseOutcomeImprovementNotePut is PUT /api/v1/courses/{course_code}/outcomes/{outcome_id}/notes.
func (d Deps) handleCourseOutcomeImprovementNotePut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.guardOutcomesReport(w) {
			return
		}
		courseCode, _, ok := d.requireCourseStaffForOutcomesReport(w, r)
		if !ok {
			return
		}
		outcomeID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "outcome_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid outcome id.")
			return
		}
		var body outcomeImprovementNoteBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		text := strings.TrimSpace(body.NoteText)
		if len(text) > 20000 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Note is too long.")
			return
		}
		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		outcomes, err := courseoutcomes.ListOutcomes(ctx, d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify outcome.")
			return
		}
		found := false
		for _, o := range outcomes {
			if o.ID == outcomeID {
				found = true
				break
			}
		}
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Outcome not found.")
			return
		}
		if err := outrep.UpsertImprovementNote(ctx, d.Pool, outcomeID, text); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save note.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"outcomeId": outcomeID.String(),
			"noteText":  text,
		})
	}
}

// handleCourseOutcomesAnalyticsRefreshPost is POST /api/v1/courses/{course_code}/analytics/outcomes/refresh.
func (d Deps) handleCourseOutcomesAnalyticsRefreshPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.guardOutcomesReport(w) {
			return
		}
		courseCode, _, ok := d.requireCourseStaffForOutcomesReport(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		start := time.Now()
		if err := outrep.RefreshCourseNow(ctx, d.Pool, *cid); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to refresh outcomes report.")
			return
		}
		d.maybeAutoAwardBadgesAfterOutcomesRefresh(ctx, *cid)
		slog.Info("outcomes_report.refresh",
			"course_id", cid.String(),
			"duration_ms", time.Since(start).Milliseconds(),
			"trigger", "manual",
		)
		meta, _ := outrep.GetRefreshMeta(ctx, d.Pool)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"refreshedAt": meta.RefreshedAt.UTC().Format(time.RFC3339),
		})
	}
}
