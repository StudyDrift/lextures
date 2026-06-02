package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
)

// handlePostFactoryResetCourse is POST /api/v1/courses/{course_code}/factory-reset.
func (d Deps) handlePostFactoryResetCourse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			log.Printf("factory-reset: permission check failed course=%q viewer=%s err=%v", courseCode, viewer.String(), err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}

		outcome, err := course.FactoryResetCourse(r.Context(), d.Pool, courseCode)
		if err != nil {
			log.Printf("factory-reset: reset failed course=%q viewer=%s err=%v", courseCode, viewer.String(), err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reset course.")
			return
		}
		if outcome == nil || outcome.Course == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		cfg := d.effectiveConfig()
		coursefiles.RemoveStoredBlobs(cfg.CourseFilesRoot, courseCode, outcome.RemovedCourseFileStorageKeys)
		d.removeFileManagerBlobs(r.Context(), courseCode, cfg.CourseFilesRoot, outcome.RemovedFileManagerStorageKeys)
		log.Printf(
			"factory-reset: success course=%q viewer=%s removed_legacy_file_blobs=%d removed_file_manager_blobs=%d",
			courseCode,
			viewer.String(),
			len(outcome.RemovedCourseFileStorageKeys),
			len(outcome.RemovedFileManagerStorageKeys),
		)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(outcome.Course)
	}
}

// removeFileManagerBlobs deletes on-disk or object-store blobs for course.file_items rows.
func (d Deps) removeFileManagerBlobs(ctx context.Context, courseCode, filesRoot string, storageKeys []string) {
	if len(storageKeys) == 0 {
		return
	}
	root := strings.TrimSpace(filesRoot)
	if root == "" {
		root = "data/course-files"
	}
	storage := d.Storage
	for _, key := range storageKeys {
		if storage != nil {
			if err := storage.DeleteObject(ctx, key); err != nil {
				log.Printf("factory-reset: file manager blob delete key=%q err=%v", key, err)
			}
			continue
		}
		if err := deleteLocalFile(root + "/" + courseCode + "/" + key); err != nil {
			log.Printf("factory-reset: file manager local delete key=%q err=%v", key, err)
		}
	}
}

