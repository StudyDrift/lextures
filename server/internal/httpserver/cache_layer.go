package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/objectcache"
	calendarsvc "github.com/lextures/lextures/server/internal/service/calendar"
)

const (
	cacheTTLCourseStructure   = 5 * time.Minute
	cacheTTLCourseEnrollments = 5 * time.Minute
	cacheTTLCatalogPage       = 10 * time.Minute
	cacheTTLUserCalendar      = 15 * time.Minute
)

func (d Deps) objectCache() *objectcache.Service {
	return d.ObjectCache
}

func (d Deps) invalidateCourseStructureCache(ctx context.Context, courseID uuid.UUID) {
	if c := d.objectCache(); c != nil {
		_ = c.InvalidateCourseStructure(ctx, courseID.String())
	}
}

func (d Deps) invalidateCourseEnrollmentsCache(ctx context.Context, courseID uuid.UUID) {
	if c := d.objectCache(); c != nil {
		_ = c.InvalidateCourseEnrollments(ctx, courseID.String())
	}
}

func (d Deps) invalidateCatalogCache(ctx context.Context) {
	if c := d.objectCache(); c != nil {
		_ = c.InvalidateCatalog(ctx)
	}
}

func (d Deps) invalidateCourseCalendarCache(ctx context.Context, courseID uuid.UUID) {
	if c := d.objectCache(); c != nil {
		_ = c.InvalidateCourseCalendar(ctx, courseID.String())
	}
	calendarsvc.DefaultFeedCache.InvalidateCourse(courseID.String())
}

func (d Deps) invalidateUserCalendarCache(ctx context.Context, userID uuid.UUID) {
	if c := d.objectCache(); c != nil {
		_ = c.InvalidateUserCalendar(ctx, userID.String())
	}
	calendarsvc.DefaultFeedCache.InvalidateUser(userID.String())
}

// authenticatedNoStoreMiddleware sets Cache-Control: no-store on authenticated API
// responses unless the handler already set Cache-Control (plan 17.5 FR-1 / AC-5).
func authenticatedNoStoreMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &cacheHeaderWriter{ResponseWriter: w, req: r}
		next.ServeHTTP(rw, r)
	})
}

type cacheHeaderWriter struct {
	http.ResponseWriter
	req          *http.Request
	wroteHeader  bool
	cacheControl string
}

func (w *cacheHeaderWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.cacheControl = w.Header().Get("Cache-Control")
		if w.cacheControl == "" && requestIsAuthenticated(w.req) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Vary", "Accept-Encoding, Authorization")
		} else if w.cacheControl != "" {
			appendVaryAcceptEncoding(w.Header())
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *cacheHeaderWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func requestIsAuthenticated(r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, "/api/v1/public/") {
		return false
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(auth, "Bearer ") {
		return true
	}
	if _, err := r.Cookie("lextures_session"); err == nil {
		return true
	}
	return false
}

func appendVaryAcceptEncoding(h http.Header) {
	vary := h.Get("Vary")
	if vary == "" {
		h.Set("Vary", "Accept-Encoding")
		return
	}
	if !strings.Contains(strings.ToLower(vary), "accept-encoding") {
		h.Set("Vary", vary+", Accept-Encoding")
	}
}

// publicCatalogCacheHeaders sets CDN-friendly cache headers with ETag support (plan 17.5 FR-2).
func publicCatalogCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=86400")
	appendVaryAcceptEncoding(w.Header())
}

func writeJSONWithETag(w http.ResponseWriter, r *http.Request, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	sum := sha256Sum(body)
	tag := `"` + hex.EncodeToString(sum[:16]) + `"`
	w.Header().Set("ETag", tag)
	if inm := strings.TrimSpace(r.Header.Get("If-None-Match")); inm == tag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func sha256Sum(b []byte) [32]byte {
	return sha256.Sum256(b)
}
