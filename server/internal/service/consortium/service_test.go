package consortium

import (
	"errors"
	"testing"
)

func TestConsortiumServiceErrors(t *testing.T) {
	if ErrAgreementNotActive == nil {
		t.Fatal("expected ErrAgreementNotActive")
	}
	if ErrCourseNotShareable == nil {
		t.Fatal("expected ErrCourseNotShareable")
	}
	if !errors.Is(ErrAlreadyEnrolled, ErrAlreadyEnrolled) {
		t.Fatal("expected ErrAlreadyEnrolled")
	}
}
