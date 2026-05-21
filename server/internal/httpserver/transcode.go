package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/transcodejobs"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/workers/transcode"
)

func (d Deps) registerTranscodeRoutes(r chi.Router) {
	r.Get("/api/v1/files/{object_id}/transcode-status", d.handleGetTranscodeStatus())
	r.Post("/api/v1/admin/files/{object_id}/retranscode", d.handleAdminRetranscode())
	r.Get("/api/v1/ws/transcode/{job_id}", d.handleTranscodeWS())
}

type transcodeStatusResponse struct {
	Status           string    `json:"status"`
	MasterPlaylistURL string   `json:"master_playlist_url,omitempty"`
	PosterURL        string    `json:"poster_url,omitempty"`
	Renditions       []string  `json:"renditions,omitempty"`
	Error            string    `json:"error,omitempty"`
	JobID            string    `json:"job_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// handleGetTranscodeStatus is GET /api/v1/files/:object_id/transcode-status.
// Returns the transcode job status for the most recent job matching this object.
func (d Deps) handleGetTranscodeStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate UUID before any other check so malformed IDs always return 400.
		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}

		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if !d.effectiveConfig().VideoTranscodingEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Video transcoding is not enabled.")
			return
		}

		job, err := transcodejobs.LoadByObjectID(r.Context(), d.Pool, objectID)
		if err != nil {
			slog.Error("transcode-status: load job", "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcode status.")
			return
		}
		if job == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "No transcode job found for this file.")
			return
		}

		resp := buildStatusResponse(r.Context(), d.Pool, d.Storage, job, d.effectiveConfig().StoragePresignTTL)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// handleAdminRetranscode is POST /api/v1/admin/files/:object_id/retranscode.
// Re-queues the transcoding job for an object (admin only).
func (d Deps) handleAdminRetranscode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.effectiveConfig().VideoTranscodingEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Video transcoding is not enabled.")
			return
		}

		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}

		// Look up the storage object to get the source key
		var sourceKey string
		err = d.Pool.QueryRow(r.Context(),
			`SELECT object_key FROM storage.objects WHERE id = $1`, objectID).Scan(&sourceKey)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
			return
		}

		jobID, err := transcode.EnqueueForObject(r.Context(), d.Pool, sourceKey, &objectID)
		if err != nil {
			slog.Error("retranscode: enqueue", "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to queue transcode job.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"job_id": jobID.String()})
	}
}

// handleTranscodeWS is GET /api/v1/ws/transcode/:job_id.
// Streams status updates (queued → processing → done) to the uploader's UI in real time.
func (d Deps) handleTranscodeWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate UUID before auth so malformed IDs always return 400.
		jobID, err := uuid.Parse(chi.URLParam(r, "job_id"))
		if err != nil {
			http.Error(w, "invalid job_id", http.StatusBadRequest)
			return
		}

		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if !d.effectiveConfig().VideoTranscodingEnabled {
			http.Error(w, "video transcoding not enabled", http.StatusNotFound)
			return
		}

		conn, wsErr := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
		if wsErr != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		ctx := r.Context()
		ttlSecs := d.effectiveConfig().StoragePresignTTL
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				job, loadErr := transcodejobs.Load(ctx, d.Pool, jobID)
				if loadErr != nil || job == nil {
					_ = conn.Close(websocket.StatusInternalError, "job not found")
					return
				}

				resp := buildStatusResponse(ctx, d.Pool, d.Storage, job, ttlSecs)
				b, _ := json.Marshal(resp)
				if writeErr := conn.Write(ctx, websocket.MessageText, b); writeErr != nil {
					return
				}

				if job.Status == transcodejobs.StatusDone || job.Status == transcodejobs.StatusFailed {
					return
				}
			}
		}
	}
}

func buildStatusResponse(
	ctx context.Context,
	pool *pgxpool.Pool,
	storage filestorage.Driver,
	job *transcodejobs.Job,
	presignTTLSecs int,
) transcodeStatusResponse {
	resp := transcodeStatusResponse{
		Status:    string(job.Status),
		JobID:     job.ID.String(),
		CreatedAt: job.CreatedAt,
	}
	if job.Error != nil {
		resp.Error = *job.Error
	}
	if job.Status == transcodejobs.StatusDone {
		resp.Renditions = []string{"360p", "720p", "1080p"}
		ttl := time.Duration(presignTTLSecs) * time.Second
		if ttl <= 0 {
			ttl = time.Hour
		}
		if job.MasterPlaylist != nil && storage != nil {
			if url, err := storage.GetPresignedURL(ctx, *job.MasterPlaylist, ttl); err == nil {
				resp.MasterPlaylistURL = url
			} else if err == filestorage.ErrNoPresignedURL {
				// Local storage: expose a relative path clients can resolve
				resp.MasterPlaylistURL = fmt.Sprintf("/api/v1/hls/%s/master.m3u8", job.ID)
			}
		}
		if job.PosterKey != nil && storage != nil {
			if url, err := storage.GetPresignedURL(ctx, *job.PosterKey, ttl); err == nil {
				resp.PosterURL = url
			}
		}
	}
	_ = pool
	return resp
}
