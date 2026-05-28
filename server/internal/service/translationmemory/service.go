// Package translationmemory implements fuzzy-match scoring and glossary helpers (plan 11.5).
package translationmemory

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
)

// SourceHash returns a SHA-256 hex digest of normalized source text for exact TM lookup.
func SourceHash(text string) string {
	n := normalizeSegment(text)
	sum := sha256.Sum256([]byte(n))
	return hex.EncodeToString(sum[:])
}

func normalizeSegment(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.ToLower(s)
}

// TrigramSimilarity returns a 0–1 similarity score using character trigram overlap (Jaccard).
func TrigramSimilarity(a, b string) float64 {
	a = normalizeSegment(a)
	b = normalizeSegment(b)
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	ta := trigrams(a)
	tb := trigrams(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	inter := 0
	for t := range ta {
		if tb[t] {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func trigrams(s string) map[string]bool {
	padded := "  " + s + "  "
	out := make(map[string]bool)
	runes := []rune(padded)
	for i := 0; i+2 < len(runes); i++ {
		out[string(runes[i:i+3])] = true
	}
	return out
}

// GlossaryEntry is a source→target term pair for a course locale pair.
type GlossaryEntry struct {
	SourceTerm string
	TargetTerm string
}

// GlossaryMatch marks a span in source text that matches a glossary term.
type GlossaryMatch struct {
	SourceTerm string
	TargetTerm string
	Start      int
	End        int
}

// FindGlossaryMatches locates glossary terms in source text (case-insensitive, ASCII word boundaries).
func FindGlossaryMatches(source string, entries []GlossaryEntry) []GlossaryMatch {
	if source == "" || len(entries) == 0 {
		return nil
	}
	var out []GlossaryMatch
	seen := make(map[string]struct{})
	for _, e := range entries {
		term := strings.TrimSpace(e.SourceTerm)
		if term == "" {
			continue
		}
		key := strings.ToLower(term)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		re, err := regexp.Compile(`(?i)\b` + regexp.QuoteMeta(term) + `\b`)
		if err != nil {
			continue
		}
		locs := re.FindAllStringIndex(source, -1)
		for _, loc := range locs {
			out = append(out, GlossaryMatch{
				SourceTerm: term,
				TargetTerm: strings.TrimSpace(e.TargetTerm),
				Start:      loc[0],
				End:        loc[1],
			})
		}
	}
	return out
}

// SuggestGlossaryTranslation pre-fills target text by replacing matched glossary terms.
func SuggestGlossaryTranslation(source string, entries []GlossaryEntry) string {
	matches := FindGlossaryMatches(source, entries)
	if len(matches) == 0 {
		return ""
	}
	// longest-first to avoid partial replacements
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Start < matches[i].Start ||
				(matches[j].Start == matches[i].Start && matches[j].End-matches[j].Start > matches[i].End-matches[i].Start) {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
	out := source
	offset := 0
	for _, m := range matches {
		if m.TargetTerm == "" {
			continue
		}
		start := m.Start + offset
		end := m.End + offset
		if start < 0 || end > len(out) || start >= end {
			continue
		}
		out = out[:start] + m.TargetTerm + out[end:]
		offset += len(m.TargetTerm) - (m.End - m.Start)
	}
	return out
}

// PrefixWords returns up to n leading words from text (for TM suggestion while typing).
func PrefixWords(text string, n int) string {
	if n <= 0 {
		return ""
	}
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) <= n {
		return strings.Join(fields, " ")
	}
	return strings.Join(fields[:n], " ")
}

// IsMostlyText reports whether s is suitable for TM indexing (not empty, mostly letters).
func IsMostlyText(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	letters := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			letters++
		}
	}
	return letters*2 >= len([]rune(s))
}
