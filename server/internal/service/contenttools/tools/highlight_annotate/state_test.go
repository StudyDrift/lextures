package highlight_annotate

import (
	"encoding/json"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	cfg := ParseConfig(nil)
	if cfg.PassageSource != PassageInline {
		t.Fatalf("passageSource=%s", cfg.PassageSource)
	}
	if cfg.UnitGranularity != UnitSentence {
		t.Fatalf("granularity=%s", cfg.UnitGranularity)
	}
	if cfg.MinAnnotations != 1 || cfg.MaxAnnotations != 20 {
		t.Fatalf("min/max=%d/%d", cfg.MinAnnotations, cfg.MaxAnnotations)
	}
}

func TestParseConfigClamp(t *testing.T) {
	raw := json.RawMessage(`{"prompt":"p","tags":[{"id":"t","label":"T","color":"#000"}],"minAnnotations":0,"maxAnnotations":99}`)
	cfg := ParseConfig(raw)
	if cfg.MinAnnotations != 1 {
		t.Fatalf("min=%d", cfg.MinAnnotations)
	}
	if cfg.MaxAnnotations != 50 {
		t.Fatalf("max=%d", cfg.MaxAnnotations)
	}
}

func TestDeriveStatus(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MinAnnotations = 3
	st := EmptyState()
	if got := DeriveStatus(cfg, st, "not_started"); got != "not_started" {
		t.Fatalf("empty: %s", got)
	}
	st.Annotations = []Annotation{
		{ID: "a1", TagID: "t", Quote: "q", CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: "a2", TagID: "t", Quote: "q2", CreatedAt: "2026-01-01T00:00:01Z"},
	}
	if got := DeriveStatus(cfg, st, "in_progress"); got != "in_progress" {
		t.Fatalf("partial: %s", got)
	}
	st.Annotations = append(st.Annotations, Annotation{ID: "a3", TagID: "t", Quote: "q3", CreatedAt: "2026-01-01T00:00:02Z"})
	if got := DeriveStatus(cfg, st, "in_progress"); got != "completed" {
		t.Fatalf("complete: %s", got)
	}
}

func TestCapAnnotations(t *testing.T) {
	st := EmptyState()
	for i := 0; i < 5; i++ {
		st.Annotations = append(st.Annotations, Annotation{ID: string(rune('a' + i)), TagID: "t"})
	}
	st = CapAnnotations(st, 3)
	if len(st.Annotations) != 3 {
		t.Fatalf("len=%d", len(st.Annotations))
	}
}

func TestDropUnknownTags(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tags = []Tag{{ID: "claim", Label: "Claim", Color: "#000"}}
	st := EmptyState()
	st.Annotations = []Annotation{
		{ID: "a", TagID: "claim", Quote: "q"},
		{ID: "b", TagID: "gone", Quote: "q2"},
	}
	st = DropUnknownTags(cfg, st)
	if len(st.Annotations) != 1 || st.Annotations[0].ID != "a" {
		t.Fatalf("got %#v", st.Annotations)
	}
}
