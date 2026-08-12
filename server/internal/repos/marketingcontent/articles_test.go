package marketingcontent

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestArticlePathBlog(t *testing.T) {
	got, err := articlePath(context.Background(), nil, "blog", "en", "release-notes", nil)
	if err != nil || got != "/blog/release-notes" {
		t.Fatalf("articlePath() = %q, %v", got, err)
	}
}

func TestArticlePathDocRequiresCategory(t *testing.T) {
	_, err := articlePath(context.Background(), nil, "doc", "en", "getting-started", nil)
	if err == nil {
		t.Fatal("expected doc without category to fail")
	}
}

func TestArticlePathPrefixesNonDefaultLocale(t *testing.T) {
	got, err := articlePath(context.Background(), nil, "blog", "es", "hola", nil)
	if err != nil || got != "/es/blog/hola" {
		t.Fatalf("articlePath() = %q, %v", got, err)
	}
}

func TestMapConstraintDuplicateArticlePath(t *testing.T) {
	err := mapConstraint(&pgconn.PgError{Code: "23505", ConstraintName: "idx_mc_articles_path_live"})
	if !errors.Is(err, ErrDuplicateSlug) {
		t.Fatalf("mapConstraint() = %v, want ErrDuplicateSlug", err)
	}
}
