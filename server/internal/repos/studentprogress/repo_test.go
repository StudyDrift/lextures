package studentprogress

import "testing"

func TestPct(t *testing.T) {
	if got := Pct(3, 5); got != 60 {
		t.Fatalf("Pct(3,5) = %v, want 60", got)
	}
	if got := Pct(0, 0); got != 0 {
		t.Fatalf("Pct(0,0) = %v, want 0", got)
	}
}
