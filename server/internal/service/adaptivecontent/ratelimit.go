package adaptivecontent

import (
	"sync"
	"time"
)

// GlobalRateLimiter is a simple token-bucket for adaptive-content model calls
// (AC.4 FR-5). Process-local; multi-instance deployments each hold a share of
// the platform ceiling (conservative enough for v1).
type GlobalRateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	last       time.Time
}

// DefaultGlobalRate is the max sustained adaptive-content completions per second platform-wide (per process).
const DefaultGlobalRate = 2.0

// DefaultGlobalBurst is the max concurrent burst of model calls.
const DefaultGlobalBurst = 8.0

var (
	globalLimiterOnce sync.Once
	globalLimiter     *GlobalRateLimiter
)

// GlobalModelLimiter returns the process-wide adaptive-content rate limiter.
func GlobalModelLimiter() *GlobalRateLimiter {
	globalLimiterOnce.Do(func() {
		globalLimiter = NewGlobalRateLimiter(DefaultGlobalRate, DefaultGlobalBurst)
	})
	return globalLimiter
}

// NewGlobalRateLimiter constructs a token bucket.
func NewGlobalRateLimiter(ratePerSec, burst float64) *GlobalRateLimiter {
	if ratePerSec <= 0 {
		ratePerSec = DefaultGlobalRate
	}
	if burst <= 0 {
		burst = DefaultGlobalBurst
	}
	return &GlobalRateLimiter{
		tokens:     burst,
		maxTokens:  burst,
		refillRate: ratePerSec,
		last:       time.Now(),
	}
}

// TryAcquire attempts to take one token. Returns false when rate-limited.
func (l *GlobalRateLimiter) TryAcquire() bool {
	if l == nil {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(l.last).Seconds()
	if elapsed > 0 {
		l.tokens += elapsed * l.refillRate
		if l.tokens > l.maxTokens {
			l.tokens = l.maxTokens
		}
		l.last = now
	}
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

// ResetForTest fills the bucket (tests only).
func (l *GlobalRateLimiter) ResetForTest() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tokens = l.maxTokens
	l.last = time.Now()
}
