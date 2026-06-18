package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

const maxInstructorCommentLen = 8000

type submissionGradeWriteBody struct {
	PointsEarned      *float64           `json:"pointsEarned"`
	RubricScores      map[string]float64 `json:"rubricScores"`
	InstructorComment *string            `json:"instructorComment"`
	ClearGrade        bool               `json:"clearGrade"`
}

func submissionGradeCellToJSON(
	submissionID *uuid.UUID,
	studentUserID uuid.UUID,
	assignRow *coursemoduleassignments.CourseItemAssignmentRow,
	cell *coursegrades.CellRow,
) map[string]any {
	out := map[string]any{
		"studentUserId": studentUserID.String(),
		"maxPoints":     assignRow.PointsWorth,
		"posted":        false,
		"excused":       false,
	}
	if submissionID != nil {
		out["submissionId"] = submissionID.String()
	}
	if cell != nil {
		if cell.PointsEarned != nil {
			out["pointsEarned"] = *cell.PointsEarned
		}
		if cell.InstructorComment != nil {
			out["instructorComment"] = *cell.InstructorComment
		}
		if cell.PostedAt != nil {
			out["posted"] = true
		}
		out["excused"] = cell.Excused
		if scores, perr := coursegrades.ParseRubricScoresMap(cell.RubricScoresJSON); perr == nil && len(scores) > 0 {
			out["rubricScores"] = scores
		}
	}
	return out
}

func (d Deps) writeSubmissionGrade(
	w http.ResponseWriter,
	r *http.Request,
	cid uuid.UUID,
	itemID uuid.UUID,
	studentUserID uuid.UUID,
	submissionID *uuid.UUID,
	assignRow *coursemoduleassignments.CourseItemAssignmentRow,
	b submissionGradeWriteBody,
) bool {
	if b.ClearGrade {
		if err := coursegrades.DeleteCell(r.Context(), d.Pool, cid, studentUserID, itemID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to clear grade.")
			return false
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	var comment *string
	if b.InstructorComment != nil {
		t := strings.TrimSpace(*b.InstructorComment)
		if len(t) > maxInstructorCommentLen {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Comment is too long.")
			return false
		}
		if t != "" {
			comment = &t
		}
	}
	rubricDef, _ := parseAssignmentRubricJSON(assignRow.RubricJSON)
	var rubricJSON []byte
	points := 0.0
	hasPoints := false
	if rubricDef != nil && len(b.RubricScores) > 0 {
		scores := make(map[uuid.UUID]float64, len(b.RubricScores))
		for k, v := range b.RubricScores {
			id, perr := uuid.Parse(strings.TrimSpace(k))
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid rubric criterion id.")
				return false
			}
			scores[id] = v
		}
		total, verr := assignmentrubric.ValidateRubricScoresForGrade(rubricDef, scores)
		if verr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, verr.Error())
			return false
		}
		points = total
		hasPoints = true
		rubricJSON, _ = json.Marshal(b.RubricScores)
	} else if b.PointsEarned != nil {
		points = *b.PointsEarned
		hasPoints = true
	}
	if !hasPoints {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Provide pointsEarned or rubricScores.")
		return false
	}
	if points < 0 || points > 1e9 {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid points value.")
		return false
	}
	posting := strings.TrimSpace(assignRow.PostingPolicy)
	if posting == "" {
		posting = "automatic"
	}
	if err := coursegrades.UpsertCell(
		r.Context(), d.Pool, cid, studentUserID, itemID,
		points, rubricJSON, comment, posting,
	); err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save grade.")
		return false
	}
	out := map[string]any{
		"studentUserId": studentUserID.String(),
		"pointsEarned":  points,
		"posted":        posting == "automatic",
	}
	if submissionID != nil {
		out["submissionId"] = submissionID.String()
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
	return true
}

func (d Deps) handleGetSubmissionGrade() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPut+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		has, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view grades.")
			return
		}
		itemID, submissionID, cid, assignRow, subRow, ok := d.loadSubmissionGradeContext(w, r, courseCode)
		if !ok {
			return
		}
		cell, err := coursegrades.GetCell(r.Context(), d.Pool, *cid, subRow.SubmittedBy, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grade.")
			return
		}
		out := submissionGradeCellToJSON(&submissionID, subRow.SubmittedBy, assignRow, cell)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) handleGetAssignmentStudentGrade() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPut+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		has, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view grades.")
			return
		}
		itemID, studentUserID, cid, assignRow, ok := d.loadAssignmentStudentGradeContext(w, r, courseCode)
		if !ok {
			return
		}
		cell, err := coursegrades.GetCell(r.Context(), d.Pool, *cid, studentUserID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grade.")
			return
		}
		out := submissionGradeCellToJSON(nil, studentUserID, assignRow, cell)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) handlePutSubmissionGrade() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPut+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		has, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to edit grades.")
			return
		}
		itemID, submissionID, cid, assignRow, subRow, ok := d.loadSubmissionGradeContext(w, r, courseCode)
		if !ok {
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var b submissionGradeWriteBody
		if err := json.Unmarshal(payload, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		_ = d.writeSubmissionGrade(w, r, *cid, itemID, subRow.SubmittedBy, &submissionID, assignRow, b)
	}
}

func (d Deps) handlePutAssignmentStudentGrade() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPut+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		has, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to edit grades.")
			return
		}
		itemID, studentUserID, cid, assignRow, ok := d.loadAssignmentStudentGradeContext(w, r, courseCode)
		if !ok {
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var b submissionGradeWriteBody
		if err := json.Unmarshal(payload, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		_ = d.writeSubmissionGrade(w, r, *cid, itemID, studentUserID, nil, assignRow, b)
	}
}

func (d Deps) loadSubmissionGradeContext(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
) (
	itemID uuid.UUID,
	submissionID uuid.UUID,
	cid *uuid.UUID,
	assignRow *coursemoduleassignments.CourseItemAssignmentRow,
	subRow *moduleassignmentsubmissions.SubmissionRow,
	ok bool,
) {
	itemID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "item_id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return
	}
	submissionID, err = uuid.Parse(strings.TrimSpace(chi.URLParam(r, "submission_id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid submission id.")
		return
	}
	cid, assignRow, ok = d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
	if !ok || cid == nil || assignRow == nil {
		return
	}
	subRow, err = moduleassignmentsubmissions.GetByIDForCourse(r.Context(), d.Pool, *cid, submissionID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submission.")
		return
	}
	if subRow == nil || subRow.ModuleItemID != itemID {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return
	}
	ok = true
	return
}

func (d Deps) loadAssignmentStudentGradeContext(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
) (
	itemID uuid.UUID,
	studentUserID uuid.UUID,
	cid *uuid.UUID,
	assignRow *coursemoduleassignments.CourseItemAssignmentRow,
	ok bool,
) {
	itemID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "item_id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return
	}
	studentUserID, err = uuid.Parse(strings.TrimSpace(chi.URLParam(r, "student_user_id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
		return
	}
	cid, assignRow, ok = d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
	if !ok || cid == nil || assignRow == nil {
		return
	}
	ok = true
	return
}

func parseAssignmentRubricJSON(raw []byte) (*assignmentrubric.RubricDefinition, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var r assignmentrubric.RubricDefinition
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	if len(r.Criteria) == 0 {
		return nil, nil
	}
	return &r, nil
}
