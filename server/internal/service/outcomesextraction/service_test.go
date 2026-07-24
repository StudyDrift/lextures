package outcomesextraction

import (
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/repos/course"
)

func TestSyllabusPromptMaterial(t *testing.T) {
	t.Parallel()
	got := SyllabusPromptMaterial([]course.SyllabusSection{
		{Heading: "  ", Markdown: "  "},
		{Heading: "Goals", Markdown: "Students will analyze data."},
		{Heading: "", Markdown: "Extra note."},
	})
	want := "## Goals\n\nStudents will analyze data.\n\nExtra note."
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if SyllabusPromptMaterial(nil) != "" {
		t.Fatal("expected empty for nil sections")
	}
}

func TestParseDraftOutcomesJSON_Success(t *testing.T) {
	t.Parallel()
	raw := "```json\n{\"outcomes\":[{\"title\":\" Analyze X \",\"description\":\"  detail  \"},{\"title\":\"\",\"description\":\"skip\"},{\"title\":\"Analyze X\",\"description\":\"dup\"}]}\n```"
	got, err := ParseDraftOutcomesJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1: %#v", len(got), got)
	}
	if got[0].Title != "Analyze X" || got[0].Description != "detail" {
		t.Fatalf("got %#v", got[0])
	}
}

func TestParseDraftOutcomesJSON_Invalid(t *testing.T) {
	t.Parallel()
	if _, err := ParseDraftOutcomesJSON("not json"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDraftOutcomesJSON_CapsCount(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	b.WriteString(`{"outcomes":[`)
	for i := 0; i < MaxOutcomes+5; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"title":"Outcome `)
		b.WriteString(strings.Repeat("x", i+1))
		b.WriteString(`","description":""}`)
	}
	b.WriteString(`]}`)
	got, err := ParseDraftOutcomesJSON(b.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != MaxOutcomes {
		t.Fatalf("len=%d want %d", len(got), MaxOutcomes)
	}
}

func TestStripJSONFences_EmbeddedObject(t *testing.T) {
	t.Parallel()
	got := stripJSONFences("Here you go:\n{\"outcomes\":[]}\nThanks")
	if got != `{"outcomes":[]}` {
		t.Fatalf("got %q", got)
	}
}
