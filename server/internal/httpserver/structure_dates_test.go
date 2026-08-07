package httpserver

import (
	"testing"
	"time"

	coursestructurerepo "github.com/lextures/lextures/server/internal/repos/coursestructure"
)

func TestParseAdjustDatesAIResponse(t *testing.T) {
	raw := "```json\n{\"reply\":\"Shifted a week\",\"proposals\":[{\"itemId\":\"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee\",\"dueAt\":\"2026-09-15T23:59:00Z\"}]}\n```"
	parsed, err := parseAdjustDatesAIResponse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Reply != "Shifted a week" {
		t.Fatalf("reply: %q", parsed.Reply)
	}
	if len(parsed.Proposals) != 1 || parsed.Proposals[0].ItemID == "" {
		t.Fatalf("proposals: %+v", parsed.Proposals)
	}
}

func TestSanitizeAdjustDatesAIProposals(t *testing.T) {
	due := time.Date(2026, 1, 10, 23, 59, 0, 0, time.UTC)
	dated := []coursestructurerepo.ItemResponse{
		{ID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Kind: "assignment", Title: "Essay", DueAt: &due},
	}
	in := []adjustDatesAIProposal{
		{ItemID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", DueAt: "2026-01-17T23:59:00Z"},
		{ItemID: "missing", DueAt: "2026-01-18T23:59:00Z"},
		{ItemID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", DueAt: "2026-01-10T23:59:00Z"}, // same as original, skip
	}
	out := sanitizeAdjustDatesAIProposals(in, dated)
	if len(out) != 1 {
		t.Fatalf("expected 1 proposal, got %+v", out)
	}
	if out[0].DueAt != "2026-01-17T23:59:00Z" {
		t.Fatalf("dueAt: %s", out[0].DueAt)
	}
}
