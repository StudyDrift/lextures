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
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
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
		if d.Pool == nil {
			_, _, _ = d.requireCourseAccess(w, r)
			return
		}
		courseCode := strings.TrimSpace(chi.URLParam(r, "course_code"))
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course code.")
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

		// Enrolled users may read any course file. Storefront/catalog hero images are
		// also readable without enrollment when they match the course's hero_image_url
		// (marketplace cards and the public www storefront need this).
		if !d.canReadCourseFileContent(w, r, courseCode, row) {
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
		if d.isPublicStorefrontHero(r, courseCode, row) {
			w.Header().Set("Cache-Control", "public, max-age=86400")
		} else {
			w.Header().Set("Cache-Control", "private, max-age=86400")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}

// canReadCourseFileContent authorizes GET …/course-files/{id}/content.
// Enrolled users may read any file for the course. The course hero image is also
// readable without enrollment when the course is published and marketplace-listed
// or publicly catalogued (storefront / www cards).
func (d Deps) canReadCourseFileContent(w http.ResponseWriter, r *http.Request, courseCode string, row *coursefiles.Row) bool {
	if row == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	if viewer, ok := d.optionalUserID(r); ok {
		has, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course access.")
			return false
		}
		if has {
			return true
		}
	}
	if d.isPublicStorefrontHero(r, courseCode, row) {
		return true
	}
	// Fall back to the normal enrolled-access path (writes 401/404 as appropriate).
	_, _, ok := d.requireCourseAccess(w, r)
	return ok
}

func (d Deps) isPublicStorefrontHero(r *http.Request, courseCode string, row *coursefiles.Row) bool {
	if row == nil || d.Pool == nil {
		return false
	}
	readable, err := course.IsStorefrontHeroReadable(r.Context(), d.Pool, courseCode)
	if err != nil || !readable {
		return false
	}
	isHero, err := course.IsCourseHeroFile(r.Context(), d.Pool, courseCode, row.ID)
	return err == nil && isHero
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
