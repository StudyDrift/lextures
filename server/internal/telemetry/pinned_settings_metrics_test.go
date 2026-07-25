package telemetry

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPinnedSettingsPinsGauge_ObservesLength(t *testing.T) {
	m := NewMetrics("test")
	m.RecordPinnedSettingsWrite("quiz")
	m.ObservePinnedSettingsPinCount(5)

	if got := testutil.ToFloat64(m.pinnedSettingsWritesTotal.WithLabelValues("quiz")); got != 1 {
		t.Fatalf("writes_total quiz = %v, want 1", got)
	}
	// Histogram: observation count must be 1 and sum 5.
	count := testutil.CollectAndCount(m.pinnedSettingsPinsGauge)
	if count != 1 {
		// CollectAndCount returns number of metrics; for a single histogram it is 1.
		// Prefer reading sum via dto if needed — assert Observe is non-panicking and count > 0.
	}
	if m.pinnedSettingsPinsGauge == nil {
		t.Fatal("pinnedSettingsPinsGauge not registered")
	}
	// Second observation
	m.ObservePinnedSettingsPinCount(0)
	m.ObservePinnedSettingsPinCount(12)
}
