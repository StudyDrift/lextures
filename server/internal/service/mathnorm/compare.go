package mathnorm

import "errors"

// Outcome of algebraic comparison.
type Outcome int

const (
	// OutcomeEquivalent — expressions match after normalisation.
	OutcomeEquivalent Outcome = iota
	// OutcomeDifferent — expressions are decidably not equivalent.
	OutcomeDifferent
	// OutcomeUndecidable — cannot safely decide; caller should fall back.
	OutcomeUndecidable
)

var errUndecidable = errors.New("undecidable")

// Compare normalises two expressions over declared variables and compares them.
// When variables is empty, any identifier is accepted (still bounded).
func Compare(got, expected string, variables []string) Outcome {
	a, err := normalize(got, variables)
	if err != nil {
		return OutcomeUndecidable
	}
	b, err := normalize(expected, variables)
	if err != nil {
		return OutcomeUndecidable
	}
	eq, err := a.equal(b)
	if err != nil {
		return OutcomeUndecidable
	}
	if eq {
		return OutcomeEquivalent
	}
	return OutcomeDifferent
}

// NormalizeCanonical returns a canonical string form for debugging/display,
// or ("", false) when undecidable.
func NormalizeCanonical(expr string, variables []string) (string, bool) {
	r, err := normalize(expr, variables)
	if err != nil {
		return "", false
	}
	nr, err := r.normalize()
	if err != nil {
		return "", false
	}
	return formatRational(nr), true
}

func normalize(expr string, variables []string) (rational, error) {
	ast, err := parseExpr(expr, variables)
	if err != nil {
		return rational{}, err
	}
	return evalAST(ast)
}

func formatRational(r rational) string {
	ns := formatPoly(r.num)
	if r.den.equal(poly{"": 1}) {
		return ns
	}
	return "(" + ns + ")/(" + formatPoly(r.den) + ")"
}

func formatPoly(p poly) string {
	c := p.clean()
	if len(c) == 0 {
		return "0"
	}
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	// Stable: constants last, then lexicographic.
	sortStringsStable(keys)
	var b stringsBuilder
	first := true
	for _, k := range keys {
		v := c[k]
		sign := "+"
		if v < 0 {
			sign = "-"
			v = -v
		}
		if first {
			if sign == "-" {
				b.WriteString("-")
			}
			first = false
		} else {
			b.WriteString(" ")
			b.WriteString(sign)
			b.WriteString(" ")
		}
		term := formatTerm(v, k)
		b.WriteString(term)
	}
	return b.String()
}

func formatTerm(coeff float64, key string) string {
	if key == "" {
		return trimFloat(coeff)
	}
	if almostOne(coeff) {
		return key
	}
	return trimFloat(coeff) + "*" + key
}

func almostOne(v float64) bool {
	return v > 1-1e-9 && v < 1+1e-9
}

func trimFloat(v float64) string {
	if v == float64(int64(v)) {
		return itoa(int(v))
	}
	// simple fixed format
	s := sprintfFloat(v)
	return s
}

func sprintfFloat(v float64) string {
	// Avoid fmt import churn; enough for tests.
	neg := v < 0
	if neg {
		v = -v
	}
	intPart := int64(v)
	frac := v - float64(intPart)
	s := itoa(int(intPart))
	if frac < 1e-9 {
		if neg {
			return "-" + s
		}
		return s
	}
	s += "."
	for i := 0; i < 6; i++ {
		frac *= 10
		d := int(frac)
		s += string(rune('0' + d))
		frac -= float64(d)
		if frac < 1e-9 {
			break
		}
	}
	if neg {
		return "-" + s
	}
	return s
}

func sortStringsStable(keys []string) {
	// constants ("") last; otherwise lexicographic.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if lessKey(keys[j], keys[i]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
}

func lessKey(a, b string) bool {
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}
	return a < b
}

// tiny strings.Builder stand-in to keep imports lean in this file.
type stringsBuilder struct {
	b []byte
}

func (s *stringsBuilder) WriteString(v string) {
	s.b = append(s.b, v...)
}

func (s *stringsBuilder) String() string {
	return string(s.b)
}
