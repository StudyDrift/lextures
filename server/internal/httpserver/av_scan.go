package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/storageobjects"
)

func (d Deps) registerAVScanRoutes(r chi.Router) {
	r.Get("/api/v1/files/{object_id}/scan-status", d.handleGetScanStatus())
}

type scanStatusResponse struct {
	Status          string     `json:"status"`
	VirusName       *string    `json:"virus_name,omitempty"`
	ScanCompletedAt *time.Time `json:"scan_completed_at,omitempty"`
	ObjectID        string     `json:"object_id"`
}

// handleGetScanStatus is GET /api/v1/files/:object_id/scan-status.
func (d Deps) handleGetScanStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if !d.effectiveConfig().AvScanningEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Antivirus scanning is not enabled.")
			return
		}
		obj, err := storageobjects.LoadByID(r.Context(), d.Pool, objectID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load scan status.")
			return
		}
		if obj == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(scanStatusResponse{
			Status:          string(obj.ScanStatus),
			VirusName:       obj.VirusName,
			ScanCompletedAt: obj.ScanCompletedAt,
			ObjectID:        obj.ID.String(),
		})
	}
}
