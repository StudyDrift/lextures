package db

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/logging"
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
	if sqlTraceEnabled() {
		cfg.ConnConfig.Tracer = logging.SQLTrace{}
	}
	applyPoolSizing(cfg)
	return pgxpool.NewWithConfig(ctx, cfg)
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
