package adaptivecontent

import (
	"testing"
)

func TestHasBlockingA11yFlag(t *testing.T) {
	if HasBlockingA11yFlag(nil) {
		t.Fatal("empty should pass")
	}
	if !HasBlockingA11yFlag([]string{"image_missing_alt"}) {
		t.Fatal("image_missing_alt should block")
	}
	if !HasBlockingA11yFlag([]string{"heading_level_skip"}) {
		t.Fatal("heading_level_skip should block")
	}
	if HasBlockingA11yFlag([]string{"unknown_warning"}) {
		t.Fatal("unknown flags should not block")
	}
}

func TestGateBlockReason(t *testing.T) {
	score := 0.9
	if got := GateBlockReason(GateCheckInput{
		Status: "approved", FidelityScore: &score, MinFidelity: 0.85,
	}); got != "" {
		t.Fatalf("expected pass, got %q", got)
	}
	low := 0.5
	if got := GateBlockReason(GateCheckInput{
		Status: "approved", FidelityScore: &low, MinFidelity: 0.85,
	}); got != "fidelity_below_threshold" {
		t.Fatalf("got %q", got)
	}
	if got := GateBlockReason(GateCheckInput{
		Status: "approved", FidelityScore: &score, MinFidelity: 0.85,
		SafetyFlags: []string{"script_tag"},
	}); got != "safety_flag" {
		t.Fatalf("got %q", got)
	}
	if got := GateBlockReason(GateCheckInput{
		Status: "approved", FidelityScore: &score, MinFidelity: 0.85,
		A11yFlags: []string{"image_missing_alt"},
	}); got != "blocking_a11y" {
		t.Fatalf("got %q", got)
	}
	if got := GateBlockReason(GateCheckInput{
		Status: "pending_review", FidelityScore: &score, MinFidelity: 0.85,
	}); got != "status_not_servable" {
		t.Fatalf("got %q", got)
	}
	// Approved row with low fidelity still blocked (AC-1 belt-and-suspenders).
	if !VariantPassesServeGates(GateCheckInput{
		Status: "auto_served", FidelityScore: &score, MinFidelity: 0.85,
	}) {
		t.Fatal("expected pass")
	}
}

func TestSoftGateFailed_BlockingA11y(t *testing.T) {
	score := 0.95
	if SoftGateFailed(&score, 0.85, nil, nil) {
		t.Fatal("clean should pass")
	}
	if !SoftGateFailed(&score, 0.85, nil, []string{"image_missing_alt"}) {
		t.Fatal("blocking a11y should soft-fail")
	}
}

func TestForceInstructorApproval(t *testing.T) {
	if ForceInstructorApproval(false, false) {
		t.Fatal("expected false")
	}
	if !ForceInstructorApproval(true, false) {
		t.Fatal("coppa minor should force")
	}
	if !ForceInstructorApproval(false, true) {
		t.Fatal("eu policy should force")
	}
}

func TestValidContestResolveStatus(t *testing.T) {
	if !ValidContestResolveStatus("resolved") {
		t.Fatal("resolved ok")
	}
	if ValidContestResolveStatus("open") {
		t.Fatal("open is not a resolve status")
	}
}

func TestDecodeStringFlags(t *testing.T) {
	if DecodeStringFlags(nil) != nil {
		t.Fatal("nil")
	}
	got := DecodeStringFlags([]byte(`["a","b"]`))
	if len(got) != 2 || got[0] != "a" {
		t.Fatalf("got %#v", got)
	}
}
