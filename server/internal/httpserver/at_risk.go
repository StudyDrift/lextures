package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/background"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/atrisk"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/service/atriskscoring"
)

func (d Deps) atRiskFeatureEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().AtRiskAlertsEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "At-risk alerts are not enabled.")
		return false
	}
	return true
}

func (d Deps) requireAtRiskInstructor(w http.ResponseWriter, r *http.Request) (string, uuid.UUID, uuid.UUID, bool) {
	courseCode, viewer, ok := d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view at-risk alerts.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	return courseCode, viewer, *cid, true
}

type atRiskAlertJSON struct {
	ID            string   `json:"id"`
	EnrollmentID  string   `json:"enrollmentId"`
	UserID        string   `json:"userId"`
	DisplayName   string   `json:"displayName"`
	Score         float32  `json:"score"`
	Status        string   `json:"status"`
	TopFactor     string   `json:"topFactor"`
	TopFactorLabel string  `json:"topFactorLabel"`
	SnoozeUntil   *string  `json:"snoozeUntil,omitempty"`
	Notes         *string  `json:"notes,omitempty"`
	TriggeredDate string   `json:"triggeredDate"`
	MissingPct    *float32 `json:"missingPct,omitempty"`
	QuizAvg       *float32 `json:"quizAvg,omitempty"`
	DaysInactive  *int     `json:"daysInactive,omitempty"`
}

func alertToJSON(r atrisk.AlertRow) atRiskAlertJSON {
	name := "Student"
	if r.DisplayName != nil && *r.DisplayName != "" {
		name = *r.DisplayName
	}
	out := atRiskAlertJSON{
		ID:             r.ID.String(),
		EnrollmentID:   r.EnrollmentID.String(),
		UserID:         r.UserID.String(),
		DisplayName:    name,
		Score:          r.Score,
		Status:         string(r.Status),
		TopFactor:      r.TopFactor,
		TopFactorLabel: topFactorLabel(r.TopFactor, r.MissingPct, r.DaysInactive),
		Notes:          r.Notes,
		TriggeredDate:  r.TriggeredDate.Format("2006-01-02"),
		MissingPct:     r.MissingPct,
		QuizAvg:        r.QuizAvg,
		DaysInactive:   r.DaysInactive,
	}
	if r.SnoozeUntil != nil {
		s := r.SnoozeUntil.Format("2006-01-02")
		out.SnoozeUntil = &s
	}
	return out
}

func topFactorLabel(key string, missing *float32, inactive *int) string {
	switch key {
	case "quiz":
		return "Failing quiz average"
	case "inactive":
		if inactive != nil {
			return fmt.Sprintf("Inactive %dd", *inactive)
		}
		return "Inactive 7+ days"
	case "trend":
		return "Declining grades"
	default:
		if missing != nil && *missing > 0 {
			return fmt.Sprintf("%.0f%% missing", *missing)
		}
		return "Missing work"
	}
}

// handleCourseAtRiskList is GET /api/v1/courses/{course_code}/at-risk
func (d Deps) handleCourseAtRiskList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.atRiskFeatureEnabled(w) {
			return
		}
		_, _, courseID, ok := d.requireAtRiskInstructor(w, r)
		if !ok {
			return
		}
		includeResolved := r.URL.Query().Get("includeResolved") == "true"
		rows, err := atrisk.ListActiveForCourse(r.Context(), d.Pool, courseID, includeResolved)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load at-risk alerts.")
			return
		}
		active := make([]atRiskAlertJSON, 0)
		resolved := make([]atRiskAlertJSON, 0)
		for _, row := range rows {
			j := alertToJSON(row)
			switch row.Status {
			case atrisk.AlertResolved, atrisk.AlertDismissed:
				resolved = append(resolved, j)
			default:
				if row.Status == atrisk.AlertSnoozed && row.SnoozeUntil != nil && row.SnoozeUntil.After(time.Now().UTC()) {
					resolved = append(resolved, j)
				} else {
					active = append(active, j)
				}
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"alerts":   active,
			"resolved": resolved,
		})
	}
}

