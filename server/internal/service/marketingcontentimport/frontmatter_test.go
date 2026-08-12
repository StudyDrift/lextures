package marketingcontentimport

import (
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

func TestRealCorpusAndTaxonomy(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", "www", "src"))
	articles, err := loadArticles(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 70 {
		t.Fatalf("got %d articles, want 70", len(articles))
	}
	categories, err := loadCategories(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(categories) != 16 {
		t.Fatalf("got %d categories", len(categories))
	}
	authors, err := loadAuthors(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(authors) != 1 || authors[0].Slug != "chase-willden" || len(authors[0].KnowsAbout) != 5 {
		t.Fatalf("unexpected authors: %#v", authors)
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
