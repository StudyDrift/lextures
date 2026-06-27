package logging

import (
	"sync/atomic"
)

// CCRMetrics tracks ccr_generated_total and ccr_verifications_total (plan 14.13).
type CCRMetrics struct {
	generated     atomic.Uint64
	verifications atomic.Uint64
}

// GlobalCCRMetrics is incremented by CCR HTTP handlers.
var GlobalCCRMetrics = &CCRMetrics{}

func (m *CCRMetrics) IncGenerated() {
	m.generated.Add(1)
}

func (m *CCRMetrics) IncVerifications() {
	m.verifications.Add(1)
}
