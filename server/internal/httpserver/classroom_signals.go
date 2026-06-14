package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/classroomsignals"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

// maxQuestionLen caps anonymous question length (plan 13.9: keeps the queue
// readable and limits spam payload size).
const maxQuestionLen = 500

// classroomSignalsEnabled writes 501 if the feature flag is off and returns
// false; on returns true.
func (d Deps) classroomSignalsEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFClassroomSignals {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented,
			"Classroom signals feature is not enabled.")
		return false
	}
	return true
}

// handlePostSectionHallPass — POST /api/v1/sections/{sectionId}/hall-passes
//
// Student in the section requests a digital pass. Body: {destination, estimatedMins}.
func (d Deps) handlePostSectionHallPass() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.classroomSignalsEnabled(w) {
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		sectionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "sectionId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid section id.")
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			Destination   string `json:"destination"`
			EstimatedMins *int   `json:"estimatedMins"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		dest := strings.TrimSpace(strings.ToLower(body.Destination))
		if !classroomsignals.IsAllowedDestination(dest) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"destination must be one of: bathroom, office, library, nurse, other.")
			return
		}
		if body.EstimatedMins != nil && (*body.EstimatedMins <= 0 || *body.EstimatedMins > 120) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"estimatedMins must be between 1 and 120.")
			return
		}
		// FR — only students enrolled in the section can submit. Use the section/course
		// enrollment check: any active enrollment in the section's course.
		if ok := d.requireStudentInSection(w, r, actorID, sectionID); !ok {
			return
		}
		pass, err := classroomsignals.CreateHallPass(r.Context(), d.Pool, actorID, sectionID, dest, body.EstimatedMins)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create hall pass.")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"pass": hallPassToJSON(pass, true)})
	}
}

// requireStudentInSection verifies actor has an active enrollment in the section's course.
// Returns false (and writes 403/500) if not.
func (d Deps) requireStudentInSection(w http.ResponseWriter, r *http.Request, actorID, sectionID uuid.UUID) bool {
	var enrolled bool
	err := d.Pool.QueryRow(r.Context(), `
SELECT EXISTS (
    SELECT 1
    FROM course.course_enrollments ce
    JOIN course.course_sections cs
        ON (ce.section_id = cs.id OR ce.course_id = cs.course_id)
    WHERE cs.id = $1 AND ce.user_id = $2 AND ce.active
)`, sectionID, actorID).Scan(&enrolled)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify enrollment.")
		return false
	}
	if !enrolled {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You are not enrolled in this section.")
		return false
	}
	return true
}

// handleGetSectionActiveHallPasses — GET /api/v1/sections/{sectionId}/hall-passes/active
//
// Teacher / admin sees currently-out students for the section.
func (d Deps) handleGetSectionActiveHallPasses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.classroomSignalsEnabled(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		sectionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "sectionId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid section id.")
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireSectionAccess(w, r, actorID, sectionID); !ok {
			return
		}
		passes, err := classroomsignals.ListActiveSectionPasses(r.Context(), d.Pool, sectionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list active hall passes.")
			return
		}
		out := make([]map[string]any, 0, len(passes))
		for i := range passes {
			out = append(out, hallPassToJSON(&passes[i], true))
		}
		writeJSON(w, http.StatusOK, map[string]any{"passes": out})
	}
}

// handlePatchHallPass — PATCH /api/v1/hall-passes/{passId}
//
// Approve/deny (teacher), or mark returned (teacher OR the student who owns it).
// Body: {status: "approved" | "denied" | "returned"}.
func (d Deps) handlePatchHallPass() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.classroomSignalsEnabled(w) {
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		passID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "passId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid pass id.")
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		newStatus := strings.TrimSpace(strings.ToLower(body.Status))
		switch newStatus {
		case classroomsignals.StatusApproved, classroomsignals.StatusDenied, classroomsignals.StatusReturned:
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"status must be approved, denied, or returned.")
			return
		}
		existing, err := classroomsignals.GetHallPass(r.Context(), d.Pool, passID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load hall pass.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Hall pass not found.")
			return
		}
		// Authorization:
		// - Students can only mark THEIR OWN pass as returned.
		// - Teachers (section access) can approve/deny/return.
		isOwner := existing.StudentID == actorID
		var approverPtr *uuid.UUID
		if newStatus == classroomsignals.StatusReturned && isOwner {
			// student "I'm back" — no teacher signature needed.
		} else {
			if _, ok := d.requireSectionAccess(w, r, actorID, existing.SectionID); !ok {
				return
			}
			if newStatus == classroomsignals.StatusApproved {
				a := actorID
				approverPtr = &a
			}
		}
		updated, err := classroomsignals.UpdateHallPassStatus(r.Context(), d.Pool, passID, newStatus, approverPtr)
		if err != nil {
			if errors.Is(err, classroomsignals.ErrInvalidTransition) {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput,
					"Hall pass is not in a state that allows this transition.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update hall pass.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"pass": hallPassToJSON(updated, true)})
	}
}

// handlePostCourseQuestion — POST /api/v1/courses/{courseId}/questions
//
// Student submits an anonymous question. Author is stored for teacher
// moderation but never echoed back to peers.
func (d Deps) handlePostCourseQuestion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.classroomSignalsEnabled(w) {
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "courseId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id.")
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			Question string `json:"question"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		q := strings.TrimSpace(body.Question)
		if q == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "question is required.")
			return
		}
		if len(q) > maxQuestionLen {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "question must be 500 characters or fewer.")
			return
		}
		hasAccess, err := enrollment.UserHasAccessByCourseID(r.Context(), d.Pool, courseID, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify enrollment.")
			return
		}
		if !hasAccess {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You are not enrolled in this course.")
			return
		}
		created, err := classroomsignals.CreateAnonymousQuestion(r.Context(), d.Pool, courseID, actorID, q)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to submit question.")
			return
		}
		// Strip author from the JSON returned to the submitter (the teacher route
		// is the only one that includes it).
		created.AuthorID = nil
		writeJSON(w, http.StatusCreated, map[string]any{"question": questionToJSON(created)})
	}
}

