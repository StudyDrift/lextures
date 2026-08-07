package readingprefs

import (
	"testing"
)

func TestPatchValidateTextScale(t *testing.T) {
	ok := 1.25
	if err := (Patch{TextScale: &ok}).Validate(); err != nil {
		t.Fatalf("expected valid textScale 1.25, got %v", err)
	}
	bad := 1.3
	if err := (Patch{TextScale: &bad}).Validate(); err == nil {
		t.Fatal("expected invalid textScale 1.3 to fail")
	}
	for _, v := range []float64{1.0, 1.125, 1.25, 1.5} {
		x := v
		if err := (Patch{TextScale: &x}).Validate(); err != nil {
			t.Fatalf("expected %v valid: %v", v, err)
		}
	}
}
