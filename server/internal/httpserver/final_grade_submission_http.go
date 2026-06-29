package httpserver

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/finalgradesub"
	"github.com/lextures/lextures/server/internal/service/gradeexport"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// handleFinalGradesPreview is GET /api/v1/courses/{course_code}/final-grades/preview.
func (d Deps) handleFinalGradesPreview() http.HandlerFunc {
	type studentGradeOut struct {
		EnrollmentID     string  `json:"enrollmentId"`
		UserID           string  `json:"userId"`
		DisplayName      string  `json:"displayName"`
		ExternalSISID    string  `json:"externalSisId,omitempty"`
		State            string  `json:"state"`
		ComputedGrade    string  `json:"computedGrade"`
		FinalGrade       string  `json:"finalGrade"`
		OverrideReason   string  `json:"overrideReason,omitempty"`
		AlreadySubmitted bool    `json:"alreadySubmitted"`
		SubmittedAt      *string `json:"submittedAt,omitempty"`
	}
	type previewResp struct {
		Grades []studentGradeOut `json:"grades"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		cfg := d.effectiveConfig()
		if !cfg.FFGradeSubmission {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Final grade submission is not enabled.")
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canGrade, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil || !canGrade {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view final grades.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID := *cid

		computed, err := gradeexport.ComputeForCourse(r.Context(), d.Pool, courseID, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute final grades.")
			return
		}

		existing, err := finalgradesub.LatestByCourse(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load prior submissions.")
			return
		}
		latestByEnrollment := make(map[uuid.UUID]finalgradesub.Submission, len(existing))
		for _, s := range existing {
			latestByEnrollment[s.EnrollmentID] = s
		}

		out := make([]studentGradeOut, 0, len(computed))
		for _, g := range computed {
			row := studentGradeOut{
				EnrollmentID:  g.EnrollmentID.String(),
				UserID:        g.UserID.String(),
				DisplayName:   g.DisplayName,
				ExternalSISID: g.ExternalSISID,
				State:         g.State,
				ComputedGrade: g.ComputedGrade,
				FinalGrade:    g.FinalGrade,
			}
			if prior, ok := latestByEnrollment[g.EnrollmentID]; ok {
				row.AlreadySubmitted = true
				ts := prior.SubmittedAt.UTC().Format(time.RFC3339)
				row.SubmittedAt = &ts
				row.FinalGrade = prior.FinalGrade
				if prior.OverrideReason != nil {
					row.OverrideReason = *prior.OverrideReason
				}
			}
			out = append(out, row)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(previewResp{Grades: out})
	}
}

// handleFinalGradesSubmit is POST /api/v1/courses/{course_code}/final-grades/submit.
func (d Deps) handleFinalGradesSubmit() http.HandlerFunc {
	type overrideIn struct {
		EnrollmentID string `json:"enrollmentId"`
		Grade        string `json:"grade"`
		Reason       string `json:"reason"`
	}
	type submitBody struct {
		Method    string       `json:"method"`
		Overrides []overrideIn `json:"overrides"`
	}
	type submitResp struct {
		Count       int    `json:"count"`
		DownloadURL string `json:"downloadUrl,omitempty"`
		AGSStatus   string `json:"agsStatus,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		cfg := d.effectiveConfig()
		if !cfg.FFGradeSubmission {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Final grade submission is not enabled.")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canGrade, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil || !canGrade {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to submit final grades.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID := *cid

		var body submitBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		method := strings.TrimSpace(body.Method)
		if method == "" {
			method = "csv"
		}
		if method != "csv" && method != "ags" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "method must be 'csv' or 'ags'.")
			return
		}

		overrideMap := make(map[uuid.UUID]overrideIn, len(body.Overrides))
		for _, ov := range body.Overrides {
			eid, err := uuid.Parse(strings.TrimSpace(ov.EnrollmentID))
			if err != nil {
				continue
			}
			if strings.TrimSpace(ov.Grade) == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Override grade must not be empty.")
				return
			}
			overrideMap[eid] = ov
		}

		computed, err := gradeexport.ComputeForCourse(r.Context(), d.Pool, courseID, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute final grades.")
			return
		}

		rows := make([]finalgradesub.Submission, 0, len(computed))
		for _, g := range computed {
			finalGrade := g.ComputedGrade
			var overrideReason *string
			if ov, ok := overrideMap[g.EnrollmentID]; ok {
				finalGrade = strings.TrimSpace(ov.Grade)
				r := strings.TrimSpace(ov.Reason)
				overrideReason = &r
			}
			rows = append(rows, finalgradesub.Submission{
				CourseID:         courseID,
				EnrollmentID:     g.EnrollmentID,
				SubmittedBy:      viewer,
				ComputedGrade:    g.ComputedGrade,
				FinalGrade:       finalGrade,
				OverrideReason:   overrideReason,
				SubmissionMethod: method,
			})
		}
		if err := finalgradesub.BulkCreate(r.Context(), d.Pool, rows); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save grade submissions.")
			return
		}
		for range rows {
			telemetry.RecordBusinessEvent("grade_submitted") // plan 17.7 FR-5e
		}

		resp := submitResp{Count: len(rows)}
		if method == "csv" {
			resp.DownloadURL = fmt.Sprintf("/api/v1/courses/%s/final-grades/export.csv", courseCode)
		} else {
			resp.AGSStatus = "submitted"
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// handleFinalGradesExportCSV is GET /api/v1/courses/{course_code}/final-grades/export.csv.
func (d Deps) handleFinalGradesExportCSV() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := d.effectiveConfig()
		if !cfg.FFGradeSubmission {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Final grade submission is not enabled.")
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canGrade, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil || !canGrade {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to export final grades.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID := *cid

		existing, err := finalgradesub.LatestByCourse(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grade submissions.")
			return
		}

		computed, err := gradeexport.ComputeForCourse(r.Context(), d.Pool, courseID, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute grades for export.")
			return
		}

		// Apply submitted final grades (overrides included).
		latestByEnrollment := make(map[uuid.UUID]finalgradesub.Submission, len(existing))
		for _, s := range existing {
			latestByEnrollment[s.EnrollmentID] = s
		}
		for i := range computed {
			if prior, ok := latestByEnrollment[computed[i].EnrollmentID]; ok {
				computed[i].FinalGrade = prior.FinalGrade
			}
		}

		filename := fmt.Sprintf("final_grades_%s_%s.csv", courseCode, time.Now().UTC().Format("20060102"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))

		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"StudentID", "StudentName", "FinalGrade", "EnrollmentState"})
		for _, g := range computed {
			sid := g.ExternalSISID
			if sid == "" {
				sid = g.UserID.String()
			}
			_ = cw.Write([]string{sid, g.DisplayName, g.FinalGrade, g.State})
		}
		cw.Flush()
	}
}

