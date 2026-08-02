package mobilelinkpolicy

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want Handling
	}{
		{"in_app", HandlingInApp},
		{"system", HandlingSystem},
		{"blocked", HandlingBlocked},
		{"IN_APP", HandlingInApp},
		{"System", HandlingSystem},
		{"", HandlingInApp},
		{"unknown", HandlingInApp},
		{"  blocked  ", HandlingBlocked},
	}
	for _, tc := range cases {
		if got := Normalize(tc.in); got != tc.want {
			t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsValid(t *testing.T) {
	if !IsValid("in_app") || !IsValid("system") || !IsValid("blocked") {
		t.Fatal("expected valid enums")
	}
	if IsValid("") || IsValid("safari") || IsValid("INAPP") {
		t.Fatal("expected invalid values rejected")
	}
}
