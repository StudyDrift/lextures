package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/reportcards"
	"github.com/lextures/lextures/server/internal/repos/userai"
	"github.com/lextures/lextures/server/internal/service/openrouter"
	"github.com/lextures/lextures/server/internal/service/reportpdf"
)

func (d Deps) reportCardsFeatureOff(w http.ResponseWriter, r *http.Request, courseCode string) bool {
	enabled, err := course.ReportCardsEnabledForCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return true
	}
	if !enabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Report cards are not enabled for this course.")
		return true
	}
	return false
}

func (d Deps) reportCardsFeatureOffForCourseID(w http.ResponseWriter, r *http.Request, courseID uuid.UUID) bool {
	enabled, err := course.ReportCardsEnabledForCourseID(r.Context(), d.Pool, courseID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return true
	}
	if !enabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Report cards are not enabled for this course.")
		return true
	}
	return false
}

func (d Deps) registerReportCardRoutes(r chi.Router) {
	// Instructor: list / manage report cards for a course+period
	r.Get("/api/v1/courses/{course_code}/report-cards/{period}", d.handleListCourseReportCards())
	// Instructor: patch a single report card (comment / status)
	r.Patch("/api/v1/report-cards/{cardId}", d.handlePatchReportCard())
	// Instructor: generate PDF for one card
	r.Post("/api/v1/report-cards/{cardId}/generate-pdf", d.handleGenerateReportCardPDF())
	// Admin: batch-release approved cards for a course+period
	r.Post("/api/v1/courses/{course_code}/report-cards/{period}/release", d.handleReleaseReportCards())
	// Authenticated user (parent/student/admin): download PDF
	r.Get("/api/v1/report-cards/{cardId}/pdf", d.handleDownloadReportCardPDF())
	// AI comment suggestion
	r.Post("/api/v1/ai/report-card-comment", d.handleAIReportCardComment())
	// Admin: comment bank CRUD
	r.Get("/api/v1/admin/orgs/{orgId}/report-cards/comment-bank", d.handleCommentBank())
	r.Post("/api/v1/admin/orgs/{orgId}/report-cards/comment-bank", d.handleCommentBank())
	r.Delete("/api/v1/admin/orgs/{orgId}/report-cards/comment-bank/{entryId}", d.handleDeleteCommentBankEntry())
	// Parent portal: list released report cards for a linked student
	r.Get("/api/v1/parent/students/{sid}/report-cards", d.handleParentReportCards())
}

// handleListCourseReportCards is GET /api/v1/courses/:course_code/report-cards/:period
func (d Deps) handleListCourseReportCards() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
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
		if d.reportCardsFeatureOff(w, r, courseCode) {
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
		period := strings.TrimSpace(chi.URLParam(r, "period"))
		if period == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "period is required.")
			return
		}
		cards, err := reportcards.ListForCoursePeriod(r.Context(), d.Pool, *cid, period)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load report cards.")
			return
		}
		out := make([]map[string]any, 0, len(cards))
		for i := range cards {
			out = append(out, reportCardToJSON(&cards[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"reportCards": out, "period": period})
	}
}

// handlePatchReportCard is PATCH /api/v1/report-cards/:cardId
func (d Deps) handlePatchReportCard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		cardID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "cardId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid card id.")
			return
		}

		var body struct {
			Comment *string `json:"comment"`
			Status  *string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.Status != nil {
			switch *body.Status {
			case "draft", "submitted", "approved", "released":
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid status value.")
				return
			}
		}

		// Verify the card exists and the actor has access (instructor of the course or org admin).
		existing, err := reportcards.GetByID(r.Context(), d.Pool, cardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load report card.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Report card not found.")
			return
		}
		if d.reportCardsFeatureOffForCourseID(w, r, existing.CourseID) {
			return
		}
		if !d.canManageReportCard(w, r, actorID, existing.CourseID) {
			return
		}

		updated, err := reportcards.PatchReportCard(r.Context(), d.Pool, cardID, body.Comment, body.Status)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update report card.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(reportCardToJSON(updated))
	}
}

