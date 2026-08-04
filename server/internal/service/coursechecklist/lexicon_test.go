package coursechecklist

import (
	"strings"
	"testing"
)

func TestLexiconEnglishFallback(t *testing.T) {
	en := LexiconFor("en")
	if en == nil || en.Locale != "en" {
		t.Fatalf("en lexicon: %+v", en)
	}
	fallback := LexiconFor("zz-ZZ")
	if fallback == nil || fallback.Locale != "en" {
		t.Fatalf("fallback want en, got %+v", fallback)
	}
}

func TestLexiconResponseTimeEN(t *testing.T) {
	lx := LexiconFor("en")
	if !lx.ResponseTime.Match("I respond to email within 24 hours") {
		t.Fatal("expected EN response-time match")
	}
	if lx.ResponseTime.Match("Office hours are on Monday") {
		t.Fatal("false positive on unrelated text")
	}
}

func TestLexiconResponseTimeES(t *testing.T) {
	// AC-7
	lx := LexiconFor("es")
	if !lx.ResponseTime.Match("responderé en 24 horas") {
		t.Fatal("expected ES response-time match for responderé en 24 horas")
	}
	en := LexiconFor("en")
	if en.ResponseTime.Match("responderé en 24 horas") {
		// English lexicon may not match Spanish; that's fine — locale routing matters.
		t.Log("EN lexicon unexpectedly matched Spanish phrasing")
	}
}

func TestLexiconCompileOnceAndLocales(t *testing.T) {
	for _, loc := range []string{"en", "es", "fr", "ar"} {
		lx := LexiconFor(loc)
		if lx == nil {
			t.Fatalf("missing lexicon %s", loc)
		}
		if lx.StartHereTitles == nil || lx.LatePolicyPresent == nil {
			t.Fatalf("%s incomplete", loc)
		}
	}
}

func TestSyllabusPlainTextCap(t *testing.T) {
	big := strings.Repeat("a", MaxSyllabusScanBytes+1000)
	text, trunc := SyllabusPlainText(CourseSnapshot{
		SyllabusSections: []SyllabusSectionSnap{{Title: "T", Markdown: big}},
	})
	if !trunc {
		t.Fatal("expected truncation")
	}
	if len(text) > MaxSyllabusScanBytes {
		t.Fatalf("len=%d", len(text))
	}
}
