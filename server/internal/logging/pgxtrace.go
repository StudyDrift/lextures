package logging

import (
	"context"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
)

// SQLTrace logs parameterized query shape only — never bound values (plan 10.14 FR-6).
type SQLTrace struct{}

func (SQLTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	sql := strings.TrimSpace(data.SQL)
	if sql == "" {
		return ctx
	}
	slog.Debug("sql query",
		"sql", sql,
		"arg_count", len(data.Args),
	)
	return ctx
}

func (SQLTrace) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if data.Err != nil {
		slog.Debug("sql query end", "err", data.Err.Error())
	}
}