// handleCourseAtRiskPatch is PATCH /api/v1/courses/{course_code}/at-risk/{alert_id}
func (d Deps) handleCourseAtRiskPatch() http.HandlerFunc {
	type body struct {
		Status      *string `json:"status"`
		SnoozeDays  *int    `json:"snoozeDays"`
		Notes       *string `json:"notes"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.atRiskFeatureEnabled(w) {
			return
		}
		_, _, courseID, ok := d.requireAtRiskInstructor(w, r)
		if !ok {
			return
		}
		alertID, err := uuid.Parse(chi.URLParam(r, "alert_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid alert id.")
			return
		}
		row, err := atrisk.GetAlertByID(r.Context(), d.Pool, courseID, alertID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load alert.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Alert not found.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var req body
		if len(b) > 0 {
			if err := json.Unmarshal(b, &req); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		var status *atrisk.AlertStatus
		var snoozeUntil *time.Time
		if req.Status != nil {
			s := atrisk.AlertStatus(strings.TrimSpace(*req.Status))
			switch s {
			case atrisk.AlertDismissed, atrisk.AlertSnoozed, atrisk.AlertSupported, atrisk.AlertResolved, atrisk.AlertActive:
				status = &s
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid status.")
				return
			}
			if s == atrisk.AlertSnoozed || s == atrisk.AlertSupported {
				days := 7
				if req.SnoozeDays != nil && (*req.SnoozeDays == 7 || *req.SnoozeDays == 14) {
					days = *req.SnoozeDays
				} else if s == atrisk.AlertSupported {
					days = 14
				}
				until := time.Now().UTC().AddDate(0, 0, days)
				snoozeUntil = &until
			}
		}
		now := time.Now().UTC()
		if err := atrisk.PatchAlert(r.Context(), d.Pool, alertID, status, snoozeUntil, req.Notes, now); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update alert.")
			return
		}
		updated, err := atrisk.GetAlertByID(r.Context(), d.Pool, courseID, alertID)
		if err != nil || updated == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load alert.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(alertToJSON(*updated))
	}
}

// handleEnrollmentAtRiskHistory is GET /api/v1/courses/{course_code}/enrollments/{enrollment_id}/at-risk-history
func (d Deps) handleEnrollmentAtRiskHistory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.atRiskFeatureEnabled(w) {
			return
		}
		_, _, courseID, ok := d.requireAtRiskInstructor(w, r)
		if !ok {
			return
		}
		eid, err := uuid.Parse(chi.URLParam(r, "enrollment_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		var belongs bool
		if err := d.Pool.QueryRow(r.Context(), `
SELECT EXISTS (SELECT 1 FROM course.course_enrollments WHERE id = $1 AND course_id = $2)
`, eid, courseID).Scan(&belongs); err != nil || !belongs {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
			return
		}
		history, err := atrisk.ListHistory(r.Context(), d.Pool, eid, 90)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load history.")
			return
		}
		type scoreOut struct {
			Date         string   `json:"date"`
			Score        float32  `json:"score"`
			MissingPct   *float32 `json:"missingPct,omitempty"`
			QuizAvg      *float32 `json:"quizAvg,omitempty"`
			DaysInactive int      `json:"daysInactive"`
			TopFactor    string   `json:"topFactor"`
		}
		out := make([]scoreOut, 0, len(history))
		for _, h := range history {
			out = append(out, scoreOut{
				Date:         h.ComputedDate.Format("2006-01-02"),
				Score:        h.Score,
				MissingPct:   h.MissingPct,
				QuizAvg:      h.QuizAvg,
				DaysInactive: h.DaysInactive,
				TopFactor:    h.TopFactor,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"scores": out})
	}
}

// handleAdminAtRiskRun is POST /api/v1/admin/at-risk/run
func (d Deps) handleAdminAtRiskRun() http.HandlerFunc {
	type reqBody struct {
		CourseCode *string `json:"courseCode"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.atRiskFeatureEnabled(w) {
			return
		}
		var body reqBody
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if len(b) > 0 {
			_ = json.Unmarshal(b, &body)
		}
		day := time.Now().UTC()
		cfg := d.effectiveConfig()
		svc := atriskscoring.Service{Pool: d.Pool, Config: cfg}
		var n int
		var err error
		if body.CourseCode != nil && strings.TrimSpace(*body.CourseCode) != "" {
			cid, e := course.GetIDByCourseCode(r.Context(), d.Pool, strings.TrimSpace(*body.CourseCode))
			if e != nil || cid == nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown course.")
				return
			}
			n, err = background.RunAtRiskForCourse(r.Context(), d.Pool, cfg, *cid, day)
		} else {
			n, err = svc.RunAllCourses(r.Context(), day)
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Scoring job failed.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"enrollmentsScored": n})
	}
}