// handleGetCourseQuestions — GET /api/v1/courses/{courseId}/questions
//
// Teacher (staff) views the question queue. Author IDs are included for
// moderation/abuse review and are NOT included in the student-facing route.
func (d Deps) handleGetCourseQuestions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.classroomSignalsEnabled(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "courseId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id.")
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		isStaff, err := enrollment.UserIsCourseStaffByID(r.Context(), d.Pool, courseID, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course role.")
			return
		}
		if !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Teacher access required.")
			return
		}
		includeAddressed := r.URL.Query().Get("includeAddressed") == "true"
		questions, err := classroomsignals.ListCourseQuestions(r.Context(), d.Pool, courseID, true, includeAddressed)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list questions.")
			return
		}
		out := make([]map[string]any, 0, len(questions))
		for i := range questions {
			out = append(out, questionToJSON(&questions[i]))
		}
		writeJSON(w, http.StatusOK, map[string]any{"questions": out})
	}
}

// handlePatchCourseQuestion — PATCH /api/v1/courses/{courseId}/questions/{questionId}
//
// Teacher marks a question as addressed.
func (d Deps) handlePatchCourseQuestion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.classroomSignalsEnabled(w) {
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "courseId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id.")
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		questionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "questionId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid question id.")
			return
		}
		isStaff, err := enrollment.UserIsCourseStaffByID(r.Context(), d.Pool, courseID, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course role.")
			return
		}
		if !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Teacher access required.")
			return
		}
		if err := classroomsignals.MarkQuestionAddressed(r.Context(), d.Pool, courseID, questionID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update question.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func hallPassToJSON(p *classroomsignals.HallPass, includeStudentID bool) map[string]any {
	if p == nil {
		return nil
	}
	m := map[string]any{
		"id":          p.ID.String(),
		"sectionId":   p.SectionID.String(),
		"destination": p.Destination,
		"status":      p.Status,
		"requestedAt": p.RequestedAt.UTC().Format(time.RFC3339Nano),
	}
	if includeStudentID {
		m["studentId"] = p.StudentID.String()
	}
	if p.EstimatedMins != nil {
		m["estimatedMins"] = *p.EstimatedMins
	} else {
		m["estimatedMins"] = nil
	}
	if p.ApprovedAt != nil {
		m["approvedAt"] = p.ApprovedAt.UTC().Format(time.RFC3339Nano)
	} else {
		m["approvedAt"] = nil
	}
	if p.ReturnedAt != nil {
		m["returnedAt"] = p.ReturnedAt.UTC().Format(time.RFC3339Nano)
	} else {
		m["returnedAt"] = nil
	}
	if p.ApprovedBy != nil {
		m["approvedBy"] = p.ApprovedBy.String()
	} else {
		m["approvedBy"] = nil
	}
	// Overdue is a derived convenience flag for the UI alert (AC-4).
	overdue := false
	if p.Status == classroomsignals.StatusApproved && p.ApprovedAt != nil && p.EstimatedMins != nil {
		elapsed := time.Since(*p.ApprovedAt)
		if elapsed > time.Duration(*p.EstimatedMins)*time.Minute {
			overdue = true
		}
	}
	m["overdue"] = overdue
	return m
}

func questionToJSON(q *classroomsignals.AnonymousQuestion) map[string]any {
	if q == nil {
		return nil
	}
	m := map[string]any{
		"id":        q.ID.String(),
		"courseId":  q.CourseID.String(),
		"question":  q.Question,
		"addressed": q.Addressed,
		"createdAt": q.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if q.AuthorID != nil {
		m["authorId"] = q.AuthorID.String()
	}
	return m
}

func (d Deps) registerClassroomSignalsRoutes(r chi.Router) {
	r.Method(http.MethodPost, "/api/v1/sections/{sectionId}/hall-passes", d.handlePostSectionHallPass())
	r.Method(http.MethodGet, "/api/v1/sections/{sectionId}/hall-passes/active", d.handleGetSectionActiveHallPasses())
	r.Method(http.MethodPatch, "/api/v1/hall-passes/{passId}", d.handlePatchHallPass())
	r.Method(http.MethodPost, "/api/v1/courses/{courseId}/questions", d.handlePostCourseQuestion())
	r.Method(http.MethodGet, "/api/v1/courses/{courseId}/questions", d.handleGetCourseQuestions())
	r.Method(http.MethodPatch, "/api/v1/courses/{courseId}/questions/{questionId}", d.handlePatchCourseQuestion())
}
