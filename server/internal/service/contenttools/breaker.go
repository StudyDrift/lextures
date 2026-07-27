package contenttools

import (
	"sync"
	"time"
)

const (
	// DefaultBreakerErrorThreshold trips after this many recent failures.
	DefaultBreakerErrorThreshold = 25
	// DefaultBreakerWindow is the sliding window for counting failures.
	DefaultBreakerWindow = 5 * time.Minute
)

// BreakerState is the circuit state for one tool.
type BreakerState struct {
	Open      bool
	OpenedAt  *time.Time
	Failures  int
	LastError string
}

// Breaker tracks per-tool failure counts and open/closed state (FR-15 / AC-8).
type Breaker struct {
	mu        sync.Mutex
	threshold int
	window    time.Duration
	byTool    map[string]*breakerEntry
}

type breakerEntry struct {
	open       bool
	openedAt   time.Time
	events     []time.Time
	lastError  string
}

// NewBreaker constructs a breaker with the given threshold and window.
func NewBreaker(threshold int, window time.Duration) *Breaker {
	if threshold <= 0 {
		threshold = DefaultBreakerErrorThreshold
	}
	if window <= 0 {
		window = DefaultBreakerWindow
	}
	return &Breaker{
		threshold: threshold,
		window:    window,
		byTool:    map[string]*breakerEntry{},
	}
}

var (
	defaultBreaker     *Breaker
	defaultBreakerOnce sync.Once
)

// DefaultBreaker returns the process-wide content-tool breaker.
func DefaultBreaker() *Breaker {
	defaultBreakerOnce.Do(func() {
		defaultBreaker = NewBreaker(DefaultBreakerErrorThreshold, DefaultBreakerWindow)
	})
	return defaultBreaker
}

func (b *Breaker) entry(toolID string) *breakerEntry {
	e, ok := b.byTool[toolID]
	if !ok {
		e = &breakerEntry{}
		b.byTool[toolID] = e
	}
	return e
}

func (b *Breaker) prune(e *breakerEntry, now time.Time) {
	cut := now.Add(-b.window)
	n := 0
	for _, t := range e.events {
		if t.After(cut) {
			e.events[n] = t
			n++
		}
	}
	e.events = e.events[:n]
}

// RecordFailure increments the failure count and may open the breaker.
func (b *Breaker) RecordFailure(toolID, errMsg string, now time.Time) BreakerState {
	if b == nil || toolID == "" {
		return BreakerState{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entry(toolID)
	b.prune(e, now)
	e.events = append(e.events, now)
	e.lastError = errMsg
	if !e.open && len(e.events) >= b.threshold {
		e.open = true
		e.openedAt = now
		SetBreakerStateGauge(toolID, 1)
	}
	return b.stateLocked(toolID, e)
}

// RecordSuccess clears recent failures when the breaker is closed (half-open soft reset).
func (b *Breaker) RecordSuccess(toolID string) {
	if b == nil || toolID == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entry(toolID)
	if e.open {
		return
	}
	e.events = e.events[:0]
}

// Reset closes the breaker (admin action).
func (b *Breaker) Reset(toolID string) BreakerState {
	if b == nil || toolID == "" {
		return BreakerState{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entry(toolID)
	e.open = false
	e.openedAt = time.Time{}
	e.events = e.events[:0]
	e.lastError = ""
	SetBreakerStateGauge(toolID, 0)
	return b.stateLocked(toolID, e)
}

// Open marks the breaker open (e.g. mirrored from DB).
func (b *Breaker) Open(toolID string, at time.Time) BreakerState {
	if b == nil || toolID == "" {
		return BreakerState{}
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entry(toolID)
	e.open = true
	e.openedAt = at
	SetBreakerStateGauge(toolID, 1)
	return b.stateLocked(toolID, e)
}

// IsOpen reports whether the tool is currently disabled by the breaker.
func (b *Breaker) IsOpen(toolID string) bool {
	if b == nil || toolID == "" {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.byTool[toolID]
	return e != nil && e.open
}

// State returns a snapshot for toolID.
func (b *Breaker) State(toolID string) BreakerState {
	if b == nil || toolID == "" {
		return BreakerState{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entry(toolID)
	b.prune(e, time.Now().UTC())
	return b.stateLocked(toolID, e)
}

func (b *Breaker) stateLocked(_ string, e *breakerEntry) BreakerState {
	st := BreakerState{Open: e.open, Failures: len(e.events), LastError: e.lastError}
	if e.open {
		t := e.openedAt
		st.OpenedAt = &t
	}
	return st
}
