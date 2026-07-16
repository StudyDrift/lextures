package board

import (
	"encoding/json"
	"testing"
)

func TestParseDefinition_Valid(t *testing.T) {
	raw := json.RawMessage(`{
		"layout": "columns",
		"reactionMode": "like",
		"sections": [{"key": "a", "title": "A", "sortIndex": 0}],
		"seedPosts": [{"key": "p1", "contentType": "text", "title": "Hi", "body": {"text": "x"}, "sectionKey": "a"}]
	}`)
	def, err := ParseDefinition(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if def.Layout != LayoutColumns {
		t.Fatalf("layout: %s", def.Layout)
	}
	if len(def.Sections) != 1 || def.Sections[0].Key != "a" {
		t.Fatalf("sections: %+v", def.Sections)
	}
	if len(def.SeedPosts) != 1 || def.SeedPosts[0].SectionKey != "a" {
		t.Fatalf("posts: %+v", def.SeedPosts)
	}
}

func TestParseDefinition_UnknownSectionKey(t *testing.T) {
	raw := json.RawMessage(`{
		"layout": "wall",
		"sections": [],
		"seedPosts": [{"contentType": "text", "title": "Hi", "sectionKey": "missing"}]
	}`)
	if _, err := ParseDefinition(raw); err == nil {
		t.Fatal("expected error for unknown sectionKey")
	}
}

func TestBoardToDefinition_StructureOnlyOmitsPosts(t *testing.T) {
	canPost := true
	b := Board{
		Layout: LayoutWall, ReactionMode: ReactionModeLike,
		Attribution: AttributionNamed, ModerationMode: ModerationOpen,
		FilterAction: FilterFlag, CanPost: canPost, CanInteract: true, CanArrange: false,
		Settings: json.RawMessage(`{}`),
	}
	secs := []Section{{ID: "s1", Title: "One", SortIndex: 0}}
	posts := []Post{{ID: "p1", ContentType: ContentTypeText, Title: "Student", Status: PostStatusApproved}}
	def := BoardToDefinition(b, secs, posts, false)
	if len(def.SeedPosts) != 0 {
		t.Fatalf("structure-only should omit posts, got %d", len(def.SeedPosts))
	}
	if len(def.Sections) != 1 || def.Sections[0].Key != "s1" {
		t.Fatalf("sections: %+v", def.Sections)
	}
}

func TestBoardToDefinition_FullSkipsRemovedAndPending(t *testing.T) {
	b := Board{Layout: LayoutWall, Settings: json.RawMessage(`{}`)}
	posts := []Post{
		{ID: "1", ContentType: ContentTypeText, Title: "ok", Status: PostStatusApproved},
		{ID: "2", ContentType: ContentTypeText, Title: "gone", Status: PostStatusApproved, Removed: true},
		{ID: "3", ContentType: ContentTypeText, Title: "pend", Status: PostStatusPending},
	}
	def := BoardToDefinition(b, nil, posts, true)
	if len(def.SeedPosts) != 1 || def.SeedPosts[0].Title != "ok" {
		t.Fatalf("posts: %+v", def.SeedPosts)
	}
}

func TestApplyBuiltinLocale_ExitTicketES(t *testing.T) {
	tmpl := &Template{
		ID:    BuiltinExitTicketID,
		Scope: TemplateScopeBuiltin,
		Title: "Exit ticket",
		Definition: json.RawMessage(`{
			"layout": "stream",
			"sections": [],
			"seedPosts": [{"key": "prompt", "contentType": "text", "title": "Exit ticket", "body": {"text": "EN"}}]
		}`),
	}
	ApplyBuiltinLocale(tmpl, "es-MX")
	if tmpl.Title != "Ticket de salida" {
		t.Fatalf("title: %s", tmpl.Title)
	}
	def, err := ParseDefinition(tmpl.Definition)
	if err != nil {
		t.Fatal(err)
	}
	if def.SeedPosts[0].Title != "Ticket de salida" {
		t.Fatalf("prompt title: %s", def.SeedPosts[0].Title)
	}
	var body map[string]string
	_ = json.Unmarshal(def.SeedPosts[0].Body, &body)
	if body["text"] == "" || body["text"] == "EN" {
		t.Fatalf("expected localized body, got %v", body)
	}
}

func TestNormalizeCopyMode(t *testing.T) {
	m, err := NormalizeCopyMode("")
	if err != nil || m != CopyModeStructure {
		t.Fatalf("default: %s %v", m, err)
	}
	m, err = NormalizeCopyMode("full")
	if err != nil || m != CopyModeFull {
		t.Fatalf("full: %s %v", m, err)
	}
	if _, err := NormalizeCopyMode("partial"); err == nil {
		t.Fatal("expected error")
	}
}
