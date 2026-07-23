package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	qbmodels "github.com/lextures/lextures/server/internal/models/questionbank"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/questionbank"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

func (d Deps) requireQuestionBankStaff(w http.ResponseWriter, r *http.Request) (courseCode string, courseID uuid.UUID, viewer uuid.UUID, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	isStaff, err := enrollment.UserIsCourseStaff(r.Context(), d.Pool, courseCode, viewer)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify access.")
		return "", uuid.Nil, uuid.Nil, false
	}
	if !isStaff {
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return "", uuid.Nil, uuid.Nil, false
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return "", uuid.Nil, uuid.Nil, false
		}
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return "", uuid.Nil, uuid.Nil, false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, uuid.Nil, false
	}
	bankOn, _, err := course.GetImportFlags(r.Context(), d.Pool, *cid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return "", uuid.Nil, uuid.Nil, false
	}
	if !bankOn {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Enable the question bank for this course before using it.")
		return "", uuid.Nil, uuid.Nil, false
	}
	return courseCode, *cid, viewer, true
}

func questionEntityToAPI(e questionbank.QuestionEntity, includeDetail bool) qbmodels.QuestionBankRowResponse {
	meta := e.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	row := qbmodels.QuestionBankRowResponse{
		ID:                     e.ID,
		CourseID:               e.CourseID,
		QuestionType:           e.QuestionType,
		Stem:                   e.Stem,
		Explanation:            e.Explanation,
		Points:                 e.Points,
		Status:                 e.Status,
		Shared:                 e.Shared,
		Source:                 e.Source,
		Metadata:               meta,
		IrtA:                   e.IrtA,
		IrtB:                   e.IrtB,
		IrtC:                   e.IrtC,
		IrtStatus:              e.IrtStatus,
		CreatedBy:              e.CreatedBy,
		CreatedAt:              e.CreatedAt.UTC(),
		UpdatedAt:              e.UpdatedAt.UTC(),
		VersionNumber:          e.VersionNumber,
		IsPublished:            e.IsPublished,
		ShuffleChoicesOverride: e.ShuffleChoicesOverride,
		SrsEligible:            e.SRSEligible,
	}
	if e.IrtSampleN > 0 {
		n := e.IrtSampleN
		row.IrtSampleN = &n
	}
	if e.IrtCalibratedAt != nil {
		t := e.IrtCalibratedAt.UTC()
		row.IrtCalibratedAt = &t
	}
	if includeDetail {
		if len(e.Options) > 0 {
			row.Options = e.Options
		}
		if len(e.CorrectAnswer) > 0 {
			row.CorrectAnswer = e.CorrectAnswer
		}
	}
	return row
}

// handleListCourseBankQuestions is GET /api/v1/courses/{course_code}/questions
func (d Deps) handleListCourseBankQuestions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, courseID, _, ok := d.requireQuestionBankStaff(w, r)
		if !ok {
			return
		}
		filter := questionbank.ListFilter{
			Query:  r.URL.Query().Get("q"),
			Type:   r.URL.Query().Get("type"),
			Status: r.URL.Query().Get("status"),
		}
		if c := strings.TrimSpace(r.URL.Query().Get("conceptId")); c != "" {
			id, err := uuid.Parse(c)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid conceptId.")
				return
			}
			filter.ConceptID = &id
		}
		rows, err := questionbank.ListQuestionsForCourse(r.Context(), d.Pool, courseID, filter)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list questions.")
			return
		}
		out := make([]qbmodels.QuestionBankRowResponse, 0, len(rows))
		for _, e := range rows {
			out = append(out, questionEntityToAPI(e, false))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleGetCourseBankQuestion is GET /api/v1/courses/{course_code}/questions/{question_id}
func (d Deps) handleGetCourseBankQuestion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, courseID, _, ok := d.requireQuestionBankStaff(w, r)
		if !ok {
			return
		}
		qid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "question_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid question id.")
			return
		}
		row, err := questionbank.GetQuestionForCourse(r.Context(), d.Pool, courseID, qid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load question.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Question not found.")
			return
		}
		out := questionEntityToAPI(*row, true)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}