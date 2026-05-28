package logging

import (
	"sync"
	"sync/atomic"
)

// RedactionMetrics tracks pii_redactions_total{field_name} (plan 10.14, cross-link 17.7).
type RedactionMetrics struct {
	mu      sync.RWMutex
	byField map[string]*atomic.Uint64
}

// GlobalRedactionMetrics is updated by the redacting slog handler.
var GlobalRedactionMetrics = &RedactionMetrics{byField: make(map[string]*atomic.Uint64)}

func (m *RedactionMetrics) Inc(field string) {
	field = normalizeFieldName(field)
	if field == "" {
		return
	}
	m.mu.RLock()
	c, ok := m.byField[field]
	m.mu.RUnlock()
	if ok {
		c.Add(1)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok = m.byField[field]; !ok {
		c = &atomic.Uint64{}
		m.byField[field] = c
	}
	c.Add(1)
}

// Snapshot returns field_name → count for exposition (e.g. redaction-status API).
func (m *RedactionMetrics) Snapshot() map[string]uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]uint64, len(m.byField))
	for k, v := range m.byField {
		out[k] = v.Load()
	}
	return out
}

// Reset clears counters (tests only).
func (m *RedactionMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byField = make(map[string]*atomic.Uint64)
}
