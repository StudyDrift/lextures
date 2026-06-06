package conferences

import (
	"testing"
	"time"
)

func TestGenerateSlotTimes_AC1(t *testing.T) {
	// AC-1: Nov 18, 4–6 PM, 15-min slots, 5-min gaps → 6 slots at 4:00, 4:20, 4:40, 5:00, 5:20, 5:40
	start := time.Date(2025, 11, 18, 16, 0, 0, 0, time.UTC)
	end := time.Date(2025, 11, 18, 18, 0, 0, 0, time.UTC)
	got := GenerateSlotTimes(start, end, 15, 5)
	if len(got) != 6 {
		t.Fatalf("expected 6 slots, got %d: %v", len(got), got)
	}
	wantHours := []int{16, 16, 16, 17, 17, 17}
	wantMins := []int{0, 20, 40, 0, 20, 40}
	for i, ts := range got {
		if ts.Hour() != wantHours[i] || ts.Minute() != wantMins[i] {
			t.Fatalf("slot %d = %v, want %02d:%02d", i, ts, wantHours[i], wantMins[i])
		}
	}
}

func TestGenerateSlotTimes_NoGap(t *testing.T) {
	start := time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	got := GenerateSlotTimes(start, end, 30, 0)
	if len(got) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(got))
	}
}

func TestGenerateSlotTimes_EmptyWindow(t *testing.T) {
	start := time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
	got := GenerateSlotTimes(start, end, 15, 5)
	if len(got) != 0 {
		t.Fatalf("expected 0 slots, got %d", len(got))
	}
}

func TestAllowedSlotDurations(t *testing.T) {
	for _, d := range []int{5, 10, 15, 20, 30} {
		if !AllowedSlotDurations[d] {
			t.Fatalf("expected %d to be allowed", d)
		}
	}
	for _, d := range []int{0, 7, 45, 60} {
		if AllowedSlotDurations[d] {
			t.Fatalf("expected %d to be rejected", d)
		}
	}
}
