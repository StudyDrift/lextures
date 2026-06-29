package telemetry

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"
)

// TraceIDHeader is added to every response so users can quote it when reporting
// issues and on-call can jump straight to the trace (plan 17.7 FR-8 / §9).
const TraceIDHeader = "X-Trace-Id"

// MetricsMiddleware records request count, latency, and in-flight gauge for
// every HTTP request, labelled by method, route group, and status class
// (plan 17.7 FR-1, AC-1). Route labels use the chi route pattern (e.g.
// /api/v1/courses/{courseId}) so cardinality stays bounded — never raw paths
// with IDs (plan 17.7 risk: high-cardinality labels).
func (m *Metrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.IncInFlight()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			m.DecInFlight()
			route := routeLabel(r)
			m.ObserveHTTP(r.Method, route, statusClass(ww.Status()), time.Since(start).Seconds())
		}()
		next.ServeHTTP(ww, r)
	})
}

// TraceIDMiddleware writes the active trace ID to the response header (FR-8). It
// must run inside the OTel HTTP middleware so a span context is present; when
// tracing is disabled the span is non-recording and no header is written.
func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sc := trace.SpanContextFromContext(r.Context()); sc.HasTraceID() {
			w.Header().Set(TraceIDHeader, sc.TraceID().String())
		}
		next.ServeHTTP(w, r)
	})
}

// routeLabel returns the matched chi route pattern, or a coarse fallback derived
// from the path when no pattern matched (404s, pre-route middleware). The
// fallback collapses anything that looks like an identifier so unmatched paths
// cannot explode label cardinality.
func routeLabel(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" && p != "/*" {
			return p
		}
	}
	return fallbackRoute(r.URL.Path)
}

func fallbackRoute(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	segs := strings.Split(strings.Trim(path, "/"), "/")
	const maxSegs = 4
	if len(segs) > maxSegs {
		segs = segs[:maxSegs]
	}
	for i, s := range segs {
		if looksLikeID(s) {
			segs[i] = "{id}"
		}
	}
	return "/" + strings.Join(segs, "/")
}

// looksLikeID reports whether a path segment is an opaque identifier (UUID,
// numeric ID, long hex) that would blow up cardinality if used as a label.
func looksLikeID(s string) bool {
	if s == "" {
		return false
	}
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	if len(s) >= 16 && isHexish(s) {
		return true
	}
	return strings.Count(s, "-") >= 4 // UUID shape
}

func isHexish(s string) bool {
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F', c == '-':
		default:
			return false
		}
	}
	return true
}

// statusClass buckets an HTTP status into 2xx/3xx/4xx/5xx to keep the status
// label low-cardinality while still distinguishing error rates (plan 17.7 FR-6
// alerting on error_rate). Status 0 (never written) maps to 2xx.
func statusClass(status int) string {
	switch {
	case status == 0:
		return "2xx"
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}
