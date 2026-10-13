package adminsearchrepo

import (
	"regexp"
	"unicode"
	"unicode/utf8"
)

// fuzzyMinRunes is the shortest query that also uses pg_trgm similarity and
// word-prefix matching. Shorter queries (e.g. "QA") must match a whole word so
// they don't surface names like "Qamets" (#791).
const fuzzyMinRunes = 3

// matchPatterns holds the Postgres ARE (~*) patterns used by admin search.
type matchPatterns struct {
	// match filters rows: whole word for short queries, word prefix otherwise.
	// Never matches in the middle of a word.
	match string
	// word matches q as a whole word; used to rank whole-word hits above prefix hits.
	word string
	// fuzzy enables trigram similarity matching.
	fuzzy bool
}

func newMatchPatterns(q string) matchPatterns {
	quoted := regexp.QuoteMeta(q)
	start, end := "", ""
	if r, _ := utf8.DecodeRuneInString(q); isWordRune(r) {
		start = `\m`
	}
	if r, _ := utf8.DecodeLastRuneInString(q); isWordRune(r) {
		end = `\M`
	}
	word := start + quoted + end
	fuzzy := utf8.RuneCountInString(q) >= fuzzyMinRunes
	match := word
	if fuzzy {
		match = start + quoted
	}
	return matchPatterns{match: match, word: word, fuzzy: fuzzy}
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
