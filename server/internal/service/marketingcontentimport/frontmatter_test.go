package marketingcontentimport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseFrontmatterArraysAndPipeCitations(t *testing.T) {
	fm, err := parseFrontmatter("post.md", "---\ntitle: 'Hello'\nroles: [instructor, admin]\ncitations: https://a.example|https://b.example\n---\n\nBody\n")
	if err != nil {
		t.Fatal(err)
	}
	if fm.Values["title"] != "Hello" || strings.Join(fm.Lists["roles"], ",") != "instructor,admin" || len(fm.Lists["citations"]) != 2 || fm.Body != "Body" {
		t.Fatalf("unexpected parse: %#v", fm)
	}
}

func TestParseFrontmatterRejectsUnknownKey(t *testing.T) {
	_, err := parseFrontmatter("bad.md", "---\ntitle: ok\nfoo: dropped\n---\nbody")
	if err == nil || !strings.Contains(err.Error(), `bad.md:3: unknown front-matter key "foo"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSeedCorpusAndTaxonomy(t *testing.T) {
	// File-based www/src/{blog,docs} corpus was archived in MC.15; the deployable
	// copy lives in the checked-in seed migration and is the source of truth now.
	seedPath := filepath.Clean(filepath.Join("..", "..", "..", "migrations", "485_marketing_content_seed.sql"))
	raw, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatal(err)
	}
	const marker = "$mc_seed$"
	text := string(raw)
	start := strings.Index(text, marker)
	if start < 0 {
		t.Fatalf("missing %s marker in %s", marker, seedPath)
	}
	start += len(marker)
	end := strings.Index(text[start:], marker)
	if end < 0 {
		t.Fatalf("missing closing %s marker in %s", marker, seedPath)
	}
	var payload struct {
		Authors    []struct {
			Slug       string   `json:"slug"`
			KnowsAbout []string `json:"knowsAbout"`
		} `json:"authors"`
		Categories []json.RawMessage `json:"categories"`
		Articles   []json.RawMessage `json:"articles"`
	}
	if err := json.Unmarshal([]byte(text[start:start+end]), &payload); err != nil {
		t.Fatalf("seed JSON: %v", err)
	}
	if len(payload.Articles) != 70 {
		t.Fatalf("got %d articles, want 70", len(payload.Articles))
	}
	if len(payload.Categories) != 16 {
		t.Fatalf("got %d categories", len(payload.Categories))
	}
	if len(payload.Authors) != 1 || payload.Authors[0].Slug != "chase-willden" || len(payload.Authors[0].KnowsAbout) != 5 {
		t.Fatalf("unexpected authors: %#v", payload.Authors)
	}
}

func TestResolveLastmodUsesGitThenPublished(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "article.md")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	published, _ := date("2020-01-02")
	got, source, err := resolveLastmod(file, "", published, "", true)
	if err != nil || source != "published" || !got.Equal(*published) {
		t.Fatalf("got %v %s %v", got, source, err)
	}
	updated, source, err := resolveLastmod(file, "2024-03-04", published, "", true)
	if err != nil || source != "frontmatter" || updated.Format(time.DateOnly) != "2024-03-04" {
		t.Fatalf("got %v %s %v", updated, source, err)
	}
}
