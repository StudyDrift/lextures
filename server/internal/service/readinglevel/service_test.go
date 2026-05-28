package readinglevel_test

import (
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/service/readinglevel"
)

func repeatWords(n int, word string) string {
	words := make([]string, n)
	for i := range words {
		words[i] = word
	}
	return strings.Join(words, " ") + ". " + strings.Join(words, " ") + "."
}

func TestAnalyze_InsufficientText(t *testing.T) {
	sc := readinglevel.Analyze("short text only here.")
	if sc.Sufficient {
		t.Fatal("expected insufficient text")
	}
	if sc.WordCount >= readinglevel.MinWordsForScore {
		t.Fatalf("word count %d", sc.WordCount)
	}
}

func TestAnalyze_KnownPassage(t *testing.T) {
	text := `The sun was warm. A small dog ran in the park. Children played on the grass. ` +
		`They laughed and waved to their friends. The dog found a red ball. It rolled fast down the hill. ` +
		`A boy picked up the ball and threw it again. The dog ran after it with joy. ` +
		`Parents sat on benches and talked. Everyone enjoyed the nice day outside. `
	sc := readinglevel.Analyze(text)
	if !sc.Sufficient {
		t.Fatalf("expected sufficient, words=%d", sc.WordCount)
	}
	if sc.FKGL < 0 || sc.FKGL > 15 {
		t.Fatalf("FKGL out of range: %v", sc.FKGL)
	}
	if sc.FRE < 0 || sc.FRE > 100 {
		t.Fatalf("FRE out of range: %v", sc.FRE)
	}
}

func TestPlainTextFromMarkdown(t *testing.T) {
	md := "# Title\n\nHello **world** with [link](https://example.com).\n\n```go\nfmt.Println(\"x\")\n```\n"
	plain := readinglevel.PlainTextFromMarkdown(md)
	if strings.Contains(plain, "```") || strings.Contains(plain, "**") {
		t.Fatalf("markdown not stripped: %q", plain)
	}
	if !strings.Contains(plain, "Hello") {
		t.Fatalf("missing content: %q", plain)
	}
}

func TestAnalyze_CompletesQuickly(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString("word ")
	}
	b.WriteString(".")
	sc := readinglevel.Analyze(b.String())
	if !sc.Sufficient {
		t.Fatal("expected sufficient for 5k words")
	}
}