// handleGenerateReportCardPDF is POST /api/v1/report-cards/:cardId/generate-pdf
func (d Deps) handleGenerateReportCardPDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		cardID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "cardId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid card id.")
			return
		}

		existing, err := reportcards.GetByID(r.Context(), d.Pool, cardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load report card.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Report card not found.")
			return
		}
		if d.reportCardsFeatureOffForCourseID(w, r, existing.CourseID) {
			return
		}
		if !d.canManageReportCard(w, r, actorID, existing.CourseID) {
			return
		}

		// Build student info for PDF.
		var studentName, studentExternalID string
		_ = d.Pool.QueryRow(r.Context(), `
SELECT COALESCE(display_name, email, ''), COALESCE(external_id, '')
FROM "user".users WHERE id = $1`, existing.StudentID).Scan(&studentName, &studentExternalID)

		var courseName, courseCode string
		_ = d.Pool.QueryRow(r.Context(), `
SELECT COALESCE(name, ''), course_code FROM course.courses WHERE id = $1`, existing.CourseID).Scan(&courseName, &courseCode)

		var orgName string
		_ = d.Pool.QueryRow(r.Context(), `
SELECT COALESCE(name, '') FROM tenant.organizations o
JOIN course.courses c ON c.org_id = o.id WHERE c.id = $1`, existing.CourseID).Scan(&orgName)

		absences := 0
		_ = d.Pool.QueryRow(r.Context(), `
SELECT COUNT(*) FROM course.attendance_records ar
JOIN course.attendance_codes ac ON ac.id = ar.code_id
JOIN course.course_sections cs ON cs.id = ar.section_id AND cs.course_id = $1
WHERE ar.student_id = $2 AND ac.category = 'absent'`,
			existing.CourseID, existing.StudentID).Scan(&absences)

		pdfIn := reportpdf.ReportCardInput{
			InstitutionName: orgName,
			CourseName:      courseName,
			CourseCode:      courseCode,
			GradingPeriod:   existing.GradingPeriod,
			GeneratedAt:     existing.UpdatedAt,
			Student: reportpdf.ReportCardStudent{
				DisplayName:   studentName,
				StudentID:     studentExternalID,
				FinalGradePct: existing.FinalGradePct,
				Absences:      absences,
			},
		}
		if existing.LetterGrade != nil {
			pdfIn.Student.LetterGrade = *existing.LetterGrade
		}
		if existing.Comment != nil {
			pdfIn.Student.Comment = *existing.Comment
		}

		pdfBytes, err := reportpdf.BuildReportCardPDF(pdfIn)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate PDF.")
			return
		}

		// Store inline as base64 data URI (v1 without object storage integration).
		pdfURL := fmt.Sprintf("/api/v1/report-cards/%s/pdf", cardID.String())
		if err := reportcards.SetPDFURL(r.Context(), d.Pool, cardID, pdfURL); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save PDF URL.")
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_%s.pdf"`, studentExternalID, existing.GradingPeriod))
		_, _ = w.Write(pdfBytes)
	}
}

// handleReleaseReportCards is POST /api/v1/courses/:course_code/report-cards/:period/release
func (d Deps) handleReleaseReportCards() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode := strings.TrimSpace(chi.URLParam(r, "course_code"))
		period := strings.TrimSpace(chi.URLParam(r, "period"))
		if period == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "period is required.")
			return
		}

		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
			return
		}
		isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
		if !isAdmin {
			hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, actorID, "course:"+courseCode+":item:create")
			if err != nil || !hasPerm {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Admin or instructor access required.")
				return
			}
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
		if d.reportCardsFeatureOff(w, r, courseCode) {
			return
		}
		released, err := reportcards.ReleaseCards(r.Context(), d.Pool, *cid, period)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to release report cards.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"released": released,
			"period":   period,
			"message":  fmt.Sprintf("%d report card(s) released.", released),
		})
	}
}

