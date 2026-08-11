package coupons

import "testing"

func TestLowEntropyWarnings(t *testing.T) {
	if w := LowEntropyWarnings("AB12"); len(w) != 1 || w[0] != WarningLowEntropy {
		t.Fatalf("short code: %v", w)
	}
	if w := LowEntropyWarnings("FREE"); len(w) != 1 {
		t.Fatalf("dictionary: %v", w)
	}
	if w := LowEntropyWarnings("LAUNCH25"); len(w) != 0 {
		t.Fatalf("good code should not warn: %v", w)
	}
	if w := LowEntropyWarnings("X9K2P7"); len(w) != 0 {
		t.Fatalf("random-ish: %v", w)
	}
}
