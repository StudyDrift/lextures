package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/marketingmedia"
)

const marketingMediaMaxBytes int64 = 10 << 20

func (d Deps) registerMarketingMediaRoutes(r chi.Router) {
	r.Post("/api/v1/admin/marketing/media", d.handleMarketingMediaUpload())
	r.Get("/api/v1/admin/marketing/media", d.handleMarketingMediaList())
	r.Get("/api/v1/admin/marketing/media/{id}", d.handleMarketingMediaGet())
	r.Patch("/api/v1/admin/marketing/media/{id}", d.handleMarketingMediaPatch())
	r.Delete("/api/v1/admin/marketing/media/{id}", d.handleMarketingMediaDelete())
}
func (d Deps) marketingMediaService() *marketingmedia.Service {
	return &marketingmedia.Service{Pool: d.Pool, Storage: d.Storage, Scanner: d.MarketingMediaScanner, MaxBytes: marketingMediaMaxBytes}
}
func (d Deps) handleMarketingMediaUpload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAuthor)
		if !ok {
			return
		}
		if d.Storage == nil {
			apierr.WriteJSON(w, 503, apierr.CodeServiceUnavailable, "Media storage is unavailable.")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, marketingMediaMaxBytes+(1<<20))
		if e := r.ParseMultipartForm(marketingMediaMaxBytes); e != nil {
			apierr.WriteJSON(w, 413, "media_too_large", "Image exceeds the 10 MB upload limit.")
			return
		}
		f, _, e := r.FormFile("file")
		if e != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "file is required.")
			return
		}
		defer func() { _ = f.Close() }()
		data, e := io.ReadAll(io.LimitReader(f, marketingMediaMaxBytes+1))
		if e != nil {
			apierr.WriteInternal(w, r, "Failed to read image.", e)
			return
		}
		decorative, _ := strconv.ParseBool(r.FormValue("decorative"))
		asset, dedup, e := d.marketingMediaService().Create(r.Context(), marketingmedia.Upload{Data: data, AltText: r.FormValue("altText"), Decorative: decorative, Title: r.FormValue("title"), Credit: r.FormValue("credit"), ActorID: actor})
		if e != nil {
			writeMediaError(w, r, e)
			return
		}
		d.recordMediaAudit(r, actor, asset)
		writeJSON(w, 201, struct {
			*repo.MediaAsset
			Deduplicated bool `json:"deduplicated"`
		}{MediaAsset: asset, Deduplicated: dedup})
	}
}
func (d Deps) handleMarketingMediaList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		unused, _ := strconv.ParseBool(r.URL.Query().Get("unusedOnly"))
		items, next, e := repo.ListMedia(r.Context(), d.Pool, repo.MediaFilter{Q: r.URL.Query().Get("q"), MIMEType: r.URL.Query().Get("mimeType"), Cursor: r.URL.Query().Get("cursor"), UnusedOnly: unused, Limit: limit})
		if e != nil {
			writeMediaError(w, r, e)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "nextCursor": next})
	}
}
func mediaID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid media id.")
		return uuid.Nil, false
	}
	return id, true
}
func (d Deps) handleMarketingMediaGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		id, ok := mediaID(w, r)
		if !ok {
			return
		}
		m, e := repo.GetMedia(r.Context(), d.Pool, id)
		if e != nil {
			writeMediaError(w, r, e)
			return
		}
		writeJSON(w, 200, m)
	}
}
func (d Deps) handleMarketingMediaPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAuthor); !ok {
			return
		}
		id, ok := mediaID(w, r)
		if !ok {
			return
		}
		var b struct {
			AltText    string `json:"altText"`
			Decorative bool   `json:"decorative"`
			Title      string `json:"title"`
			Credit     string `json:"credit"`
		}
		if !readMarketingJSON(w, r, &b) {
			return
		}
		if !b.Decorative && strings.TrimSpace(b.AltText) == "" {
			apierr.WriteJSON(w, 422, "alt_text_required", "Alt text is required unless the image is decorative.")
			return
		}
		m, e := repo.UpdateMedia(r.Context(), d.Pool, id, strings.TrimSpace(b.AltText), b.Decorative, strings.TrimSpace(b.Title), strings.TrimSpace(b.Credit))
		if e != nil {
			writeMediaError(w, r, e)
			return
		}
		writeJSON(w, 200, m)
	}
}
func (d Deps) handleMarketingMediaDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAdmin); !ok {
			return
		}
		id, ok := mediaID(w, r)
		if !ok {
			return
		}
		m, e := repo.GetMedia(r.Context(), d.Pool, id)
		if e != nil {
			writeMediaError(w, r, e)
			return
		}
		if m.UsageCount > 0 {
			writeJSON(w, 409, map[string]any{"error": map[string]string{"code": "media_in_use", "message": "Image is referenced by content."}, "usedBy": m.UsedBy})
			return
		}
		if e = repo.SoftDeleteMedia(r.Context(), d.Pool, id); e != nil {
			writeMediaError(w, r, e)
			return
		}
		w.WriteHeader(204)
	}
}
func writeMediaError(w http.ResponseWriter, r *http.Request, e error) {
	switch {
	case errors.Is(e, marketingmedia.ErrAltRequired):
		apierr.WriteJSON(w, 422, "alt_text_required", "Alt text is required unless the image is decorative.")
	case errors.Is(e, marketingmedia.ErrTooLarge):
		apierr.WriteJSON(w, 413, "media_too_large", "Image exceeds the upload limit.")
	case errors.Is(e, marketingmedia.ErrUnsupported):
		apierr.WriteJSON(w, 415, "unsupported_media_type", "Unsupported or invalid image.")
	case errors.Is(e, marketingmedia.ErrInfected):
		slog.Warn("marketing media malware rejected", "err", e)
		apierr.WriteJSON(w, 422, "media_infected", "The image failed the malware scan.")
	case errors.Is(e, pgx.ErrNoRows):
		apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Media not found.")
	default:
		apierr.WriteInternal(w, r, "Marketing media operation failed.", e)
	}
}
func (d Deps) recordMediaAudit(r *http.Request, actor uuid.UUID, m *repo.MediaAsset) {
	target := "marketing_media"
	after, _ := json.Marshal(map[string]any{"checksum": m.Checksum, "bytes": m.ByteSize, "mime": m.MIMEType})
	_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{EventType: adminaudit.EventMarketingContent, ActorID: actor, TargetType: &target, TargetID: &m.ID, AfterValue: after})
}

func (d Deps) handlePublicMarketingMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicContentOff(w) {
			return
		}
		id, ok := mediaID(w, r)
		if !ok {
			return
		}
		m, e := repo.GetPublicMedia(r.Context(), d.Pool, id)
		if e != nil {
			writeMediaError(w, r, e)
			return
		}
		rend, ok := marketingmedia.Rendition(m, chi.URLParam(r, "file"))
		if !ok {
			for _, candidate := range m.Renditions {
				if candidate.Name == "original" {
					rend = candidate
					ok = true
					break
				}
			}
		}
		if !ok {
			apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Media not found.")
			return
		}
		if s3, yes := d.Storage.(*filestorage.S3Driver); yes {
			if u, e := s3.GetPresignedURL(r.Context(), rend.Key, 5*time.Minute); e == nil {
				http.Redirect(w, r, u, http.StatusTemporaryRedirect)
				return
			}
		}
		obj, e := d.Storage.GetObject(r.Context(), rend.Key)
		if e != nil {
			apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Media not found.")
			return
		}
		defer func() { _ = obj.Close() }()
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Content-Type", rend.MIME)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Length", fmt.Sprint(rend.Bytes))
		_, _ = io.Copy(w, obj)
	}
}
