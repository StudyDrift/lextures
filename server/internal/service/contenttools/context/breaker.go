package context

import (
	"sync"
	"time"
)

const (
	hostFailThreshold = 3
	hostOpenDuration  = 5 * time.Minute
)

type hostBreakerState struct {
	failures int
	openUntil time.Time
}

var (
	hostBreakersMu sync.Mutex
	hostBreakers   = map[string]*hostBreakerState{}
)

// HostBreakerOpen reports whether fetches to host should be skipped (AC-9).
func HostBreakerOpen(host string) bool {
	host = stringsToLower(host)
	if host == "" {
		return false
	}
	hostBreakersMu.Lock()
	defer hostBreakersMu.Unlock()
	st := hostBreakers[host]
	if st == nil {
		return false
	}
	if st.openUntil.IsZero() {
		return false
	}
	if time.Now().Before(st.openUntil) {
		return true
	}
	// Half-open: allow a probe.
	st.openUntil = time.Time{}
	st.failures = 0
	return false
}

// HostBreakerRecordFailure increments consecutive failures; opens after threshold.
func HostBreakerRecordFailure(host string) {
	host = stringsToLower(host)
	if host == "" {
		return
	}
	hostBreakersMu.Lock()
	defer hostBreakersMu.Unlock()
	st := hostBreakers[host]
	if st == nil {
		st = &hostBreakerState{}
		hostBreakers[host] = st
	}
	st.failures++
	if st.failures >= hostFailThreshold {
		st.openUntil = time.Now().Add(hostOpenDuration)
	}
}

// HostBreakerRecordSuccess resets the breaker.
func HostBreakerRecordSuccess(host string) {
	host = stringsToLower(host)
	if host == "" {
		return
	}
	hostBreakersMu.Lock()
	defer hostBreakersMu.Unlock()
	delete(hostBreakers, host)
}

func stringsToLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
