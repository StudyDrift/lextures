package logging

import (
	"sync/atomic"
)

// CredentialMetrics tracks credentials_issued_total and credential_verifications_total (plan 15.5).
type CredentialMetrics struct {
	issued         atomic.Uint64
	verifications  atomic.Uint64
}

// GlobalCredentialMetrics is incremented by credential HTTP handlers and issuance.
var GlobalCredentialMetrics = &CredentialMetrics{}

func (m *CredentialMetrics) IncIssued(sourceType string) {
	m.issued.Add(1)
	_ = sourceType
}

func (m *CredentialMetrics) IncVerifications() {
	m.verifications.Add(1)
}

func (m *CredentialMetrics) Snapshot() map[string]uint64 {
	return map[string]uint64{
		"credentials_issued_total":      m.issued.Load(),
		"credential_verifications_total": m.verifications.Load(),
	}
}

func (m *CredentialMetrics) Reset() {
	m.issued.Store(0)
	m.verifications.Store(0)
}