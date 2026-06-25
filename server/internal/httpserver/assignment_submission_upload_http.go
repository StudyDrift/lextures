package httpserver

import (
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/gradingagentqueue"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/repos/submissionattachments"
	webhooksvc "github.com/lextures/lextures/server/internal/service/webhooks"
)

// handlePostAssignmentSubmissionUpload is POST .../assignments/{item_id}/submissions/upload
func (d Deps) handlePostAssignmentSubmissionUpload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, assignRow, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok || assignRow == nil || cid == nil {
			return
		}
		if !d.enforceConditionalReleaseForLearner(w, r, courseCode, *cid, viewer, itemID) {
			return
		}
		if !assignRow.SubmissionAllowFileUpload && !assignRow.SubmissionAllowText {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This assignment does not accept submissions.")
			return
		}
		if err := r.ParseMultipartForm(12 << 20); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid form or file too large.")
			return
		}
		f, header, err := r.FormFile("file")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing 'file' part.")
			return
		}
		defer func() { _ = f.Close() }()

		ct := strings.TrimSpace(header.Header.Get("Content-Type"))
		if ct == "" {
			ct = mime.TypeByExtension(filepath.Ext(header.Filename))
		}
		if ct == "" {
			ct = "application/octet-stream"
		}
		allowed := strings.HasPrefix(ct, "text/") ||
			ct == "application/pdf" ||
			strings.HasPrefix(ct, "image/") ||
			ct == "application/octet-stream"
		if !allowed {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unsupported submission file type.")
			return
		}
		if header.Size <= 0 || header.Size > 20<<20 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File must be between 1 byte and 20MB.")
			return
		}

		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".txt"
		}
		fileUUID := uuid.New().String()
		storageKey := fmt.Sprintf("submissions/%s/%s%s", courseCode, fileUUID, ext)
		cfg := d.effectiveConfig()
		if d.Storage != nil {
			if perr := d.Storage.PutObject(r.Context(), storageKey, f, header.Size, ct); perr != nil {
				log.Printf("submission-upload: PutObject key=%s err=%v", storageKey, perr)
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store submission.")
				return
			}
		} else {
			root := strings.TrimSpace(cfg.CourseFilesRoot)
			if root == "" {
				root = "data/course-files"
			}
			p := coursefiles.BlobDiskPath(root, courseCode, storageKey)
			if werr := writeLocalFile(p, f); werr != nil {
				log.Printf("submission-upload: local write key=%s err=%v", storageKey, werr)
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store submission.")
				return
			}
		}
		fileID, err := coursefiles.Create(r.Context(), d.Pool, *cid, viewer, storageKey, header.Filename, ct, header.Size)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record upload.")
			return
		}
		subRow, err := moduleassignmentsubmissions.UpsertAttachment(r.Context(), d.Pool, *cid, itemID, viewer, fileID)
		if err != nil || subRow == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save submission.")
			return
		}
		if err := submissionattachments.ReplaceForSubmission(r.Context(), d.Pool, subRow.ID, []uuid.UUID{fileID}); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save submission files.")
			return
		}
		d.maybeEnqueueAutoGrade(r, courseCode, *cid, itemID, subRow.ID)
		webhooksvc.EmitAssignmentSubmittedEvent(r.Context(), d.Pool, d.effectiveConfig(), *cid, courseCode, itemID, subRow.ID, viewer)
		out := d.submissionToJSON(r.Context(), courseCode, *subRow, false, 0, "")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"submission": out})
	}
}

func (d Deps) maybeEnqueueAutoGrade(r *http.Request, courseCode string, courseID, itemID, submissionID uuid.UUID) {
	if !d.graderAgentEnabled() || d.GradingAgentQueue == nil || d.Pool == nil {
		return
	}
	cfg, err := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
	if err != nil || cfg == nil || cfg.Status != gradingagentrepo.StatusAccepted || !cfg.AutoGradeNew {
		return
	}
	run, err := gradingagentrepo.CreateRun(r.Context(), d.Pool, cfg.ID, gradingagentrepo.RunScopeAuto, gradingagentrepo.RunModeApply, nil, nil, 1, nil, nil)
	if err != nil || run == nil {
		return
	}
	_ = gradingagentrepo.MarkRunRunning(r.Context(), d.Pool, run.ID)
	_ = d.GradingAgentQueue.Publish(r.Context(), gradingagentqueue.QueueMessage{
		RunID: run.ID, ConfigID: cfg.ID, SubmissionID: submissionID,
		CourseID: courseID, ItemID: itemID, CourseCode: courseCode,
	})
}