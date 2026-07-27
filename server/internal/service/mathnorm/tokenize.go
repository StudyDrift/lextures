package mathnorm

import (
	"fmt"
	"strings"
	"unicode"
)

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokNumber
	tokIdent
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokCaret
	tokLParen
	tokRParen
)

type token struct {
	kind tokenKind
	text string
}

func tokenize(input string) ([]token, error) {
	s := strings.TrimSpace(input)
	if len(s) > MaxInputBytes {
		return nil, fmt.Errorf("input too long")
	}
	var out []token
	i := 0
	for i < len(s) {
		if len(out) >= MaxTokens {
			return nil, fmt.Errorf("too many tokens")
		}
		r := rune(s[i])
		switch {
		case unicode.IsSpace(r):
			i++
		case r == '+':
			out = append(out, token{kind: tokPlus, text: "+"})
			i++
		case r == '-':
			out = append(out, token{kind: tokMinus, text: "-"})
			i++
		case r == '*':
			out = append(out, token{kind: tokStar, text: "*"})
			i++
		case r == '/':
			out = append(out, token{kind: tokSlash, text: "/"})
			i++
		case r == '^':
			out = append(out, token{kind: tokCaret, text: "^"})
			i++
		case r == '(':
			out = append(out, token{kind: tokLParen, text: "("})
			i++
		case r == ')':
			out = append(out, token{kind: tokRParen, text: ")"})
			i++
		case unicode.IsDigit(r) || r == '.':
			start := i
			dot := 0
			for i < len(s) {
				c := rune(s[i])
				if c == '.' {
					dot++
					if dot > 1 {
						return nil, fmt.Errorf("invalid number")
					}
					i++
					continue
				}
				if !unicode.IsDigit(c) {
					break
				}
				i++
			}
			text := s[start:i]
			if text == "." || text == "" {
				return nil, fmt.Errorf("invalid number")
			}
			out = append(out, token{kind: tokNumber, text: text})
		case unicode.IsLetter(r) || r == '_':
			start := i
			for i < len(s) {
				c := rune(s[i])
				if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' {
					i++
					continue
				}
				break
			}
			ident := s[start:i]
			if len(ident) > MaxVarNameLen {
				return nil, fmt.Errorf("identifier too long")
			}
			out = append(out, token{kind: tokIdent, text: ident})
		default:
			return nil, fmt.Errorf("unexpected character %q", string(r))
		}
	}
	out = append(out, token{kind: tokEOF})
	return out, nil
}
