package gdpr

import (
	"testing"
	"time"
)

func TestPermissionConstants_FourSegments(t *testing.T) {
	for _, perm := range []string{AdminPermission, DPOPermission} {
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

func TestGenerateDPATemplate_Fields(t *testing.T) {
	tpl := GenerateDPATemplate("Acme University", "https://example.com/privacy")
	if tpl.ControllerName != "Acme University" {
		t.Errorf("ControllerName = %q want %q", tpl.ControllerName, "Acme University")
	}
	if tpl.ProcessorName == "" {
		t.Error("ProcessorName must not be empty")
	}
	if tpl.PrivacyPolicyURL != "https://example.com/privacy" {
		t.Errorf("PrivacyPolicyURL = %q", tpl.PrivacyPolicyURL)
	}
	if len(tpl.SubProcessors) == 0 {
		t.Error("SubProcessors must not be empty")
	}
	if len(tpl.ProcessingPurposes) == 0 {
		t.Error("ProcessingPurposes must not be empty")
	}
	if len(tpl.TechnicalSafeguards) == 0 {
		t.Error("TechnicalSafeguards must not be empty")
	}
	if tpl.GeneratedAt == "" {
		t.Error("GeneratedAt must be set")
	}
	if _, err := time.Parse(time.RFC3339, tpl.GeneratedAt); err != nil {
		t.Errorf("GeneratedAt %q is not RFC3339: %v", tpl.GeneratedAt, err)
	}
}

func TestOptRFC3339_NilReturnsNil(t *testing.T) {
	if optRFC3339(nil) != nil {
		t.Error("optRFC3339(nil) should return nil")
	}
}

func TestOptRFC3339_TimeReturnsString(t *testing.T) {
	now := time.Now().UTC()
	s := optRFC3339(&now)
	if s == nil {
		t.Fatal("optRFC3339 returned nil for non-nil time")
	}
	if _, err := time.Parse(time.RFC3339, *s); err != nil {
		t.Errorf("optRFC3339 returned invalid RFC3339 %q: %v", *s, err)
	}
}

func TestBuildErasureConfirmationURL(t *testing.T) {
	id := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	url := buildErasureConfirmationURL(id)
	if url == "" {
		t.Error("erasure confirmation URL must not be empty")
	}
}

func TestDSARDeadlineWarning_FiveDays(t *testing.T) {
	if DSARDeadlineWarning != 5*24*time.Hour {
		t.Errorf("DSARDeadlineWarning = %v want 120h", DSARDeadlineWarning)
	}
}

func TestArchiveLinkTTL_SeventyTwoHours(t *testing.T) {
	if ArchiveLinkTTL != 72*time.Hour {
		t.Errorf("ArchiveLinkTTL = %v want 72h", ArchiveLinkTTL)
	}
}

func TestPurposeAIProcessing_NotEmpty(t *testing.T) {
	if PurposeAIProcessing == "" {
		t.Error("PurposeAIProcessing must not be empty")
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
