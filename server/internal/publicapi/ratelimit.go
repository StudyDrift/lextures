package publicapi

import (
	"sync"
	"time"
)

// TokenLimiter enforces per-token request quotas with an in-memory sliding window.
// Production deployments should replace this with Redis-backed counters (plan 17.6).
type TokenLimiter struct {
	mu       sync.Mutex
	windows  map[string]*window
	limit    int
	windowDur time.Duration
}

type window struct {
	count    int
	resetAt  time.Time
}

// NewTokenLimiter creates a limiter allowing limit requests per window duration.
func NewTokenLimiter(limit int, windowDur time.Duration) *TokenLimiter {
	if limit <= 0 {
		limit = 1000
	}
	if windowDur <= 0 {
		windowDur = time.Minute
	}
	return &TokenLimiter{
		windows:   make(map[string]*window),
		limit:     limit,
		windowDur: windowDur,
	}
}

// Allow reports whether the key may proceed and seconds until reset when denied.
func (l *TokenLimiter) Allow(key string, now time.Time) (ok bool, retryAfterSec int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, exists := l.windows[key]
	if !exists || now.After(w.resetAt) {
		l.windows[key] = &window{count: 1, resetAt: now.Add(l.windowDur)}
		return true, 0
	}
	if w.count >= l.limit {
		sec := int(w.resetAt.Sub(now).Seconds())
		if sec < 1 {
			sec = 1
		}
		return false, sec
	}
	w.count++
	return true, 0
}

// DefaultLimiter is the process-wide public API token rate limiter.
var DefaultLimiter = NewTokenLimiter(1000, time.Minute)
