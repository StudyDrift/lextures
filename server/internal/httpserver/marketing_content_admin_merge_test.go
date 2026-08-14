package httpserver

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

func TestMergeMarketingArticleAppliesSearchMetadata(t *testing.T) {
	t.Parallel()
	actor := uuid.New()
	old := &mcrepo.Article{
		Kind: "blog", Slug: "advice", Title: "Advice", AuthorSlug: "chase",
		PrimaryQuestion: "", Cluster: "", Pillar: "", VerifiedAgainst: "",
	}
	rawJSON := []byte(`{
		"title":"Advice",
		"cluster":"learning",
		"primaryQuestion":"How do I start?",
		"pillar":"home-education",
		"verifiedAgainst":"2026.8",
		"expectedRevisionNo":3
	}`)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		t.Fatal(err)
	}
	var body marketingArticleBody
	if err := json.Unmarshal(rawJSON, &body); err != nil {
		t.Fatal(err)
	}

	got := mergeMarketingArticle(old, body, raw, actor)
	if got.Cluster != "learning" {
		t.Fatalf("cluster: got %q", got.Cluster)
	}
	if got.PrimaryQuestion != "How do I start?" {
		t.Fatalf("primaryQuestion: got %q", got.PrimaryQuestion)
	}
	if got.Pillar != "home-education" {
		t.Fatalf("pillar: got %q", got.Pillar)
	}
	if got.VerifiedAgainst != "2026.8" {
		t.Fatalf("verifiedAgainst: got %q", got.VerifiedAgainst)
	}
	if got.Title != "Advice" {
		t.Fatalf("title: got %q", got.Title)
	}
}

func TestMergeMarketingArticleLeavesOmittedSearchMetadata(t *testing.T) {
	t.Parallel()
	old := &mcrepo.Article{Title: "Advice", Cluster: "learning", PrimaryQuestion: "How?", Pillar: "home", VerifiedAgainst: "1.0"}
	rawJSON := []byte(`{"title":"Advice revised","expectedRevisionNo":2}`)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		t.Fatal(err)
	}
	var body marketingArticleBody
	if err := json.Unmarshal(rawJSON, &body); err != nil {
		t.Fatal(err)
	}

	got := mergeMarketingArticle(old, body, raw, uuid.New())
	if got.Cluster != "learning" || got.PrimaryQuestion != "How?" || got.Pillar != "home" || got.VerifiedAgainst != "1.0" {
		t.Fatalf("omitted fields were cleared: %+v", got)
	}
	if got.Title != "Advice revised" {
		t.Fatalf("title: got %q", got.Title)
	}
}
