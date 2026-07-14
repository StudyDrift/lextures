package platformpeople

import "testing"

func TestNormalizeFilter(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"signups_7d", FilterSignups7d},
		{"SIGNUPS_7D", FilterSignups7d},
		{"signups_last_7_days", FilterSignups7d},
		{"active", FilterActive},
		{"active_accounts", FilterActive},
		{"recent_30d", FilterRecent30d},
		{"recently_active_30_days", FilterRecent30d},
		{"total", FilterTotal},
		{"total_accounts", FilterTotal},
		{"suspended", FilterSuspended},
		{"suspended_accounts", FilterSuspended},
		{"nope", ""},
		{"  active  ", FilterActive},
	}
	for _, tc := range cases {
		if got := NormalizeFilter(tc.in); got != tc.want {
			t.Errorf("NormalizeFilter(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
