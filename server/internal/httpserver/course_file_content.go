package httpserver

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/imageproxy"
)

// handleGetCourseFileContent is GET /api/v1/courses/{course_code}/course-files/{file_id}/content
func (d Deps) handleGetCourseFileContent() http.HandlerFunc {
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
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		fileID, err := uuid.Parse(chi.URLParam(r, "file_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid file id.")
			return
		}
		row, err := coursefiles.GetForCourse(r.Context(), d.Pool, courseCode, fileID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load file.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		if !d.gateObjectDownload(w, r, row.StorageKey) {
			return
		}

		resizeOpts := parseCourseFileImageResizeOpts(r)
		cfg := d.effectiveConfig()

		// S3-backed: generate presigned URL and redirect (skip when serving a resized thumbnail)
		if d.Storage != nil && resizeOpts.MaxWidth <= 0 && resizeOpts.MaxHeight <= 0 {
			ttl := time.Duration(cfg.StoragePresignTTL) * time.Second
			if ttl <= 0 {
				ttl = time.Hour
			}
			presignURL, presignErr := d.Storage.GetPresignedURL(r.Context(), row.StorageKey, ttl)
			if presignErr != nil && !errors.Is(presignErr, filestorage.ErrNoPresignedURL) {
				log.Printf("course-file-content: presign key=%q err=%v", row.StorageKey, presignErr)
				apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "File temporarily unavailable — try again in a moment.")
				return
			}
			if presignURL != "" {
				http.Redirect(w, r, presignURL, http.StatusFound)
				return
			}
			// local driver falls through to GetObject / disk below
		}

		b, err := d.readCourseFileRowBytes(r.Context(), courseCode, row)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		ct := strings.TrimSpace(row.MimeType)
		if ct == "" {
			ct = "application/octet-stream"
		}
		if resizeOpts.MaxWidth > 0 || resizeOpts.MaxHeight > 0 {
			resized, resizedCT, err := imageproxy.ResizeIfNeeded(b, ct, resizeOpts)
			if err != nil {
				if errors.Is(err, imageproxy.ErrNotImage) {
					// SVG and other non-raster formats: serve the original (e.g. course hero banners).
				} else {
					apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInternal, "Could not resize image.")
					return
				}
			} else {
				b = resized
				ct = resizedCT
			}
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "private, max-age=86400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}

const courseFileImageResizeMaxDim = 2048

func parseCourseFileImageResizeOpts(r *http.Request) imageproxy.ResizeOpts {
	q := r.URL.Query()
	maxW := clampCourseFileImageResizeDim(q.Get("w"))
	maxH := clampCourseFileImageResizeDim(q.Get("h"))
	quality := 85
	if raw := strings.TrimSpace(q.Get("q")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 100 {
			quality = n
		}
	}
	return imageproxy.ResizeOpts{
		MaxWidth:  maxW,
		MaxHeight: maxH,
		Quality:   quality,
	}
}

func clampCourseFileImageResizeDim(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0
	}
	if n > courseFileImageResizeMaxDim {
		return courseFileImageResizeMaxDim
	}
	return n
}
