package logging

import (
	"sync"
	"sync/atomic"
)

// CCRMetrics tracks ccr_generated_total and ccr_verifications_total (plan 14.13).
type CCRMetrics struct {
	generated     atomic.Uint64
	verifications atomic.Uint64
	mu            sync.RWMutex
}

// GlobalCCRMetrics is incremented by CCR HTTP handlers.
var GlobalCCRMetrics = &CCRMetrics{}

func (m *CCRMetrics) IncGenerated() {
	m.generated.Add(1)
}

func (m *CCRMetrics) IncVerifications() {
	m.verifications.Add(1)
}

func (m *CCRMetrics) Snapshot() map[string]uint64 {
	return map[string]uint64{
		"ccr_generated_total":     m.generated.Load(),
		"ccr_verifications_total": m.verifications.Load(),
	}
}

func (m *CCRMetrics) Reset() {
	m.generated.Store(0)
	m.verifications.Store(0)
}
