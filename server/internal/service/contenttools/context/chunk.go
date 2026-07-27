package context

import (
	"strings"
	"unicode/utf8"
)

const (
	chunkCharTarget = 1100
	chunkCharStride = 720
)

// EstimateTokens is a cheap ~4 chars/token heuristic for budgeting.
func EstimateTokens(text string) int {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return 0
	}
	toks := (n + 3) / 4
	if toks < 1 {
		return 1
	}
	return toks
}

// ChunkText splits extracted text into overlapping passages (notebookrag-style).
func ChunkText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	runes := []rune(text)
	if len(runes) <= chunkCharTarget {
		return []string{string(runes)}
	}
	var out []string
	for start := 0; start < len(runes); start += chunkCharStride {
		end := start + chunkCharTarget
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			out = append(out, chunk)
		}
		if end >= len(runes) {
			break
		}
	}
	return out
}

// LexicalScore ranks a chunk against a query (activity-scoped retrieval, FR-10).
func LexicalScore(query, chunk string) int {
	q := tokenize(query)
	if len(q) == 0 {
		return 0
	}
	lower := strings.ToLower(chunk)
	score := 0
	for tok := range q {
		if strings.Contains(lower, tok) {
			score++
		}
	}
	return score
}

func tokenize(s string) map[string]struct{} {
	s = strings.ToLower(s)
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	out := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		if len(f) < 2 {
			continue
		}
		out[f] = struct{}{}
	}
	return out
}
