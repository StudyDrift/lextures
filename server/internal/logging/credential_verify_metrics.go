package logging

import "sync/atomic"

// CredentialVerifyMetrics tracks credential_verify_total{type,result} style counters (T08).
type CredentialVerifyMetrics struct {
	genuine  atomic.Uint64
	tampered atomic.Uint64
	revoked  atomic.Uint64
	notFound atomic.Uint64
	upload   atomic.Uint64
}

// GlobalCredentialVerifyMetrics is incremented by unified verify handlers.
var GlobalCredentialVerifyMetrics = &CredentialVerifyMetrics{}

func (m *CredentialVerifyMetrics) IncResult(result string) {
	switch result {
	case "genuine":
		m.genuine.Add(1)
	case "tampered":
		m.tampered.Add(1)
	case "revoked":
		m.revoked.Add(1)
	case "not_found":
		m.notFound.Add(1)
	}
}

func (m *CredentialVerifyMetrics) IncUpload() {
	m.upload.Add(1)
}
