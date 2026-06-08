package plagiarism

import (
	"testing"
)

func TestParseScorePercent(t *testing.T) {
	tests := []struct {
		in   string
		want *float64
	}{
		{"72", ptr(72)},
		{"Score: 45.5%", ptr(45)},
		{"n/a", nil},
	}
	for _, tc := range tests {
		got := parseScorePercent(tc.in)
		if tc.want == nil {
			if got != nil {
				t.Fatalf("parseScorePercent(%q) = %v, want nil", tc.in, *got)
			}
			continue
		}
		if got == nil || int(*got) != int(*tc.want) {
			t.Fatalf("parseScorePercent(%q) = %v, want %v", tc.in, got, *tc.want)
		}
	}
}

func TestProvidersForMode(t *testing.T) {
	got := providersForMode("both", "copyleaks")
	if len(got) != 2 || got[0] != ProviderInternal || got[1] != ProviderCopyleaks {
		t.Fatalf("providersForMode(both,copyleaks) = %#v", got)
	}
	got = providersForMode("plagiarism", "none")
	if len(got) != 1 || got[0] != ProviderTurnitin {
		t.Fatalf("providersForMode(plagiarism,none) = %#v", got)
	}
}

func TestStubSimilarityScore(t *testing.T) {
	a := stubSimilarityScore("hello world")
	b := stubSimilarityScore("hello world")
	if a != b {
		t.Fatalf("stub score not deterministic: %v vs %v", a, b)
	}
	if a < 5 || a > 85 {
		t.Fatalf("stub score out of range: %v", a)
	}
}

func ptr(n float64) *float64 { return &n }