// handleAdminAtRiskConfigPut is PUT /api/v1/admin/at-risk/config
func (d Deps) handleAdminAtRiskConfigGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.atRiskFeatureEnabled(w) {
			return
		}
		instStr := strings.TrimSpace(r.URL.Query().Get("institutionId"))
		var orgID uuid.UUID
		if instStr != "" {
			parsed, err := uuid.Parse(instStr)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid institutionId.")
				return
			}
			orgID = parsed
		} else {
			err := d.Pool.QueryRow(r.Context(), `
SELECT id FROM tenant.organizations ORDER BY created_at LIMIT 1
`).Scan(&orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load organization.")
				return
			}
		}
		cfg, err := atrisk.LoadEffective(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(adminAtRiskConfigJSON(cfg))
	}
}

func (d Deps) handleAdminAtRiskConfigPut() http.HandlerFunc {
	type body struct {
		InstitutionID    *string  `json:"institutionId"`
		Threshold        *float32 `json:"threshold"`
		WeightMissing    *float32 `json:"weightMissing"`
		WeightQuiz       *float32 `json:"weightQuiz"`
		WeightInactive   *float32 `json:"weightInactive"`
		WeightTrend      *float32 `json:"weightTrend"`
		QuizAvgThreshold *float32 `json:"quizAvgThreshold"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.atRiskFeatureEnabled(w) {
			return
		}
		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var orgID uuid.UUID
		if b.InstitutionID != nil && *b.InstitutionID != "" {
			parsed, err := uuid.Parse(*b.InstitutionID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid institutionId.")
				return
			}
			orgID = parsed
		} else {
			_ = d.Pool.QueryRow(r.Context(), `SELECT id FROM tenant.organizations ORDER BY created_at LIMIT 1`).Scan(&orgID)
		}
		cfg := atrisk.DefaultConfig(orgID)
		existing, _ := atrisk.LoadEffective(r.Context(), d.Pool, orgID)
		cfg = existing
		if b.Threshold != nil {
			cfg.Threshold = *b.Threshold
		}
		if b.WeightMissing != nil {
			cfg.WeightMissing = *b.WeightMissing
		}
		if b.WeightQuiz != nil {
			cfg.WeightQuiz = *b.WeightQuiz
		}
		if b.WeightInactive != nil {
			cfg.WeightInactive = *b.WeightInactive
		}
		if b.WeightTrend != nil {
			cfg.WeightTrend = *b.WeightTrend
		}
		if b.QuizAvgThreshold != nil {
			cfg.QuizAvgThreshold = *b.QuizAvgThreshold
		}
		sum := cfg.WeightMissing + cfg.WeightQuiz + cfg.WeightInactive + cfg.WeightTrend
		if sum < 0.999 || sum > 1.001 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Weights must sum to 1.")
			return
		}
		if err := atrisk.Upsert(r.Context(), d.Pool, cfg); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(adminAtRiskConfigJSON(cfg))
	}
}

func adminAtRiskConfigJSON(cfg atrisk.Config) map[string]any {
	return map[string]any{
		"institutionId":    cfg.OrgID.String(),
		"threshold":        cfg.Threshold,
		"weightMissing":    cfg.WeightMissing,
		"weightQuiz":       cfg.WeightQuiz,
		"weightInactive":   cfg.WeightInactive,
		"weightTrend":      cfg.WeightTrend,
		"quizAvgThreshold": cfg.QuizAvgThreshold,
	}
}