// handleDownloadReportCardPDF is GET /api/v1/report-cards/:cardId/pdf
func (d Deps) handleDownloadReportCardPDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		cardID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "cardId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid card id.")
			return
		}

		existing, err := reportcards.GetByID(r.Context(), d.Pool, cardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load report card.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Report card not found.")
			return
		}
		if d.reportCardsFeatureOffForCourseID(w, r, existing.CourseID) {
			return
		}

		// Access control: student themselves, parent linked to student, or instructor/admin of the course.
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
			return
		}
		isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
		isSelf := actorID == existing.StudentID
		isParent := false
		if !isAdmin && !isSelf {
			_ = d.Pool.QueryRow(r.Context(), `
SELECT EXISTS(
    SELECT 1 FROM "user".parent_links
    WHERE parent_id = $1 AND student_id = $2 AND status = 'active'
)`, actorID, existing.StudentID).Scan(&isParent)
		}
		isInstructor := false
		if !isAdmin && !isSelf && !isParent {
			_ = d.Pool.QueryRow(r.Context(), `
SELECT EXISTS(
    SELECT 1 FROM course.course_enrollments
    WHERE user_id = $1 AND course_id = $2 AND active
      AND role IN ('teacher', 'instructor', 'owner', 'ta')
)`, actorID, existing.CourseID).Scan(&isInstructor)
		}
		if !isAdmin && !isSelf && !isParent && !isInstructor {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have access to this report card.")
			return
		}

		// Non-admins and non-instructors can only access released cards.
		if !isAdmin && !isInstructor && existing.Status != "released" {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Report card is not yet released.")
			return
		}

		// Re-generate PDF on demand (v1: no object storage).
		var studentName, studentExternalID string
		_ = d.Pool.QueryRow(r.Context(), `
SELECT COALESCE(display_name, email, ''), COALESCE(external_id, '')
FROM "user".users WHERE id = $1`, existing.StudentID).Scan(&studentName, &studentExternalID)

		var courseName, courseCode string
		_ = d.Pool.QueryRow(r.Context(), `
SELECT COALESCE(name, ''), course_code FROM course.courses WHERE id = $1`, existing.CourseID).Scan(&courseName, &courseCode)

		var orgName string
		_ = d.Pool.QueryRow(r.Context(), `
SELECT COALESCE(o.name, '') FROM tenant.organizations o
JOIN course.courses c ON c.org_id = o.id WHERE c.id = $1`, existing.CourseID).Scan(&orgName)

		absences := 0
		_ = d.Pool.QueryRow(r.Context(), `
SELECT COUNT(*) FROM course.attendance_records ar
JOIN course.attendance_codes ac ON ac.id = ar.code_id
JOIN course.course_sections cs ON cs.id = ar.section_id AND cs.course_id = $1
WHERE ar.student_id = $2 AND ac.category = 'absent'`,
			existing.CourseID, existing.StudentID).Scan(&absences)

		pdfIn := reportpdf.ReportCardInput{
			InstitutionName: orgName,
			CourseName:      courseName,
			CourseCode:      courseCode,
			GradingPeriod:   existing.GradingPeriod,
			GeneratedAt:     existing.UpdatedAt,
			Student: reportpdf.ReportCardStudent{
				DisplayName:   studentName,
				StudentID:     studentExternalID,
				FinalGradePct: existing.FinalGradePct,
				Absences:      absences,
			},
		}
		if existing.LetterGrade != nil {
			pdfIn.Student.LetterGrade = *existing.LetterGrade
		}
		if existing.Comment != nil {
			pdfIn.Student.Comment = *existing.Comment
		}

		pdfBytes, err := reportpdf.BuildReportCardPDF(pdfIn)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate PDF.")
			return
		}

		filename := fmt.Sprintf("%s_%s.pdf", studentExternalID, existing.GradingPeriod)
		if studentExternalID == "" {
			filename = fmt.Sprintf("report_card_%s.pdf", cardID.String()[:8])
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		_, _ = w.Write(pdfBytes)
	}
}

