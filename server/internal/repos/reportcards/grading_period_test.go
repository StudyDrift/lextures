package reportcards

import (
	"testing"
	"time"
)

func TestParseGradingPeriodLabel_Quarters(t *testing.T) {
	dr, ok := parseGradingPeriodLabel("Q1-2026")
	if !ok {
		t.Fatal("expected Q1-2026 to parse")
	}
	if dr.Start != time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("start: got %v", dr.Start)
	}
	if dr.End != time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("end: got %v", dr.End)
	}

	dr, ok = parseGradingPeriodLabel("q3-2025")
	if !ok {
		t.Fatal("expected q3-2025 to parse")
	}
	if dr.Start.Month() != time.July || dr.End.Month() != time.September {
		t.Fatalf("Q3 range: %v – %v", dr.Start, dr.End)
	}
}

func TestParseGradingPeriodLabel_Semesters(t *testing.T) {
	dr, ok := parseGradingPeriodLabel("S1-2026")
	if !ok {
		t.Fatal("expected S1-2026 to parse")
	}
	if dr.Start.Month() != time.January || dr.End.Month() != time.June {
		t.Fatalf("S1 range: %v – %v", dr.Start, dr.End)
	}

	dr, ok = parseGradingPeriodLabel("S2-2026")
	if !ok {
		t.Fatal("expected S2-2026 to parse")
	}
	if dr.Start.Month() != time.July || dr.End.Month() != time.December {
		t.Fatalf("S2 range: %v – %v", dr.Start, dr.End)
	}
}

func TestParseGradingPeriodLabel_Unknown(t *testing.T) {
	if _, ok := parseGradingPeriodLabel("Fall 2026"); ok {
		t.Fatal("expected unknown label to fail")
	}
}