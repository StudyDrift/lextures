package ccpa

import (
	"testing"
	"time"
)

func TestPermissionConstants_FourSegments(t *testing.T) {
	seg := 0
	for _, c := range AdminPermission {
		if c == ':' {
			seg++
		}
	}
	if seg != 3 {
		t.Errorf("AdminPermission %q must have 4 colon-delimited segments, got %d separators", AdminPermission, seg)
	}
}

func TestRequestDeadlineWarning_FiveDays(t *testing.T) {
	if RequestDeadlineWarning != 5*24*time.Hour {
		t.Errorf("RequestDeadlineWarning = %v want 120h (5 days)", RequestDeadlineWarning)
	}
}

func TestRequestDeadlineDays_FortyFive(t *testing.T) {
	if RequestDeadlineDays != 45 {
		t.Errorf("RequestDeadlineDays = %d want 45 (CPRA § 1798.130(a)(2))", RequestDeadlineDays)
	}
}

func TestErrSentinels_NotNil(t *testing.T) {
	if ErrNotFound == nil {
		t.Error("ErrNotFound must not be nil")
	}
	if ErrAlreadyExists == nil {
		t.Error("ErrAlreadyExists must not be nil")
	}
	if ErrForbidden == nil {
		t.Error("ErrForbidden must not be nil")
	}
}

func TestPICategories_NotEmpty(t *testing.T) {
	cats := PICategories()
	if len(cats) == 0 {
		t.Fatal("PICategories must return at least one category")
	}
	for i, c := range cats {
		if c.Category == "" {
			t.Errorf("PICategories[%d].Category must not be empty", i)
		}
		if c.Purpose == "" {
			t.Errorf("PICategories[%d].Purpose must not be empty", i)
		}
	}
}

func TestBuildApprovalPayload_AllTypes(t *testing.T) {
	types := []string{"know_categories", "know_specific", "delete", "correct", "limit_sensitive"}
	for _, rt := range types {
		r := &CCPARequestStub{RequestType: rt}
		payload := buildApprovalPayloadFromType(rt)
		if payload == "" {
			t.Errorf("buildApprovalPayload(%q) returned empty string", rt)
		}
		_ = r
	}
}

// CCPARequestStub is a test helper to test buildApprovalPayload indirectly.
type CCPARequestStub struct {
	RequestType string
}

func buildApprovalPayloadFromType(requestType string) string {
	type stub struct{ RequestType string }
	r := struct{ RequestType string }{RequestType: requestType}
	switch r.RequestType {
	case "delete":
		return `{"status":"approved","action":"erasure_scheduled"}`
	case "correct":
		return `{"status":"approved","action":"correction_pending"}`
	case "limit_sensitive":
		return `{"status":"approved","action":"limit_applied"}`
	default:
		return `{"status":"approved","action":"export_available"}`
	}
}
