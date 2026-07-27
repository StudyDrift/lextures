package parameter_explorer

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

const (
	maxExprLen   = 500
	maxASTDepth  = 32
	maxEvalSteps = 2000
)

// EvalError is a parse/eval failure with a stable code for authoring UI.
type EvalError struct {
	Code    string
	Message string
}

func (e *EvalError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// EvalExpression evaluates a sandboxed arithmetic/predicate expression.
// vars may include declared parameters and built-in constants (pi, e).
// No eval/Function, no property access, no loops, no assignments.
func EvalExpression(expr string, vars map[string]float64) (float64, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, &EvalError{Code: "empty", Message: "expression is empty"}
	}
	if len(expr) > maxExprLen {
		return 0, &EvalError{Code: "too_long", Message: "expression exceeds length limit"}
	}
	toks, err := tokenize(expr)
	if err != nil {
		return 0, err
	}
	p := &parser{toks: toks}
	ast, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	if p.pos < len(p.toks) {
		return 0, &EvalError{Code: "trailing", Message: "unexpected trailing tokens"}
	}
	if depth := astDepth(ast); depth > maxASTDepth {
		return 0, &EvalError{Code: "too_deep", Message: "expression nesting too deep"}
	}
	env := map[string]float64{
		"pi": math.Pi,
		"e":  math.E,
	}
	for k, v := range vars {
		env[k] = v
	}
	steps := 0
	return evalNode(ast, env, &steps)
}

// EvalPredicate returns whether a boolean/predicate expression is true.
func EvalPredicate(expr string, vars map[string]float64) (bool, error) {
	v, err := EvalExpression(expr, vars)
	if err != nil {
		return false, err
	}
	return v != 0 && !math.IsNaN(v), nil
}

// ValidateExpression parses without evaluating (authoring gate).
func ValidateExpression(expr string) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return &EvalError{Code: "empty", Message: "expression is empty"}
	}
	if len(expr) > maxExprLen {
		return &EvalError{Code: "too_long", Message: "expression exceeds length limit"}
	}
	toks, err := tokenize(expr)
	if err != nil {
		return err
	}
	p := &parser{toks: toks}
	ast, err := p.parseExpr()
	if err != nil {
		return err
	}
	if p.pos < len(p.toks) {
		return &EvalError{Code: "trailing", Message: "unexpected trailing tokens"}
	}
	if depth := astDepth(ast); depth > maxASTDepth {
		return &EvalError{Code: "too_deep", Message: "expression nesting too deep"}
	}
	return nil
}

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokNum
	tokIdent
	tokOp
	tokLParen
	tokRParen
	tokComma
)

type token struct {
	kind tokenKind
	text string
	num  float64
}

func tokenize(s string) ([]token, error) {
	var out []token
	i := 0
	for i < len(s) {
		r := rune(s[i])
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if s[i] == '(' {
			out = append(out, token{kind: tokLParen, text: "("})
			i++
			continue
		}
		if s[i] == ')' {
			out = append(out, token{kind: tokRParen, text: ")"})
			i++
			continue
		}
		if s[i] == ',' {
			out = append(out, token{kind: tokComma, text: ","})
			i++
			continue
		}
		// multi-char ops
		if i+1 < len(s) {
			two := s[i : i+2]
			switch two {
			case ">=", "<=", "==", "!=", "&&", "||":
				out = append(out, token{kind: tokOp, text: two})
				i += 2
				continue
			}
		}
		switch s[i] {
		case '+', '-', '*', '/', '%', '^', '>', '<', '!':
			out = append(out, token{kind: tokOp, text: string(s[i])})
			i++
			continue
		}
		if unicode.IsDigit(r) || s[i] == '.' {
			j := i
			for j < len(s) && (unicode.IsDigit(rune(s[j])) || s[j] == '.' || s[j] == 'e' || s[j] == 'E') {
				if (s[j] == 'e' || s[j] == 'E') && j+1 < len(s) && (s[j+1] == '+' || s[j+1] == '-') {
					j += 2
					continue
				}
				j++
			}
			var n float64
			_, err := fmt.Sscanf(s[i:j], "%f", &n)
			if err != nil {
				return nil, &EvalError{Code: "number", Message: "invalid number"}
			}
			out = append(out, token{kind: tokNum, text: s[i:j], num: n})
			i = j
			continue
		}
		if unicode.IsLetter(r) || s[i] == '_' {
			j := i
			for j < len(s) && (unicode.IsLetter(rune(s[j])) || unicode.IsDigit(rune(s[j])) || s[j] == '_') {
				j++
			}
			out = append(out, token{kind: tokIdent, text: s[i:j]})
			i = j
			continue
		}
		return nil, &EvalError{Code: "char", Message: fmt.Sprintf("disallowed character %q", s[i])}
	}
	return out, nil
}

