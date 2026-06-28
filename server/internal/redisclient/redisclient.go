// Package redisclient provides the shared Redis connection used by all app
// instances for cross-node state: JWT revocation blocklist (session:*),
// rate-limit counters (rate:*), token caches (token:*), feed caches (feed:*),
// and background job queues (jobs:*) — plan 17.2 (horizontal scaling).
//
// Redis is the shared-memory layer that makes the Axum/Go app tier horizontally
// scalable: any per-process mutable state that must be coherent across instances
// is moved here. When REDIS_URL is unset, New returns (nil, nil) and callers must
// degrade to single-instance behaviour (DB-backed where applicable).
package redisclient

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Default connection-pool sizing per app instance (plan 17.2 §6: min 5, max 20).
const (
	DefaultPoolMin = 5
	DefaultPoolMax = 20

	defaultDialTimeout  = 5 * time.Second
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second
)

// Config controls how the shared Redis client connects.
type Config struct {
	// URL is the connection string (redis:// or rediss:// for TLS). When empty,
	// New returns a nil client and Redis features are disabled.
	URL string
	// PoolMax is the maximum connections per instance (PoolSize). Defaults to 20.
	PoolMax int
	// PoolMin is the minimum idle connections (MinIdleConns). Defaults to 5.
	PoolMin int
}

// Client wraps the go-redis client with the namespaces used by Lextures.
type Client struct {
	rdb *redis.Client
}

// New builds the shared Redis client from cfg. It returns (nil, nil) when
// cfg.URL is empty so the server can run single-instance without Redis.
//
// rediss:// URLs enable TLS (DigitalOcean / ElastiCache managed Redis requires
// TLS — plan 17.2 §6 Security). New verifies connectivity with a PING.
func New(ctx context.Context, cfg Config) (*Client, error) {
	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		return nil, nil
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redisclient: parse URL: %w", err)
	}
	// ParseURL only sets a TLS config for rediss://. Managed Redis requires TLS;
	// ensure a sane minimum version when TLS is in play.
	if opts.TLSConfig != nil {
		opts.TLSConfig.MinVersion = tls.VersionTLS12
	}
	opts.PoolSize = poolMax(cfg.PoolMax)
	opts.MinIdleConns = poolMin(cfg.PoolMin)
	if opts.DialTimeout == 0 {
		opts.DialTimeout = defaultDialTimeout
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = defaultReadTimeout
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = defaultWriteTimeout
	}

	rdb := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, opts.DialTimeout)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redisclient: ping: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

// Redis exposes the underlying client for packages that need raw commands.
func (c *Client) Redis() *redis.Client {
	if c == nil {
		return nil
	}
	return c.rdb
}

// Ping checks connectivity; used by the readiness probe (plan 17.2 FR-1 / 17.8).
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("redisclient: not configured")
	}
	return c.rdb.Ping(ctx).Err()
}

// Close releases pooled connections.
func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

// Set stores a value with a TTL (ttl <= 0 means no expiry).
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("redisclient: not configured")
	}
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// Get returns the string value at key. Missing keys return ("", nil).
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if c == nil || c.rdb == nil {
		return "", fmt.Errorf("redisclient: not configured")
	}
	val, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// Del removes one or more keys. No-op when keys is empty.
func (c *Client) Del(ctx context.Context, keys ...string) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("redisclient: not configured")
	}
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

// DelByPrefix deletes all keys matching prefix*. Uses SCAN to avoid blocking Redis.
func (c *Client) DelByPrefix(ctx context.Context, prefix string) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("redisclient: not configured")
	}
	if prefix == "" {
		return nil
	}
	return c.DelByPattern(ctx, prefix+"*")
}

// DelByPattern deletes all keys matching a glob-style pattern (e.g. cache:user:*:calendar:course:abc).
func (c *Client) DelByPattern(ctx context.Context, pattern string) error {
	if c == nil || c.rdb == nil {
		return fmt.Errorf("redisclient: not configured")
	}
	if pattern == "" {
		return nil
	}
	var cursor uint64
	for {
		keys, next, err := c.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

// Exists reports whether key is present.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	if c == nil || c.rdb == nil {
		return false, fmt.Errorf("redisclient: not configured")
	}
	n, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// IncrWindow increments a fixed-window counter at key and returns the new count.
// The TTL is set to window on the first increment of a window (plan 17.2 §8
// rate:* namespace), so the counter is shared across all app instances (AC-3).
func (c *Client) IncrWindow(ctx context.Context, key string, window time.Duration) (int64, error) {
	if c == nil || c.rdb == nil {
		return 0, fmt.Errorf("redisclient: not configured")
	}
	n, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 && window > 0 {
		// Best-effort expiry; a missed TTL only widens the window, never leaks the limit open.
		_ = c.rdb.Expire(ctx, key, window).Err()
	}
	return n, nil
}

func poolMax(v int) int {
	if v <= 0 {
		return DefaultPoolMax
	}
	return v
}

func poolMin(v int) int {
	if v < 0 {
		return DefaultPoolMin
	}
	if v == 0 {
		return DefaultPoolMin
	}
	return v
}
