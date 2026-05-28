package logredaction

import (
	"testing"

	"github.com/lextures/lextures/server/internal/repos/rbac"
)

func TestReadPermission_ValidFourSegmentForm(t *testing.T) {
	if err := rbac.ValidatePermissionString(ReadPermission); err != nil {
		t.Fatalf("ReadPermission invalid: %v", err)
	}
}
