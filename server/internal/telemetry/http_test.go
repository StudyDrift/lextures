package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

func TestMetricsMiddleware_RecordsRouteAndStatus(t *testing.T) {
	m := NewMetrics()
	r := chi.NewRouter()
	r.Use(m.MetricsMiddleware)
	r.Get("/api/v1/courses/{courseId}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(r)
	defer srv.Close()
	// Two requests with different IDs must collapse to one labelled series
	// (low cardinality — plan 17.7 risk mitigation).
	for _, id := range []string{"123", "456"} {
		resp, err := http.Get(srv.URL + "/api/v1/courses/" + id)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		_ = resp.Body.Close()
	}

	body := scrape(t, m)
	want := `lextures_http_requests_total{method="GET",route="/api/v1/courses/{courseId}",status="2xx"} 2`
	if !strings.Contains(body, want) {
		t.Errorf("expected aggregated series %q in:\n%s", want, body)
	}
}

func TestMetricsMiddleware_5xx(t *testing.T) {
	m := NewMetrics()
	r := chi.NewRouter()
	r.Use(m.MetricsMiddleware)
	r.Get("/boom", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(r)
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/boom")
	_ = resp.Body.Close()
	if !strings.Contains(scrape(t, m), `route="/boom",status="5xx"`) {
		t.Error("expected 5xx status class series")
	}
}

func TestTraceIDMiddleware_SetsHeaderWhenSpanPresent(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	spanID, _ := trace.SpanIDFromHex("0123456789abcdef")
	sc := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})

	h := TraceIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(trace.ContextWithSpanContext(context.Background(), sc))
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get(TraceIDHeader); got != traceID.String() {
		t.Errorf("X-Trace-Id = %q, want %q", got, traceID.String())
	}
}

func TestTraceIDMiddleware_NoHeaderWithoutSpan(t *testing.T) {
	h := TraceIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rr.Header().Get(TraceIDHeader) != "" {
		t.Error("no trace header expected when no span in context")
	}
}

func TestStatusClass(t *testing.T) {
	cases := map[int]string{0: "2xx", 200: "2xx", 204: "2xx", 301: "3xx", 404: "4xx", 429: "4xx", 500: "5xx", 503: "5xx"}
	for code, want := range cases {
		if got := statusClass(code); got != want {
			t.Errorf("statusClass(%d) = %s, want %s", code, got, want)
		}
	}
}

func TestFallbackRoute_CollapsesIDs(t *testing.T) {
	cases := map[string]string{
		"/":                   "/",
		"/api/v1/courses/123": "/api/v1/courses/{id}",
		"/health":             "/health",
		"/files/9f86d081884c7d659a2feaa0c55ad015": "/files/{id}",
		"/a/b/c/d/e/f": "/a/b/c/d", // capped at 4 segments
	}
	for in, want := range cases {
		if got := fallbackRoute(in); got != want {
			t.Errorf("fallbackRoute(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLooksLikeID(t *testing.T) {
	ids := []string{"123", "9f86d081884c7d659a2feaa0c55ad015", "550e8400-e29b-41d4-a716-446655440000"}
	for _, s := range ids {
		if !looksLikeID(s) {
			t.Errorf("looksLikeID(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"courses", "v1", "", "abc"} {
		if looksLikeID(s) {
			t.Errorf("looksLikeID(%q) = true, want false", s)
		}
	}
}
