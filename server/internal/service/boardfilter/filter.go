// Package boardfilter provides deterministic profanity/blocklist matching for board posts (VC.7).
package boardfilter

import (
	"regexp"
	"strings"
	"unicode"
)

// Result is the outcome of screening text.
type Result struct {
	Matched bool
	Term    string // matched canonical term (for managers/audit; not shown to students)
}

// DefaultEnglish is a small built-in English blocklist for v1.
var DefaultEnglish = []string{
	"fuck", "shit", "asshole", "bitch", "cunt", "nigger", "faggot", "retard",
}

var leetMap = map[rune]rune{
	'0': 'o', '1': 'i', '3': 'e', '4': 'a', '5': 's', '7': 't', '@': 'a', '$': 's',
}

// Normalize folds case, strips zero-width/punctuation separators, and maps common leetspeak.
func Normalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range strings.ToLower(s) {
		if mapped, ok := leetMap[r]; ok {
			r = mapped
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		// Collapse separators so "f.u.c.k" / "f u c k" still match.
		if !prevSpace {
			b.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// Match reports whether text contains any term from the word list (substring on normalized form,
// with word-boundary preference for short terms).
func Match(text string, words []string) Result {
	if len(words) == 0 {
		words = DefaultEnglish
	}
	norm := " " + Normalize(text) + " "
	compact := strings.ReplaceAll(norm, " ", "")
	for _, raw := range words {
		term := strings.TrimSpace(strings.ToLower(raw))
		if term == "" {
			continue
		}
		termNorm := Normalize(term)
		termCompact := strings.ReplaceAll(termNorm, " ", "")
		if termCompact == "" {
			continue
		}
		// Prefer word-boundary match on spaced normalization.
		if strings.Contains(norm, " "+termNorm+" ") {
			return Result{Matched: true, Term: term}
		}
		// Also catch concatenated evasion (fuck as f u c k already handled above).
		if len(termCompact) >= 3 && strings.Contains(compact, termCompact) {
			return Result{Matched: true, Term: term}
		}
	}
	return Result{}
}

// ExtractPlainText pulls human-readable text from a board body JSON document plus title.
func ExtractPlainText(title string, bodyJSON []byte) string {
	parts := []string{title}
	if len(bodyJSON) > 0 && string(bodyJSON) != "null" {
		var obj map[string]any
		if err := jsonUnmarshal(bodyJSON, &obj); err == nil {
			if t, ok := obj["text"].(string); ok {
				parts = append(parts, t)
			}
			if h, ok := obj["html"].(string); ok {
				parts = append(parts, stripTags(h))
			}
		} else {
			var s string
			if err := jsonUnmarshal(bodyJSON, &s); err == nil {
				parts = append(parts, stripTags(s))
			}
		}
	}
	return strings.Join(parts, " ")
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string {
	return tagRe.ReplaceAllString(s, " ")
}

// jsonUnmarshal is a tiny indirection so tests can stay in this package without importing encoding/json in every call site name.
func jsonUnmarshal(data []byte, v any) error {
	return jsonUnmarshalImpl(data, v)
}
