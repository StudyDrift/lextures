package emailtemplates

import "sync/atomic"

var (
	savesTotal     atomic.Uint64
	testSendsTotal atomic.Uint64
)

// RecordSave increments email_template_saves_total.
func RecordSave() {
	savesTotal.Add(1)
}

// RecordTestSend increments email_template_test_sends_total.
func RecordTestSend() {
	testSendsTotal.Add(1)
}

// SavesTotal returns the save counter (for tests/metrics).
func SavesTotal() uint64 {
	return savesTotal.Load()
}

// TestSendsTotal returns the test-send counter (for tests/metrics).
func TestSendsTotal() uint64 {
	return testSendsTotal.Load()
}
