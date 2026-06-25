package logging

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/lextures/lextures/server/internal/apierr"
)

// AccessLog logs HTTP requests with redacted paths (plan 10.14 FR-5).
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		r = apierr.WithServerErrorTracking(r)
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		status := ww.Status()
		attrs := []any{
			"method", r.Method,
			"path", RedactRequestPath(r.URL.Path),
			"status", status,
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		}
		if status >= http.StatusInternalServerError {
			msg, serverErr := apierr.ServerErrorFromRequest(r)
			if msg != "" {
				attrs = append(attrs, "error_message", msg)
			}
			if serverErr != nil {
				attrs = append(attrs, "err", serverErr)
			}
			slog.Error("http request", attrs...)
			return
		}
		slog.Info("http request", attrs...)
	})
}
