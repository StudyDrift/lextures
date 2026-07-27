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

func TestMergeAnnotationsByID(t *testing.T) {
	server := []Annotation{
		{ID: "a", TagID: "t1", Quote: "s", CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: "b", TagID: "t1", Quote: "s2", CreatedAt: "2026-01-01T00:00:01Z"},
	}
	client := []Annotation{
		{ID: "b", TagID: "t2", Quote: "edited", CreatedAt: "2026-01-01T00:00:02Z"},
		{ID: "c", TagID: "t1", Quote: "new", CreatedAt: "2026-01-01T00:00:03Z"},
	}
	merged := MergeAnnotationsByID(client, server)
	if len(merged) != 3 {
		t.Fatalf("len=%d", len(merged))
	}
	byID := map[string]Annotation{}
	for _, a := range merged {
		byID[a.ID] = a
	}
	if byID["b"].Quote != "edited" {
		t.Fatalf("b quote=%s", byID["b"].Quote)
	}
	if byID["c"].Quote != "new" {
		t.Fatalf("missing c")
	}
	if byID["a"].Quote != "s" {
		t.Fatalf("missing a")
	}
}

func TestCapAnnotations(t *testing.T) {
	st := EmptyState()
	for i := 0; i < 5; i++ {
		st.Annotations = append(st.Annotations, Annotation{ID: string(rune('a' + i))})
	}
	st = CapAnnotations(st, 3)
	if len(st.Annotations) != 3 {
		t.Fatalf("len=%d", len(st.Annotations))
	}
}
