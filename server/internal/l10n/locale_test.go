package l10n

import "testing"

func TestNormalizeLocale(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"de", "de", true},
		{"fr-CA", "fr-CA", true},
		{"EN-us", "en-US", true},
		{"", "", false},
		{"x", "", false},
		{"not valid!", "", false},
	}
	for _, tc := range cases {
		got, err := NormalizeLocale(tc.in)
		if tc.ok && err != nil {
			t.Fatalf("%q: unexpected err %v", tc.in, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%q: expected error", tc.in)
		}
		if tc.ok && got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeTimezone(t *testing.T) {
	t.Parallel()
	if _, err := NormalizeTimezone("America/New_York"); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeTimezone("Not/A_Zone"); err == nil {
		t.Fatal("expected error for invalid zone")
	}
}
