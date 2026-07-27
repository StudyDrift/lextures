package mathnorm

type nodeKind int

const (
	nodeNumber nodeKind = iota
	nodeVar
	nodeUnary
	nodeBinary
)

type astNode struct {
	kind  nodeKind
	op    tokenKind // for unary/binary
	value float64   // number
	name  string    // variable
	left  *astNode
	right *astNode
}

type parser struct {
	toks  []token
	pos   int
	nodes int
	vars  map[string]struct{}
}

func (p *parser) peek() token {
	if p.pos >= len(p.toks) {
		return token{kind: tokEOF}
	}
	return p.toks[p.pos]
}

func (p *parser) next() token {
	t := p.peek()
	if t.kind != tokEOF {
		p.pos++
	}
	return t
}

func (p *parser) alloc() error {
	p.nodes++
	if p.nodes > MaxASTNodes {
		return errUndecidable
	}
	return nil
}

func parseExpr(input string, declared []string) (*astNode, error) {
	toks, err := tokenize(input)
	if err != nil {
		return nil, errUndecidable
	}
	vars := map[string]struct{}{}
	for _, v := range declared {
		v = stringsTrim(v)
		if v == "" {
			continue
		}
		if len(vars) >= MaxVariables {
			return nil, errUndecidable
		}
		vars[v] = struct{}{}
	}
	p := &parser{toks: toks, vars: vars}
	n, err := p.parseAdd(0)
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokEOF {
		return nil, errUndecidable
	}
	return n, nil
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func (p *parser) parseAdd(depth int) (*astNode, error) {
	if depth > MaxASTDepth {
		return nil, errUndecidable
	}
	left, err := p.parseMul(depth + 1)
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.kind != tokPlus && t.kind != tokMinus {
			break
		}
		op := p.next().kind
		right, err := p.parseMul(depth + 1)
		if err != nil {
			return nil, err
		}
		if err := p.alloc(); err != nil {
			return nil, err
		}
		left = &astNode{kind: nodeBinary, op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseMul(depth int) (*astNode, error) {
	if depth > MaxASTDepth {
		return nil, errUndecidable
	}
	left, err := p.parsePow(depth + 1)
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		explicit := t.kind == tokStar || t.kind == tokSlash
		implicit := isImplicitMulStart(t.kind)
		if !explicit && !implicit {
			break
		}
		op := tokStar
		if explicit {
			op = p.next().kind
		}
		right, err := p.parsePow(depth + 1)
		if err != nil {
			return nil, err
		}
		if err := p.alloc(); err != nil {
			return nil, err
		}
		left = &astNode{kind: nodeBinary, op: op, left: left, right: right}
	}
	return left, nil
}

func isImplicitMulStart(k tokenKind) bool {
	return k == tokNumber || k == tokIdent || k == tokLParen
}

func (p *parser) parsePow(depth int) (*astNode, error) {
	if depth > MaxASTDepth {
		return nil, errUndecidable
	}
	left, err := p.parseUnary(depth + 1)
	if err != nil {
		return nil, err
	}
	if p.peek().kind == tokCaret {
		p.next()
		right, err := p.parseUnary(depth + 1)
		if err != nil {
			return nil, err
		}
		if err := p.alloc(); err != nil {
			return nil, err
		}
		return &astNode{kind: nodeBinary, op: tokCaret, left: left, right: right}, nil
	}
	return left, nil
}

func (p *parser) parseUnary(depth int) (*astNode, error) {
	if depth > MaxASTDepth {
		return nil, errUndecidable
	}
	t := p.peek()
	if t.kind == tokPlus || t.kind == tokMinus {
		op := p.next().kind
		child, err := p.parseUnary(depth + 1)
		if err != nil {
			return nil, err
		}
		if err := p.alloc(); err != nil {
			return nil, err
		}
		return &astNode{kind: nodeUnary, op: op, left: child}, nil
	}
	return p.parsePrimary(depth + 1)
}

func (p *parser) parsePrimary(depth int) (*astNode, error) {
	if depth > MaxASTDepth {
		return nil, errUndecidable
	}
	t := p.next()
	switch t.kind {
	case tokNumber:
		var f float64
		if _, err := fmtSscanf(t.text, &f); err != nil {
			return nil, errUndecidable
		}
		if err := p.alloc(); err != nil {
			return nil, err
		}
		return &astNode{kind: nodeNumber, value: f}, nil
	case tokIdent:
		if len(p.vars) > 0 {
			if _, ok := p.vars[t.text]; !ok {
				return nil, errUndecidable
			}
		}
		if err := p.alloc(); err != nil {
			return nil, err
		}
		return &astNode{kind: nodeVar, name: t.text}, nil
	case tokLParen:
		n, err := p.parseAdd(depth + 1)
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRParen {
			return nil, errUndecidable
		}
		p.next()
		return n, nil
	default:
		return nil, errUndecidable
	}
}

func fmtSscanf(s string, f *float64) (int, error) {
	n := 0
	dot := false
	frac := 1.0
	sign := 1.0
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		if s[i] == '-' {
			sign = -1
		}
		i++
	}
	if i >= len(s) {
		return 0, errUndecidable
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			if dot {
				return 0, errUndecidable
			}
			dot = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, errUndecidable
		}
		d := float64(c - '0')
		if !dot {
			n = 1
			*f = *f*10 + d
		} else {
			n = 1
			frac *= 0.1
			*f += d * frac
		}
	}
	*f *= sign
	if n == 0 {
		return 0, errUndecidable
	}
	return 1, nil
}
