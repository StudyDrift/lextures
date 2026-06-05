package readingprefs

import "testing"

func ptr(s string) *string { return &s }

func TestGradeToUIMode(t *testing.T) {
	cases := []struct {
		grade *string
		want  string
	}{
		{nil, UIModeStandard},
		{ptr("K"), UIModeK2},
		{ptr("1"), UIModeK2},
		{ptr("2"), UIModeK2},
		{ptr("3"), UIModeElementary},
		{ptr("4"), UIModeElementary},
		{ptr("5"), UIModeElementary},
		{ptr("6"), UIModeStandard},
		{ptr("7"), UIModeStandard},
		{ptr("8"), UIModeStandard},
		{ptr("12"), UIModeStandard},
		{ptr("K-2"), UIModeStandard},  // aggregate grade bands → standard
		{ptr("3-5"), UIModeStandard},
		{ptr("K-12"), UIModeStandard},
	}
	for _, c := range cases {
		got := GradeToUIMode(c.grade)
		g := "<nil>"
		if c.grade != nil {
			g = *c.grade
		}
		if got != c.want {
			t.Errorf("GradeToUIMode(%q) = %q, want %q", g, got, c.want)
		}
	}
}

func TestEffectiveUIMode_OverrideBeatGrade(t *testing.T) {
	grade := ptr("1") // would be k2
	override := ptr("elementary")
	got := EffectiveUIMode(grade, override)
	if got != UIModeElementary {
		t.Errorf("expected elementary override to beat grade k2, got %q", got)
	}
}

func TestEffectiveUIMode_NilOverrideFallsBack(t *testing.T) {
	grade := ptr("2")
	got := EffectiveUIMode(grade, nil)
	if got != UIModeK2 {
		t.Errorf("expected k2 from grade 2 with nil override, got %q", got)
	}
}

func TestEffectiveUIMode_InvalidOverrideIgnored(t *testing.T) {
	grade := ptr("4")
	invalid := ptr("bogus")
	got := EffectiveUIMode(grade, invalid)
	if got != UIModeElementary {
		t.Errorf("expected elementary from grade 4 when override is invalid, got %q", got)
	}
}

func TestEffectiveUIMode_StandardOverride(t *testing.T) {
	grade := ptr("K")
	override := ptr("standard")
	got := EffectiveUIMode(grade, override)
	if got != UIModeStandard {
		t.Errorf("expected standard override to beat grade k2, got %q", got)
	}
}

func TestValidUIMode(t *testing.T) {
	valid := []string{"k2", "elementary", "standard"}
	for _, v := range valid {
		if !ValidUIMode(v) {
			t.Errorf("ValidUIMode(%q) should be true", v)
		}
	}
	invalid := []string{"", "K2", "ELEMENTARY", "Standard", "k-2", "bogus"}
	for _, v := range invalid {
		if ValidUIMode(v) {
			t.Errorf("ValidUIMode(%q) should be false", v)
		}
	}
}
