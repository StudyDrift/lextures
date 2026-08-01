package contentpagegeneration

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseDraftSectionsJSON_Success(t *testing.T) {
	t.Parallel()
	raw := "```json\n{\"sections\":[{\"heading\":\" ## Intro \",\"markdown\":\"  Hello  \"},{\"heading\":\"\",\"markdown\":\"\"},{\"heading\":\"Next\",\"markdown\":\"Body\"}]}\n```"
	got, err := ParseDraftSectionsJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2: %#v", len(got), got)
	}
	if got[0].Heading != "Intro" || got[0].Markdown != "Hello" {
		t.Fatalf("got %#v", got[0])
	}
	if got[1].Heading != "Next" || got[1].Markdown != "Body" {
		t.Fatalf("got %#v", got[1])
	}
}

func TestParseDraftSectionsJSON_Invalid(t *testing.T) {
	t.Parallel()
	if _, err := ParseDraftSectionsJSON("not json"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDraftSectionsJSON_CapsCount(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	b.WriteString(`{"sections":[`)
	for i := 0; i < MaxSections+5; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"heading":"Section `)
		b.WriteString(strings.Repeat("x", i+1))
		b.WriteString(`","markdown":"body"}`)
	}
	b.WriteString(`]}`)
	got, err := ParseDraftSectionsJSON(b.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != MaxSections {
		t.Fatalf("len=%d want %d", len(got), MaxSections)
	}
}

func TestStripJSONFences_EmbeddedObject(t *testing.T) {
	t.Parallel()
	got := stripJSONFences("Here you go:\n{\"sections\":[]}\nThanks")
	if got != `{"sections":[]}` {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeDraftSections_TrimsEmpty(t *testing.T) {
	t.Parallel()
	got := normalizeDraftSections([]DraftSection{
		{Heading: "  ", Markdown: "  "},
		{Heading: "", Markdown: "Only body"},
	})
	if len(got) != 1 || got[0].Markdown != "Only body" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseDraftSectionsJSON_WithTools(t *testing.T) {
	t.Parallel()
	raw := `{
	  "sections": [
	    {"heading":"Intro","markdown":"Hello"},
	    {"heading":"","markdown":"","tools":[{"toolId":"flashcards","config":{"title":"Deck","cards":[{"id":"c1","front":"A","back":"B"}]}}]},
	    {"heading":"End","markdown":"Bye","tools":[{"toolId":"inline_questions","config":{"questions":[{"id":"q1","type":"single","prompt":"Q?","options":[{"id":"a","text":"A","correct":true},{"id":"b","text":"B","correct":false}]}]}}]}
	  ]
	}`
	got, err := ParseDraftSectionsJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d: %#v", len(got), got)
	}
	if len(got[0].Tools) != 0 {
		t.Fatalf("section0 tools: %#v", got[0].Tools)
	}
	if len(got[1].Tools) != 1 || got[1].Tools[0].ToolID != "flashcards" {
		t.Fatalf("section1 tools: %#v", got[1].Tools)
	}
	if len(got[2].Tools) != 1 || got[2].Tools[0].ToolID != "inline_questions" {
		t.Fatalf("section2 tools: %#v", got[2].Tools)
	}
}

func TestNormalizeDraftSectionTools_DropsUnknownAndInvalid(t *testing.T) {
	t.Parallel()
	sections := []DraftSection{
		{
			Heading:  "A",
			Markdown: "body",
			Tools: []DraftTool{
				{ToolID: "diagram_hotspot", Config: json.RawMessage(`{}`)},
				{ToolID: "flashcards", Config: json.RawMessage(`{"title":"T","cards":[{"id":"c1","front":"F","back":"B"}]}`)},
				{ToolID: "inline_questions", Config: json.RawMessage(`{"questions":[]}`)}, // invalid: empty questions
			},
		},
	}
	validate := func(toolID string, config json.RawMessage) error {
		// Simulate schema: reject empty questions array.
		if toolID == "inline_questions" {
			var m map[string]any
			_ = json.Unmarshal(config, &m)
			qs, _ := m["questions"].([]any)
			if len(qs) == 0 {
				return errTestInvalidConfig
			}
		}
		return nil
	}
	got := NormalizeDraftSectionToolsWith(sections, DefaultAIToolIDs, validate)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if len(got[0].Tools) != 1 || got[0].Tools[0].ToolID != "flashcards" {
		t.Fatalf("tools=%#v", got[0].Tools)
	}
}

type testInvalidConfigError struct{}

func (testInvalidConfigError) Error() string { return "invalid config" }

var errTestInvalidConfig = testInvalidConfigError{}

func TestNormalizeDraftSectionTools_CapsPerDraft(t *testing.T) {
	t.Parallel()
	// Always accept configs so we only exercise caps.
	validate := func(string, json.RawMessage) error { return nil }
	var sections []DraftSection
	for i := 0; i < MaxToolsPerDraft+3; i++ {
		sections = append(sections, DraftSection{
			Heading:  "S",
			Markdown: "m",
			Tools: []DraftTool{
				{ToolID: "ask_questions", Config: json.RawMessage(`{"intro":"hi","placeholder":"q"}`)},
			},
		})
	}
	got := NormalizeDraftSectionToolsWith(sections, []string{"ask_questions"}, validate)
	total := 0
	for _, s := range got {
		total += len(s.Tools)
	}
	if total != MaxToolsPerDraft {
		t.Fatalf("total tools=%d want %d", total, MaxToolsPerDraft)
	}
}

func TestStripAllTools(t *testing.T) {
	t.Parallel()
	in := []DraftSection{
		{Heading: "A", Markdown: "b", Tools: []DraftTool{{ToolID: "flashcards", Config: json.RawMessage(`{}`)}}},
	}
	got := StripAllTools(in)
	if len(got) != 1 || len(got[0].Tools) != 0 {
		t.Fatalf("%#v", got)
	}
}

func TestIntersectAllowedToolIDs(t *testing.T) {
	t.Parallel()
	got := IntersectAllowedToolIDs([]string{"flashcards", "diagram_hotspot", "inline_questions"}, []string{"flashcards", "code_sandbox"})
	if len(got) != 1 || got[0] != "flashcards" {
		t.Fatalf("%#v", got)
	}
	// Empty course allowlist means all AI tools allowed.
	all := IntersectAllowedToolIDs(nil, nil)
	if len(all) != len(DefaultAIToolIDs) {
		t.Fatalf("len=%d", len(all))
	}
}

func TestPrepareToolConfig_AssignsIDs(t *testing.T) {
	t.Parallel()
	cfg, ok := prepareToolConfig("flashcards", json.RawMessage(`{"title":"Deck","cards":[{"front":"F","back":"B"}]}`))
	if !ok {
		t.Fatal("prepare failed")
	}
	var m map[string]any
	if err := json.Unmarshal(cfg, &m); err != nil {
		t.Fatal(err)
	}
	cards, _ := m["cards"].([]any)
	if len(cards) != 1 {
		t.Fatalf("cards=%#v", cards)
	}
	c0 := cards[0].(map[string]any)
	if strings.TrimSpace(c0["id"].(string)) == "" {
		t.Fatal("expected card id")
	}
}
