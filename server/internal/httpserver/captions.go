package httpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	captionsrepo "github.com/lextures/lextures/server/internal/repos/captions"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/workers/captioning"
)

func (d Deps) registerCaptionRoutes(r chi.Router) {
	r.Get("/api/v1/files/{object_id}/captions", d.handleListCaptions())
	r.Get("/api/v1/files/{object_id}/captions/{caption_id}/vtt", d.handleGetCaptionVTT())
	r.Put("/api/v1/files/{object_id}/captions/{caption_id}", d.handleUpdateCaption())
	r.Post("/api/v1/files/{object_id}/captions/retrigger", d.handleRetriggerCaption())
	r.Get("/api/v1/reports/caption-coverage", d.handleCaptionCoverageReport())
}

type captionResponse struct {
	ID               string     `json:"id"`
	StorageObjectID  string     `json:"storage_object_id"`
	Lang             string     `json:"lang"`
	Status           string     `json:"status"`
	HasLowConfidence bool       `json:"has_low_confidence"`
	ConfidenceAvg    *float32   `json:"confidence_avg,omitempty"`
	Backend          string     `json:"backend"`
	CreatedAt        time.Time  `json:"created_at"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
}

func captionToResponse(c *captionsrepo.Caption) captionResponse {
	return captionResponse{
		ID:               c.ID.String(),
		StorageObjectID:  c.StorageObjectID.String(),
		Lang:             c.Lang,
		Status:           string(c.Status),
		HasLowConfidence: c.HasLowConfidence,
		ConfidenceAvg:    c.ConfidenceAvg,
		Backend:          c.Backend,
		CreatedAt:        c.CreatedAt,
		ReviewedAt:       c.ReviewedAt,
	}
}

// handleListCaptions is GET /api/v1/files/:object_id/captions
func (d Deps) handleListCaptions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if !d.effectiveConfig().AutoCaptioningEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Auto-captioning is not enabled.")
			return
		}

		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}

		list, err := captionsrepo.ListByObjectID(r.Context(), d.Pool, objectID)
		if err != nil {
			slog.Error("captions: list", "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load captions.")
			return
		}

		resp := make([]captionResponse, 0, len(list))
		for _, c := range list {
			resp = append(resp, captionToResponse(c))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// handleGetCaptionVTT is GET /api/v1/files/:object_id/captions/:caption_id/vtt
// Redirects to a pre-signed VTT download URL.
func (d Deps) handleGetCaptionVTT() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if !d.effectiveConfig().AutoCaptioningEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Auto-captioning is not enabled.")
			return
		}

		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}
		captionID, err := uuid.Parse(chi.URLParam(r, "caption_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid caption_id.")
			return
		}

		cap, err := captionsrepo.Load(r.Context(), d.Pool, captionID)
		if err != nil {
			slog.Error("captions: load vtt", "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load caption.")
			return
		}
		if cap == nil || cap.StorageObjectID != objectID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Caption not found.")
			return
		}
		if cap.VTTKey == nil || *cap.VTTKey == "" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "VTT not yet generated.")
			return
		}

		cfg := d.effectiveConfig()
		ttl := time.Duration(cfg.StoragePresignTTL) * time.Second
		if ttl <= 0 {
			ttl = time.Hour
		}
		if d.Storage != nil {
			url, presignErr := d.Storage.GetPresignedURL(r.Context(), *cap.VTTKey, ttl)
			if presignErr != nil && presignErr != filestorage.ErrNoPresignedURL {
				slog.Error("captions: presign vtt", "err", presignErr)
				apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Storage unavailable.")
				return
			}
			if url != "" {
				http.Redirect(w, r, url, http.StatusFound)
				return
			}
			// Local driver: serve via GetObject
			rc, getErr := d.Storage.GetObject(r.Context(), *cap.VTTKey)
			if getErr != nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "VTT file not found.")
				return
			}
			defer func() { _ = rc.Close() }()
			w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
			w.Header().Set("Cache-Control", "private, max-age=300")
			_, _ = io.Copy(w, rc)
			return
		}

		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Storage not configured.")
	}
}

type updateCaptionRequest struct {
	TranscriptText string `json:"transcript_text"`
}

// handleUpdateCaption is PUT /api/v1/files/:object_id/captions/:caption_id (instructor only)
func (d Deps) handleUpdateCaption() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.effectiveConfig().AutoCaptioningEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Auto-captioning is not enabled.")
			return
		}

		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}
		captionID, err := uuid.Parse(chi.URLParam(r, "caption_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid caption_id.")
			return
		}

		var body updateCaptionRequest
		if decErr := json.NewDecoder(r.Body).Decode(&body); decErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.TranscriptText == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "transcript_text is required.")
			return
		}

		cap, err := captionsrepo.Load(r.Context(), d.Pool, captionID)
		if err != nil {
			slog.Error("captions: load for update", "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load caption.")
			return
		}
		if cap == nil || cap.StorageObjectID != objectID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Caption not found.")
			return
		}

		vttKey := ""
		if cap.VTTKey != nil {
			vttKey = *cap.VTTKey
		} else {
			vttKey = "captions/" + cap.StorageObjectID.String() + "/" + cap.ID.String() + "/" + cap.Lang + ".vtt"
		}

		if updateErr := captionsrepo.UpdateTranscript(r.Context(), d.Pool, captionID, userID, body.TranscriptText, vttKey); updateErr != nil {
			slog.Error("captions: update transcript", "err", updateErr)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update caption.")
			return
		}

		updated, err := captionsrepo.Load(r.Context(), d.Pool, captionID)
		if err != nil || updated == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reload caption.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(captionToResponse(updated))
	}
}

// handleRetriggerCaption is POST /api/v1/files/:object_id/captions/retrigger (instructor only)
func (d Deps) handleRetriggerCaption() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		cfg := d.effectiveConfig()
		if !cfg.AutoCaptioningEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Auto-captioning is not enabled.")
			return
		}

		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}

		backend := cfg.WhisperBackend
		if backend == "" {
			backend = string(captioning.BackendWhisperAPI)
		}

		jobID, err := captionsrepo.Enqueue(r.Context(), d.Pool, objectID, backend)
		if err != nil {
			slog.Error("captions: retrigger enqueue", "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to queue caption job.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"caption_id": jobID.String()})
	}
}

type captionCoverageRow struct {
	ObjectID   string     `json:"object_id"`
	ObjectKey  string     `json:"object_key"`
	MimeType   string     `json:"mime_type"`
	CaptionID  *string    `json:"caption_id,omitempty"`
	Status     *string    `json:"caption_status,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
}

