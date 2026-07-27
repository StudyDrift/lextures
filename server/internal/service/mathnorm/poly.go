package mathnorm

import (
	"math"
	"sort"
	"strings"
)

// poly is a map from monomial key -> coefficient.
// Monomial key is sorted "var^exp" joined by "*", empty string = constant.
type poly map[string]float64

func monoKey(exps map[string]int) string {
	if len(exps) == 0 {
		return ""
	}
	names := make([]string, 0, len(exps))
	for n, e := range exps {
		if e == 0 {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, n := range names {
		e := exps[n]
		if e == 1 {
			parts = append(parts, n)
		} else {
			parts = append(parts, n+"^"+itoa(e))
		}
	}
	return strings.Join(parts, "*")
}

func parseMonoKey(key string) map[string]int {
	out := map[string]int{}
	if key == "" {
		return out
	}
	for _, part := range strings.Split(key, "*") {
		if part == "" {
			continue
		}
		if i := strings.IndexByte(part, '^'); i >= 0 {
			out[part[:i]] = atoi(part[i+1:])
		} else {
			out[part] = 1
		}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func (p poly) add(q poly) poly {
	out := make(poly, len(p)+len(q))
	for k, v := range p {
		out[k] = v
	}
	for k, v := range q {
		out[k] += v
	}
	return out.clean()
}

func (p poly) scale(c float64) poly {
	if c == 0 {
		return poly{}
	}
	out := make(poly, len(p))
	for k, v := range p {
		out[k] = v * c
	}
	return out.clean()
}

func (p poly) neg() poly {
	return p.scale(-1)
}

func (p poly) mul(q poly) (poly, error) {
	if len(p)*len(q) > MaxTerms*MaxTerms {
		return nil, errUndecidable
	}
	out := poly{}
	for ak, av := range p {
		ae := parseMonoKey(ak)
		for bk, bv := range q {
			be := parseMonoKey(bk)
			merged := map[string]int{}
			for n, e := range ae {
				merged[n] = e
			}
			for n, e := range be {
				merged[n] += e
				if merged[n] > MaxExponent*2 {
					return nil, errUndecidable
				}
			}
			key := monoKey(merged)
			out[key] += av * bv
		}
		if len(out) > MaxTerms {
			return nil, errUndecidable
		}
	}
	return out.clean(), nil
}

func (p poly) pow(exp int) (poly, error) {
	if exp < 0 || exp > MaxExponent {
		return nil, errUndecidable
	}
	if exp == 0 {
		return poly{"": 1}, nil
	}
	out := poly{"": 1}
	base := p
	for i := 0; i < exp; i++ {
		var err error
		out, err = out.mul(base)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (p poly) clean() poly {
	out := poly{}
	for k, v := range p {
		if math.Abs(v) < 1e-12 {
			continue
		}
		out[k] = v
	}
	return out
}

func (p poly) equal(q poly) bool {
	pc := p.clean()
	qc := q.clean()
	if len(pc) != len(qc) {
		return false
	}
	for k, v := range pc {
		w, ok := qc[k]
		if !ok || math.Abs(v-w) > 1e-9 {
			return false
		}
	}
	return true
}

func (p poly) isZero() bool {
	return len(p.clean()) == 0
}

func (p poly) isConst() (float64, bool) {
	c := p.clean()
	if len(c) == 0 {
		return 0, true
	}
	if len(c) == 1 {
		if v, ok := c[""]; ok {
			return v, true
		}
	}
	return 0, false
}

// rational is num/den in polynomial form.
type rational struct {
	num poly
	den poly
}

func ratFromPoly(p poly) rational {
	return rational{num: p, den: poly{"": 1}}
}

func (r rational) normalize() (rational, error) {
	if r.den.isZero() {
		return rational{}, errUndecidable
	}
	// Scale so leading den coeff is positive 1 when den is constant.
	if c, ok := r.den.isConst(); ok {
		if math.Abs(c) < 1e-12 {
			return rational{}, errUndecidable
		}
		return rational{num: r.num.scale(1 / c), den: poly{"": 1}}, nil
	}
	return r, nil
}

func (r rational) equal(o rational) (bool, error) {
	a, err := r.normalize()
	if err != nil {
		return false, err
	}
	b, err := o.normalize()
	if err != nil {
		return false, err
	}
	// a.num/a.den == b.num/b.den  <=>  a.num*b.den == b.num*a.den
	left, err := a.num.mul(b.den)
	if err != nil {
		return false, err
	}
	right, err := b.num.mul(a.den)
	if err != nil {
		return false, err
	}
	return left.equal(right), nil
}

func evalAST(n *astNode) (rational, error) {
	if n == nil {
		return rational{}, errUndecidable
	}
	switch n.kind {
	case nodeNumber:
		return ratFromPoly(poly{"": n.value}), nil
	case nodeVar:
		return ratFromPoly(poly{monoKey(map[string]int{n.name: 1}): 1}), nil
	case nodeUnary:
		child, err := evalAST(n.left)
		if err != nil {
			return rational{}, err
		}
		if n.op == tokMinus {
			return rational{num: child.num.neg(), den: child.den}, nil
		}
		return child, nil
	case nodeBinary:
		left, err := evalAST(n.left)
		if err != nil {
			return rational{}, err
		}
		right, err := evalAST(n.right)
		if err != nil {
			return rational{}, err
		}
		switch n.op {
		case tokPlus:
			// a/b + c/d = (a*d + c*b)/(b*d)
			ad, err := left.num.mul(right.den)
			if err != nil {
				return rational{}, err
			}
			cb, err := right.num.mul(left.den)
			if err != nil {
				return rational{}, err
			}
			bd, err := left.den.mul(right.den)
			if err != nil {
				return rational{}, err
			}
			return rational{num: ad.add(cb), den: bd}, nil
		case tokMinus:
			ad, err := left.num.mul(right.den)
			if err != nil {
				return rational{}, err
			}
			cb, err := right.num.mul(left.den)
			if err != nil {
				return rational{}, err
			}
			bd, err := left.den.mul(right.den)
			if err != nil {
				return rational{}, err
			}
			return rational{num: ad.add(cb.neg()), den: bd}, nil
		case tokStar:
			num, err := left.num.mul(right.num)
			if err != nil {
				return rational{}, err
			}
			den, err := left.den.mul(right.den)
			if err != nil {
				return rational{}, err
			}
			return rational{num: num, den: den}, nil
		case tokSlash:
			num, err := left.num.mul(right.den)
			if err != nil {
				return rational{}, err
			}
			den, err := left.den.mul(right.num)
			if err != nil {
				return rational{}, err
			}
			if den.isZero() {
				return rational{}, errUndecidable
			}
			return rational{num: num, den: den}, nil
		case tokCaret:
			// Only integer constant exponents.
			c, ok := right.num.isConst()
			if !ok || !right.den.equal(poly{"": 1}) {
				return rational{}, errUndecidable
			}
			exp := int(math.Round(c))
			if math.Abs(c-float64(exp)) > 1e-9 || exp < 0 || exp > MaxExponent {
				return rational{}, errUndecidable
			}
			// (a/b)^n = a^n / b^n
			num, err := left.num.pow(exp)
			if err != nil {
				return rational{}, err
			}
			den, err := left.den.pow(exp)
			if err != nil {
				return rational{}, err
			}
			return rational{num: num, den: den}, nil
		default:
			return rational{}, errUndecidable
		}
	default:
		return rational{}, errUndecidable
	}
}
