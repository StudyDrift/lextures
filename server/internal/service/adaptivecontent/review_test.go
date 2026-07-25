package adaptivecontent

import (
	"errors"
	"testing"
	"time"
)

func TestHasHardKeyTermFailure(t *testing.T) {
	if HasHardKeyTermFailure([]string{"safety:pii"}) {
		t.Fatal("expected soft only")
	}
	if !HasHardKeyTermFailure([]string{"missing_key_term:Photosynthesis"}) {
		t.Fatal("expected hard fail")
	}
}

func TestSoftGateFailed(t *testing.T) {
	score := 0.5
	if !SoftGateFailed(&score, 0.85, nil, nil) {
		t.Fatal("low fidelity should soft-fail")
	}
	score = 0.9
	if SoftGateFailed(&score, 0.85, nil, nil) {
		t.Fatal("high fidelity should pass")
	}
	if !SoftGateFailed(&score, 0.85, []string{"safety:toxicity"}, nil) {
		t.Fatal("safety flag should soft-fail")
	}
	// Hard key-term flags alone are not soft-gate (handled separately).
	if SoftGateFailed(&score, 0.85, []string{"missing_key_term:X"}, nil) {
		t.Fatal("hard key term alone is not a soft gate failure")
	}
	// AC.8: blocking a11y flags are soft-gate failures.
	if !SoftGateFailed(&score, 0.85, nil, []string{"image_missing_alt"}) {
		t.Fatal("blocking a11y should soft-fail")
	}
}

func TestValidateBulkAction(t *testing.T) {
	if err := ValidateBulkAction("approve", 1); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBulkAction("nope", 1); !errors.Is(err, ErrInvalidReviewAction) {
		t.Fatalf("got %v", err)
	}
	if err := ValidateBulkAction("reject", 0); !errors.Is(err, ErrBulkEmpty) {
		t.Fatalf("got %v", err)
	}
	if err := ValidateBulkAction("approve", MaxBulkReviewVariants+1); !errors.Is(err, ErrBulkTooLarge) {
		t.Fatalf("got %v", err)
	}
}

func TestCanApproveRejectRevoke(t *testing.T) {
	if !CanApproveStatus("pending_review") || CanApproveStatus("approved") {
		t.Fatal("approve status")
	}
	if !CanRejectStatus("pending_review") || CanRejectStatus("superseded") {
		t.Fatal("reject status")
	}
	if !CanRevokeStatus("approved") || !CanRevokeStatus("auto_served") || CanRevokeStatus("pending_review") {
		t.Fatal("revoke status")
	}
}

func TestTimeInQueueMs(t *testing.T) {
	if TimeInQueueMs(time.Time{}) != 0 {
		t.Fatal("zero time")
	}
	past := time.Now().Add(-2 * time.Second)
	ms := TimeInQueueMs(past)
	if ms < 1000 {
		t.Fatalf("expected ~2000ms, got %v", ms)
	}
}

func TestReviewNotePtr(t *testing.T) {
	if ReviewNotePtr("  ") != nil {
		t.Fatal("blank")
	}
	p := ReviewNotePtr("  reason ")
	if p == nil || *p != "reason" {
		t.Fatalf("got %#v", p)
	}
}
