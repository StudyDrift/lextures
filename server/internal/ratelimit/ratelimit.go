// Package ratelimit implements Redis-backed request rate limiting with a
// sliding-window-log algorithm (plan 17.6). Counters live in Redis so limits
// are enforced consistently across all app instances (AC-3); when Redis is
// unavailable the limiter fails open (AC-4) so a Redis outage never takes the
// service down.
package ratelimit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/lextures/lextures/server/internal/config"
)

// slidingWindow is an atomic sliding-window-log check. It removes entries older
// than the window, counts what remains, and (only when under the limit) records
// the new request. Denied requests are NOT recorded, so a flood of blocked
// requests cannot extend the window or be used to enumerate the limit
// (plan 17.6 open question #3). Returns {allowed, remaining, retry_after_ms}.
var slidingWindow = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)
if count < limit then
  redis.call('ZADD', key, now, member)
  redis.call('PEXPIRE', key, window)
  return {1, limit - count - 1, 0}
end
local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
local retry = window
if oldest[2] then
  retry = (tonumber(oldest[2]) + window) - now
  if retry < 1 then retry = 1 end
end
redis.call('PEXPIRE', key, window)
return {0, 0, retry}
`)

// Limiter enforces rate limits against a shared Redis instance.
type Limiter struct {
	rdb       *redis.Client
	hmacKey   []byte
	allowlist []*net.IPNet
	trusted   []*net.IPNet
}

// New builds a Limiter. rdb may be nil (the limiter then always fails open).
// hmacKey keys the HMAC used to hash IPs in Redis keys so the key space cannot
// be enumerated back to raw IPs (plan 17.6 NFR privacy).
func New(rdb *redis.Client, hmacKey string, cfg config.RateLimits) *Limiter {
	return &Limiter{
		rdb:       rdb,
		hmacKey:   []byte(hmacKey),
		allowlist: ParseCIDRs(cfg.IPAllowlist),
		trusted:   ParseCIDRs(cfg.TrustedProxies),
	}
}

// TrustedProxies returns the configured trusted-proxy networks for IP extraction.
func (l *Limiter) TrustedProxies() []*net.IPNet { return l.trusted }

// Allowlisted reports whether ip bypasses IP-based limits (plan 17.6 FR-8 / AC-5).
func (l *Limiter) Allowlisted(ip string) bool {
	return ipInNets(ip, l.allowlist)
}

// Decision is the outcome of a limit check, including the values needed for the
// X-RateLimit-* and Retry-After response headers (plan 17.6 FR-4).
type Decision struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter int   // seconds until a slot frees (0 when allowed)
	Reset      int64 // unix seconds when the window resets
}

// hashIP returns an HMAC-SHA256 hex digest of the IP for use in Redis keys.
func (l *Limiter) hashIP(ip string) string {
	mac := hmac.New(sha256.New, l.hmacKey)
	_, _ = mac.Write([]byte(ip))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// IPKey builds the Redis key for a per-IP limit group (rl:ip:{hash}:{group}).
func (l *Limiter) IPKey(ip, group string) string {
	return "rl:ip:" + l.hashIP(ip) + ":" + group
}

// UserKey builds the Redis key for a per-user limit (rl:user:{id}:{group}).
func (l *Limiter) UserKey(userID, group string) string {
	return "rl:user:" + userID + ":" + group
}

// TokenKey builds the Redis key for a per-token limit (rl:token:{id}:api).
func (l *Limiter) TokenKey(tokenID, group string) string {
	return "rl:token:" + tokenID + ":" + group
}

// Allow runs the sliding-window check for key under rule. On any Redis error it
// fails open (Allowed=true) and records rate_limit_redis_miss_total (AC-4).
func (l *Limiter) Allow(ctx context.Context, key string, rule config.RateLimitRule, lt LimitType) Decision {
	now := time.Now()
	if l.rdb == nil || rule.Limit <= 0 {
		return failOpen(rule, now)
	}
	windowMs := rule.Window.Milliseconds()
	res, err := slidingWindow.Run(ctx, l.rdb, []string{key},
		now.UnixMilli(), windowMs, rule.Limit, uuid.NewString()).Result()
	if err != nil {
		RecordRedisMiss(lt)
		return failOpen(rule, now)
	}
	vals, ok := res.([]interface{})
	if !ok || len(vals) < 3 {
		RecordRedisMiss(lt)
		return failOpen(rule, now)
	}
	allowed := toInt(vals[0]) == 1
	remaining := toInt(vals[1])
	retryMs := toInt(vals[2])
	d := Decision{
		Allowed:   allowed,
		Limit:     rule.Limit,
		Remaining: remaining,
	}
	if allowed {
		d.Reset = now.Add(rule.Window).Unix()
		return d
	}
	retrySec := (retryMs + 999) / 1000
	if retrySec < 1 {
		retrySec = 1
	}
	d.RetryAfter = retrySec
	d.Reset = now.Add(time.Duration(retrySec) * time.Second).Unix()
	return d
}

func failOpen(rule config.RateLimitRule, now time.Time) Decision {
	rem := rule.Limit
	if rem < 0 {
		rem = 0
	}
	return Decision{
		Allowed:   true,
		Limit:     rule.Limit,
		Remaining: rem,
		Reset:     now.Add(rule.Window).Unix(),
	}
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case int:
		return n
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}
