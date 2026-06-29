package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TestObserveChain_EndToEnd installs the full observation middleware chain with a
// real (always-sampling) tracer provider and asserts that a request produces a
// trace span, sets X-Trace-Id, and is recorded in Prometheus — covering FR-1,
// FR-2, FR-8, and AC-1 without needing an external collector.
func TestObserveChain_EndToEnd(t *testing.T) {
	prev := otel.GetTracerProvider()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev); _ = tp.Shutdown(context.Background()) })

	tel := Init(context.Background(), Config{ServiceName: "test", Version: "1", Environment: "test"})
	r := chi.NewRouter()
	for _, mw := range tel.ObserveMiddlewares() {
		r.Use(mw)
	}
	var sawTraceCtx bool
	r.Get("/api/v1/courses/{id}", func(w http.ResponseWriter, req *http.Request) {
		sawTraceCtx = trace.SpanContextFromContext(req.Context()).IsValid()
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(r)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/v1/courses/7")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// FR-8: response carries a 32-hex-char trace id.
	tid := resp.Header.Get(TraceIDHeader)
	if len(tid) != 32 {
		t.Fatalf("X-Trace-Id = %q (len %d), want 32 hex chars", tid, len(tid))
	}
	if !sawTraceCtx {
		t.Error("handler did not observe an active span context (FR-2)")
	}

	// FR-1 / AC-1: the request is recorded under the route group, not the raw path.
	body := scrape(t, tel.Metrics)
	if !strings.Contains(body, `route="/api/v1/courses/{id}",status="2xx"`) {
		t.Errorf("metrics missing route-grouped series:\n%s", body)
	}
}
