package transcripts

import (
	"bytes"
	"strings"
	"testing"
)

func TestNetRevenueMinor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		total, refunded, want int64
	}{
		{0, 0, 0},
		{1500, 0, 1500},
		{1500, 500, 1000},
		{1500, 1500, 0},
		{1500, 2000, 0}, // clamped
	}
	for _, tc := range cases {
		got := NetRevenueMinor(tc.total, tc.refunded)
		if got != tc.want {
			t.Fatalf("NetRevenueMinor(%d,%d)=%d want %d", tc.total, tc.refunded, got, tc.want)
		}
	}
}

func TestPercentileHours(t *testing.T) {
	t.Parallel()
	if PercentileHours(nil, 50) != 0 {
		t.Fatal("empty should be 0")
	}
	one := []float64{10}
	if PercentileHours(one, 50) != 10 || PercentileHours(one, 95) != 10 {
		t.Fatalf("single sample: got p50=%v p95=%v", PercentileHours(one, 50), PercentileHours(one, 95))
	}
	sorted := []float64{1, 2, 3, 4, 5}
	if got := PercentileHours(sorted, 0); got != 1 {
		t.Fatalf("p0=%v", got)
	}
	if got := PercentileHours(sorted, 100); got != 5 {
		t.Fatalf("p100=%v", got)
	}
	if got := PercentileHours(sorted, 50); got != 3 {
		t.Fatalf("p50=%v want 3", got)
	}
}

func TestComputeTurnaroundStats(t *testing.T) {
	t.Parallel()
	empty := ComputeTurnaroundStats(nil)
	if empty.SampleSize != 0 || empty.AvgHours != 0 {
		t.Fatalf("empty: %+v", empty)
	}
	// Unsorted input should still produce correct percentiles.
	st := ComputeTurnaroundStats([]float64{40, 10, 20, 30, 50})
	if st.SampleSize != 5 {
		t.Fatalf("sample=%d", st.SampleSize)
	}
	if st.AvgHours != 30 {
		t.Fatalf("avg=%v want 30", st.AvgHours)
	}
	if st.P50Hours != 30 {
		t.Fatalf("p50=%v want 30", st.P50Hours)
	}
	if st.P95Hours < 45 || st.P95Hours > 50 {
		t.Fatalf("p95=%v expected near 48-50", st.P95Hours)
	}
}

func TestRate(t *testing.T) {
	t.Parallel()
	if Rate(0, 0) != 0 || Rate(5, 0) != 0 {
		t.Fatal("zero total")
	}
	if got := Rate(1, 4); got != 0.25 {
		t.Fatalf("got %v", got)
	}
	if got := Rate(10, 4); got != 1 {
		t.Fatalf("clamp got %v", got)
	}
}

func TestWriteDashboardCSV_ReconcilesSummary(t *testing.T) {
	t.Parallel()
	sum := DashboardSummary{
		OrgID:           "org-1",
		From:            "2026-01-01",
		To:              "2026-01-31",
		Orders:          10,
		Items:           12,
		Delivered:       8,
		OnHold:          1,
		Rejected:        1,
		Refunded:        2,
		NetRevenueMinor: 4500,
		HoldRate:        0.1,
		RejectionRate:   0.1,
		RefundRate:      0.2,
		Turnaround:      TurnaroundStats{SampleSize: 5, AvgHours: 12, P50Hours: 10, P90Hours: 20, P95Hours: 24},
		MethodMix:       []MethodMixBucket{{Method: "electronic_pdf", Count: 7}},
		TopDestinations: []DestinationBucket{{RecipientName: "State U", Count: 4}},
		Currency:        "usd",
	}
	var buf bytes.Buffer
	if err := WriteDashboardCSV(&buf, sum); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"orders,10",
		"net_revenue_minor,4500",
		"method_mix,electronic_pdf,7",
		"destination,State U,4",
		"turnaround_p95_hours,24.0000",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("csv missing %q\n%s", want, out)
		}
	}
}
