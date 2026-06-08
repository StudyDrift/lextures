package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/gradingredaction"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
)

func submissionAttachmentContentPath(courseCode string, fileID uuid.UUID) string {
	return "/api/v1/courses/" + courseCode + "/course-files/" + fileID.String() + "/content"
}

func (d Deps) loadAssignmentForSubmissions(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
	itemID uuid.UUID,
) (*uuid.UUID, *coursemoduleassignments.CourseItemAssignmentRow, bool) {
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
		return nil, nil, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return nil, nil, false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return nil, nil, false
	}
	row, err := coursemoduleassignments.GetForCourseItem(r.Context(), d.Pool, *cid, itemID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load assignment.")
		return nil, nil, false
	}
	if row == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return nil, nil, false
	}
	return cid, row, true
}

func (d Deps) submissionToJSON(
	ctx context.Context,
	courseCode string,
	s moduleassignmentsubmissions.SubmissionRow,
	redactPII bool,
	blindRank int,
) map[string]any {
	out := map[string]any{
		"id":               s.ID.String(),
		"attachmentFileId": nil,
		"submittedAt":      s.SubmittedAt.UTC().Format(time.RFC3339),
		"updatedAt":        s.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if s.AttachmentFileID != nil {
		out["attachmentFileId"] = s.AttachmentFileID.String()
		if file, err := coursefiles.GetForCourse(ctx, d.Pool, courseCode, *s.AttachmentFileID); err == nil && file != nil {
			out["attachmentContentPath"] = submissionAttachmentContentPath(courseCode, file.ID)
			out["attachmentMimeType"] = file.MimeType
			out["attachmentFilename"] = file.OriginalFilename
		}
	}
	if s.ResubmissionRequested {
		out["resubmissionRequested"] = true
	}
	if s.RevisionDueAt != nil {
		out["revisionDueAt"] = s.RevisionDueAt.UTC().Format(time.RFC3339)
	}
	if s.RevisionFeedback != nil && strings.TrimSpace(*s.RevisionFeedback) != "" {
		out["revisionFeedback"] = *s.RevisionFeedback
	}
	if s.VersionNumber > 0 {
		out["versionNumber"] = s.VersionNumber
	}
	if redactPII {
		out["blindLabel"] = gradingredaction.BlindStudentLabel(blindRank)
	} else {
		out["submittedBy"] = s.SubmittedBy.String()
	}
	return out
}

func parseSubmissionGradedFilter(q string) moduleassignmentsubmissions.GradedFilter {
	switch strings.ToLower(strings.TrimSpace(q)) {
	case "graded":
		return moduleassignmentsubmissions.GradedFilterGraded
	case "ungraded":
		return moduleassignmentsubmissions.GradedFilterUngraded
	default:
		return moduleassignmentsubmissions.GradedFilterAll
	}
}

// handleListAssignmentSubmissions is GET /api/v1/courses/{course_code}/assignments/{item_id}/submissions.
func (d Deps) handleListAssignmentSubmissions() http.HandlerFunc {
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
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view submissions.")
			return
		}
		itemID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "item_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		cid, assignRow, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok {
			return
		}
		filter := parseSubmissionGradedFilter(r.URL.Query().Get("graded"))
		rows, err := moduleassignmentsubmissions.ListForAssignment(r.Context(), d.Pool, *cid, itemID, filter)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submissions.")
			return
		}
		cfg := d.effectiveConfig()
		identitiesRevealed := assignRow.IdentitiesRevealedAt != nil
		redact := gradingredaction.ShouldRedactSubmissionPiiForStaff(
			cfg.BlindGradingEnabled,
			assignRow.BlindGrading,
			identitiesRevealed,
		)
		newestFirst := make([]uuid.UUID, len(rows))
		for i := range rows {
			newestFirst[len(rows)-1-i] = rows[i].ID
		}
		rankByID := gradingredaction.SubmissionRankByID(newestFirst)
		items := make([]map[string]any, 0, len(rows))
		for _, s := range rows {
			rank := rankByID[s.ID]
			items = append(items, d.submissionToJSON(r.Context(), courseCode, s, redact, rank))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"submissions": items})
	}
}

// handleGetMyAssignmentSubmission is GET /api/v1/courses/{course_code}/assignments/{item_id}/submissions/mine.
func (d Deps) handleGetMyAssignmentSubmission() http.HandlerFunc {
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
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "item_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		cid, _, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok {
			return
		}
		row, err := moduleassignmentsubmissions.GetForCourseItemUser(r.Context(), d.Pool, *cid, itemID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submission.")
			return
		}
		if row == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"submission": nil})
			return
		}
		payload := d.submissionToJSON(r.Context(), courseCode, *row, false, 0)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"submission": payload})
	}
}