// handleAIReportCardComment is POST /api/v1/ai/report-card-comment
func (d Deps) handleAIReportCardComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}

		var body struct {
			CourseName string  `json:"courseName"`
			GradePct   float64 `json:"gradePct"`
			Absences   int     `json:"absences"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}

		or := d.openRouterClient()
		if or == nil || d.effectiveConfig().OpenRouterAPIKey == "" {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "AI comment generation is not available.")
			return
		}

		model, err := userai.GetCourseSetupModelID(r.Context(), d.Pool, actorID)
		if err != nil {
			model = userai.DefaultCourseSetupModelID
		}

		courseName := strings.TrimSpace(body.CourseName)
		if courseName == "" {
			courseName = "this course"
		}
		prompt := fmt.Sprintf(
			"Write a 2-sentence report card comment for a student with a %.1f%% average in %s and %d absence(s). Be encouraging and specific. Do not use the student's name.",
			body.GradePct, courseName, body.Absences,
		)

		msgs := []openrouter.Message{
			{Role: "system", Content: "You are a helpful teacher assistant writing concise, professional report card comments."},
			{Role: "user", Content: prompt},
		}
		suggestion, err := or.ChatCompletion(model, msgs)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "AI comment generation failed.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"suggestion": strings.TrimSpace(suggestion.Text)})
	}
}

// handleCommentBank is GET/POST /api/v1/admin/orgs/:orgId/report-cards/comment-bank
func (d Deps) handleCommentBank() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		switch r.Method {
		case http.MethodGet:
			category := r.URL.Query().Get("category")
			entries, err := reportcards.ListCommentBank(r.Context(), d.Pool, orgID, category)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list comment bank.")
				return
			}
			out := make([]map[string]any, 0, len(entries))
			for i := range entries {
				out = append(out, commentBankEntryToJSON(&entries[i]))
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"entries": out})

		case http.MethodPost:
			var body struct {
				Category string `json:"category"`
				Text     string `json:"text"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			category := strings.TrimSpace(body.Category)
			text := strings.TrimSpace(body.Text)
			if category == "" || text == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "category and text are required.")
				return
			}
			entry, err := reportcards.UpsertCommentBankEntry(r.Context(), d.Pool, orgID, category, text)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save comment bank entry.")
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(commentBankEntryToJSON(entry))

		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleDeleteCommentBankEntry is DELETE /api/v1/admin/orgs/:orgId/report-cards/comment-bank/:entryId
func (d Deps) handleDeleteCommentBankEntry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		entryID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "entryId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid entry id.")
			return
		}
		deleted, err := reportcards.DeleteCommentBankEntry(r.Context(), d.Pool, orgID, entryID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete comment bank entry.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Entry not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleParentReportCards is GET /api/v1/parent/students/:sid/report-cards
func (d Deps) handleParentReportCards() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		studentID, ok := d.parseStudentIDParam(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireParentLink(w, r, parentID, orgID, studentID); !ok {
			return
		}

		cards, err := reportcards.ListReleasedForStudent(r.Context(), d.Pool, studentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load report cards.")
			return
		}
		out := make([]map[string]any, 0, len(cards))
		for i := range cards {
			out = append(out, reportCardToJSON(&cards[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"reportCards": out})
	}
}

// canManageReportCard checks that actorID is an instructor/admin for the given courseID.
func (d Deps) canManageReportCard(w http.ResponseWriter, r *http.Request, actorID, courseID uuid.UUID) bool {
	orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
		return false
	}
	isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
	if isAdmin {
		return true
	}
	var isInstructor bool
	_ = d.Pool.QueryRow(r.Context(), `
SELECT EXISTS(
    SELECT 1 FROM course.course_enrollments
    WHERE user_id = $1 AND course_id = $2 AND active
      AND role IN ('teacher', 'instructor', 'owner', 'ta')
)`, actorID, courseID).Scan(&isInstructor)
	if !isInstructor {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor or admin access required.")
		return false
	}
	return true
}

// ─── JSON helpers ─────────────────────────────────────────────────────────────

func reportCardToJSON(rc *reportcards.ReportCard) map[string]any {
	if rc == nil {
		return nil
	}
	m := map[string]any{
		"id":            rc.ID.String(),
		"studentId":     rc.StudentID.String(),
		"courseId":      rc.CourseID.String(),
		"gradingPeriod": rc.GradingPeriod,
		"status":        rc.Status,
		"createdAt":     rc.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		"updatedAt":     rc.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if rc.FinalGradePct != nil {
		m["finalGradePct"] = *rc.FinalGradePct
	}
	if rc.LetterGrade != nil {
		m["letterGrade"] = *rc.LetterGrade
	}
	if rc.Comment != nil {
		m["comment"] = *rc.Comment
	}
	if rc.PDFURL != nil {
		m["pdfUrl"] = *rc.PDFURL
	}
	if rc.GeneratedAt != nil {
		m["generatedAt"] = rc.GeneratedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	if rc.ReleasedAt != nil {
		m["releasedAt"] = rc.ReleasedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return m
}

func commentBankEntryToJSON(e *reportcards.CommentBankEntry) map[string]any {
	if e == nil {
		return nil
	}
	return map[string]any{
		"id":       e.ID.String(),
		"orgId":    e.OrgID.String(),
		"category": e.Category,
		"text":     e.Text,
		"active":   e.Active,
	}
}
