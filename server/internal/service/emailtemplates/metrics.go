package emailtemplates

import "sync/atomic"

var (
	savesTotal       atomic.Uint64
	testSendsTotal   atomic.Uint64
	compileOKTotal   atomic.Uint64
	compileFailTotal atomic.Uint64
	// fallbackTotal counts render fall-throughs to built-in code defaults.
	fallbackTotal atomic.Uint64
)

// RecordSave increments email_template_saves_total.
func RecordSave() {
	savesTotal.Add(1)
}

// RecordTestSend increments email_template_test_sends_total.
func RecordTestSend() {
	testSendsTotal.Add(1)
}

// RecordCompile records a compile attempt (success or failure).
func RecordCompile(ok bool) {
	if ok {
		compileOKTotal.Add(1)
	} else {
		compileFailTotal.Add(1)
	}
}

// RecordFallback increments email_template_render_fallback_total.
func RecordFallback() {
	fallbackTotal.Add(1)
}

// SavesTotal returns the save counter (for tests/metrics).
func SavesTotal() uint64 {
	return savesTotal.Load()
}

// TestSendsTotal returns the test-send counter (for tests/metrics).
func TestSendsTotal() uint64 {
	return testSendsTotal.Load()
}

// CompileOKTotal returns successful compile count.
func CompileOKTotal() uint64 {
	return compileOKTotal.Load()
}

// CompileFailTotal returns failed compile count.
func CompileFailTotal() uint64 {
	return compileFailTotal.Load()
}

// FallbackTotal returns render-fallback count (for tests/metrics).
func FallbackTotal() uint64 {
	return fallbackTotal.Load()
}
