package navprefs

import "testing"

func TestValidScope(t *testing.T) {
	ok := []string{"global", "settings", "admin", "course:MATH101", "course-settings:MATH101"}
	for _, s := range ok {
		if !ValidScope(s) {
			t.Errorf("expected valid: %q", s)
		}
	}
	bad := []string{"", "foo", "course:", "course:a b", "../../../etc"}
	for _, s := range bad {
		if ValidScope(s) {
			t.Errorf("expected invalid: %q", s)
		}
	}
}

func TestValidateIDs(t *testing.T) {
	got := ValidateIDs([]string{"course.gradebook", " course.gradebook ", "BAD ID", "", "ok.id"})
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
	if got[0] != "course.gradebook" || got[1] != "ok.id" {
		t.Fatalf("got %#v", got)
	}
}
