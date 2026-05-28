package securityreports

import (
	"testing"
	"time"
)

func TestPatchSLADays(t *testing.T) {
	tests := []struct {
		severity string
		want     int
	}{
		{"critical", 7},
		{"high", 30},
		{"medium", 90},
		{"low", 0},
		{"informational", 0},
	}
	for _, tc := range tests {
		if got := PatchSLADays(tc.severity); got != tc.want {
			t.Errorf("PatchSLADays(%q) = %d want %d", tc.severity, got, tc.want)
		}
	}
}

func TestComputeSLAMet_CriticalWithinSLA(t *testing.T) {
	reportDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	patchDate := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	severity := "critical"
	met := ComputeSLAMet(reportDate, &patchDate, severity)
	if met == nil || !*met {
		t.Fatalf("expected sla_met true, got %v", met)
	}
}

func TestComputeSLAMet_CriticalMissedSLA(t *testing.T) {
	reportDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	patchDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	severity := "critical"
	met := ComputeSLAMet(reportDate, &patchDate, severity)
	if met == nil || *met {
		t.Fatalf("expected sla_met false, got %v", met)
	}
}

func TestComputeSLAMet_LowNotApplicable(t *testing.T) {
	reportDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	patchDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	met := ComputeSLAMet(reportDate, &patchDate, "low")
	if met != nil {
		t.Fatalf("expected nil sla_met for low severity, got %v", *met)
	}
}
