package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/gradingredaction"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/submissionattachments"
	"github.com/lextures/lextures/server/internal/repos/user"
)

var errAssignmentNotFound = errors.New("assignment not found")

func submissionAttachmentContentPath(courseCode string, fileID uuid.UUID) string {
	return "/api/v1/courses/" + courseCode + "/course-files/" + fileID.String() + "/content"
}

func (d Deps) viewerCanViewAssignmentSubmissions(ctx context.Context, courseCode string, viewer uuid.UUID) (bool, error) {
	canGradebook, err := courseroles.UserHasPermission(ctx, d.Pool, viewer, "course:"+courseCode+":gradebook:view")
	if err != nil {
		return false, err
	}
	if canGradebook {
		return true, nil
	}
	// Course designers can configure grading agents but may lack gradebook:view.
	return rbac.UserHasPermission(ctx, d.Pool, viewer, "course:"+courseCode+":item:create")
}

func loadAssignmentForSubmissionsByIDs(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode string,
	itemID uuid.UUID,
) (*uuid.UUID, *coursemoduleassignments.CourseItemAssignmentRow, error) {
	cid, err := course.GetIDByCourseCode(ctx, pool, courseCode)
	if err != nil {
		return nil, nil, err
	}
	if cid == nil {
		return nil, nil, errAssignmentNotFound
	}
	row, err := coursemoduleassignments.GetForCourseItem(ctx, pool, *cid, itemID)
	if err != nil {
		return nil, nil, err
	}
	if row == nil {
		return nil, nil, errAssignmentNotFound
	}
	return cid, row, nil
}

func (d Deps) loadAssignmentForSubmissions(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
	itemID uuid.UUID,
) (*uuid.UUID, *coursemoduleassignments.CourseItemAssignmentRow, bool) {
	if d.Pool == nil {
		apierr.WriteJSONWithErr(w, r, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.", nil)
		return nil, nil, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteInternal(w, r, "Failed to load course.", err)
		return nil, nil, false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return nil, nil, false
	}
	row, err := coursemoduleassignments.GetForCourseItem(r.Context(), d.Pool, *cid, itemID)
	if err != nil {
		apierr.WriteInternal(w, r, "Failed to load assignment.", err)
		return nil, nil, false
	}
	if row == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return nil, nil, false
	}
	return cid, row, true
}

func submissionAttachmentToJSON(courseCode string, fileID uuid.UUID, filename, mimeType string) map[string]any {
	return map[string]any{
		"fileId":      fileID.String(),
		"filename":    filename,
		"mimeType":    mimeType,
		"contentPath": submissionAttachmentContentPath(courseCode, fileID),
	}
}

func (d Deps) submissionAttachmentsToJSON(ctx context.Context, courseCode string, submissionID uuid.UUID) []map[string]any {
	if d.Pool == nil {
		return nil
	}
	rows, err := submissionattachments.ListForSubmission(ctx, d.Pool, submissionID)
	if err != nil || len(rows) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, submissionAttachmentToJSON(courseCode, row.FileID, row.OriginalFilename, row.MimeType))
	}
	return out
}

