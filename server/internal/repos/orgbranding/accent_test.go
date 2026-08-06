package orgbranding

import (
	"strings"
	"testing"
)

func TestValidateOklch(t *testing.T) {
	norm, okl, err := ValidateOklch("oklch(0.55 0.18 264)")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(norm, "oklch(") {
		t.Fatalf("norm=%q", norm)
	}
	if okl.L < 0.54 || okl.L > 0.56 {
		t.Fatalf("L=%v", okl.L)
	}
	if _, _, err := ValidateOklch("oklch(0.55 0.18 264);url(x)"); err == nil {
		t.Fatal("expected reject injection")
	}
	if _, _, err := ValidateOklch("red"); err == nil {
		t.Fatal("expected reject")
	}
}

func TestDeriveAccentRamp(t *testing.T) {
	_, okl, err := ValidateOklch("oklch(0.51 0.22 277)")
	if err != nil {
		t.Fatal(err)
	}
	ramp := DeriveAccentRamp(okl)
	if len(ramp) != 11 {
		t.Fatalf("len=%d", len(ramp))
	}
	if ramp["600"] == "" {
		t.Fatal("missing 600")
	}
}

func TestValidateAccentRampAAIndigo(t *testing.T) {
	_, okl, err := ValidateOklch("oklch(0.51 0.22 277)")
	if err != nil {
		t.Fatal(err)
	}
	_, failing, _ := ValidateAccentRampAA(okl)
	if len(failing) != 0 {
		t.Fatalf("indigo-like seed should pass AA: %+v", failing)
	}
}
