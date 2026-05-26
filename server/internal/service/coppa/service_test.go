package coppa

import (
	"testing"
	"time"
)

func TestClassifyMinor(t *testing.T) {
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		dob   time.Time
		minor bool
	}{
		{time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC), true},  // age 11
		{time.Date(2013, 5, 26, 0, 0, 0, 0, time.UTC), false}, // exactly 13 today — not under 13
		{time.Date(2013, 5, 25, 0, 0, 0, 0, time.UTC), false}, // turned 13 yesterday
		{time.Date(2014, 5, 27, 0, 0, 0, 0, time.UTC), true},  // turns 12 tomorrow
		{time.Date(2010, 6, 1, 0, 0, 0, 0, time.UTC), false},  // age 15
		{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), false},  // adult
	}
	for _, tc := range cases {
		got := ClassifyMinor(tc.dob, now)
		if got != tc.minor {
			t.Errorf("ClassifyMinor(%v, %v) = %v, want %v", tc.dob.Format("2006-01-02"), now.Format("2006-01-02"), got, tc.minor)
		}
	}
}

func TestClassifyMinor_BirthdayEdge(t *testing.T) {
	// Someone born exactly 13 years ago today is no longer < 13.
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	dob := time.Date(2013, 5, 26, 0, 0, 0, 0, time.UTC)
	if ClassifyMinor(dob, now) {
		t.Errorf("expected not minor on exact 13th birthday")
	}
}

func TestGenerateToken_Unique(t *testing.T) {
	a, err := generateToken()
	if err != nil {
		t.Fatal(err)
	}
	b, err := generateToken()
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Errorf("generateToken produced duplicate tokens")
	}
	if len(a) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(a))
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	raw := "abc123"
	h1 := hashToken(raw)
	h2 := hashToken(raw)
	if h1 != h2 {
		t.Errorf("hashToken not deterministic")
	}
	if h1 == raw {
		t.Errorf("hashToken returned raw input")
	}
}

func TestHashToken_DifferentInputs(t *testing.T) {
	if hashToken("foo") == hashToken("bar") {
		t.Errorf("hashToken collision for different inputs")
	}
}