func (d Deps) submissionToJSON(
	ctx context.Context,
	courseCode string,
	s moduleassignmentsubmissions.SubmissionRow,
	redactPII bool,
	blindRank int,
	submitterDisplayName string,
) map[string]any {
	out := map[string]any{
		"id":               s.ID.String(),
		"attachmentFileId": nil,
		"attachments":      []map[string]any{},
		"submittedAt":      s.SubmittedAt.UTC().Format(time.RFC3339),
		"updatedAt":        s.UpdatedAt.UTC().Format(time.RFC3339),
	}
	attachmentItems := d.submissionAttachmentsToJSON(ctx, courseCode, s.ID)
	if len(attachmentItems) > 0 {
		out["attachments"] = attachmentItems
		first := attachmentItems[0]
		out["attachmentFileId"] = first["fileId"]
		out["attachmentContentPath"] = first["contentPath"]
		out["attachmentMimeType"] = first["mimeType"]
		out["attachmentFilename"] = first["filename"]
	} else if s.AttachmentFileID != nil {
		out["attachmentFileId"] = s.AttachmentFileID.String()
		if file, err := coursefiles.GetForCourse(ctx, d.Pool, courseCode, *s.AttachmentFileID); err == nil && file != nil {
			out["attachmentContentPath"] = submissionAttachmentContentPath(courseCode, file.ID)
			out["attachmentMimeType"] = file.MimeType
			out["attachmentFilename"] = file.OriginalFilename
			item := submissionAttachmentToJSON(courseCode, file.ID, file.OriginalFilename, file.MimeType)
			out["attachments"] = []map[string]any{item}
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
	if strings.TrimSpace(s.BodyText) != "" {
		out["bodyText"] = s.BodyText
	}
	if redactPII {
		out["blindLabel"] = gradingredaction.BlindStudentLabel(blindRank)
	} else {
		out["submittedBy"] = s.SubmittedBy.String()
		if strings.TrimSpace(submitterDisplayName) != "" {
			out["submittedByDisplayName"] = strings.TrimSpace(submitterDisplayName)
		}
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

type assignmentRosterEntry struct {
	UserID      uuid.UUID
	DisplayName string
	Submission  *moduleassignmentsubmissions.SubmissionRow
}

func submissionMatchesGradedFilter(isGraded bool, filter moduleassignmentsubmissions.GradedFilter) bool {
	switch filter {
	case moduleassignmentsubmissions.GradedFilterGraded:
		return isGraded
	case moduleassignmentsubmissions.GradedFilterUngraded:
		return !isGraded
	default:
		return true
	}
}

func buildAssignmentRosterEntries(
	students []struct {
		UserID      uuid.UUID
		DisplayName string
	},
	submissions []moduleassignmentsubmissions.SubmissionRow,
) []assignmentRosterEntry {
	subByUser := make(map[uuid.UUID]moduleassignmentsubmissions.SubmissionRow, len(submissions))
	for _, s := range submissions {
		subByUser[s.SubmittedBy] = s
	}
	seen := make(map[uuid.UUID]struct{}, len(students)+len(submissions))
	out := make([]assignmentRosterEntry, 0, len(students)+len(submissions))
	for _, st := range students {
		entry := assignmentRosterEntry{UserID: st.UserID, DisplayName: st.DisplayName}
		if sub, ok := subByUser[st.UserID]; ok {
			subCopy := sub
			entry.Submission = &subCopy
		}
		out = append(out, entry)
		seen[st.UserID] = struct{}{}
	}
	for _, s := range submissions {
		if _, ok := seen[s.SubmittedBy]; ok {
			continue
		}
		out = append(out, assignmentRosterEntry{
			UserID:     s.SubmittedBy,
			Submission: &s,
		})
	}
	return out
}

func sortAssignmentRosterEntries(entries []assignmentRosterEntry, displayNames map[uuid.UUID]string) {
	sort.Slice(entries, func(i, j int) bool {
		labelI := strings.TrimSpace(displayNames[entries[i].UserID])
		if labelI == "" {
			labelI = entries[i].DisplayName
		}
		labelJ := strings.TrimSpace(displayNames[entries[j].UserID])
		if labelJ == "" {
			labelJ = entries[j].DisplayName
		}
		if labelI != labelJ {
			return strings.ToLower(labelI) < strings.ToLower(labelJ)
		}
		return entries[i].UserID.String() < entries[j].UserID.String()
	})
}

func blindRanksForRoster(entries []assignmentRosterEntry) map[uuid.UUID]int {
	sorted := append([]assignmentRosterEntry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].UserID.String() < sorted[j].UserID.String()
	})
	out := make(map[uuid.UUID]int, len(sorted))
	for i, e := range sorted {
		out[e.UserID] = i + 1
	}
	return out
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
		has, err := d.viewerCanViewAssignmentSubmissions(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to verify permissions.", err)
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
		students, err := enrollment.ListStudentUsersForCourseCode(r.Context(), d.Pool, courseCode, nil)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load course roster.", err)
			return
		}
		rows, err := moduleassignmentsubmissions.ListForAssignment(
			r.Context(), d.Pool, *cid, itemID, moduleassignmentsubmissions.GradedFilterAll,
		)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load submissions.", err)
			return
		}
		roster := buildAssignmentRosterEntries(students, rows)
		cfg := d.effectiveConfig()
		identitiesRevealed := assignRow.IdentitiesRevealedAt != nil
		redact := gradingredaction.ShouldRedactSubmissionPiiForStaff(
			cfg.BlindGradingEnabled,
			assignRow.BlindGrading,
			identitiesRevealed,
		)
		displayNames := map[uuid.UUID]string{}
		if !redact {
			userIDs := make([]uuid.UUID, 0, len(roster))
			for _, entry := range roster {
				userIDs = append(userIDs, entry.UserID)
			}
			displayNames, err = user.DisplayLabelsByIDs(r.Context(), d.Pool, userIDs)
			if err != nil {
				apierr.WriteInternal(w, r, "Failed to load submitter names.", err)
				return
			}
			for _, entry := range roster {
				if strings.TrimSpace(displayNames[entry.UserID]) == "" && strings.TrimSpace(entry.DisplayName) != "" {
					displayNames[entry.UserID] = strings.TrimSpace(entry.DisplayName)
				}
			}
		}
		sortAssignmentRosterEntries(roster, displayNames)
		blindRanks := blindRanksForRoster(roster)
		gradedMap := make(map[uuid.UUID]bool)
		gradeRows, err := d.Pool.Query(r.Context(), `
			SELECT student_user_id 
			FROM course.course_grades 
			WHERE course_id = $1 AND module_item_id = $2
		`, *cid, itemID)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load grades.", err)
			return
		}
		defer gradeRows.Close()
		for gradeRows.Next() {
			var sID uuid.UUID
			if err := gradeRows.Scan(&sID); err != nil {
				apierr.WriteInternal(w, r, "Failed to scan grades.", err)
				return
			}
			gradedMap[sID] = true
		}

		items := make([]map[string]any, 0, len(roster))
		for _, entry := range roster {
			isGraded := gradedMap[entry.UserID]
			if !submissionMatchesGradedFilter(isGraded, filter) {
				continue
			}
			label := strings.TrimSpace(displayNames[entry.UserID])
			if label == "" {
				label = strings.TrimSpace(entry.DisplayName)
			}
			var item map[string]any
			if entry.Submission != nil {
				item = d.submissionToJSON(
					r.Context(), courseCode, *entry.Submission, redact, blindRanks[entry.UserID], label,
				)
			} else {
				item = map[string]any{
					"attachmentFileId": nil,
				}
				if redact {
					item["blindLabel"] = gradingredaction.BlindStudentLabel(blindRanks[entry.UserID])
				} else {
					item["submittedBy"] = entry.UserID.String()
					if label != "" {
						item["submittedByDisplayName"] = label
					}
				}
			}
			item["isGraded"] = isGraded
			items = append(items, item)
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
			apierr.WriteInternal(w, r, "Failed to load submission.", err)
			return
		}
		if row == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"submission": nil})
			return
		}
		displayName := ""
		if u, err := user.FindByID(r.Context(), d.Pool, viewer); err == nil && u != nil {
			displayName = user.DisplayLabel(u.DisplayName, u.Email)
		}
		payload := d.submissionToJSON(r.Context(), courseCode, *row, false, 0, displayName)
		var exists bool
		err = d.Pool.QueryRow(r.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM course.course_grades 
				WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
			)
		`, *cid, viewer, itemID).Scan(&exists)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to check grading status.", err)
			return
		}
		payload["isGraded"] = exists

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"submission": payload})
	}
}
