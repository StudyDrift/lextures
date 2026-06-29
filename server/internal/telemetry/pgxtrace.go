package telemetry

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// PgxTracer creates a child span for each SQL query so a request trace shows
// its DB calls (plan 17.7 FR-2, AC-2: "child spans for each DB query"). It
// records only the operation/table — never bound argument values — to avoid
// PII in trace attributes (plan 17.7 NFR Privacy).
type PgxTracer struct {
	tracer trace.Tracer
}

// NewPgxTracer returns a pgx QueryTracer backed by the global OTel provider.
func NewPgxTracer() *PgxTracer {
	return &PgxTracer{tracer: Tracer("pgx")}
}

type pgxSpanKey struct{}

// TraceQueryStart starts a span when the request context already carries a
// recording span; otherwise it is effectively a no-op (no orphan spans).
func (t *PgxTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if !trace.SpanFromContext(ctx).SpanContext().IsValid() {
		return ctx
	}
	op := sqlOperation(data.SQL)
	ctx, span := t.tracer.Start(ctx, "db.query "+op)
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", op),
		attribute.Int("db.args_count", len(data.Args)),
	)
	return context.WithValue(ctx, pgxSpanKey{}, span)
}

// TraceQueryEnd closes the span started in TraceQueryStart, recording any error.
func (t *PgxTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	span, ok := ctx.Value(pgxSpanKey{}).(trace.Span)
	if !ok {
		return
	}
	if data.Err != nil {
		span.SetStatus(codes.Error, "query failed")
		span.RecordError(data.Err)
	}
	span.End()
}

// sqlOperation extracts the leading verb (SELECT/INSERT/...) for a low-cardinality
// span name. Never returns argument values.
func sqlOperation(sql string) string {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return "unknown"
	}
	if i := strings.IndexAny(sql, " \t\n"); i > 0 {
		return strings.ToUpper(sql[:i])
	}
	return strings.ToUpper(sql)
}
