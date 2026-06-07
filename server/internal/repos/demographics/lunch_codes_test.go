package demographics

import "testing"

func TestMapLunchCode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code     string
		wantFree bool
		wantRed  bool
		ok       bool
	}{
		{"F", true, false, true},
		{"free", true, false, true},
		{"R", false, true, true},
		{"reduced", false, true, true},
		{"N", false, false, true},
		{"PAID", false, false, true},
		{"UNKNOWN", false, false, false},
	}
	for _, c := range cases {
		got := MapLunchCode(c.code)
		if c.ok {
			if got.FreeLunch == nil || *got.FreeLunch != c.wantFree {
				t.Fatalf("MapLunchCode(%q) free = %v, want %v", c.code, got.FreeLunch, c.wantFree)
			}
			if got.ReducedLunch == nil || *got.ReducedLunch != c.wantRed {
				t.Fatalf("MapLunchCode(%q) reduced = %v, want %v", c.code, got.ReducedLunch, c.wantRed)
			}
		} else if got.FreeLunch != nil || got.ReducedLunch != nil {
			t.Fatalf("MapLunchCode(%q) expected empty flags, got %+v", c.code, got)
		}
	}
}
