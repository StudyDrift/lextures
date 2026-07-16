package boardfilter

import (
	"strings"
	"testing"
)

func TestMatch_BasicAndEvasion(t *testing.T) {
	cases := []struct {
		text     string
		want     bool
		wantTerm string
	}{
		{"hello world", false, ""},
		{"what the fuck", true, "fuck"},
		{"f.u.c.k this", true, "fuck"},
		{"f u c k off", true, "fuck"},
		{"sh1t happens", true, "shit"},
		{"classy assignment", false, ""},
	}
	for _, tc := range cases {
		got := Match(tc.text, DefaultEnglish)
		if got.Matched != tc.want {
			t.Errorf("Match(%q) matched=%v want %v (term=%q)", tc.text, got.Matched, tc.want, got.Term)
		}
		if tc.want && got.Term != tc.wantTerm {
			t.Errorf("Match(%q) term=%q want %q", tc.text, got.Term, tc.wantTerm)
		}
	}
}

func TestExtractPlainText(t *testing.T) {
	body := []byte(`{"html":"<p>Hello <strong>world</strong></p>","text":"Hello world"}`)
	got := ExtractPlainText("Title", body)
	if !strings.Contains(got, "Title") || !strings.Contains(got, "Hello world") {
		t.Fatalf("ExtractPlainText got %q", got)
	}
}
