package httpserver

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/search"
	"github.com/lextures/lextures/server/internal/service/managedlearners"
)

func TestManagedLearnerQueryItem(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	grade := "4"
	item := managedLearnerQueryItem(managedlearners.SearchHit{
		UserID:      id,
		DisplayName: "QA Hunt Kid",
		GradeLevel:  &grade,
		Score:       0.95,
	})
	if item.Title != "QA Hunt Kid" || item.Type != "person" {
		t.Fatalf("item: %+v", item)
	}
	if item.Subtitle != "Learner · Grade 4" {
		t.Fatalf("subtitle: %q", item.Subtitle)
	}
	if item.Path != "/learners?learner="+id.String() {
		t.Fatalf("path: %q", item.Path)
	}
	if item.ID != "person:"+id.String()+":learner" {
		t.Fatalf("id: %q", item.ID)
	}
}

func TestMergeSearchPersonResults_PrefersNewLearnerAndDedupesRoster(t *testing.T) {
	learnerID := "22222222-2222-2222-2222-222222222222"
	roster := []search.QueryResultItem{{
		ID:    "person:" + learnerID + ":BIO101:Student",
		Type:  "person",
		Title: "QA Hunt Kid",
		Path:  "/courses/BIO101/enrollments",
		Score: 0.4,
	}}
	extra := []search.QueryResultItem{{
		ID:       "person:" + learnerID + ":learner",
		Type:     "person",
		Title:    "QA Hunt Kid",
		Subtitle: "Learner",
		Path:     "/learners?learner=" + learnerID,
		Score:    0.95,
	}, {
		ID:       "person:33333333-3333-3333-3333-333333333333:learner",
		Type:     "person",
		Title:    "Other Kid",
		Subtitle: "Learner",
		Path:     "/learners?learner=33333333-3333-3333-3333-333333333333",
		Score:    0.95,
	}}
	got, total := mergeSearchPersonResults(roster, 1, extra, 5)
	if total != 2 {
		t.Fatalf("total %d", total)
	}
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Title != "Other Kid" {
		t.Fatalf("first: %+v", got[0])
	}
	if got[1].Path != "/courses/BIO101/enrollments" {
		t.Fatalf("kept roster path: %+v", got[1])
	}
}
