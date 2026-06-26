// Package jwtblocklist implements the Redis-backed JWT revocation blocklist
// (plan 17.2 FR-4). Revoked access-token `jti` claims are stored under
// session:jti:{jti} with a TTL equal to the token's remaining lifetime, so any
// app instance enforces revocation within the propagation window and entries
// self-expire without a sweep job.
package jwtblocklist

import (
	"context"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/redisclient"
)

// keyPrefix namespaces revocation entries (plan 17.2 §8 data model).
const keyPrefix = "session:jti:"

// Redis is a redisclient-backed auth.Blocklist.
type Redis struct {
	c *redisclient.Client
}

// New returns a blocklist backed by c, or nil when c is nil so callers can
// detect the disabled (single-instance) configuration.
func New(c *redisclient.Client) *Redis {
	if c == nil {
		return nil
	}
	return &Redis{c: c}
}

// IsRevoked reports whether jti is present in the blocklist.
func (r *Redis) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if r == nil || r.c == nil || strings.TrimSpace(jti) == "" {
		return false, nil
	}
	return r.c.Exists(ctx, key(jti))
}

// Revoke records jti as revoked for ttl. ttl <= 0 is a no-op (already expired).
func (r *Redis) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if r == nil || r.c == nil || strings.TrimSpace(jti) == "" || ttl <= 0 {
		return nil
	}
	return r.c.Set(ctx, key(jti), "revoked", ttl)
}

func key(jti string) string {
	return keyPrefix + jti
}
