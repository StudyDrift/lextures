package httpserver

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursestandards"
	"github.com/lextures/lextures/server/internal/service/masterytranscriptpdf"
)

// handleMasteryTranscriptPDF is GET
// /api/v1/courses/{course_code}/students/{student_id}/mastery-transcript.pdf.
// The standards gradebook Transcript column downloads this file.
func (d Deps) handleMasteryTranscriptPDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "student_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		perm := "course:" + courseCode + ":gradebook:view"
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Gradebook access required.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		transcript, err := coursestandards.LoadMasteryTranscript(r.Context(), d.Pool, *cid, courseCode, studentID)
		if errors.Is(err, coursestandards.ErrStudentNotEnrolled) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Student not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load mastery transcript.")
			return
		}
		rows := make([]masterytranscriptpdf.Row, 0, len(transcript.Rows))
		for _, row := range transcript.Rows {
			code := ""
			if row.ExternalID != nil {
				code = *row.ExternalID
			}
			level := row.LevelLabel
			if strings.TrimSpace(level) == "" && row.Proficiency != nil {
				level = fmt.Sprintf("%.2f", *row.Proficiency)
			}
			rows = append(rows, masterytranscriptpdf.Row{
				Code:         code,
				Description:  row.Description,
				Level:        level,
				LastAssessed: row.LastAssessed,
			})
		}
		pdfBytes, err := masterytranscriptpdf.Build(masterytranscriptpdf.Input{
			CourseTitle: transcript.CourseTitle,
			CourseCode:  transcript.CourseCode,
			StudentName: transcript.StudentName,
			GeneratedAt: time.Now().UTC(),
			Rows:        rows,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build mastery transcript PDF.")
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="mastery-transcript.pdf"`)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(pdfBytes)
	}
}
