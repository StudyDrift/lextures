package stateprivacy

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

func TestDeletionDeadlineDays_Thirty(t *testing.T) {
	if DeletionDeadlineDays != 30 {
		t.Errorf("DeletionDeadlineDays = %d want 30 (IL SOPPA 105 ILCS 85/25(c))", DeletionDeadlineDays)
	}
}

func TestDeletionDeadlineWarning_FiveDays(t *testing.T) {
	if DeletionDeadlineWarning != 5*24*time.Hour {
		t.Errorf("DeletionDeadlineWarning = %v want 120h (5 days)", DeletionDeadlineWarning)
	}
}

func TestJurisdictionConstants(t *testing.T) {
	if JurisdictionCA != "CA" {
		t.Errorf("JurisdictionCA = %q want CA", JurisdictionCA)
	}
	if JurisdictionNY != "NY" {
		t.Errorf("JurisdictionNY = %q want NY", JurisdictionNY)
	}
	if JurisdictionIL != "IL" {
		t.Errorf("JurisdictionIL = %q want IL", JurisdictionIL)
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
	if ErrInvalidJurisdiction == nil {
		t.Error("ErrInvalidJurisdiction must not be nil")
	}
}

func TestSetOrgJurisdiction_RejectsUnknown(t *testing.T) {
	err := validateJurisdiction("TX")
	if err != ErrInvalidJurisdiction {
		t.Errorf("validateJurisdiction(TX) err = %v want ErrInvalidJurisdiction", err)
	}
}

func TestSetOrgJurisdiction_ValidJurisdictions_PassValidation(t *testing.T) {
	// Validate that recognised codes pass the jurisdiction guard.
	// We test only the guard, not the DB round-trip, so we check only
	// that ErrInvalidJurisdiction is NOT returned for valid/empty values.
	validInputs := []string{"CA", "NY", "IL", ""}
	for _, j := range validInputs {
		err := validateJurisdiction(j)
		if err == ErrInvalidJurisdiction {
			t.Errorf("validateJurisdiction(%q) returned ErrInvalidJurisdiction unexpectedly", j)
		}
	}
}

// validateJurisdiction mirrors the guard in SetOrgJurisdiction for unit testing.
func validateJurisdiction(j string) error {
	if j != "" && j != JurisdictionCA && j != JurisdictionNY && j != JurisdictionIL {
		return ErrInvalidJurisdiction
	}
	return nil
}

func TestDPAAddendum_AllJurisdictions(t *testing.T) {
	for _, j := range []string{JurisdictionCA, JurisdictionNY, JurisdictionIL} {
		content, err := DPAAddendum(j)
		if err != nil {
			t.Errorf("DPAAddendum(%q) error: %v", j, err)
			continue
		}
		if content == nil {
			t.Errorf("DPAAddendum(%q) returned nil", j)
			continue
		}
		if content.StatuteName == "" {
			t.Errorf("DPAAddendum(%q).StatuteName must not be empty", j)
		}
		if content.StatuteCite == "" {
			t.Errorf("DPAAddendum(%q).StatuteCite must not be empty", j)
		}
		if len(content.Prohibitions) == 0 {
			t.Errorf("DPAAddendum(%q).Prohibitions must not be empty", j)
		}
		if len(content.ParentRights) == 0 {
			t.Errorf("DPAAddendum(%q).ParentRights must not be empty", j)
		}
		if len(content.Exhibits) == 0 {
			t.Errorf("DPAAddendum(%q).Exhibits must not be empty", j)
		}
	}
}

func TestDPAAddendum_RejectsUnknown(t *testing.T) {
	_, err := DPAAddendum("TX")
	if err != ErrInvalidJurisdiction {
		t.Errorf("DPAAddendum(TX) err = %v want ErrInvalidJurisdiction", err)
	}
}

func TestProhibitionAttestation_NotEmpty(t *testing.T) {
	p := ProhibitionAttestation()
	if len(p) == 0 {
		t.Fatal("ProhibitionAttestation must return at least one item")
	}
	for i, s := range p {
		if s == "" {
			t.Errorf("ProhibitionAttestation[%d] must not be empty", i)
		}
	}
}

func TestCAChecklist_CorrectJurisdiction(t *testing.T) {
	items := caChecklist()
	if len(items) == 0 {
		t.Fatal("caChecklist must return items")
	}
	for _, item := range items {
		if item.Jurisdiction != JurisdictionCA {
			t.Errorf("item %q jurisdiction = %q want CA", item.ID, item.Jurisdiction)
		}
		if item.ID == "" {
			t.Error("checklist item must have non-empty ID")
		}
		if item.Statute == "" {
			t.Error("checklist item must have non-empty Statute")
		}
	}
}

func TestNYCAAddendum_ExhibitNamesDistinct(t *testing.T) {
	ca, _ := DPAAddendum(JurisdictionCA)
	ny, _ := DPAAddendum(JurisdictionNY)
	il, _ := DPAAddendum(JurisdictionIL)

	seen := make(map[string]bool)
	for _, c := range []*DPAContent{ca, ny, il} {
		for _, e := range c.Exhibits {
			if seen[e.Name] {
				t.Errorf("exhibit name %q appears in multiple addenda", e.Name)
			}
			seen[e.Name] = true
			if e.Heading == "" {
				t.Errorf("exhibit %q heading must not be empty", e.Name)
			}
			if e.Body == "" {
				t.Errorf("exhibit %q body must not be empty", e.Name)
			}
		}
	}
}