// handleAdminFinalGradesStatus is GET /api/v1/admin/final-grades/status.
func (d Deps) handleAdminFinalGradesStatus() http.HandlerFunc {
	type courseStatusOut struct {
		CourseID        string  `json:"courseId"`
		CourseCode      string  `json:"courseCode"`
		CourseTitle     string  `json:"courseTitle"`
		InstructorName  string  `json:"instructorName"`
		TotalStudents   int     `json:"totalStudents"`
		SubmittedCount  int     `json:"submittedCount"`
		AllSubmitted    bool    `json:"allSubmitted"`
		LastSubmittedAt *string `json:"lastSubmittedAt,omitempty"`
	}
	type statusResp struct {
		Courses []courseStatusOut `json:"courses"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		cfg := d.effectiveConfig()
		if !cfg.FFGradeSubmission {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Final grade submission is not enabled.")
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		termIDStr := strings.TrimSpace(r.URL.Query().Get("term_id"))
		if termIDStr == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "term_id is required.")
			return
		}
		termID, err := uuid.Parse(termIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "term_id must be a valid UUID.")
			return
		}

		statuses, err := finalgradesub.ListStatusByTerm(r.Context(), d.Pool, termID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grade submission status.")
			return
		}

		out := make([]courseStatusOut, 0, len(statuses))
		for _, s := range statuses {
			row := courseStatusOut{
				CourseID:       s.CourseID.String(),
				CourseCode:     s.CourseCode,
				CourseTitle:    s.CourseTitle,
				InstructorName: s.InstructorName,
				TotalStudents:  s.TotalStudents,
				SubmittedCount: s.SubmittedCount,
				AllSubmitted:   s.TotalStudents > 0 && s.SubmittedCount >= s.TotalStudents,
			}
			if s.SubmittedAt != nil {
				ts := s.SubmittedAt.UTC().Format(time.RFC3339)
				row.LastSubmittedAt = &ts
			}
			out = append(out, row)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(statusResp{Courses: out})
	}
}

func (d Deps) registerFinalGradeRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/final-grades/preview", d.handleFinalGradesPreview())
	r.Post("/api/v1/courses/{course_code}/final-grades/submit", d.handleFinalGradesSubmit())
	r.Get("/api/v1/courses/{course_code}/final-grades/export.csv", d.handleFinalGradesExportCSV())
	r.Get("/api/v1/admin/final-grades/status", d.handleAdminFinalGradesStatus())
}
