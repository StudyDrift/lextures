package user

import "testing"

func TestNormalizeLocalePrimary(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"en-US", "en"},
		{"ar-SA", "ar"},
		{"", ""},
		{"!!!", ""},
	}
	for _, tc := range tests {
		if got := NormalizeLocalePrimary(tc.in); got != tc.want {
			t.Errorf("NormalizeLocalePrimary(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
