package logging

import "sync/atomic"

// DiplomaMetrics tracks diploma issuance/revocation/batch counters (T11).
type DiplomaMetrics struct {
	issuedDiploma     atomic.Uint64
	issuedCertificate atomic.Uint64
	revoked           atomic.Uint64
	batchOK           atomic.Uint64
	batchFail         atomic.Uint64
	batchSkip         atomic.Uint64
}

// GlobalDiplomaMetrics is incremented by diploma issue handlers and jobs.
var GlobalDiplomaMetrics = &DiplomaMetrics{}

func (m *DiplomaMetrics) IncIssued(kind string) {
	switch kind {
	case "certificate":
		m.issuedCertificate.Add(1)
	default:
		m.issuedDiploma.Add(1)
	}
}

func (m *DiplomaMetrics) IncRevoked()    { m.revoked.Add(1) }
func (m *DiplomaMetrics) IncBatchOK()    { m.batchOK.Add(1) }
func (m *DiplomaMetrics) IncBatchFail()  { m.batchFail.Add(1) }
func (m *DiplomaMetrics) IncBatchSkip()  { m.batchSkip.Add(1) }

func (m *DiplomaMetrics) Snapshot() (issuedDiploma, issuedCertificate, revoked, batchOK, batchFail, batchSkip uint64) {
	return m.issuedDiploma.Load(), m.issuedCertificate.Load(), m.revoked.Load(),
		m.batchOK.Load(), m.batchFail.Load(), m.batchSkip.Load()
}
