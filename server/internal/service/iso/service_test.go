package iso

import "testing"

func TestAnnexAControls_CountIs93(t *testing.T) {
	if len(AnnexAControls) != 93 {
		t.Fatalf("AnnexAControls len=%d want 93", len(AnnexAControls))
	}
}

func TestAnnexAControls_UniqueIDs(t *testing.T) {
	seen := make(map[string]struct{}, len(AnnexAControls))
	for _, c := range AnnexAControls {
		if _, ok := seen[c.ID]; ok {
			t.Fatalf("duplicate control id %q", c.ID)
		}
		seen[c.ID] = struct{}{}
	}
}

func TestValidFindingType(t *testing.T) {
	if !validFindingType("nonconformity") || validFindingType("invalid") {
		t.Fatal("finding type validation failed")
	}
}

func TestValidTreatment(t *testing.T) {
	if !validTreatment("mitigate") || validTreatment("unknown") {
		t.Fatal("treatment validation failed")
	}
}
