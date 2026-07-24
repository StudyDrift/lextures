package httpserver

import (
	"testing"

	"github.com/lextures/lextures/server/internal/service/badgesextraction"
)

func TestBadgesExtractionParse_Empty(t *testing.T) {
	t.Parallel()
	got, err := badgesextraction.ParseDraftBadgesJSON(`{"badges":[]}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestBadgesExtractionParse_WithOutcomeID(t *testing.T) {
	t.Parallel()
	valid := map[string]struct{}{"abc": {}}
	got, err := badgesextraction.ParseDraftBadgesJSON(
		`{"badges":[{"outcomeId":"abc","name":"Inquiry","description":"Demonstrates inquiry skills."}]}`,
		valid,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].OutcomeID == nil || *got[0].OutcomeID != "abc" {
		t.Fatalf("got %#v", got)
	}
}