// handleCaptionCoverageReport is GET /api/v1/reports/caption-coverage (admin only)
func (d Deps) handleCaptionCoverageReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.effectiveConfig().AutoCaptioningEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Auto-captioning is not enabled.")
			return
		}

		rows, err := d.Pool.Query(r.Context(), `
			SELECT o.id, o.object_key, o.mime_type,
			       c.id, c.status, c.reviewed_at
			FROM storage.objects o
			LEFT JOIN storage.captions c ON c.storage_object_id = o.id
			WHERE o.deleted_at IS NULL
			  AND o.mime_type LIKE 'video/%'
			  AND (c.id IS NULL OR c.status NOT IN ('done', 'instructor_reviewed'))
			ORDER BY o.created_at DESC
			LIMIT 500`)
		if err != nil {
			slog.Error("captions: coverage report query", "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load coverage report.")
			return
		}
		defer rows.Close()

		var result []captionCoverageRow
		for rows.Next() {
			var row captionCoverageRow
			var capID *uuid.UUID
			var status *string
			var reviewedAt *time.Time
			if scanErr := rows.Scan(&row.ObjectID, &row.ObjectKey, &row.MimeType, &capID, &status, &reviewedAt); scanErr != nil {
				slog.Error("captions: coverage report scan", "err", scanErr)
				continue
			}
			if capID != nil {
				s := capID.String()
				row.CaptionID = &s
			}
			row.Status = status
			row.ReviewedAt = reviewedAt
			result = append(result, row)
		}
		if result == nil {
			result = []captionCoverageRow{}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"rows": result})
	}
}
