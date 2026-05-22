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
	"github.com/lextures/lextures/server/internal/repos/reportschedules"
	"github.com/lextures/lextures/server/internal/service/reportpdf"
)

func (d Deps) reportExportFeatureEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().ReportExportEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Report export is not enabled.")
		return false
	}
	return true
}

// handleExportCoursePDF is POST /api/v1/courses/{course_code}/reports/{report_type}/export
// Synchronously generates a PDF and streams it back as an attachment.
func (d Deps) handleExportCoursePDF() http.HandlerFunc {
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
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.reportExportFeatureEnabled(w) {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		reportType := chi.URLParam(r, "report_type")
		ctx := r.Context()

		has, err := courseroles.UserHasPermission(ctx, d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check permissions.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to export reports for this course.")
			return
		}

		now := time.Now().UTC()
		var pdfBytes []byte
		switch reportType {
		case "gradebook":
			pdfBytes, err = reportpdf.BuildGradebookPDF(reportpdf.GradebookInput{
				CourseName:  r.URL.Query().Get("course_name"),
				CourseCode:  courseCode,
				GeneratedAt: now,
			})
		case "progress":
			pdfBytes, err = reportpdf.BuildProgressPDF(reportpdf.ProgressInput{
				CourseName:  r.URL.Query().Get("course_name"),
				CourseCode:  courseCode,
				StudentName: r.URL.Query().Get("student_name"),
				GeneratedAt: now,
			})
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				fmt.Sprintf("Unknown report type %q. Valid types: gradebook, progress.", reportType))
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate PDF report.")
			return
		}

		filename := fmt.Sprintf("%s-%s-%s.pdf", reportType, courseCode, now.Format("20060102"))
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
		_, _ = w.Write(pdfBytes)
	}
}

// handleExportLearningActivityPDF is GET /api/v1/reports/learning-activity/export.pdf
func (d Deps) handleExportLearningActivityPDF() http.HandlerFunc {
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
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if !d.reportExportFeatureEnabled(w) {
			return
		}
		now := time.Now().UTC()
		pdfBytes, err := reportpdf.BuildLearningActivityPDF(reportpdf.LearningActivityInput{
			GeneratedAt: now,
			From:        now.AddDate(0, 0, -30),
			To:          now,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate PDF report.")
			return
		}
		filename := fmt.Sprintf("learning-activity-%s.pdf", now.Format("20060102"))
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
		_, _ = w.Write(pdfBytes)
	}
}

// scheduleJSON is the API representation of a report schedule.
type scheduleJSON struct {
	ID            string            `json:"id"`
	ReportType    string            `json:"reportType"`
	CourseID      *string           `json:"courseId,omitempty"`
	Parameters    map[string]string `json:"parameters"`
	Recipients    []string          `json:"recipients"`
	Cadence       string            `json:"cadence"`
	CadenceDetail map[string]any    `json:"cadenceDetail,omitempty"`
	Enabled       bool              `json:"enabled"`
	LastRunAt     *string           `json:"lastRunAt,omitempty"`
	NextRunAt     string            `json:"nextRunAt"`
	CreatedAt     string            `json:"createdAt"`
}

func toScheduleJSON(s reportschedules.Schedule) scheduleJSON {
	j := scheduleJSON{
		ID:            s.ID.String(),
		ReportType:    s.ReportType,
		Parameters:    s.Parameters,
		Recipients:    s.Recipients,
		Cadence:       s.Cadence,
		CadenceDetail: s.CadenceDetail,
		Enabled:       s.Enabled,
		NextRunAt:     s.NextRunAt.UTC().Format(time.RFC3339),
		CreatedAt:     s.CreatedAt.UTC().Format(time.RFC3339),
	}
	if s.CourseID != nil {
		cid := s.CourseID.String()
		j.CourseID = &cid
	}
	if s.LastRunAt != nil {
		t := s.LastRunAt.UTC().Format(time.RFC3339)
		j.LastRunAt = &t
	}
	if j.Parameters == nil {
		j.Parameters = map[string]string{}
	}
	if j.Recipients == nil {
		j.Recipients = []string{}
	}
	return j
}

