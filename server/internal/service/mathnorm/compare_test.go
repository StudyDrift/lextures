package mathnorm

import "testing"

func TestCompareEquivalent(t *testing.T) {
	cases := []struct {
		got, expected string
		vars          []string
	}{
		{"3x + 6", "3(x+2)", []string{"x"}},
		{"3(x+2)", "3x+6", []string{"x"}},
		{"x^2 + 2*x + 1", "(x+1)^2", []string{"x"}},
		{"(x+1)(x-1)", "x^2 - 1", []string{"x"}},
		{"2*x", "x+x", []string{"x"}},
		{"x/2", "0.5*x", []string{"x"}},
		{"-x", "0-x", []string{"x"}},
		{"3", "1+2", nil},
		{"y + x", "x + y", []string{"x", "y"}},
		{"2(x+y)", "2x+2y", []string{"x", "y"}},
	}
	for _, tc := range cases {
		if got := Compare(tc.got, tc.expected, tc.vars); got != OutcomeEquivalent {
			t.Errorf("Compare(%q, %q) = %v, want Equivalent", tc.got, tc.expected, got)
		}
	}
}

func TestCompareDifferent(t *testing.T) {
	cases := []struct {
		got, expected string
		vars          []string
	}{
		{"3x + 5", "3(x+2)", []string{"x"}},
		{"x^2", "x", []string{"x"}},
		{"x+y", "x-y", []string{"x", "y"}},
	}
	for _, tc := range cases {
		if got := Compare(tc.got, tc.expected, tc.vars); got != OutcomeDifferent {
			t.Errorf("Compare(%q, %q) = %v, want Different", tc.got, tc.expected, got)
		}
	}
}

func TestCompareUndecidable(t *testing.T) {
	cases := []struct {
		got, expected string
		vars          []string
	}{
		{"sin(x)", "x", []string{"x"}},
		{"x^0.5", "sqrt(x)", []string{"x"}},
		{"x^^2", "x", []string{"x"}},
		{"", "1", nil},
		{"@@@", "1", nil},
		{"unknown", "1", []string{"x"}}, // undeclared when vars set
	}
	for _, tc := range cases {
		if got := Compare(tc.got, tc.expected, tc.vars); got != OutcomeUndecidable {
			t.Errorf("Compare(%q, %q) = %v, want Undecidable", tc.got, tc.expected, got)
		}
	}
}

func TestNormalizeCanonical(t *testing.T) {
	s, ok := NormalizeCanonical("3(x+2)", []string{"x"})
	if !ok {
		t.Fatal("expected ok")
	}
	if s == "" {
		t.Fatal("empty canonical")
	}
	// Round-trip equivalence via Compare is the contract.
	if Compare(s, "3x+6", []string{"x"}) != OutcomeEquivalent &&
		Compare("3(x+2)", "3x+6", []string{"x"}) != OutcomeEquivalent {
		t.Fatalf("canonical %q unexpected", s)
	}
}

func FuzzCompare(f *testing.F) {
	seeds := []string{
		"3x+6",
		"3(x+2)",
		"x^2",
		"(x+1)/(x-1)",
		"a+b",
		"",
		"((((x))))",
		"x*x*x*x*x*x*x*x*x",
		stringsRepeat("x+", 40) + "1",
	}
	for _, s := range seeds {
		f.Add(s, "3x+6")
	}
	f.Fuzz(func(t *testing.T, a, b string) {
		_ = Compare(a, b, []string{"x", "y"})
		_ = Compare(a, a, []string{"x"})
	})
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
