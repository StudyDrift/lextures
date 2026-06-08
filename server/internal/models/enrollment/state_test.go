package enrollment

import (
	"testing"
	"time"
)

func TestValidateTransition_DropAfterDeadline(t *testing.T) {
	deadline := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2027, 1, 20, 12, 0, 0, 0, time.UTC)
	err := ValidateTransition(StateActive, StateDropped, DeadlineContext{
		AddDropDeadline: &deadline,
		Now:             now,
	})
	if err == nil {
		t.Fatal("expected error for drop after deadline")
	}
}

func TestValidateTransition_WithdrawBeforeAddDrop(t *testing.T) {
	deadline := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2027, 1, 10, 12, 0, 0, 0, time.UTC)
	err := ValidateTransition(StateActive, StateWithdrawn, DeadlineContext{
		AddDropDeadline: &deadline,
		Now:             now,
	})
	if err == nil {
		t.Fatal("expected error for withdraw before add/drop deadline")
	}
}

func TestValidateTransition_WithdrawInWindow(t *testing.T) {
	addDrop := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	withdraw := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2027, 2, 10, 12, 0, 0, 0, time.UTC)
	err := ValidateTransition(StateActive, StateWithdrawn, DeadlineContext{
		AddDropDeadline:    &addDrop,
		WithdrawalDeadline: &withdraw,
		Now:                now,
	})
	if err != nil {
		t.Fatalf("expected allowed withdraw: %v", err)
	}
}

func TestValidateTransition_AdminOverride(t *testing.T) {
	deadline := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2027, 2, 1, 12, 0, 0, 0, time.UTC)
	err := ValidateTransition(StateActive, StateDropped, DeadlineContext{
		AddDropDeadline:   &deadline,
		OverrideDeadlines: true,
		Now:               now,
	})
	if err != nil {
		t.Fatalf("override should allow drop: %v", err)
	}
}

func TestStateLISStatusCode(t *testing.T) {
	if StateAudit.LISStatusCode() != "Auditor" {
		t.Fatalf("audit LIS code: %s", StateAudit.LISStatusCode())
	}
	if StateWithdrawn.LISStatusCode() != "Withdrawn" {
		t.Fatalf("withdrawn LIS code: %s", StateWithdrawn.LISStatusCode())
	}
}
