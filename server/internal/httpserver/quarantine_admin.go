package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/avscanjobs"
	"github.com/lextures/lextures/server/internal/repos/storageobjects"
	"github.com/lextures/lextures/server/internal/service/clamav"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/workers/avscan"
)

type quarantineListItem struct {
	ObjectID      string  `json:"object_id"`
	ObjectKey     string  `json:"object_key"`
	VirusName     *string `json:"virus_name,omitempty"`
	UploaderID    *string `json:"uploader_id,omitempty"`
	UploaderName  *string `json:"uploader_name,omitempty"`
	UploaderEmail *string `json:"uploader_email,omitempty"`
	CourseCode    *string `json:"course_code,omitempty"`
	CourseTitle   *string `json:"course_title,omitempty"`
	UploadedAt    string  `json:"uploaded_at"`
}

func (d Deps) handleAdminListQuarantine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.effectiveConfig().AvScanningEnabled {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Antivirus scanning is not enabled.")
			return
		}
		rows, err := storageobjects.ListQuarantined(r.Context(), d.Pool, 200)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list quarantine.")
			return
		}
		out := make([]quarantineListItem, 0, len(rows))
		for _, row := range rows {
			item := quarantineListItem{
				ObjectID:  row.ObjectID.String(),
				ObjectKey: row.ObjectKey,
				VirusName: row.VirusName,
				UploadedAt: row.CreatedAt.UTC().Format(time.RFC3339),
			}
			if row.UploadedBy != nil {
				s := row.UploadedBy.String()
				item.UploaderID = &s
			}
			item.UploaderName = row.UploaderName
			item.UploaderEmail = row.UploaderEmail
			item.CourseCode = row.CourseCode
			item.CourseTitle = row.CourseTitle
			out = append(out, item)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
	}
}

func (d Deps) handleAdminDeleteQuarantine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.effectiveConfig().AvScanningEnabled {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Antivirus scanning is not enabled.")
			return
		}
		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}
		obj, err := storageobjects.LoadByID(r.Context(), d.Pool, objectID)
		if err != nil || obj == nil || obj.ScanStatus != storageobjects.ScanQuarantined {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quarantined file not found.")
			return
		}
		if d.Storage != nil {
			_ = d.Storage.DeleteObject(r.Context(), obj.ObjectKey)
		}
		if err := storageobjects.SoftDelete(r.Context(), d.Pool, objectID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleAdminReleaseQuarantine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.effectiveConfig().AvScanningEnabled {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Antivirus scanning is not enabled.")
			return
		}
		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}
		obj, err := storageobjects.LoadByID(r.Context(), d.Pool, objectID)
		if err != nil || obj == nil || obj.ScanStatus != storageobjects.ScanQuarantined {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quarantined file not found.")
			return
		}
		releaseKey := clamav.ReleaseKey(obj.ObjectKey)
		if d.Storage != nil {
			if s3d, ok := d.Storage.(*filestorage.S3Driver); ok {
				err = filestorage.CopyObjectS3(r.Context(), s3d, obj.ObjectKey, releaseKey)
			} else if root := d.effectiveConfig().CourseFilesRoot; root != "" {
				err = filestorage.MoveObjectLocal(root, obj.ObjectKey, releaseKey)
			} else {
				err = filestorage.MoveObject(r.Context(), d.Storage, obj.ObjectKey, releaseKey)
			}
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to release file.")
				return
			}
		}
		if err := storageobjects.ReleaseFromQuarantine(r.Context(), d.Pool, objectID, releaseKey); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update record.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"object_key": releaseKey, "status": "clean"})
	}
}

func (d Deps) handleAdminQuarantineDownload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.effectiveConfig().AvScanningEnabled || d.Storage == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Not available.")
			return
		}
		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}
		obj, err := storageobjects.LoadByID(r.Context(), d.Pool, objectID)
		if err != nil || obj == nil || obj.ScanStatus != storageobjects.ScanQuarantined {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quarantined file not found.")
			return
		}
		ttl := time.Duration(d.effectiveConfig().StoragePresignTTL) * time.Second
		if ttl <= 0 {
			ttl = time.Hour
		}
		url, err := d.Storage.GetPresignedURL(r.Context(), obj.ObjectKey, ttl)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Failed to generate download URL.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"download_url": url})
	}
}

func (d Deps) handleAdminBulkAVScan() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.effectiveConfig().AvScanningEnabled {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Antivirus scanning is not enabled.")
			return
		}
		ids, err := storageobjects.ListPendingLegacy(r.Context(), d.Pool, 500)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list pending files.")
			return
		}
		n, err := avscanjobs.BulkEnqueuePending(r.Context(), d.Pool, ids)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to queue scans.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"queued": n, "pending": len(ids)})
	}
}

// gateObjectDownload blocks downloads when AV scanning has not cleared the object.
func (d Deps) gateObjectDownload(w http.ResponseWriter, r *http.Request, objectKey string) bool {
	cfg := d.effectiveConfig()
	if !cfg.AvScanningEnabled {
		return true
	}
	blocked, reason, err := avscan.IsBlockedDownload(r.Context(), d.Pool, objectKey, cfg.AvScanningEnabled)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify file status.")
		return false
	}
	if blocked {
		msg := "File is not available — virus scan in progress."
		code := apierr.CodeConflict
		switch reason {
		case "quarantined":
			msg = "This file was quarantined due to a security threat."
			code = apierr.CodeForbidden
		case "scan_pending":
			msg = "File is being scanned for viruses. Try again shortly."
		}
		apierr.WriteJSON(w, http.StatusForbidden, code, msg)
		return false
	}
	return true
}
