package adaptivecontent

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestActiveForCourse(t *testing.T) {
	t.Cleanup(func() { SetKillSwitchForTest(nil) })

	falseV := false
	trueV := true

	SetKillSwitchForTest(&falseV)
	if !ActiveForCourse(true) {
		t.Fatal("flag on + kill off => active")
	}
	if ActiveForCourse(false) {
		t.Fatal("flag off + kill off => inactive")
	}

	SetKillSwitchForTest(&trueV)
	if ActiveForCourse(true) {
		t.Fatal("flag on + kill on => inactive")
	}
	if ActiveForCourse(false) {
		t.Fatal("flag off + kill on => inactive")
	}
}

func TestKillSwitchEngaged_DefaultOff(t *testing.T) {
	t.Cleanup(func() { SetKillSwitchForTest(nil) })
	SetKillSwitchForTest(nil)
	// Without override, depends on env; force via override for determinism.
	off := false
	SetKillSwitchForTest(&off)
	if KillSwitchEngaged() {
		t.Fatal("expected disengaged")
	}
	on := true
	SetKillSwitchForTest(&on)
	if !KillSwitchEngaged() {
		t.Fatal("expected engaged")
	}
}

func TestValidateSettings(t *testing.T) {
	if err := ValidateSettings([]string{"emphasis", "scaffolding"}, "balanced", 25, 1000); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSettings(nil, "balanced", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSettings(nil, "balanced", 100, 0); !errors.Is(err, ErrHoldoutOutOfRange) {
		t.Fatalf("holdout 100: got %v", err)
	}
	if err := ValidateSettings(nil, "balanced", -1, 0); !errors.Is(err, ErrHoldoutOutOfRange) {
		t.Fatalf("holdout -1: got %v", err)
	}
	if err := ValidateSettings(nil, "balanced", 50, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSettings(nil, "nope", 0, 0); !errors.Is(err, ErrInvalidStrategy) {
		t.Fatalf("strategy: got %v", err)
	}
	if err := ValidateSettings([]string{"unknown_axis"}, "balanced", 0, 0); !errors.Is(err, ErrInvalidAxis) {
		t.Fatalf("axis: got %v", err)
	}
	if err := ValidateSettings(nil, "balanced", 0, -5); !errors.Is(err, ErrNegativeBudget) {
		t.Fatalf("budget: got %v", err)
	}
}

func TestValidateUnitTargetShape(t *testing.T) {
	mod := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	out := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	if err := ValidateUnitTargetShape("module", &mod, nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateUnitTargetShape("outcome", nil, &out); err != nil {
		t.Fatal(err)
	}
	if err := ValidateUnitTargetShape("module", nil, nil); !errors.Is(err, ErrTargetShape) {
		t.Fatalf("got %v", err)
	}
	if err := ValidateUnitTargetShape("module", &mod, &out); !errors.Is(err, ErrTargetShape) {
		t.Fatalf("got %v", err)
	}
	if err := ValidateUnitTargetShape("outcome", &mod, &out); !errors.Is(err, ErrTargetShape) {
		t.Fatalf("got %v", err)
	}
	if err := ValidateUnitTargetShape("page", &mod, nil); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateUnitStatus(t *testing.T) {
	for _, s := range []string{"", "draft", "active", "paused", "archived"} {
		if err := ValidateUnitStatus(s); err != nil {
			t.Fatalf("%q: %v", s, err)
		}
	}
	if err := ValidateUnitStatus("live"); !errors.Is(err, ErrInvalidUnitStatus) {
		t.Fatalf("got %v", err)
	}
}

func TestProfileSignature_Stable(t *testing.T) {
	payload := map[string]any{"emphasis": "remediate", "gaps": []string{"a", "b"}}
	s1, err := ProfileSignature(payload)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := ProfileSignature(payload)
	if err != nil {
		t.Fatal(err)
	}
	if s1 != s2 {
		t.Fatalf("signature not stable: %s vs %s", s1, s2)
	}
	if len(s1) != 64 {
		t.Fatalf("want 64 hex chars, got %d", len(s1))
	}
	other, err := ProfileSignature(map[string]any{"emphasis": "compress"})
	if err != nil {
		t.Fatal(err)
	}
	if other == s1 {
		t.Fatal("different payloads should differ")
	}
}

func TestNormalizeAxes(t *testing.T) {
	got := NormalizeAxes([]string{" emphasis ", "emphasis", "scaffolding", ""})
	if len(got) != 2 || got[0] != "emphasis" || got[1] != "scaffolding" {
		t.Fatalf("got %#v", got)
	}
}
