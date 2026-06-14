package course

import "testing"

func TestValidHeroObjectPosition(t *testing.T) {
	tests := []struct {
		pos  string
		want bool
	}{
		{"50% 50%", true},
		{"0% 100%", true},
		{"50% 0%", true},
		{"", false},
		{"center", false},
		{"50%", false},
	}
	for _, tc := range tests {
		if got := ValidHeroObjectPosition(tc.pos); got != tc.want {
			t.Fatalf("ValidHeroObjectPosition(%q) = %v, want %v", tc.pos, got, tc.want)
		}
	}
}