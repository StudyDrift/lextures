package context

import (
	"encoding/json"
	"testing"
)

func TestDiscoverLinks_markdownAndConfig(t *testing.T) {
	md := `# Lesson
Read [Standards](https://standards.example/a) and https://news.example/b.
Also <https://lab.example/protocol>.
`
	cfg, _ := json.Marshal(map[string]any{
		"links": []string{"https://config.example/extra"},
	})
	links := DiscoverLinks(md, cfg, nil)
	if len(links) < 4 {
		t.Fatalf("expected >=4 links, got %v", links)
	}
	want := map[string]bool{
		"https://standards.example/a":   false,
		"https://news.example/b":        false,
		"https://lab.example/protocol":  false,
		"https://config.example/extra":  false,
	}
	for _, l := range links {
		if _, ok := want[l]; ok {
			want[l] = true
		}
	}
	for u, ok := range want {
		if !ok {
			t.Fatalf("missing %s in %v", u, links)
		}
	}
}

func TestSectionBodyForInstance(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	md := `# Intro
hello

# Main
See https://example.com/x
` + "```lex-tool\n" + `{"id": "` + id + `", "toolId": "noop_probe"}
` + "```\n" + `
# Outro
bye
`
	sec := SectionBodyForInstance(md, id)
	if sec == "" || sec == md {
		t.Fatalf("expected section slice, got %q", sec)
	}
	if !contains(sec, "example.com") {
		t.Fatalf("section should include link context: %q", sec)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (stringIndex(s, sub) >= 0)))
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
