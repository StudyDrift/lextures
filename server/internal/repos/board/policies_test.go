package board

import (
	"testing"

	"github.com/google/uuid"
)

func TestDefaultOrgPolicies(t *testing.T) {
	orgID := uuid.MustParse("a0000000-0000-4000-8000-0000000000a0")
	p := DefaultOrgPolicies(orgID)
	if p.ExternalSharing {
		t.Fatal("external sharing should default off")
	}
	if !p.MinorModerationFloor {
		t.Fatal("minor moderation floor should default on")
	}
	if p.DefaultAttribution != AttributionNamed {
		t.Fatalf("attribution = %q, want named", p.DefaultAttribution)
	}
	if p.BoardCapPerCourse != nil {
		t.Fatal("board cap should default unlimited")
	}
}

func TestExternalSharingAllowed(t *testing.T) {
	pol := DefaultOrgPolicies(uuid.New())
	if ExternalSharingAllowed(true, pol) {
		t.Fatal("org default off should block even when platform flag on")
	}
	pol.ExternalSharing = true
	if !ExternalSharingAllowed(true, pol) {
		t.Fatal("both on should allow")
	}
	if ExternalSharingAllowed(false, pol) {
		t.Fatal("platform flag off should block")
	}
}

func TestApplyMinorsModerationFloor_WithOrgPolicy(t *testing.T) {
	mode, filter := ApplyMinorsModerationFloor(ModerationOpen, FilterFlag, true)
	if mode != ModerationApproval || filter != FilterBlock {
		t.Fatalf("got %s/%s", mode, filter)
	}
	mode, filter = ApplyMinorsModerationFloor(ModerationOpen, FilterFlag, false)
	if mode != ModerationOpen || filter != FilterFlag {
		t.Fatalf("got %s/%s", mode, filter)
	}
}
