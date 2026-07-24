package contentpagegeneration

import (
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
