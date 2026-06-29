package config

import "time"

// Rate-limit tier defaults (plan 17.6 §5 functional requirements). These can be
// overridden per deployment via the RATE_LIMIT_* environment variables.
const (
	// DefaultAuthPerMin is the per-IP limit on authentication endpoints (FR-1).
	DefaultAuthPerMin = 10
	// DefaultAuthPerHour is the per-IP hourly cap on authentication endpoints (FR-1).
	DefaultAuthPerHour = 50
	// DefaultGlobalPerMin is the per-IP limit across all non-API routes (FR-3).
	DefaultGlobalPerMin = 300
	// DefaultAPITokenPerMin is the default per-token public API quota (FR-2).
	DefaultAPITokenPerMin = 1000
)

// RateLimitRule is a request ceiling over a rolling window.
type RateLimitRule struct {
	// Limit is the maximum number of requests allowed within Window.
	Limit int
	// Window is the rolling window the limit applies over.
	Window time.Duration
}

// RateLimits holds the rate-limiting configuration (plan 17.6). All limiting is
// gated behind Enabled (feature flag rate_limiting_enabled, default false).
type RateLimits struct {
	// Enabled turns rate limiting on. Default false (monitoring-only rollout).
	Enabled bool
	// MonitorOnly logs would-be 429s without blocking (phase-1 rollout / AC observability).
	MonitorOnly bool
	// Auth is the per-minute per-IP limit for authentication endpoints (FR-1).
	Auth RateLimitRule
	// AuthHourly is the per-hour per-IP cap for authentication endpoints (FR-1).
	AuthHourly RateLimitRule
	// Global is the per-minute per-IP limit for general (non-API-token) traffic (FR-3).
	Global RateLimitRule
	// APIToken is the default per-minute per-token quota for the public API (FR-2).
	APIToken RateLimitRule
	// TrustedProxies are CIDRs whose X-Forwarded-For / X-Real-IP headers are
	// honoured for client-IP extraction (NFR security). Requests from peers
	// outside this set use the raw connection address, defeating IP spoofing.
	TrustedProxies []string
	// IPAllowlist are CIDRs that bypass all IP-based rate limits (FR-8 / AC-5).
	IPAllowlist []string
}

// DefaultRateLimits returns the rate-limit tiers built from the standard defaults.
func DefaultRateLimits() RateLimits {
	return RateLimits{
		Auth:       RateLimitRule{Limit: DefaultAuthPerMin, Window: time.Minute},
		AuthHourly: RateLimitRule{Limit: DefaultAuthPerHour, Window: time.Hour},
		Global:     RateLimitRule{Limit: DefaultGlobalPerMin, Window: time.Minute},
		APIToken:   RateLimitRule{Limit: DefaultAPITokenPerMin, Window: time.Minute},
	}
}

// rateLimitsFromEnv builds the rate-limit config from the standard defaults plus
// RATE_LIMIT_* overrides (plan 17.6 §15 rollout via feature flag).
func rateLimitsFromEnv() RateLimits {
	rl := DefaultRateLimits()
	rl.Enabled = boolEnv("RATE_LIMITING_ENABLED")
	rl.MonitorOnly = boolEnv("RATE_LIMIT_MONITOR_ONLY")
	rl.Auth.Limit = intEnvDefault("RATE_LIMIT_AUTH_PER_MIN", DefaultAuthPerMin)
	rl.AuthHourly.Limit = intEnvDefault("RATE_LIMIT_AUTH_PER_HOUR", DefaultAuthPerHour)
	rl.Global.Limit = intEnvDefault("RATE_LIMIT_GLOBAL_PER_MIN", DefaultGlobalPerMin)
	rl.APIToken.Limit = intEnvDefault("RATE_LIMIT_API_TOKEN_PER_MIN", DefaultAPITokenPerMin)
	rl.TrustedProxies = commaSeparatedEnv("RATE_LIMIT_TRUSTED_PROXIES")
	rl.IPAllowlist = commaSeparatedEnv("RATE_LIMIT_IP_ALLOWLIST")
	return rl
}
