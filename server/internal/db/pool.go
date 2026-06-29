package db

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// NewPool opens a single shared pgx connection pool.
//
// For multi-instance (horizontally scaled) deployments, cap connections per
// instance via DB_POOL_MAX_CONNS so that instances × pool size stays under the
// Postgres max_connections limit (plan 17.2 FR-7 / AC-5). DB_POOL_MIN_CONNS
// keeps warm connections. Both default to the pgx library defaults when unset.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("db: empty DSN")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	var tracers []pgx.QueryTracer
	if sqlTraceEnabled() {
		tracers = append(tracers, logging.SQLTrace{})
	}
	// OTel DB spans are no-ops unless a request trace is active (plan 17.7 FR-2,
	// AC-2). The tracer self-checks for a recording span, so this adds no
	// overhead when tracing is disabled.
	tracers = append(tracers, telemetry.NewPgxTracer())
	switch len(tracers) {
	case 1:
		cfg.ConnConfig.Tracer = tracers[0]
	default:
		cfg.ConnConfig.Tracer = multiQueryTracer(tracers)
	}
	applyPoolSizing(cfg)
	return pgxpool.NewWithConfig(ctx, cfg)
}

// multiQueryTracer fans a pgx query trace out to several tracers (e.g. the slog
// SQL shape logger and the OpenTelemetry span tracer).
type multiQueryTracer []pgx.QueryTracer

func (m multiQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	for _, t := range m {
		ctx = t.TraceQueryStart(ctx, conn, data)
	}
	return ctx
}

func (m multiQueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	for _, t := range m {
		t.TraceQueryEnd(ctx, conn, data)
	}
}

// applyPoolSizing overrides pgx pool min/max from the environment. The DSN's own
// pool_max_conns query param (if present) is honored by ParseConfig; the env
// vars take precedence so deployments can tune sizing without rewriting the DSN.
func applyPoolSizing(cfg *pgxpool.Config) {
	if maxConns := intEnv("DB_POOL_MAX_CONNS"); maxConns > 0 {
		cfg.MaxConns = int32(maxConns)
	}
	if minConns := intEnv("DB_POOL_MIN_CONNS"); minConns > 0 {
		// MinConns may not exceed MaxConns.
		if cfg.MaxConns > 0 && int32(minConns) > cfg.MaxConns {
			minConns = int(cfg.MaxConns)
		}
		cfg.MinConns = int32(minConns)
	}
}

func intEnv(key string) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func sqlTraceEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_SQL"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