// handleListReportSchedules is GET /api/v1/reports/schedules
func (d Deps) handleListReportSchedules() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.reportExportFeatureEnabled(w) {
			return
		}
		list, err := reportschedules.List(r.Context(), d.Pool, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list schedules.")
			return
		}
		out := make([]scheduleJSON, 0, len(list))
		for _, s := range list {
			out = append(out, toScheduleJSON(s))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

type createScheduleBody struct {
	ReportType    string            `json:"reportType"`
	CourseID      *string           `json:"courseId"`
	Parameters    map[string]string `json:"parameters"`
	Recipients    []string          `json:"recipients"`
	Cadence       string            `json:"cadence"`
	CadenceDetail map[string]any    `json:"cadenceDetail"`
}

// handleCreateReportSchedule is POST /api/v1/reports/schedules
func (d Deps) handleCreateReportSchedule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.reportExportFeatureEnabled(w) {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body createScheduleBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := validateScheduleInput(body.ReportType, body.Cadence, body.Recipients); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		sched := reportschedules.Schedule{
			OwnerID:       viewer,
			ReportType:    body.ReportType,
			Parameters:    body.Parameters,
			Recipients:    body.Recipients,
			Cadence:       body.Cadence,
			CadenceDetail: body.CadenceDetail,
			Enabled:       true,
			NextRunAt:     background.NextRunAt(body.Cadence, time.Now().UTC()),
		}
		if body.CourseID != nil {
			cid, err := uuid.Parse(*body.CourseID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
				return
			}
			sched.CourseID = &cid
		}
		created, err := reportschedules.Create(r.Context(), d.Pool, sched)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create schedule.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(toScheduleJSON(created))
	}
}

type updateScheduleBody struct {
	Recipients    []string       `json:"recipients"`
	Cadence       string         `json:"cadence"`
	CadenceDetail map[string]any `json:"cadenceDetail"`
	Enabled       *bool          `json:"enabled"`
}

// handleUpdateReportSchedule is PUT /api/v1/reports/schedules/{id}
func (d Deps) handleUpdateReportSchedule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.reportExportFeatureEnabled(w) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid schedule ID.")
			return
		}
		existing, err := reportschedules.Get(r.Context(), d.Pool, id)
		if err != nil || existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Schedule not found.")
			return
		}
		if existing.OwnerID != viewer {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not own this schedule.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body updateScheduleBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cadence := existing.Cadence
		if strings.TrimSpace(body.Cadence) != "" {
			cadence = body.Cadence
		}
		recipients := existing.Recipients
		if len(body.Recipients) > 0 {
			recipients = body.Recipients
		}
		enabled := existing.Enabled
		if body.Enabled != nil {
			enabled = *body.Enabled
		}
		detail := existing.CadenceDetail
		if body.CadenceDetail != nil {
			detail = body.CadenceDetail
		}
		in := reportschedules.UpdateInput{
			Recipients:    recipients,
			Cadence:       cadence,
			CadenceDetail: detail,
			Enabled:       enabled,
			NextRunAt:     background.NextRunAt(cadence, time.Now().UTC()),
		}
		if err := reportschedules.Update(r.Context(), d.Pool, id, in); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update schedule.")
			return
		}
		updated, err := reportschedules.Get(r.Context(), d.Pool, id)
		if err != nil || updated == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reload schedule.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(toScheduleJSON(*updated))
	}
}

// handleDeleteReportSchedule is DELETE /api/v1/reports/schedules/{id}
func (d Deps) handleDeleteReportSchedule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.reportExportFeatureEnabled(w) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid schedule ID.")
			return
		}
		existing, err := reportschedules.Get(r.Context(), d.Pool, id)
		if err != nil || existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Schedule not found.")
			return
		}
		if existing.OwnerID != viewer {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not own this schedule.")
			return
		}
		if err := reportschedules.Delete(r.Context(), d.Pool, id); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete schedule.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

var validCadences = map[string]bool{"daily": true, "weekly": true, "monthly": true}

func validateScheduleInput(reportType, cadence string, recipients []string) error {
	if strings.TrimSpace(reportType) == "" {
		return fmt.Errorf("reportType is required")
	}
	if !validCadences[strings.ToLower(cadence)] {
		return fmt.Errorf("cadence must be one of: daily, weekly, monthly")
	}
	if len(recipients) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if len(recipients) > 20 {
		return fmt.Errorf("maximum 20 recipients per schedule")
	}
	return nil
}
