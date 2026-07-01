package httpserver

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	prrepo "github.com/lextures/lextures/server/internal/repos/peerreview"
	"github.com/lextures/lextures/server/internal/repos/submissionattachments"
)

// handleGetSubmissionAttachmentsArchive is GET .../submissions/{submission_id}/attachments/archive
func (d Deps) handleGetSubmissionAttachmentsArchive() http.HandlerFunc {
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
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		submissionID, err := uuid.Parse(chi.URLParam(r, "submission_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid submission id.")
			return
		}
		cid, _, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok || cid == nil {
			return
		}
		subRow, err := moduleassignmentsubmissions.GetByIDForCourse(r.Context(), d.Pool, *cid, submissionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submission.")
			return
		}
		if subRow == nil || subRow.ModuleItemID != itemID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		canViewAll, err := d.viewerCanViewAssignmentSubmissions(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify access.")
			return
		}
		canPeerReview := false
		if !canViewAll && subRow.SubmittedBy != viewer {
			canPeerReview, err = prrepo.ReviewerHasAllocationForSubmission(r.Context(), d.Pool, viewer, submissionID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify access.")
				return
			}
		}
		if !canViewAll && subRow.SubmittedBy != viewer && !canPeerReview {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Not allowed.")
			return
		}
		attachments, err := submissionattachments.ListForSubmission(r.Context(), d.Pool, submissionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submission files.")
			return
		}
		if len(attachments) == 0 {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "No files attached to this submission.")
			return
		}
		if len(attachments) == 1 {
			row, err := coursefiles.GetForCourse(r.Context(), d.Pool, courseCode, attachments[0].FileID)
			if err != nil || row == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
				return
			}
			if !d.gateObjectDownload(w, r, row.StorageKey) {
				return
			}
			data, err := d.readCourseFileRowBytes(r.Context(), courseCode, row)
			if err != nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
				return
			}
			filename := strings.TrimSpace(row.OriginalFilename)
			if filename == "" {
				filename = "submission"
			}
			w.Header().Set("Content-Type", strings.TrimSpace(row.MimeType))
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "application/octet-stream")
			}
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
			_, _ = w.Write(data)
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="submission-files.zip"`)
		zw := zip.NewWriter(w)
		defer func() { _ = zw.Close() }()
		usedNames := make(map[string]int)
		for _, att := range attachments {
			row, err := coursefiles.GetForCourse(r.Context(), d.Pool, courseCode, att.FileID)
			if err != nil || row == nil {
				continue
			}
			if !d.gateObjectDownload(w, r, row.StorageKey) {
				return
			}
			data, err := d.readCourseFileRowBytes(r.Context(), courseCode, row)
			if err != nil {
				continue
			}
			name := zipEntryName(row.OriginalFilename, usedNames)
			fw, err := zw.Create(name)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build archive.")
				return
			}
			if _, err := io.Copy(fw, bytes.NewReader(data)); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build archive.")
				return
			}
		}
	}
}

func zipEntryName(original string, used map[string]int) string {
	base := strings.TrimSpace(filepath.Base(original))
	if base == "" || base == "." {
		base = "submission"
	}
	count := used[base]
	if count > 0 {
		ext := filepath.Ext(base)
		stem := strings.TrimSuffix(base, ext)
		name := fmt.Sprintf("%s (%d)%s", stem, count+1, ext)
		used[base] = count + 1
		return name
	}
	used[base] = 1
	return base
}
