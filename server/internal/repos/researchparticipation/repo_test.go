package researchparticipation

import "testing"

func TestParticipationValid(t *testing.T) {
	for _, tc := range []struct {
		value Participation
		want  bool
	}{{OptIn, true}, {OptOut, true}, {"", false}, {"yes", false}} {
		if got := tc.value.Valid(); got != tc.want {
			t.Fatalf("Valid(%q)=%v want %v", tc.value, got, tc.want)
		}
	}
}
