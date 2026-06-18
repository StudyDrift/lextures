package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
)

type archivedCoursesResponse struct {
	Courses []course.ArchivedCourseRow `json:"courses"`
}

// handleGetArchivedCourses is GET /api/v1/settings/archived-courses.
func (d Deps) handleGetArchivedCourses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.viewerOrgID(w, r, viewer)
		if !ok {
			return
		}
		rows, err := course.ListArchivedInOrg(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load archived courses.")
			return
		}
		if rows == nil {
			rows = []course.ArchivedCourseRow{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(archivedCoursesResponse{Courses: rows})
	}
}

// handleRestoreArchivedCourse is POST /api/v1/settings/archived-courses/{course_code}/restore.
func (d Deps) handleRestoreArchivedCourse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.viewerOrgID(w, r, viewer)
		if !ok {
			return
		}
		courseCode := strings.TrimSpace(chi.URLParam(r, "course_code"))
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course code.")
			return
		}
		if !d.archivedCourseBelongsToOrg(w, r, orgID, courseCode) {
			return
		}
		out, err := course.SetArchived(r.Context(), d.Pool, courseCode, false, nil)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to restore course.")
			return
		}
		if out == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		d.notifyCourses(viewer)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleDeleteArchivedCourse is DELETE /api/v1/settings/archived-courses/{course_code}.
func (d Deps) handleDeleteArchivedCourse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.viewerOrgID(w, r, viewer)
		if !ok {
			return
		}
		courseCode := strings.TrimSpace(chi.URLParam(r, "course_code"))
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course code.")
			return
		}

		outcome, err := course.PermanentlyDeleteCourse(r.Context(), d.Pool, orgID, courseCode)
		if err != nil {
			if strings.Contains(err.Error(), "not archived") {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput, "Only archived courses can be permanently deleted.")
				return
			}
			log.Printf("archived-courses-delete: course=%q viewer=%s err=%v", courseCode, viewer.String(), err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete course.")
			return
		}
		if outcome == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		cfg := d.effectiveConfig()
		root := strings.TrimSpace(cfg.CourseFilesRoot)
		if root == "" {
			root = "data/course-files"
		}
		coursefiles.RemoveStoredBlobs(root, courseCode, outcome.RemovedCourseFileStorageKeys)
		d.removeFileManagerBlobs(r.Context(), courseCode, root, outcome.RemovedFileManagerStorageKeys)
		d.removeStorageObjectBlobs(r.Context(), outcome.RemovedStorageObjectKeys)
		d.removeFeedbackMediaBlobs(root, courseCode, outcome.RemovedFeedbackMediaKeys, outcome.RemovedFeedbackCaptionKeys)
		d.removeCourseLocalDirectories(root, courseCode)

		targetID := outcome.CourseID
		targetType := "course"
		if _, err := adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
			OrgID:      &orgID,
			EventType:  adminaudit.EventCourseDelete,
			ActorID:    viewer,
			TargetType: &targetType,
			TargetID:   &targetID,
		}); err != nil {
			log.Printf("archived-courses-delete: audit record failed course=%q err=%v", courseCode, err)
		}

		d.notifyCourses(viewer)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) viewerOrgID(w http.ResponseWriter, r *http.Request, viewer uuid.UUID) (uuid.UUID, bool) {
	orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, viewer)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
		return uuid.UUID{}, false
	}
	return orgID, true
}

func (d Deps) archivedCourseBelongsToOrg(w http.ResponseWriter, r *http.Request, orgID uuid.UUID, courseCode string) bool {
	courseOrg, err := course.CourseOrgID(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return false
	}
	if courseOrg == nil || *courseOrg != orgID {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return false
	}
	return true
}

func (d Deps) removeStorageObjectBlobs(ctx context.Context, objectKeys []string) {
	if len(objectKeys) == 0 || d.Storage == nil {
		return
	}
	for _, key := range objectKeys {
		if err := d.Storage.DeleteObject(ctx, key); err != nil {
			log.Printf("archived-courses-delete: storage object delete key=%q err=%v", key, err)
		}
	}
}

func (d Deps) removeFeedbackMediaBlobs(root, courseCode string, mediaKeys, captionKeys []string) {
	feedbackRoot := filepath.Join(root, "feedback", courseCode)
	for _, key := range append(append([]string{}, mediaKeys...), captionKeys...) {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		_ = os.Remove(filepath.Join(feedbackRoot, filepath.FromSlash(key)))
	}
	_ = os.RemoveAll(feedbackRoot)
}

func (d Deps) removeCourseLocalDirectories(root, courseCode string) {
	if strings.TrimSpace(courseCode) == "" {
		return
	}
	_ = os.RemoveAll(filepath.Dir(coursefiles.BlobDiskPath(root, courseCode, "cleanup")))
	_ = os.RemoveAll(filepath.Join(root, courseCode))
	_ = os.RemoveAll(filepath.Join(root, "submissions", "import", courseCode))
	_ = os.RemoveAll(filepath.Join(root, "submissions", courseCode))
}

