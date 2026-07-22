package httpserver

import (
	"net/http"
	"strconv"

	"github.com/redis/go-redis/v9"

	"github.com/lextures/lextures/server/internal/publicapi"
	"github.com/lextures/lextures/server/internal/ratelimit"
)

// authSensitivePaths get the stricter per-IP auth limit (plan 17.6 FR-1):
// login, registration, and password-reset request endpoints.
var authSensitivePaths = map[string]bool{
	"/api/v1/auth/login":              true,
	"/api/v1/auth/signup":             true,
	"/api/v1/auth/forgot-password":    true,
	"/api/v1/auth/reset-password":     true,
	"/api/v1/auth/parent-invite/consume": true,
	"/api/v1/auth/magic-link/request": true,
	"/api/v1/auth/oidc/apple/native":  true,
	"/api/v1/auth/oidc/google/native": true,
}

// buildRateLimiter constructs the request limiter from config + Redis. IPs are
// hashed with the platform secret so rate-limit keys cannot be enumerated back
// to raw IPs (plan 17.6 NFR privacy).
func (d Deps) buildRateLimiter() *ratelimit.Limiter {
	var rdb *redis.Client
	if d.Redis != nil {
		rdb = d.Redis.Redis()
	}
	return ratelimit.New(rdb, d.apiTokenIPHashKey(), d.effectiveConfig().RateLimits)
}

// rateLimitMiddleware enforces per-IP auth and global limits (plan 17.6 FR-1,
// FR-3). It must run before chi's RealIP so it sees the genuine TCP peer and can
// reject forged X-Forwarded-For headers from untrusted clients (NFR security).
// Per-token API limits are enforced separately in publicAPIMiddleware (FR-2).
func (d Deps) rateLimitMiddleware(limiter *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := d.effectiveConfig().RateLimits
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}
			ip := ratelimit.ClientIP(r.RemoteAddr, r.Header, limiter.TrustedProxies())
			if ip == "" || limiter.Allowlisted(ip) {
				next.ServeHTTP(w, r)
				return
			}
			ctx := r.Context()

			// Authentication endpoints: strict per-IP minute + hourly caps (FR-1).
			if authSensitivePaths[r.URL.Path] {
				perMin := limiter.Allow(ctx, limiter.IPKey(ip, "auth"), cfg.Auth, ratelimit.LimitTypeAuth)
				perHour := limiter.Allow(ctx, limiter.IPKey(ip, "auth:h"), cfg.AuthHourly, ratelimit.LimitTypeAuth)
				dec := mostRestrictive(perMin, perHour)
				if !dec.Allowed {
					if d.rejectRateLimited(w, r, dec, "auth", ratelimit.LimitTypeAuth, cfg.MonitorOnly) {
						return
					}
				} else {
					writeRateLimitHeaders(w, dec)
				}
			}

			// Global per-IP limit for browser/session traffic (FR-3). API-token
			// clients are governed by their per-token quota instead (FR-7), so we
			// skip the global IP limit for Bearer-token API requests.
			if !publicapi.IsAccessKeyRequest(r) {
				dec := limiter.Allow(ctx, limiter.IPKey(ip, "global"), cfg.Global, ratelimit.LimitTypeGlobal)
				if !dec.Allowed {
					if d.rejectRateLimited(w, r, dec, "global", ratelimit.LimitTypeGlobal, cfg.MonitorOnly) {
						return
					}
				} else {
					writeRateLimitHeaders(w, dec)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// mostRestrictive returns the denied decision when either is denied, otherwise
// the one advertising fewer remaining requests.
func mostRestrictive(a, b ratelimit.Decision) ratelimit.Decision {
	if !a.Allowed {
		return a
	}
	if !b.Allowed {
		return b
	}
	if b.Remaining < a.Remaining {
		return b
	}
	return a
}

// rejectRateLimited records the metric and, unless monitor-only mode is on,
// writes a 429. Returns true when the request was rejected (caller must stop).
func (d Deps) rejectRateLimited(w http.ResponseWriter, r *http.Request, dec ratelimit.Decision, group string, lt ratelimit.LimitType, monitorOnly bool) bool {
	ratelimit.RecordExceeded(group, lt)
	if monitorOnly {
		return false
	}
	writeRateLimitHeaders(w, dec)
	w.Header().Set("Retry-After", strconv.Itoa(dec.RetryAfter))
	detail := "Rate limit of " + strconv.Itoa(dec.Limit) + " requests exceeded. Retry after " + strconv.Itoa(dec.RetryAfter) + " seconds."
	publicapi.WriteProblem(w, publicapi.Problem{
		Type:     "https://lextures.io/errors/rate-limited",
		Title:    "Too Many Requests",
		Status:   http.StatusTooManyRequests,
		Detail:   detail,
		Instance: r.URL.Path,
	})
	return true
}

// writeRateLimitHeaders sets the standard X-RateLimit-* headers (FR-4).
func writeRateLimitHeaders(w http.ResponseWriter, dec ratelimit.Decision) {
	h := w.Header()
	h.Set("X-RateLimit-Limit", strconv.Itoa(dec.Limit))
	h.Set("X-RateLimit-Remaining", strconv.Itoa(dec.Remaining))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(dec.Reset, 10))
}
