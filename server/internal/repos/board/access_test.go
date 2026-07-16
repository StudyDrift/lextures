package board

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeVisibility(t *testing.T) {
	t.Parallel()
	ok, err := NormalizeVisibility(" Course ")
	if err != nil || ok != VisibilityCourse {
		t.Fatalf("got %q %v", ok, err)
	}
	if _, err := NormalizeVisibility("secret"); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeAttribution(t *testing.T) {
	t.Parallel()
	ok, err := NormalizeAttribution("anon_to_peers")
	if err != nil || ok != AttributionAnonToPeers {
		t.Fatalf("got %q %v", ok, err)
	}
	if _, err := NormalizeAttribution("hidden"); err == nil {
		t.Fatal("expected error")
	}
}

func TestRevealAuthor(t *testing.T) {
	t.Parallel()
	manager := Capabilities{CanManage: true}
	peer := Capabilities{CanView: true}
	if !RevealAuthor(AttributionNamed, peer) {
		t.Fatal("named should reveal to peers")
	}
	if RevealAuthor(AttributionAnonToPeers, peer) {
		t.Fatal("anon_to_peers should hide from peers")
	}
	if !RevealAuthor(AttributionAnonToPeers, manager) {
		t.Fatal("anon_to_peers should reveal to managers")
	}
	if RevealAuthor(AttributionAnonymous, manager) {
		t.Fatal("anonymous should never reveal via API")
	}
}

func TestResolveShareLinkCaps(t *testing.T) {
	t.Parallel()
	b := &Board{Visibility: VisibilityLink, CanPost: true}
	view := resolveShareLinkCaps(b, ResolveOpts{
		ExternalSharingAllowed: true,
		ShareCapability:        ShareCapabilityView,
	})
	if !view.CanView || view.CanPost || view.CanInteract {
		t.Fatalf("view link caps: %+v", view)
	}
	contrib := resolveShareLinkCaps(b, ResolveOpts{
		ExternalSharingAllowed: true,
		ShareCapability:        ShareCapabilityContribute,
	})
	if !contrib.CanView || !contrib.CanPost || contrib.CanInteract {
		t.Fatalf("contribute link caps: %+v", contrib)
	}
	blocked := resolveShareLinkCaps(b, ResolveOpts{
		ExternalSharingAllowed:  false,
		ShareCapability:         ShareCapabilityContribute,
	})
	if blocked.CanView {
		t.Fatal("external sharing disabled should deny")
	}
	minors := resolveShareLinkCaps(b, ResolveOpts{
		ExternalSharingAllowed:  true,
		ForbidExternalForMinors: true,
		ShareCapability:         ShareCapabilityView,
	})
	if minors.CanView {
		t.Fatal("minors policy should deny")
	}
}

func TestGenerateShareTokenEntropy(t *testing.T) {
	t.Parallel()
	raw, hash, err := GenerateShareToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 40 {
		t.Fatalf("token too short: %d", len(raw))
	}
	if !TokenMatches(raw, hash) {
		t.Fatal("token should match its hash")
	}
	if TokenMatches(raw+"x", hash) {
		t.Fatal("mutated token should not match")
	}
	_ = uuid.New() // keep uuid import used if needed by future cases
}

func TestNormalizeMemberRoleAndShareCapability(t *testing.T) {
	t.Parallel()
	if _, err := NormalizeMemberRole("editor"); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeMemberRole("admin"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := NormalizeShareCapability("contribute"); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeShareCapability("edit"); err == nil {
		t.Fatal("expected error")
	}
}
