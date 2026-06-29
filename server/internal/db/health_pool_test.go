package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewHealthPool(t *testing.T) {
	if testing.Short() {
		t.Skip("requires DATABASE_URL")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL for Postgres")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, err := NewHealthPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewHealthPool: %v", err)
	}
	defer p.Close()
	if p.Config().MaxConns != 1 {
		t.Fatalf("MaxConns = %d, want 1", p.Config().MaxConns)
	}
	if err := p.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}
