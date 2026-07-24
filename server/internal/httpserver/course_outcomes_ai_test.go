package httpserver

import (
	"testing"

	"github.com/lextures/lextures/server/internal/service/outcomesextraction"
)

func TestOutcomesExtractionParse_EmptyArray(t *testing.T) {
	t.Parallel()
	got, err := outcomesextraction.ParseDraftOutcomesJSON(`{"outcomes":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestOutcomesExtractionParse_ValidDrafts(t *testing.T) {
	t.Parallel()
	got, err := outcomesextraction.ParseDraftOutcomesJSON(`{"outcomes":[{"title":"Explain photosynthesis","description":"Describe the light-dependent reactions."}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "Explain photosynthesis" {
		t.Fatalf("got %#v", got)
	}
}
