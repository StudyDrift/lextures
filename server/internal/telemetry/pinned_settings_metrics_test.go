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
	if m.pinnedSettingsPinsGauge == nil {
		t.Fatal("pinnedSettingsPinsGauge not registered")
	}
	// Histogram: CollectAndCount returns metric family count (1 for a single histogram).
	if count := testutil.CollectAndCount(m.pinnedSettingsPinsGauge); count < 1 {
		t.Fatalf("pinnedSettingsPinsGauge metrics count = %d, want >= 1", count)
	}
	// Further observations must not panic.
	m.ObservePinnedSettingsPinCount(0)
	m.ObservePinnedSettingsPinCount(12)
}
