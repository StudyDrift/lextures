package db

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/logging"
)

// NewPool opens a single shared pgx connection pool.
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
	// default pool settings are fine; tune max conns in production
	return pgxpool.NewWithConfig(ctx, cfg)
}

func sqlTraceEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_SQL"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
