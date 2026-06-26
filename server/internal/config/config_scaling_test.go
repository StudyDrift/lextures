package config

import (
	"testing"

	"github.com/lextures/lextures/server/internal/redisclient"
)

func TestLoad_ScalingDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", "0123456789012345678901234567890123")
	// Ensure scaling vars are unset for the defaults case.
	for _, k := range []string{"REDIS_URL", "REDIS_POOL_MIN", "REDIS_POOL_MAX", "DB_POOL_MAX_CONNS", "DB_POOL_MIN_CONNS", "SHUTDOWN_TIMEOUT_SECS"} {
		t.Setenv(k, "")
	}

	c := Load()
	if c.RedisURL != "" {
		t.Fatalf("RedisURL default = %q, want empty", c.RedisURL)
	}
	if c.RedisPoolMin != redisclient.DefaultPoolMin {
		t.Fatalf("RedisPoolMin = %d, want %d", c.RedisPoolMin, redisclient.DefaultPoolMin)
	}
	if c.RedisPoolMax != redisclient.DefaultPoolMax {
		t.Fatalf("RedisPoolMax = %d, want %d", c.RedisPoolMax, redisclient.DefaultPoolMax)
	}
	if c.DBPoolMaxConns != 0 || c.DBPoolMinConns != 0 {
		t.Fatalf("DB pool conns default should be 0, got max=%d min=%d", c.DBPoolMaxConns, c.DBPoolMinConns)
	}
	if c.ShutdownTimeoutSecs != defaultShutdownTimeoutSecs {
		t.Fatalf("ShutdownTimeoutSecs = %d, want %d", c.ShutdownTimeoutSecs, defaultShutdownTimeoutSecs)
	}
}

func TestLoad_ScalingOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", "0123456789012345678901234567890123")
	t.Setenv("REDIS_URL", "rediss://cache.example:6379")
	t.Setenv("REDIS_POOL_MIN", "8")
	t.Setenv("REDIS_POOL_MAX", "40")
	t.Setenv("DB_POOL_MAX_CONNS", "20")
	t.Setenv("DB_POOL_MIN_CONNS", "4")
	t.Setenv("SHUTDOWN_TIMEOUT_SECS", "45")

	c := Load()
	if c.RedisURL != "rediss://cache.example:6379" {
		t.Fatalf("RedisURL = %q", c.RedisURL)
	}
	if c.RedisPoolMin != 8 || c.RedisPoolMax != 40 {
		t.Fatalf("redis pool = %d/%d", c.RedisPoolMin, c.RedisPoolMax)
	}
	if c.DBPoolMaxConns != 20 || c.DBPoolMinConns != 4 {
		t.Fatalf("db pool = %d/%d", c.DBPoolMaxConns, c.DBPoolMinConns)
	}
	if c.ShutdownTimeoutSecs != 45 {
		t.Fatalf("shutdown = %d", c.ShutdownTimeoutSecs)
	}
}

func TestIntEnvDefault(t *testing.T) {
	t.Setenv("X_INT", "")
	if got := intEnvDefault("X_INT", 7); got != 7 {
		t.Fatalf("empty -> %d want 7", got)
	}
	t.Setenv("X_INT", "abc")
	if got := intEnvDefault("X_INT", 7); got != 7 {
		t.Fatalf("invalid -> %d want 7", got)
	}
	t.Setenv("X_INT", "-3")
	if got := intEnvDefault("X_INT", 7); got != 7 {
		t.Fatalf("negative -> %d want 7", got)
	}
	t.Setenv("X_INT", "11")
	if got := intEnvDefault("X_INT", 7); got != 11 {
		t.Fatalf("valid -> %d want 11", got)
	}
}
