package ferpa

import "testing"

func TestErrNotFound_Error(t *testing.T) {
	if ErrNotFound.Error() == "" {
		t.Fatal("ErrNotFound.Error() must not be empty")
	}
}

func TestPermissionConstants_FourSegments(t *testing.T) {
	for _, perm := range []string{LEIPermission, AdminPermission} {
		seg := 0
		for _, c := range perm {
			if c == ':' {
				seg++
			}
		}
		if seg != 3 {
			t.Errorf("permission %q must have 4 colon-delimited segments, got %d separators", perm, seg)
		}
	}
}
