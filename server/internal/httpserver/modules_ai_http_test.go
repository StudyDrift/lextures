package httpserver

import (
	"testing"

	coursestructurerepo "github.com/lextures/lextures/server/internal/repos/coursestructure"
)

func TestParseModulesAIChatResponse(t *testing.T) {
	raw := "```json\n{\"reply\":\"Added a module\",\"proposals\":[{\"op\":\"create_module\",\"title\":\"Week 2\"}]}\n```"
	got, err := parseModulesAIChatResponse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Reply != "Added a module" {
		t.Fatalf("reply=%q", got.Reply)
	}
	if len(got.Proposals) != 1 || got.Proposals[0].Op != "create_module" || got.Proposals[0].Title != "Week 2" {
		t.Fatalf("proposals=%+v", got.Proposals)
	}
}

func TestSanitizeModulesAIProposals(t *testing.T) {
	modID := "11111111-1111-4111-8111-111111111111"
	itemID := "22222222-2222-4222-8222-222222222222"
	items := []coursestructurerepo.ItemResponse{
		{ID: modID, Kind: "module", Title: "M1"},
		{ID: itemID, Kind: "content_page", Title: "Page", ParentID: &modID},
	}
	pub := true
	in := []modulesAIProposal{
		{Op: "create_quiz", ModuleTitle: "Week 3", Title: "Week 3 Quiz"},
		{Op: "create_module", Title: "Week 3"},
		{Op: "rename", ItemID: itemID, Title: "Renamed"},
		{Op: "set_published", ItemID: "missing", Published: &pub},
		{Op: "create_quiz", ModuleID: modID, Title: "Quiz 1"},
		{Op: "create_quiz", ModuleID: "missing", Title: "Bad"},
		{Op: "create_assignment", ModuleTitle: "Unknown Module", Title: "No"},
		{Op: "delete", ItemID: itemID},
	}
	out := sanitizeModulesAIProposals(in, items)
	if len(out) != 4 {
		t.Fatalf("want 4 proposals, got %+v", out)
	}
	if out[0].Op != "create_module" || out[0].Title != "Week 3" {
		t.Fatalf("expected create_module first, got %+v", out[0])
	}
	foundTitleRef := false
	for _, p := range out {
		if p.Op == "create_quiz" && p.ModuleTitle == "Week 3" && p.Title == "Week 3 Quiz" {
			foundTitleRef = true
		}
	}
	if !foundTitleRef {
		t.Fatalf("expected moduleTitle child create, got %+v", out)
	}
}