type nodeKind int

const (
	nodeNum nodeKind = iota
	nodeIdent
	nodeUnary
	nodeBinary
	nodeCall
)

type astNode struct {
	kind  nodeKind
	num   float64
	name  string
	op    string
	left  *astNode
	right *astNode
	args  []*astNode
}

type parser struct {
	toks []token
	pos  int
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

func (p *parser) parseExpr() (*astNode, error) { return p.parseOr() }

func (p *parser) parseOr() (*astNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOp && p.peek().text == "||" {
		op := p.next().text
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &astNode{kind: nodeBinary, op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (*astNode, error) {
	left, err := p.parseCmp()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOp && p.peek().text == "&&" {
		op := p.next().text
		right, err := p.parseCmp()
		if err != nil {
			return nil, err
		}
		left = &astNode{kind: nodeBinary, op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseCmp() (*astNode, error) {
	left, err := p.parseSum()
	if err != nil {
		return nil, err
	}
	if p.peek().kind == tokOp {
		switch p.peek().text {
		case ">", "<", ">=", "<=", "==", "!=":
			op := p.next().text
			right, err := p.parseSum()
			if err != nil {
				return nil, err
			}
			return &astNode{kind: nodeBinary, op: op, left: left, right: right}, nil
		}
	}
	return left, nil
}

func (p *parser) parseSum() (*astNode, error) {
	left, err := p.parseProduct()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOp && (p.peek().text == "+" || p.peek().text == "-") {
		op := p.next().text
		right, err := p.parseProduct()
		if err != nil {
			return nil, err
		}
		left = &astNode{kind: nodeBinary, op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseProduct() (*astNode, error) {
	left, err := p.parsePower()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOp && (p.peek().text == "*" || p.peek().text == "/" || p.peek().text == "%") {
		op := p.next().text
		right, err := p.parsePower()
		if err != nil {
			return nil, err
		}
		left = &astNode{kind: nodeBinary, op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parsePower() (*astNode, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	if p.peek().kind == tokOp && p.peek().text == "^" {
		p.next()
		right, err := p.parsePower() // right-assoc
		if err != nil {
			return nil, err
		}
		return &astNode{kind: nodeBinary, op: "^", left: left, right: right}, nil
	}
	return left, nil
}

func (p *parser) parseUnary() (*astNode, error) {
	if p.peek().kind == tokOp {
		switch p.peek().text {
		case "+", "-", "!":
			op := p.next().text
			child, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			return &astNode{kind: nodeUnary, op: op, left: child}, nil
		}
	}
	return p.parseCall()
}

func (p *parser) parseCall() (*astNode, error) {
	if p.peek().kind == tokIdent {
		name := p.next().text
		if p.peek().kind == tokLParen {
			p.next()
			var args []*astNode
			if p.peek().kind != tokRParen {
				for {
					a, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, a)
					if p.peek().kind == tokComma {
						p.next()
						continue
					}
					break
				}
			}
			if p.peek().kind != tokRParen {
				return nil, &EvalError{Code: "paren", Message: "expected closing parenthesis"}
			}
			p.next()
			if !allowedFn(name) {
				return nil, &EvalError{Code: "unknown_fn", Message: fmt.Sprintf("unknown function %q", name)}
			}
			return &astNode{kind: nodeCall, name: name, args: args}, nil
		}
		return &astNode{kind: nodeIdent, name: name}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (*astNode, error) {
	t := p.peek()
	switch t.kind {
	case tokNum:
		p.next()
		return &astNode{kind: nodeNum, num: t.num}, nil
	case tokLParen:
		p.next()
		n, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRParen {
			return nil, &EvalError{Code: "paren", Message: "expected closing parenthesis"}
		}
		p.next()
		return n, nil
	default:
		return nil, &EvalError{Code: "syntax", Message: "expected number, identifier, or '('"}
	}
}

func allowedFn(name string) bool {
	switch strings.ToLower(name) {
	case "abs", "sqrt", "sin", "cos", "tan", "asin", "acos", "atan",
		"ln", "log", "log10", "exp", "floor", "ceil", "round",
		"min", "max", "pow", "hypot", "sign", "clamp":
		return true
	default:
		return false
	}
}

func astDepth(n *astNode) int {
	if n == nil {
		return 0
	}
	d := 1
	d = max(d, 1+astDepth(n.left))
	d = max(d, 1+astDepth(n.right))
	for _, a := range n.args {
		d = max(d, 1+astDepth(a))
	}
	return d
}

func evalNode(n *astNode, env map[string]float64, steps *int) (float64, error) {
	*steps++
	if *steps > maxEvalSteps {
		return 0, &EvalError{Code: "steps", Message: "expression exceeded evaluation step limit"}
	}
	switch n.kind {
	case nodeNum:
		return n.num, nil
	case nodeIdent:
		name := strings.ToLower(n.name)
		if name == "true" {
			return 1, nil
		}
		if name == "false" {
			return 0, nil
		}
		// prefer exact then lower-case key
		if v, ok := env[n.name]; ok {
			return v, nil
		}
		if v, ok := env[name]; ok {
			return v, nil
		}
		return 0, &EvalError{Code: "unknown_var", Message: fmt.Sprintf("unknown variable %q", n.name)}
	case nodeUnary:
		v, err := evalNode(n.left, env, steps)
		if err != nil {
			return 0, err
		}
		switch n.op {
		case "+":
			return v, nil
		case "-":
			return -v, nil
		case "!":
			if v == 0 {
				return 1, nil
			}
			return 0, nil
		}
	case nodeBinary:
		// short-circuit && / ||
		if n.op == "&&" || n.op == "||" {
			l, err := evalNode(n.left, env, steps)
			if err != nil {
				return 0, err
			}
			if n.op == "&&" && l == 0 {
				return 0, nil
			}
			if n.op == "||" && l != 0 {
				return 1, nil
			}
			r, err := evalNode(n.right, env, steps)
			if err != nil {
				return 0, err
			}
			if r != 0 {
				return 1, nil
			}
			return 0, nil
		}
		l, err := evalNode(n.left, env, steps)
		if err != nil {
			return 0, err
		}
		r, err := evalNode(n.right, env, steps)
		if err != nil {
			return 0, err
		}
		switch n.op {
		case "+":
			return l + r, nil
		case "-":
			return l - r, nil
		case "*":
			return l * r, nil
		case "/":
			if r == 0 {
				return math.NaN(), nil
			}
			return l / r, nil
		case "%":
			if r == 0 {
				return math.NaN(), nil
			}
			return math.Mod(l, r), nil
		case "^":
			// Cap huge exponents to avoid hangs.
			if math.Abs(r) > 1000 {
				return 0, &EvalError{Code: "exponent", Message: "exponent too large"}
			}
			return math.Pow(l, r), nil
		case ">":
			return boolNum(l > r), nil
		case "<":
			return boolNum(l < r), nil
		case ">=":
			return boolNum(l >= r), nil
		case "<=":
			return boolNum(l <= r), nil
		case "==":
			return boolNum(l == r), nil
		case "!=":
			return boolNum(l != r), nil
		}
	case nodeCall:
		args := make([]float64, len(n.args))
		for i, a := range n.args {
			v, err := evalNode(a, env, steps)
			if err != nil {
				return 0, err
			}
			args[i] = v
		}
		return callFn(strings.ToLower(n.name), args)
	}
	return 0, &EvalError{Code: "internal", Message: "unevaluable node"}
}
