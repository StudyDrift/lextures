package publicapi

import (
	"sync"
	"time"
)

const (
	defaultQuotaPerMinute = 1000
	rateWindow            = time.Minute
)

type tokenBucket struct {
	count   int
	resetAt time.Time
}

var (
	rateMu sync.Mutex
	rates  = map[string]*tokenBucket{}
)

// AllowToken returns whether the token may proceed and seconds until reset when denied.
func AllowToken(tokenKey string, quota int) (allowed bool, retryAfterSec int) {
	if tokenKey == "" {
		return true, 0
	}
	if quota <= 0 {
		quota = defaultQuotaPerMinute
	}
	now := time.Now().UTC()
	rateMu.Lock()
	defer rateMu.Unlock()
	b, ok := rates[tokenKey]
	if !ok || now.After(b.resetAt) {
		rates[tokenKey] = &tokenBucket{count: 1, resetAt: now.Add(rateWindow)}
		return true, 0
	}
	if b.count >= quota {
		sec := int(time.Until(b.resetAt).Seconds())
		if sec < 1 {
			sec = 1
		}
		return false, sec
	}
	b.count++
	return true, 0
}

// ResetRateLimits clears in-memory counters (tests).
func ResetRateLimits() {
	rateMu.Lock()
	rates = map[string]*tokenBucket{}
	rateMu.Unlock()
}
