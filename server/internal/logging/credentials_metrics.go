package logging

import "sync/atomic"

// CredentialsMetrics tracks credentials_issued_total and credential_verifications_total (plans 15.5, 15.6).
type CredentialsMetrics struct {
	issued        atomic.Uint64
	verifications atomic.Uint64
	shares        atomic.Uint64
}

// GlobalCredentialsMetrics is incremented by credentials HTTP handlers.
var GlobalCredentialsMetrics = &CredentialsMetrics{}

func (m *CredentialsMetrics) IncIssued() {
	m.issued.Add(1)
}

func (m *CredentialsMetrics) IncVerifications() {
	m.verifications.Add(1)
}

func (m *CredentialsMetrics) IncShares() {
	m.shares.Add(1)
}
