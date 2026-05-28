// Package readinglevel computes Flesch-Kincaid Grade Level and Flesch Reading Ease for English text.
package readinglevel

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

const MinWordsForScore = 50

// Score holds readability metrics for a passage.
type Score struct {
	FKGL       float64 // Flesch-Kincaid Grade Level
	FRE        float64 // Flesch Reading Ease
	WordCount  int
	Sufficient bool    // true when word count >= MinWordsForScore
}

var (
	sentenceEndRe = regexp.MustCompile(`[.!?]+`)
	whitespaceRe  = regexp.MustCompile(`\s+`)
)

// PlainTextFromMarkdown strips common markdown syntax for scoring.
func PlainTextFromMarkdown(md string) string {
	s := md
	// Remove fenced code blocks.
	for {
		start := strings.Index(s, "```")
		if start < 0 {
			break
		}
		end := strings.Index(s[start+3:], "```")
		if end < 0 {
			s = s[:start]
			break
		}
		s = s[:start] + " " + s[start+3+end+3:]
	}
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Drop ATX headings markers.
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		// Strip list markers.
		if len(line) > 0 && (line[0] == '-' || line[0] == '*') {
			line = strings.TrimSpace(line[1:])
		}
		// Remove inline markdown links/images.
		line = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`).ReplaceAllString(line, " ")
		line = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`).ReplaceAllString(line, " ")
		line = regexp.MustCompile(`[*_~` + "`" + `]`).ReplaceAllString(line, "")
		b.WriteString(line)
		b.WriteByte(' ')
	}
	return strings.TrimSpace(b.String())
}

// Analyze computes FKGL and FRE for plain text. Returns Sufficient=false when < MinWordsForScore.
func Analyze(text string) Score {
	words := countWords(text)
	if words < MinWordsForScore {
		return Score{WordCount: words, Sufficient: false}
	}
	sentences := countSentences(text)
	if sentences < 1 {
		sentences = 1
	}
	syllables := countSyllables(text)
	wps := float64(words) / float64(sentences)
	spw := float64(syllables) / float64(words)
	fkgl := 0.39*wps + 11.8*spw - 15.59
	fre := 206.835 - 1.015*wps - 84.6*spw
	return Score{
		FKGL:       round1(fkgl),
		FRE:        round1(fre),
		WordCount:  words,
		Sufficient: true,
	}
}

func round1(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*10) / 10
}

func countWords(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	return len(whitespaceRe.Split(text, -1))
}

func countSentences(text string) int {
	parts := sentenceEndRe.Split(text, -1)
	n := 0
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			n++
		}
	}
	if n == 0 && strings.TrimSpace(text) != "" {
		return 1
	}
	return n
}

func countSyllables(text string) int {
	total := 0
	for _, w := range strings.Fields(text) {
		total += syllablesInWord(w)
	}
	return total
}

func syllablesInWord(word string) int {
	word = strings.ToLower(strings.Trim(word, ".,!?;:\"'()[]{}—–-"))
	if word == "" {
		return 0
	}
	// Common short words.
	switch word {
	case "a", "an", "the", "i":
		return 1
	}
	var runes []rune
	for _, r := range word {
		if unicode.IsLetter(r) {
			runes = append(runes, r)
		}
	}
	if len(runes) == 0 {
		return 1
	}
	count := 0
	prevVowel := false
	for i, r := range runes {
		v := isVowel(r)
		if v && !prevVowel {
			count++
		}
		prevVowel = v
		if i == len(runes)-1 && r == 'e' && count > 1 {
			count--
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u', 'y':
		return true
	default:
		return false
	}
}
