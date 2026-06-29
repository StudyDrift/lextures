package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewHealthPool opens a dedicated 1-connection pool for readiness probes so
// health checks do not compete with request handlers for main-pool connections
// (plan 17.8 NFR Reliability).
func NewHealthPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("db: empty DSN")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 1
	cfg.MinConns = 0
	return pgxpool.NewWithConfig(ctx, cfg)
}
