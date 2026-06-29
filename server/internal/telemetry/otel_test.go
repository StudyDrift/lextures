package telemetry

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestSetupTracing_DisabledNoEndpoint(t *testing.T) {
	shutdown, err := setupTracing(context.Background(), OTelConfig{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown must be non-nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("no-op shutdown err: %v", err)
	}
}

func TestServiceNameOr(t *testing.T) {
	if serviceNameOr("") != "lextures-api" {
		t.Error("empty service name should default")
	}
	if serviceNameOr("custom") != "custom" {
		t.Error("explicit service name should be kept")
	}
}

func TestSpanRouteName_Fallback(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/courses/42", nil)
	if got := spanRouteName(req); got != "/api/v1/courses/{id}" {
		t.Errorf("spanRouteName = %q", got)
	}
}

func TestSQLOperation(t *testing.T) {
	cases := map[string]string{
		"SELECT * FROM users":    "SELECT",
		"  insert into x values": "INSERT",
		"UPDATE":                 "UPDATE",
		"":                       "unknown",
	}
	for in, want := range cases {
		if got := sqlOperation(in); got != want {
			t.Errorf("sqlOperation(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPgxTracer_NoSpanNoPanic(t *testing.T) {
	tr := NewPgxTracer()
	// Without a recording span in context, TraceQueryStart returns the same ctx
	// and TraceQueryEnd is a no-op (no orphan spans, zero overhead).
	ctx := tr.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
	tr.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
}
